// Package integration ..
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/newrelic/go-telemetry-sdk/cumulative"
	"github.com/newrelic/go-telemetry-sdk/telemetry"
)

const (
	// Ideally these 2 values should be configurable as integer multiples of the scrape interval.
	deltaExpirationAge            = 30 * time.Second
	deltaExpirationCheckInternval = 30 * time.Second
)

// Emitter is an interface representing the ability to emit metrics.
type Emitter interface {
	Name() string
	Emit([]Metric) error
}

// TelemetryEmitter emits metrics using the go-telemetry-sdk.
type TelemetryEmitter struct {
	name            string
	apiKey          string
	harvester       *telemetry.Harvester
	deltaCalculator *cumulative.DeltaCalculator
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip is the implementation for http.RoundTripper.
func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// CancelRequest is an optional interface required for go1.4 and go1.5
func (fn roundTripperFunc) CancelRequest(*http.Request) {}

// NewTelemetryEmitterWithRoundTripper returns a new TelemetryEmitter with the
// given http.RoundTripper.
func NewTelemetryEmitterWithRoundTripper(apiURL, apiKey string, rt roundTripperFunc) *TelemetryEmitter {
	dc := cumulative.NewDeltaCalculator()
	dc.SetExpirationAge(deltaExpirationAge)
	dc.SetExpirationCheckInterval(deltaExpirationCheckInternval)
	return &TelemetryEmitter{
		name:   "telemetry",
		apiKey: apiKey,
		harvester: telemetry.NewHarvester(
			func(cfg *telemetry.Config) {
				cfg.MetricsURLOverride = apiURL
				cfg.Client.Transport = rt
			},
			telemetry.ConfigAPIKey(apiKey),
			telemetry.ConfigBasicErrorLogger(os.Stdout)),
		deltaCalculator: dc,
	}
}

// NewTelemetryEmitter returns a new TelemetryEmitter.
func NewTelemetryEmitter(apiURL, apiKey string, harvestPeriod time.Duration) *TelemetryEmitter {
	dc := cumulative.NewDeltaCalculator()
	dc.SetExpirationAge(deltaExpirationAge)
	dc.SetExpirationCheckInterval(deltaExpirationCheckInternval)
	return &TelemetryEmitter{
		name:   "telemetry",
		apiKey: apiKey,
		harvester: telemetry.NewHarvester(
			func(cfg *telemetry.Config) {
				cfg.MetricsURLOverride = apiURL
				cfg.HarvestPeriod = harvestPeriod
				cfg.Client.Transport = newInfraTransport(cfg.Client.Transport, apiKey)
			},
			telemetry.ConfigAPIKey(apiKey),
			telemetry.ConfigBasicErrorLogger(os.Stdout)),
		deltaCalculator: dc,
	}
}

// Name returns the emitter name.
func (te *TelemetryEmitter) Name() string {
	return te.name
}

// Emit makes the mapping between Prometheus and NR metrics and records them
// into the NR telemetry harvester.
func (te *TelemetryEmitter) Emit(metrics []Metric) error {
	for _, metric := range metrics {
		switch metric.metricType {
		case metricType_GAUGE:
			te.harvester.RecordMetric(telemetry.Gauge{
				Name:       metric.name,
				Attributes: metric.attributes,
				Value:      metric.value.(float64),
				Timestamp:  time.Now(),
			})
		case metricType_COUNTER:
			if m, ok := te.deltaCalculator.CountMetric(metric.name, metric.attributes, metric.value.(float64), time.Now()); ok {
				te.harvester.RecordMetric(m)
			}
		}
	}
	return nil
}

// StdoutEmitter emits metrics to stdout.
type StdoutEmitter struct {
	name string
}

// NewStdoutEmitter returns a NewStdoutEmitter.
func NewStdoutEmitter() *StdoutEmitter {
	return &StdoutEmitter{
		name: "stdout",
	}
}

// Name is the StdoutEmitter name.
func (se *StdoutEmitter) Name() string {
	return se.name
}

// Emit prints the metrics into stdout.
func (se *StdoutEmitter) Emit(metrics []Metric) error {
	b, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
