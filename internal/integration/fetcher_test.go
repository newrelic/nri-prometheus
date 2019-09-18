// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/newrelic/nri-prometheus/internal/pkg/endpoints"
	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	"github.com/newrelic/nri-prometheus/internal/pkg/prometheus"
)

const (
	fetchDuration  = 1 * time.Millisecond
	fetchTimeout   = time.Second * 5
	maxConnections = 4
	queueLength    = 100
)

func TestFetcher(t *testing.T) {
	// Given a fetcher
	fetcher := NewFetcher(fetchDuration, fetchTimeout, maxConnections, "", "", true, queueLength)
	var invokedURL string
	fetcher.(*prometheusFetcher).getMetrics = func(client prometheus.HTTPDoer, url string) (names prometheus.MetricFamiliesByName, e error) {
		invokedURL = url
		return prometheus.MetricFamiliesByName{
			"some-name": dto.MetricFamily{},
		}, nil
	}

	// When it fetches data synchronously
	addr := url.URL{Scheme: "http", Path: "hello/metrics"}
	pairsCh := fetcher.Fetch([]endpoints.Target{endpoints.New("", addr, endpoints.Object{})})

	var pair TargetMetrics
	select {
	case pair = <-pairsCh:
	case <-time.After(fetchTimeout):
		t.Fatal("can't fetch data")
	}

	// Then the fetched metric families are submitted
	assert.Equal(t, "http://hello/metrics", pair.Target.URL.String())

	// and the URL is invoked
	assert.Equal(t, "http://hello/metrics", invokedURL)

	invokedURL = ""
}

func TestFetcher_Error(t *testing.T) {
	// Given a fetcher
	fetcher := NewFetcher(time.Millisecond, fetchTimeout, maxConnections, "", "", true, queueLength)

	// That fails retrieving data from one of the metrics endpoint
	invokedURLs := make([]string, 0)
	fetcher.(*prometheusFetcher).getMetrics = func(client prometheus.HTTPDoer, url string) (names prometheus.MetricFamiliesByName, e error) {
		if strings.Contains(url, "fail") {
			return nil, errors.New("catapun")
		}
		invokedURLs = append(invokedURLs, url)
		return prometheus.MetricFamiliesByName{
			"some-name": dto.MetricFamily{},
		}, nil
	}

	fail := url.URL{Scheme: "http", Path: "fail/metrics"}
	hello := url.URL{Scheme: "http", Path: "hello/metrics"}
	pairsCh := fetcher.Fetch([]endpoints.Target{
		endpoints.New("", fail, endpoints.Object{}),
		endpoints.New("", hello, endpoints.Object{}),
	})

	var pair TargetMetrics
	select {
	case pair = <-pairsCh:
	case <-time.After(fetchTimeout):
		t.Fatal("can't fetch data")
	}

	// No more data is forwarded
	select {
	case p := <-pairsCh: // channel is closed
		assert.Empty(t, p.Target.URL, "no more data should have been submitted", "%#v", p)
	case <-time.After(100 * time.Millisecond):
		require.Fail(t, "fetcher channel should have been closed")
	}

	assert.Equal(t, "http://hello/metrics", pair.Target.URL.String())
	assert.Len(t, invokedURLs, 1)
	assert.Equal(t, "http://hello/metrics", invokedURLs[0])
}

func TestFetcher_ConcurrencyLimit(t *testing.T) {
	// This test fetches a lot of targets and verifies that no more than "maxConnections" are executed in
	// parallel
	parallelTasks := int32(0)
	reportedParallel := make(chan int32, queueLength)

	// Given a Fetcher
	fetcher := NewFetcher(time.Millisecond, fetchTimeout, maxConnections, "", "", true, queueLength)

	fetcher.(*prometheusFetcher).getMetrics = func(client prometheus.HTTPDoer, url string) (names prometheus.MetricFamiliesByName, e error) {
		defer atomic.AddInt32(&parallelTasks, -1)
		atomic.AddInt32(&parallelTasks, 1)
		reportedParallel <- atomic.LoadInt32(&parallelTasks)
		time.Sleep(10 * time.Millisecond)
		return prometheus.MetricFamiliesByName{"some-name": dto.MetricFamily{}}, nil
	}

	// WHEN it fetches data from a big number of targets
	targets := make([]endpoints.Target, 0, queueLength)
	for i := 0; i < queueLength; i++ {
		addr := url.URL{Scheme: "http", Host: fmt.Sprintf("target%v", i), Path: "/metrics"}
		targets = append(targets, endpoints.New("", addr, endpoints.Object{}))
	}
	fetcher.Fetch(targets)

	maxParallel := 0
	timeout := time.After(5 * time.Second)
	for i := 0; i < queueLength; i++ {
		select {
		case val := <-reportedParallel:
			if maxParallel < int(val) {
				maxParallel = int(val)
			}
		case <-timeout:
			require.Fail(t, "timeout while waiting for the targets output")
		}
	}
	// THEN no more than "maxConnections" are executed in parallel
	require.True(t, maxParallel == maxConnections,
		"no more nor less than %v connections should run in parallel. Actually %v", maxConnections, maxParallel)
}

func TestConvertPromMetrics(t *testing.T) {
	tests := []struct {
		target string
		mfs    prometheus.MetricFamiliesByName
		want   []Metric
	}{
		{
			"hotdog-stand",
			prometheus.MetricFamiliesByName{
				"sales": dto.MetricFamily{
					// use anonymous struct to return *dto.MetricType literal.
					Type: &(&struct{ x dto.MetricType }{dto.MetricType_COUNTER}).x,
					Metric: []*dto.Metric{
						{
							Label: []*dto.LabelPair{
								{
									// use anonymous struct to return *string literal.
									Name:  &(&struct{ x string }{"location"}).x,
									Value: &(&struct{ x string }{"downtown"}).x,
								},
							},
							Counter: &dto.Counter{
								// use anonymous struct to return *float64 literal.
								Value: &(&struct{ x float64 }{137}).x,
							},
						},
					},
				},
				"temperature": dto.MetricFamily{
					Type: &(&struct{ x dto.MetricType }{dto.MetricType_GAUGE}).x,
					Metric: []*dto.Metric{
						{
							Label: []*dto.LabelPair{
								{
									Name:  &(&struct{ x string }{"filling"}).x,
									Value: &(&struct{ x string }{"beef"}).x,
								},
							},
							Gauge: &dto.Gauge{
								Value: &(&struct{ x float64 }{165}).x,
							},
						},
					},
				},
			},
			[]Metric{
				{
					name:       "sales",
					metricType: metricType_COUNTER,
					value:      float64(137),
					attributes: labels.Set{
						"location":       "downtown",
						"targetName":     "hotdog-stand",
						"nrMetricType":   "count",
						"promMetricType": "counter",
					},
				},
				{
					name:       "temperature",
					metricType: metricType_GAUGE,
					value:      float64(165),
					attributes: labels.Set{
						"filling":        "beef",
						"targetName":     "hotdog-stand",
						"nrMetricType":   "gauge",
						"promMetricType": "gauge",
					},
				},
			},
		},
		{
			"hotdog-stand",
			prometheus.MetricFamiliesByName{
				"sales": dto.MetricFamily{
					// use anonymous struct to return *dto.MetricType literal.
					Type: &(&struct{ x dto.MetricType }{dto.MetricType_COUNTER}).x,
					Metric: []*dto.Metric{
						{
							Label: []*dto.LabelPair{
								{
									Name:  &(&struct{ x string }{"location"}).x,
									Value: &(&struct{ x string }{"downtown"}).x,
								},
							},
							Counter: &dto.Counter{
								Value: &(&struct{ x float64 }{140}).x,
							},
						},
					},
				},
				"temperature": dto.MetricFamily{
					Type: &(&struct{ x dto.MetricType }{dto.MetricType_GAUGE}).x,
					Metric: []*dto.Metric{
						{
							Label: []*dto.LabelPair{
								{
									Name:  &(&struct{ x string }{"filling"}).x,
									Value: &(&struct{ x string }{"beef"}).x,
								},
							},
							Gauge: &dto.Gauge{
								Value: &(&struct{ x float64 }{135}).x,
							},
						},
					},
				},
			},
			[]Metric{
				{
					name:       "sales",
					metricType: metricType_COUNTER,
					value:      float64(140),
					attributes: labels.Set{
						"location":       "downtown",
						"targetName":     "hotdog-stand",
						"nrMetricType":   "count",
						"promMetricType": "counter",
					},
				},
				{
					name:       "temperature",
					metricType: metricType_GAUGE,
					value:      float64(135),
					attributes: labels.Set{
						"filling":        "beef",
						"targetName":     "hotdog-stand",
						"nrMetricType":   "gauge",
						"promMetricType": "gauge",
					},
				},
			},
		},
	}

	for _, test := range tests {
		assert.ElementsMatch(t, test.want, convertPromMetrics(nil, test.target, test.mfs))
	}
}

func TestConvertPromMetricsMultiTargetCollisions(t *testing.T) {
	metric := dto.Metric{
		Label: []*dto.LabelPair{
			{
				// use anonymous struct to return *string literal.
				Name:  &(&struct{ x string }{"name"}).x,
				Value: &(&struct{ x string }{"common-name"}).x,
			},
		},
		Counter: &dto.Counter{
			// use anonymous struct to return *float64 literal.
			Value: &(&struct{ x float64 }{137}).x,
		},
	}

	mfbn := prometheus.MetricFamiliesByName{
		"common-name": dto.MetricFamily{
			// use anonymous struct to return *dto.MetricType literal.
			Type:   &(&struct{ x dto.MetricType }{dto.MetricType_COUNTER}).x,
			Metric: []*dto.Metric{&metric},
		},
	}

	// Process metric scraped from `target-a`.
	convertPromMetrics(nil, "target-a", mfbn)

	// Process similarly named and labeled metric scrapped from `target-b` but with a different value.
	metric.Counter.Value = &(&struct{ x float64 }{100}).x
	convertPromMetrics(nil, "target-b", mfbn)

	// Again process metric scraped from `target-a`.
	// The value of the accumulated count has increased by 1.
	metric.Counter.Value = &(&struct{ x float64 }{138}).x
	nrMetrics := convertPromMetrics(nil, "target-a", mfbn)

	if len(nrMetrics) != 1 {
		t.Errorf("expected a single metric got %d", len(nrMetrics))
		return
	}

	want := Metric{
		name:       "common-name",
		metricType: metricType_COUNTER,
		// Here the delta calculation didn't happen yet.
		value: float64(138),
		attributes: labels.Set{
			"name":           "common-name",
			"targetName":     "target-a",
			"nrMetricType":   "count",
			"promMetricType": "counter",
		},
	}
	assert.Equal(t, nrMetrics[0], want)
}
