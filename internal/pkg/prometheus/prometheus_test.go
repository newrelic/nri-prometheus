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

func TestGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "testdata/simple-metrics")
	}))
	defer ts.Close()
	expected := []string{"go_goroutines", "go_memstats_heap_idle_bytes", "go_gc_duration_seconds", "http_requests_total"}
	mfs, err := prometheus.Get(http.DefaultClient, ts.URL)
	actual := []string{}
	for k := range mfs {
		actual = append(actual, k)
	}
	assert.NoError(t, err)
	assert.ElementsMatch(t, expected, actual)
}
