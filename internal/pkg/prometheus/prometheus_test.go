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

var result = `
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 8
# HELP go_memstats_heap_idle_bytes Number of heap bytes waiting to be used.
# TYPE go_memstats_heap_idle_bytes gauge
go_memstats_heap_idle_bytes 2.301952e+06
# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 7.5235e-05
go_gc_duration_seconds{quantile="0.25"} 7.5235e-05
go_gc_duration_seconds{quantile="0.5"} 0.000200349
go_gc_duration_seconds{quantile="0.75"} 0.000200349
go_gc_duration_seconds{quantile="1"} 0.000200349
go_gc_duration_seconds_sum 0.000275584
go_gc_duration_seconds_count 2
# HELP http_requests_total Total number of HTTP requests made.
# TYPE http_requests_total counter
http_requests_total{code="200",handler="prometheus",method="get"} 2
`

func TestGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(result))
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
