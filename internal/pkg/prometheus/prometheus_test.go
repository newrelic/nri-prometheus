// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/newrelic/nri-prometheus/internal/pkg/prometheus"
)

const testHeader = "application/openmetrics-text"

func TestGetHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if accept != testHeader {
			t.Errorf("Expected Accept header to prefer application/openmetrics-text, got %q", accept)
		}

		_, _ = w.Write([]byte("metric_a 1\nmetric_b 2\n"))
	}))
	defer ts.Close()

	expected := []string{"metric_a", "metric_b"}
	mfs, err := prometheus.Get(http.DefaultClient, ts.URL, testHeader)
	actual := []string{}
	for k := range mfs {
		actual = append(actual, k)
	}

	assert.NoError(t, err)
	assert.ElementsMatch(t, expected, actual)
}
