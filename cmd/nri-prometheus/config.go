// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/newrelic/nri-prometheus/internal/cmd/scraper"
	"github.com/spf13/viper"
)

func loadConfig() (*scraper.Config, error) {
	cfg := viper.New()
	cfg.SetConfigName("config")
	cfg.SetConfigType("yaml")
	cfg.AddConfigPath("/etc/nri-prometheus/")
	cfg.AddConfigPath(".")
	loadViperDefaults(cfg)

	err := cfg.ReadInConfig()
	if err != nil {
		return nil, err
	}

	var scraperCfg scraper.Config
	bindViperEnv(cfg, scraperCfg)
	err = cfg.Unmarshal(&scraperCfg)

	if err != nil {
		return nil, err
	}

	if scraperCfg.LicenseKey == "" {
		return nil, errors.New("LICENSE_KEY is required and can't be empty")
	}

	if scraperCfg.MetricAPIURL != "" {
		scraperCfg.MetricAPIURL = determineMetricAPIURL(scraperCfg.LicenseKey)
	}

	return &scraperCfg, err
}

// LoadViperDefaults loads the default configuration into the given Viper loader.
func loadViperDefaults(viper *viper.Viper) {
	viper.SetDefault("debug", false)
	viper.SetDefault("verbose", false)
	viper.SetDefault("emitters", []string{"telemetry"})
	viper.SetDefault("scrape_enabled_label", "prometheus.io/scrape")
	viper.SetDefault("require_scrape_enabled_label_for_nodes", false)
	viper.SetDefault("scrape_timeout", time.Duration(5000000000))
	viper.SetDefault("scrape_duration", "30s")
	viper.SetDefault("emitter_harvest_period", "1s")
	viper.SetDefault("auto_decorate", false)
	viper.SetDefault("insecure_skip_verify", false)
}

// BindViperEnv automatically binds the variables in given configuration struct to environment variables.
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
