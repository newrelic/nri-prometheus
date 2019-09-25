// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"

	"github.com/newrelic/go-telemetry-sdk/telemetry"
	"github.com/newrelic/nri-prometheus/internal/pkg/prometheus"
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
			TelemetryHarvesterWithMetricsURL("nilapiurl"),
			telemetry.ConfigBasicErrorLogger(os.Stdout),
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		emitter := NewTelemetryEmitter(c)
		err = emitter.Emit(superMetrics)
		assert.NoError(b, err)
		// Need to trigger a manual harvest here otherwise the benchmark is useless.
		emitter.harvester.HarvestNow()
	}
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
