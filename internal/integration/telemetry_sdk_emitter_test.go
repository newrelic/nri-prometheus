// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	"github.com/newrelic/nri-prometheus/internal/pkg/prometheus"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"

	assert "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkTelemetrySDKEmitter(b *testing.B) {
	contents, err := ioutil.ReadFile("test/cadvisor.txt")
	cachedFile := bytes.NewBuffer(contents)
	assert.NoError(b, err)
	contentLength := strconv.Itoa(cachedFile.Len())
	b.Log("payload size", contentLength)

	mfByName, err := decodePromMetrics(cachedFile)
	assert.NoError(b, err)
	assert.NotNil(b, mfByName)

	cachedMetrics := convertPromMetrics(nil, "fakeTarget", *mfByName)
	b.Logf("Number of metrics in sample: %d", len(cachedMetrics))

	multiplyFactor := 20
	superMetrics := make([]Metric, 0, len(cachedMetrics)*multiplyFactor)
	for i := 0; i < multiplyFactor; i++ {
		for j, m := range cachedMetrics {
			m.name = "Metric " + strconv.Itoa(i) + strconv.Itoa(j-1)
			superMetrics = append(superMetrics, m)
		}
	}
	b.Logf("Number of metrics in supersized sample: %d", len(superMetrics))

	c := TelemetryEmitterConfig{
		HarvesterOpts: []TelemetryHarvesterOpt{
			func(cfg *telemetry.Config) {
				cfg.Client.Transport = nilRoundTripper()
			},
			telemetry.ConfigAPIKey("api key"),
			TelemetryHarvesterWithMetricsURL("nilapiurl"),
			telemetry.ConfigBasicErrorLogger(os.Stdout),
		},
		DisableBoundedHarvester: true,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		emitter, err := NewTelemetryEmitter(c)
		assert.NoError(b, err)
		err = emitter.Emit(superMetrics)
		assert.NoError(b, err)
		// Need to trigger a manual harvest here otherwise the benchmark is useless.
		emitter.harvester.HarvestNow(context.Background())
	}
}

func TestTelemetryEmitterEmit(t *testing.T) {
	t.Parallel()

	hist, err := newHistogram([]int64{0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}

	summary, err := newSummary(3, 10, []*quantile{{0.5, 10}, {0.999, 100}})
	if err != nil {
		t.Fatal(err)
	}

	metrics := []Metric{
		{
			name:       "common-name",
			metricType: metricType_COUNTER,
			value:      float64(1),
			attributes: labels.Set{
				"name":           "common-name",
				"targetName":     "target-a",
				"nrMetricType":   "count",
				"promMetricType": "counter",
			},
		},
		{
			name:       "common-name2",
			metricType: metricType_COUNTER,
			value:      float64(1),
			attributes: labels.Set{
				"name":           "common-name2",
				"targetName":     "target-b",
				"nrMetricType":   "count",
				"promMetricType": "counter",
			},
		},
		{
			name:       "common-name3",
			metricType: metricType_GAUGE,
			value:      float64(1),
			attributes: labels.Set{
				"name":           "common-name3",
				"targetName":     "target-c",
				"nrMetricType":   "gauge",
				"promMetricType": "gauge",
			},
		},
		{
			name:       "histogram-1",
			metricType: metricType_HISTOGRAM,
			value:      hist,
			attributes: labels.Set{
				"name":           "histogram-1",
				"targetName":     "target-d",
				"nrMetricType":   "histogram",
				"promMetricType": "histogram",
			},
		},
		{
			name:       "summary-1",
			metricType: metricType_SUMMARY,
			value:      summary,
			attributes: labels.Set{},
		},
		{
			name:       "not a number, NAN",
			metricType: metricType_COUNTER,
			value:      math.NaN(),
			attributes: labels.Set{},
		},
	}

	var rawMetrics []interface{}
	c := TelemetryEmitterConfig{
		HarvesterOpts: []TelemetryHarvesterOpt{
			telemetry.ConfigAPIKey("api key"),
			TelemetryHarvesterWithMetricsURL("nilapiurl"),
			func(cfg *telemetry.Config) {
				cfg.Client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
					var reader io.ReadCloser
					// The telemetry SDK sends encoded data, but be safe and check
					switch req.Header.Get("Content-Encoding") {
					case "gzip":
						var err error
						if reader, err = gzip.NewReader(req.Body); err != nil {
							t.Fatal(err)
						}
						defer func() {
							_ = reader.Close()
						}()
					default:
						reader = ioutil.NopCloser(req.Body)
					}
					var decoder []map[string]interface{}
					if err := json.NewDecoder(reader).Decode(&decoder); err != nil {
						t.Fatal(err)
					}
					var ok bool
					if rawMetrics, ok = decoder[0]["metrics"].([]interface{}); !ok {
						t.Fatal(errors.New("failed to decode telemetry post request body metrics"))
					}

					return emptyResponse(200), nil
				})
			},
			telemetry.ConfigBasicErrorLogger(os.Stdout),
		},
		BoundedHarvesterCfg: BoundedHarvesterCfg{
			DisablePeriodicReporting: true,
		},
	}

	e, err := NewTelemetryEmitter(c)
	assert.NoError(t, err)

	// Emit and force a harvest to clear.
	assert.NoError(t, e.Emit(metrics))
	e.harvester.HarvestNow(context.Background())
	time.Sleep(100 * time.Millisecond) // boundedHarvester.HarvestNow is asynchronous

	// Set new summary values so counts will be non-zero.
	summary2, err := newSummary(4, 15, []*quantile{{0.5, 10}, {0.999, 100}})
	if err != nil {
		t.Fatal(err)
	}
	*summary = *summary2

	// Set new histogram values so counts will be non-zero.
	hist2, err := newHistogram([]int64{1, 2, 10})
	if err != nil {
		t.Fatal(err)
	}
	*hist = *hist2

	// Run twice so delta counts are sent.
	assert.NoError(t, e.Emit(metrics))
	e.harvester.HarvestNow(context.Background())
	time.Sleep(100 * time.Millisecond)
	purgeTimestamps(rawMetrics)

	expectedMetrics := []interface{}{
		map[string]interface{}{
			"attributes": map[string]interface{}{
				"name":           "common-name",
				"nrMetricType":   "count",
				"promMetricType": "counter",
				"targetName":     "target-a",
			},
			"name":  "common-name",
			"type":  "count",
			"value": float64(0),
		},
		map[string]interface{}{
			"attributes": map[string]interface{}{
				"name":           "common-name2",
				"nrMetricType":   "count",
				"promMetricType": "counter",
				"targetName":     "target-b",
			},
			"name":  "common-name2",
			"type":  "count",
			"value": float64(0),
		},
		map[string]interface{}{
			"attributes": map[string]interface{}{
				"name":           "common-name3",
				"nrMetricType":   "gauge",
				"promMetricType": "gauge",
				"targetName":     "target-c",
			},
			"name":  "common-name3",
			"type":  "gauge",
			"value": float64(1),
		},
		map[string]interface{}{
			"attributes": map[string]interface{}{
				"name":           "histogram-1",
				"targetName":     "target-d",
				"nrMetricType":   "histogram",
				"promMetricType": "histogram",
			},
			"name": "histogram-1_sum",
			"type": "summary",
			"value": map[string]interface{}{
				"sum":   float64(10),
				"count": float64(1),
				"min":   nil,
				"max":   nil,
			},
		},
		map[string]interface{}{
			"attributes": map[string]interface{}{
				"name":           "histogram-1",
				"targetName":     "target-d",
				"nrMetricType":   "histogram",
				"promMetricType": "histogram",
			},
			"name":  "histogram-1_count",
			"type":  "count",
			"value": float64(0),
		},
		map[string]interface{}{
			"attributes": map[string]interface{}{
				"name":           "histogram-1",
				"targetName":     "target-d",
				"nrMetricType":   "histogram",
				"promMetricType": "histogram",
				"le":             "0",
			},
			"name":  "histogram-1_bucket",
			"type":  "count",
			"value": float64(1),
		},
		map[string]interface{}{
			"attributes": map[string]interface{}{
				"name":           "histogram-1",
				"targetName":     "target-d",
				"nrMetricType":   "histogram",
				"promMetricType": "histogram",
				"le":             "1",
			},
			"name":  "histogram-1_bucket",
			"type":  "count",
			"value": float64(2),
		},
		map[string]interface{}{
			"attributes": map[string]interface{}{
				"name":           "histogram-1",
				"targetName":     "target-d",
				"nrMetricType":   "histogram",
				"promMetricType": "histogram",
				"le":             "+Inf",
			},
			"name":  "histogram-1_bucket",
			"type":  "count",
			"value": float64(10),
		},
		map[string]interface{}{
			"attributes": map[string]interface{}{},
			"name":       "summary-1_sum",
			"type":       "summary",
			"value": map[string]interface{}{
				"sum":   float64(5),
				"count": float64(1),
				"min":   nil,
				"max":   nil,
			},
		},
		map[string]interface{}{
			"attributes": map[string]interface{}{},
			"name":       "summary-1_count",
			"type":       "count",
			"value":      float64(1),
		},
		map[string]interface{}{
			"attributes": map[string]interface{}{
				"quantile": "0.5",
			},
			"name":  "summary-1",
			"type":  "gauge",
			"value": float64(10),
		},
		map[string]interface{}{
			"attributes": map[string]interface{}{
				"quantile": "0.999",
			},
			"name":  "summary-1",
			"type":  "gauge",
			"value": float64(100),
		},
	}
	assert.Equal(t, expectedMetrics, rawMetrics)
}

func TestTelemetryHarvesterWithTLSConfig(t *testing.T) {
	t.Parallel()

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	cfg := &telemetry.Config{Client: &http.Client{}}
	TelemetryHarvesterWithTLSConfig(tlsConfig)(cfg)
	rt := cfg.Client.Transport
	tr := rt.(*http.Transport)
	assert.True(t, tr.TLSClientConfig.InsecureSkipVerify)

	tlsConfig.InsecureSkipVerify = false
	TelemetryHarvesterWithTLSConfig(tlsConfig)(cfg)
	rt = cfg.Client.Transport
	tr = rt.(*http.Transport)
	assert.False(t, tr.TLSClientConfig.InsecureSkipVerify)
}

func TestTelemetryHarvesterWithProxy(t *testing.T) {
	t.Parallel()

	proxyStr := "http://myproxy:444"
	proxyURL, err := url.Parse(proxyStr)
	require.NoError(t, err)
	cfg := &telemetry.Config{Client: &http.Client{}}
	TelemetryHarvesterWithProxy(proxyURL)(cfg)
	rt := cfg.Client.Transport
	tr, ok := rt.(*http.Transport)
	assert.True(t, ok)
	actualProxyURL, err := tr.Proxy(&http.Request{})
	require.NoError(t, err)
	assert.Equal(t, proxyURL, actualProxyURL)
}

func emptyResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       ioutil.NopCloser(&bytes.Buffer{}),
	}
}

func nilRoundTripper() roundTripperFunc {
	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return emptyResponse(200), nil
	})
	return rt
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip is the implementation for http.RoundTripper.
func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// CancelRequest is an optional interface required for go1.4 and go1.5
func (fn roundTripperFunc) CancelRequest(*http.Request) {}

func decodePromMetrics(src io.Reader) (*prometheus.MetricFamiliesByName, error) {
	mfs := prometheus.MetricFamiliesByName{}
	d := expfmt.NewDecoder(src, expfmt.FmtText)
	for {
		var mf dto.MetricFamily
		if err := d.Decode(&mf); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		mfs[mf.GetName()] = mf
	}
	return &mfs, nil
}

// quantile groups Quantile values so they can be passed as an ordered pair.
type quantile struct {
	Quantile float64
	Value    float64
}

// newSummary returns a Prometheus Summary for testing.
func newSummary(count uint64, sum float64, quantiles []*quantile) (*dto.Summary, error) {
	raw := fmt.Sprintf(`{
		"sample_count": %d,
		"sample_sum": %g,
		"quantile": [`, count, float64(sum))
	for i, q := range quantiles {
		raw += fmt.Sprintf(`{"quantile": %g, "value": %g}`, q.Quantile, q.Value)
		if i != len(quantiles)-1 {
			raw += ","
		}
	}
	raw += `]
	}`

	summary := &dto.Summary{}
	if err := json.Unmarshal([]byte(raw), summary); err != nil {
		return nil, err
	}
	return summary, nil
}

// newHistogram returns a Prometheus Histogram for testing.
func newHistogram(buckets []int64) (*dto.Histogram, error) {
	count := len(buckets)
	sum := buckets[count-1]
	raw := fmt.Sprintf(`{
		"sample_count": %d,
		"sample_sum": %g,
		"bucket": [`, count, float64(sum))
	for i, v := range buckets {
		raw += fmt.Sprintf(`{"upper_bound": %g, "cumulative_count": %d}`, float64(i), v)
		if i != count-1 {
			raw += ","
		}
	}
	raw += `]
	}`

	hist := &dto.Histogram{}
	if err := json.Unmarshal([]byte(raw), hist); err != nil {
		return nil, err
	}

	// Work around infinity not being supported in JSON.
	inf := math.Inf(1)
	hist.Bucket[count-1].UpperBound = &inf

	return hist, nil
}

// purgeTimestamps removes all `timestamp` amd `interval.ms` key/values
// from metrics.
//
// The passed metrics are the raw values the telemetry SDK passes to
// New Relic (hence no struct). This structure keeps the "timestamp"
// and "interval.ms" keys at the top-level of each "metric" in the
// interface{} slice.
func purgeTimestamps(metrics []interface{}) {
	for _, m := range metrics {
		assertedM, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		delete(assertedM, "timestamp")
		delete(assertedM, "interval.ms")
	}
}
