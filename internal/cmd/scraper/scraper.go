// Package scraper ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package scraper

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/newrelic/nri-prometheus/internal/integration"
	"github.com/newrelic/nri-prometheus/internal/pkg/endpoints"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Config is the config struct for the scraper.
type Config struct {
	ConfigFile                        string
	MetricAPIURL                      string                       `mapstructure:"metric_api_url"`
	LicenseKey                        string                       `mapstructure:"license_key"`
	ClusterName                       string                       `mapstructure:"cluster_name"`
	Debug                             bool                         `mapstructure:"debug"`
	Verbose                           bool                         `mapstructure:"verbose"`
	Emitters                          []string                     `mapstructure:"emitters"`
	ScrapeEnabledLabel                string                       `mapstructure:"scrape_enabled_label"`
	RequireScrapeEnabledLabelForNodes bool                         `mapstructure:"require_scrape_enabled_label_for_nodes"`
	ScrapeTimeout                     time.Duration                `mapstructure:"scrape_timeout"`
	ScrapeDuration                    string                       `mapstructure:"scrape_duration"`
	EmitterHarvestPeriod              string                       `mapstructure:"emitter_harvest_period"`
	TargetConfigs                     []endpoints.TargetConfig     `mapstructure:"targets"`
	AutoDecorate                      bool                         `mapstructure:"auto_decorate" default:"false"`
	CaFile                            string                       `mapstructure:"ca_file"`
	BearerTokenFile                   string                       `mapstructure:"bearer_token_file"`
	InsecureSkipVerify                bool                         `mapstructure:"insecure_skip_verify" default:"false"`
	ProcessingRules                   []integration.ProcessingRule `mapstructure:"transformations"`
	DecorateFile                      bool
}

// Number of /metrics targets that can be fetched in parallel
const maxTargetConnections = 4

// channel length for entities
const queueLength = 100

func validateConfig(cfg *Config) error {
	requiredMsg := "%s is required and can't be empty"
	if cfg.ClusterName == "" {
		return fmt.Errorf(requiredMsg, "cluster_name")
	}
	if cfg.LicenseKey == "" {
		return fmt.Errorf(requiredMsg, "license_key")
	}

	return nil
}

// RunWithEmitters runs the scraper with preselected emitters.
func RunWithEmitters(cfg *Config, emitters []integration.Emitter) {
	logrus.Infof("Starting New Relic's Prometheus OpenMetrics Integration version %s", integration.Version)
	if cfg.Verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.Debugf("Config: %#v", cfg)

	if len(emitters) == 0 {
		logrus.Fatal("you need to configure at least one valid emitter.")
	}

	err := validateConfig(cfg)
	if err != nil { // Handle errors validating the config file
		logrus.WithError(err).Fatal("while validating configuration options")
	}

	selfRetriever, err := endpoints.SelfRetriever()
	if err != nil {
		logrus.WithError(err).Fatal("while parsing provided endpoints")
	}
	var retrievers []endpoints.TargetRetriever
	fixedRetriever, err := endpoints.FixedRetriever(cfg.TargetConfigs...)
	if err != nil {
		logrus.WithError(err).Fatal("while parsing provided endpoints")
	}
	retrievers = append(retrievers, fixedRetriever)

	kubernetesRetriever, err := endpoints.NewKubernetesTargetRetriever(cfg.ScrapeEnabledLabel, cfg.RequireScrapeEnabledLabelForNodes)
	if err != nil {
		logrus.WithError(err).Errorf("not possible to get a Kubernetes client. If you aren't running this integration in a Kubernetes cluster, you can ignore this error")
	} else {
		retrievers = append(retrievers, kubernetesRetriever)
	}
	defaultTransformations := integration.ProcessingRule{
		Description: "Default transformation rules",
		AddAttributes: []integration.AddAttributesRule{
			{
				MetricPrefix: "",
				Attributes: map[string]interface{}{
					"k8s.cluster.name":   cfg.ClusterName,
					"clusterName":        cfg.ClusterName,
					"integrationVersion": integration.Version,
					"integrationName":    integration.Name,
				},
			},
		},
	}
	processingRules := append(cfg.ProcessingRules, defaultTransformations)

	scrapeDuration, err := time.ParseDuration(cfg.ScrapeDuration)
	if err != nil {
		log.Fatalf("parsing scrape_duration value (%v): %v", cfg.ScrapeDuration, err.Error())
	}

	go integration.Execute(
		scrapeDuration,
		selfRetriever,
		retrievers,
		integration.NewFetcher(scrapeDuration, cfg.ScrapeTimeout, maxTargetConnections, cfg.BearerTokenFile, cfg.CaFile, cfg.InsecureSkipVerify, queueLength),
		integration.RuleProcessor(processingRules, queueLength),
		emitters)

	r := http.NewServeMux()
	r.Handle("/metrics", promhttp.Handler())
	if cfg.Debug {
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	log.Fatal(http.ListenAndServe(":8080", r))
}

// Run runs the scraper
func Run(cfg *Config) {
	err := validateConfig(cfg)
	if err != nil { // Handle errors reading the config file
		logrus.WithError(err).Fatal("while getting configuration options")
	}

	var emitters []integration.Emitter
	for _, e := range cfg.Emitters {
		switch e {
		case "stdout":
			emitters = append(emitters, integration.NewStdoutEmitter())
		case "telemetry":
			h, err := time.ParseDuration(cfg.EmitterHarvestPeriod)
			if err != nil {
				logrus.Fatalf("invalid telemetry emitter harvest period: %s", cfg.EmitterHarvestPeriod)
			}

			logrus.Debugf("telemetry emitter configured with API endpoint %s, harvest period of %s", cfg.MetricAPIURL, cfg.EmitterHarvestPeriod)
			emitters = append(emitters, integration.NewTelemetryEmitter(cfg.MetricAPIURL, cfg.LicenseKey, h))
		default:
			logrus.Debugf("unknown emitter: %s", e)
			continue
		}
	}

	RunWithEmitters(cfg, emitters)
}
