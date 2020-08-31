// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/nri-prometheus/internal/cmd/scraper"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// ArgumentList Available Arguments
type ArgumentList struct {
	ConfigPath string `default:"" help:"Path to the config file"`
	Configfile string `default:"" help:"Deprecated. --config_path takes precedence if both are set"`
}

func loadConfig() (*scraper.Config, error) {

	c := ArgumentList{}
	err := args.SetupArgs(&c)
	if err != nil {
		return nil, err
	}

	cfg := viper.New()
	cfg.SetConfigType("yaml")

	if c.Configfile != "" && c.ConfigPath == "" {
		c.ConfigPath = c.Configfile
	}

	if c.ConfigPath != "" {
		cfg.AddConfigPath(filepath.Dir(c.ConfigPath))
		cfg.SetConfigName(filepath.Base(c.ConfigPath))
	} else {
		cfg.SetConfigName("config")
		cfg.AddConfigPath("/etc/nri-prometheus/")
		cfg.AddConfigPath(".")
	}

	setViperDefaults(cfg)

	err = cfg.ReadInConfig()
	if err != nil {
		return nil, errors.Wrap(err, "could not read configuration")
	}

	var scraperCfg scraper.Config
	bindViperEnv(cfg, scraperCfg)
	err = cfg.Unmarshal(&scraperCfg)

	if err != nil {
		return nil, errors.Wrap(err, "could not parse configuration file")
	}

	if scraperCfg.MetricAPIURL == "" {
		scraperCfg.MetricAPIURL = determineMetricAPIURL(string(scraperCfg.LicenseKey))
	}

	return &scraperCfg, nil
}

// setViperDefaults loads the default configuration into the given Viper registry.
func setViperDefaults(viper *viper.Viper) {
	viper.SetDefault("debug", false)
	viper.SetDefault("verbose", false)
	viper.SetDefault("emitters", []string{"telemetry"})
	viper.SetDefault("scrape_enabled_label", "prometheus.io/scrape")
	viper.SetDefault("require_scrape_enabled_label_for_nodes", true)
	viper.SetDefault("scrape_timeout", 5*time.Second)
	viper.SetDefault("scrape_duration", "30s")
	viper.SetDefault("emitter_harvest_period", "1s")
	viper.SetDefault("auto_decorate", false)
	viper.SetDefault("insecure_skip_verify", false)
	viper.SetDefault("standalone", true)
	viper.SetDefault("disable_autodiscovery", false)
	viper.SetDefault("percentiles", []float64{50.0, 95.0, 99.0})
}

// bindViperEnv automatically binds the variables in given configuration struct to environment variables.
// This is needed because Viper only takes environment variables into consideration for unmarshalling if they are also
// defined in the configuration file. We need to be able to use environment variables even if such variable is not in
// the config file.
// For more information see https://github.com/spf13/viper/issues/188.
func bindViperEnv(vCfg *viper.Viper, iface interface{}, parts ...string) {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)
	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		t := ift.Field(i)
		tv, ok := t.Tag.Lookup("mapstructure")
		if !ok {
			continue
		}
		switch v.Kind() {
		case reflect.Struct:
			bindViperEnv(vCfg, v.Interface(), append(parts, tv)...)
		default:
			_ = vCfg.BindEnv(strings.Join(append(parts, tv), "_"))
		}
	}
}

var (
	regionLicenseRegex = regexp.MustCompile(`^([a-z]{2,3})[0-9]{2}x{1,2}`)
	metricAPIRegionURL = "https://metric-api.%s.newrelic.com/metric/v1/infra"
	// for historical reasons the US datacenter is the default Metric API
	defaultMetricAPIURL = "https://metric-api.newrelic.com/metric/v1/infra"
)

// determineMetricAPIURL determines the Metric API URL based on the license key.
// The first 5 characters of the license URL indicates the region.
func determineMetricAPIURL(license string) string {
	m := regionLicenseRegex.FindStringSubmatch(license)
	if len(m) > 1 {
		return fmt.Sprintf(metricAPIRegionURL, m[1])
	}

	return defaultMetricAPIURL
}
