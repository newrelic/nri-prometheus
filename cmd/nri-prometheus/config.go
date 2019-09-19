package main

import (
	"errors"
	"fmt"
	"github.com/newrelic/nri-prometheus/internal/cmd/scraper"
	"github.com/spf13/viper"
	"regexp"
)

func loadConfig() (*scraper.Config, error) {
	cfg := viper.New()
	cfg.SetConfigName("config")
	cfg.SetConfigType("yaml")
	cfg.AddConfigPath("/etc/nri-prometheus/")
	cfg.AddConfigPath(".")
	scraper.LoadViperDefaults(cfg)

	err := cfg.ReadInConfig()
	if err != nil {
		return nil, err
	}

	var scraperCfg scraper.Config
	scraper.BindViperEnv(cfg, scraperCfg)
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
