// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/newrelic/nri-prometheus/internal/pkg/endpoints"
)

var prometheusInput = `# HELP redis_exporter_build_info redis exporter build_info
# TYPE redis_exporter_build_info gauge
redis_exporter_build_info{build_date="2018-07-03-14:18:56",commit_sha="3e15af27aac37e114b32a07f5e9dc0510f4cbfc4",golang_version="go1.9.4",version="v0.20.2"} 1
# HELP redis_exporter_scrapes_total Current total redis scrapes.
# TYPE redis_exporter_scrapes_total counter
redis_exporter_scrapes_total{cosa="fina"} 42
# HELP redis_instance_info Information about the Redis instance
# TYPE redis_instance_info gauge
redis_instance_info{addr="ohai-playground-redis-master:6379",alias="ohai-playground-redis",os="Linux 4.15.0 x86_64",redis_build_id="c701a4acd98ea64a",redis_mode="standalone",redis_version="4.0.10",role="master"} 1
redis_instance_info{addr="ohai-playground-redis-slave:6379",alias="ohai-playground-redis",os="Linux 4.15.0 x86_64",redis_build_id="c701a4acd98ea64a",redis_mode="standalone",redis_version="4.0.10",role="slave"} 1
# HELP redis_instantaneous_input_kbps instantaneous_input_kbpsmetric
# TYPE redis_instantaneous_input_kbps gauge
redis_instantaneous_input_kbps{addr="ohai-playground-redis-master:6379",alias="ohai-playground-redis"} 0.05
redis_instantaneous_input_kbps{addr="ohai-playground-redis-slave:6379",alias="ohai-playground-redis"} 0
`

func scrapeString(t *testing.T, inputMetrics string) TargetMetrics {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(inputMetrics))
	}))
	defer ts.Close()
	server, err := endpoints.FixedRetriever(endpoints.TargetConfig{URLs: []string{ts.URL}})
	require.NoError(t, err)
	target, err := server.GetTargets()
	require.NoError(t, err)

	metricsCh := NewFetcher(time.Millisecond, 1*time.Second, "", workerThreads, "", "", true, queueLength).Fetch(target)

	var pair TargetMetrics
	select {
	case pair = <-metricsCh:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout while waiting for a simple entity")
	}

	// we expect that only one entity is sent from the fetcher, then the channel is closed
	select {
	case p := <-metricsCh: // channel is closed
		require.Empty(t, p.Metrics, "no more data should have been submitted", "%#v", p)
	case <-time.After(100 * time.Millisecond):
		require.Fail(t, "scraper channel should have been closed after all entities were processed")
	}

	return pair
}
