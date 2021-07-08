// Package scraper ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package scraper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/newrelic/nri-prometheus/internal/integration"
	"github.com/newrelic/nri-prometheus/internal/pkg/endpoints"
	"github.com/newrelic/nri-prometheus/internal/synthesis"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Config is the config struct for the scraper.
type Config struct {
	MetricAPIURL                      string                       `mapstructure:"metric_api_url"`
	LicenseKey                        LicenseKey                   `mapstructure:"license_key"`
	ClusterName                       string                       `mapstructure:"cluster_name"`
	Debug                             bool                         `mapstructure:"debug"`
	Verbose                           bool                         `mapstructure:"verbose"`
	Audit                             bool                         `mapstructure:"audit"`
	Emitters                          []string                     `mapstructure:"emitters"`
	ScrapeEnabledLabel                string                       `mapstructure:"scrape_enabled_label"`
	RequireScrapeEnabledLabelForNodes bool                         `mapstructure:"require_scrape_enabled_label_for_nodes"`
	ScrapeTimeout                     time.Duration                `mapstructure:"scrape_timeout"`
	Standalone                        bool                         `mapstructure:"standalone"`
	DisableAutodiscovery              bool                         `mapstructure:"disable_autodiscovery"`
	ScrapeServices                    bool                         `mapstructure:"scrape_services"`
	ScrapeEndpoints                   bool                         `mapstructure:"scrape_endpoints"`
	ScrapeDuration                    string                       `mapstructure:"scrape_duration"`
	EmitterHarvestPeriod              string                       `mapstructure:"emitter_harvest_period"`
	MinEmitterHarvestPeriod           string                       `mapstructure:"min_emitter_harvest_period"`
	MaxStoredMetrics                  int                          `mapstructure:"max_stored_metrics"`
	TargetConfigs                     []endpoints.TargetConfig     `mapstructure:"targets"`
	AutoDecorate                      bool                         `mapstructure:"auto_decorate" default:"false"`
	CaFile                            string                       `mapstructure:"ca_file"`
	BearerTokenFile                   string                       `mapstructure:"bearer_token_file"`
	InsecureSkipVerify                bool                         `mapstructure:"insecure_skip_verify" default:"false"`
	ProcessingRules                   []integration.ProcessingRule `mapstructure:"transformations"`
	DecorateFile                      bool
	EmitterProxy                      string `mapstructure:"emitter_proxy"`
	// Parsed version of `EmitterProxy`
	EmitterProxyURL                              *url.URL
	EmitterCAFile                                string                 `mapstructure:"emitter_ca_file"`
	EmitterInsecureSkipVerify                    bool                   `mapstructure:"emitter_insecure_skip_verify" default:"false"`
	TelemetryEmitterDeltaExpirationAge           time.Duration          `mapstructure:"telemetry_emitter_delta_expiration_age"`
	TelemetryEmitterDeltaExpirationCheckInterval time.Duration          `mapstructure:"telemetry_emitter_delta_expiration_check_interval"`
	WorkerThreads                                int                    `mapstructure:"worker_threads"`
	EntityDefinitions                            []synthesis.Definition `mapstructure:"entity_definitions"`
	IntegrationMetadata                          integration.Metadata   `mapstructure:"integration_metadata"`
}

const maskedLicenseKey = "****"

// LicenseKey is a New Relic license key that will be masked when printed using standard formatters
type LicenseKey string

// String ensures that the LicenseKey will be masked in functions like fmt.Println(licenseKey)
func (l LicenseKey) String() string {
	return maskedLicenseKey
}

// GoString ensures that the LicenseKey will be masked in functions like fmt.Printf("%#v", licenseKey)
func (l LicenseKey) GoString() string {
	return maskedLicenseKey
}

// channel length for entities
const queueLength = 100

func validateConfig(cfg *Config) error {
	requiredMsg := "%s is required and can't be empty"
	if cfg.ClusterName == "" && cfg.Standalone {
		return fmt.Errorf(requiredMsg, "cluster_name")
	}
	if cfg.LicenseKey == "" && cfg.Standalone {
		return fmt.Errorf(requiredMsg, "license_key")
	}

	if cfg.EmitterProxy != "" {
		proxyURL, err := url.Parse(cfg.EmitterProxy)
		if err != nil {
			return fmt.Errorf("couldn't parse emitter proxy url: %w", err)
		}
		cfg.EmitterProxyURL = proxyURL
	}

	if cfg.EmitterCAFile != "" {
		_, err := ioutil.ReadFile(cfg.EmitterCAFile)
		if err != nil {
			return fmt.Errorf("couldn't read emitter CA file: %w", err)
		}
	}

	if cfg.WorkerThreads < 4 {
		logrus.Infof("Minimum amount of 4 worker threads required, %d given. Setting to 4.", cfg.WorkerThreads)
		cfg.WorkerThreads = 4
	}

	return nil
}

// RunWithEmitters runs the scraper with preselected emitters.
func RunWithEmitters(cfg *Config, emitters []integration.Emitter) error {
	if len(emitters) == 0 {
		return fmt.Errorf("you need to configure at least one valid emitter")
	}

	selfRetriever, err := endpoints.SelfRetriever()
	if err != nil {
		return fmt.Errorf("while parsing provided endpoints: %w", err)
	}
	var retrievers []endpoints.TargetRetriever
	fixedRetriever, err := endpoints.FixedRetriever(cfg.TargetConfigs...)
	if err != nil {
		return fmt.Errorf("while parsing provided endpoints: %w", err)
	}
	retrievers = append(retrievers, fixedRetriever)

	if !cfg.DisableAutodiscovery {
		kubernetesRetriever, err := endpoints.NewKubernetesTargetRetriever(cfg.ScrapeEnabledLabel, cfg.RequireScrapeEnabledLabelForNodes, cfg.ScrapeServices, cfg.ScrapeEndpoints, endpoints.WithInClusterConfig())
		if err != nil {
			logrus.WithError(err).Errorf("not possible to get a Kubernetes client. If you aren't running this integration in a Kubernetes cluster, you can ignore this error")
		} else {
			retrievers = append(retrievers, kubernetesRetriever)
		}
	}
	defaultTransformations := integration.ProcessingRule{
		Description: "Default transformation rules",
		AddAttributes: []integration.AddAttributesRule{
			{
				MetricPrefix: "",
				Attributes: map[string]interface{}{
					"k8s.cluster.name": cfg.ClusterName,
					"clusterName":      cfg.ClusterName,
					// Keeping these for backward compatibility
					"integrationVersion": integration.Version,
					"integrationName":    integration.Name,
					// Since the agent is not used we add the attributes manually
					"collector.name":           integration.Name,
					"collector.version":        integration.Version,
					"instrumentation.name":     integration.Name,
					"instrumentation.version":  integration.Version,
					"instrumentation.provider": "newRelic",
				},
			},
		},
	}
	processingRules := append(cfg.ProcessingRules, defaultTransformations)

	scrapeDuration, err := time.ParseDuration(cfg.ScrapeDuration)
	if err != nil {
		return fmt.Errorf(
			"parsing scrape_duration value (%v): %w",
			cfg.ScrapeDuration,
			err,
		)
	}

	go integration.Execute(
		scrapeDuration,
		selfRetriever,
		retrievers,
		integration.NewFetcher(scrapeDuration, cfg.ScrapeTimeout, cfg.WorkerThreads, cfg.BearerTokenFile, cfg.CaFile, cfg.InsecureSkipVerify, queueLength),
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
	return http.ListenAndServe(":8080", r)
}

// RunOnceWithEmitters runs the scraper with preselected emitters once.
func RunOnceWithEmitters(cfg *Config, emitters []integration.Emitter) error {
	if len(emitters) == 0 {
		return fmt.Errorf("you need to configure at least one valid emitter")
	}

	var retrievers []endpoints.TargetRetriever
	fixedRetriever, err := endpoints.FixedRetriever(cfg.TargetConfigs...)
	if err != nil {
		return fmt.Errorf("while parsing provided endpoints: %w", err)
	}
	retrievers = append(retrievers, fixedRetriever)

	scrapeDuration, err := time.ParseDuration(cfg.ScrapeDuration)
	if err != nil {
		return fmt.Errorf(
			"parsing scrape_duration value (%v): %w",
			cfg.ScrapeDuration,
			err,
		)
	}

	// Fetch duration is hardcoded to 1 since the target is scraped only once
	integration.ExecuteOnce(
		retrievers,
		integration.NewFetcher(scrapeDuration, cfg.ScrapeTimeout, cfg.WorkerThreads, cfg.BearerTokenFile, cfg.CaFile, cfg.InsecureSkipVerify, queueLength),
		integration.RuleProcessor(cfg.ProcessingRules, queueLength),
		emitters)

	return nil
}

// Run runs the scraper. If Standalone=true it keeps running otherwise runs once and exits
func Run(cfg *Config) error {
	err := validateConfig(cfg)
	if err != nil {
		return fmt.Errorf("while getting configuration options: %w", err)
	}
	if cfg.Verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	var emitters []integration.Emitter
	for _, e := range cfg.Emitters {
		switch e {
		case "stdout":
			emitters = append(emitters, integration.NewStdoutEmitter())
		case "telemetry":
			harvesterOpts := []func(*telemetry.Config){
				telemetry.ConfigAPIKey(string(cfg.LicenseKey)),
				telemetry.ConfigBasicErrorLogger(os.Stdout),
				integration.TelemetryHarvesterWithMetricsURL(cfg.MetricAPIURL),
			}

			if cfg.EmitterProxyURL != nil {
				harvesterOpts = append(
					harvesterOpts,
					integration.TelemetryHarvesterWithProxy(cfg.EmitterProxyURL),
				)
			}

			if cfg.EmitterCAFile != "" {
				tlsConfig, err := integration.NewTLSConfig(
					cfg.EmitterCAFile,
					cfg.EmitterInsecureSkipVerify,
				)
				if err != nil {
					return fmt.Errorf("invalid TLS configuration: %w", err)
				}
				harvesterOpts = append(
					harvesterOpts,
					integration.TelemetryHarvesterWithTLSConfig(tlsConfig),
				)
			}

			// Options that rely on modifying the emitter Client Transport
			// should go before this one, as this changes the type of the
			// Transport to `integration.licenseKeyRoundTripper`.
			harvesterOpts = append(
				harvesterOpts,
				integration.TelemetryHarvesterWithLicenseKeyRoundTripper(string(cfg.LicenseKey)),
			)

			if cfg.Verbose {
				harvesterOpts = append(harvesterOpts, telemetry.ConfigBasicDebugLogger(os.Stdout))
			}

			if cfg.Audit {
				harvesterOpts = append(harvesterOpts, telemetry.ConfigBasicAuditLogger(os.Stdout))
			}

			hTime, err := time.ParseDuration(cfg.EmitterHarvestPeriod)
			if err != nil {
				return fmt.Errorf(
					"invalid telemetry emitter harvest period %s: %w",
					cfg.EmitterHarvestPeriod,
					err,
				)
			}
			mhTime, err := time.ParseDuration(cfg.MinEmitterHarvestPeriod)
			if err != nil {
				return fmt.Errorf(
					"invalid minimum telemetry emitter harvest period %s: %w",
					cfg.MinEmitterHarvestPeriod,
					err,
				)
			}

			c := integration.TelemetryEmitterConfig{
				HarvesterOpts:                 harvesterOpts,
				DeltaExpirationAge:            cfg.TelemetryEmitterDeltaExpirationAge,
				DeltaExpirationCheckInternval: cfg.TelemetryEmitterDeltaExpirationCheckInterval,
				BoundedHarvesterCfg: integration.BoundedHarvesterCfg{
					HarvestPeriod:     hTime,
					MinReportInterval: mhTime,
					MetricCap:         cfg.MaxStoredMetrics,
				},
			}

			emitter, err := integration.NewTelemetryEmitter(c)
			if err != nil {
				return errors.Wrap(err, "could not create new TelemetryEmitter")
			}
			emitters = append(emitters, emitter)
		case "infra-sdk":
			s := synthesis.NewSynthesizer(cfg.EntityDefinitions)
			emitter := integration.NewInfraSdkEmitter(s)
			if err := emitter.SetIntegrationMetadata(cfg.IntegrationMetadata); err != nil {
				logrus.WithError(err).Debugf("could not set emitter metadata: %v", cfg.IntegrationMetadata)
			}
			emitters = append(emitters, emitter)
		default:
			logrus.Debugf("unknown emitter: %s", e)
			continue
		}
	}

	if cfg.Standalone {
		logrus.Info("Running in standalone mode...")
		err = RunWithEmitters(cfg, emitters)
	} else {
		logrus.Info("Running in run-once mode...")
		err = RunOnceWithEmitters(cfg, emitters)
	}
	return err
}
