// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/newrelic/nri-prometheus/internal/cmd/scraper"
	"github.com/newrelic/nri-prometheus/internal/pkg/endpoints"
)

func TestDetermineMetricAPIURL(t *testing.T) {
	testCases := []struct {
		license     string
		expectedURL string
	}{
		// empty license
		{license: "", expectedURL: defaultMetricAPIURL},
		// non-region license
		{license: "0123456789012345678901234567890123456789", expectedURL: defaultMetricAPIURL},
		// four letter region
		{license: "eu01xx6789012345678901234567890123456789", expectedURL: fmt.Sprintf(metricAPIRegionURL, "eu")},
		// five letter region
		{license: "gov01x6789012345678901234567890123456789", expectedURL: fmt.Sprintf(metricAPIRegionURL, "gov")},
	}

	for _, tt := range testCases {
		actualURL := determineMetricAPIURL(tt.license)
		if actualURL != tt.expectedURL {
			t.Fatalf("URL does not match expected URL, got=%s, expected=%s", actualURL, tt.expectedURL)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	expectedScrapper := scraper.Config{
		MetricAPIURL:                      "https://metric-api.newrelic.com/metric/v1/infra",
		Verbose:                           true,
		Emitters:                          []string{"infra-sdk"},
		ScrapeEnabledLabel:                "prometheus.io/scrape",
		RequireScrapeEnabledLabelForNodes: true,
		ScrapeTimeout:                     5 * time.Second,
		ScrapeServices:                    true,
		ScrapeDuration:                    "5s",
		ScrapeAcceptHeader:                "text/plain;version=0.0.4",
		EmitterHarvestPeriod:              "1s",
		MinEmitterHarvestPeriod:           "200ms",
		MaxStoredMetrics:                  10000,
		TargetConfigs: []endpoints.TargetConfig{
			{
				Description: "AAA",
				URLs:        []string{"localhost:9121"},
				TLSConfig:   endpoints.TLSConfig{},
			},
		},
		InsecureSkipVerify: true,
		WorkerThreads:      4,
		HostID:             "awesome-host",
	}
	t.Setenv("CONFIG_PATH", "testdata/config-with-legacy-entity-definitions.yaml")
	t.Setenv("NRI_HOST_ID", "awesome-host")
	scraperCfg, err := loadConfig()
	if err != nil {
		t.Fatalf("error was not expected %v", err)
	}
	if !reflect.DeepEqual(*scraperCfg, expectedScrapper) {
		t.Fatalf("scraper retrieved not as expected, got=%v, expected=%v", *scraperCfg, expectedScrapper)
	}
}
