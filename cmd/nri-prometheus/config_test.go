// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"testing"
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
