// Package integration ..
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/newrelic/go-telemetry-sdk/cumulative"
	"github.com/newrelic/go-telemetry-sdk/telemetry"
	"github.com/sirupsen/logrus"
)

const (
	// Ideally these 2 values should be configurable as integer multiples of the scrape interval.
	defaultDeltaExpirationAge           = 30 * time.Second
	defaultDeltaExpirationCheckInterval = 30 * time.Second
)

// Emitter is an interface representing the ability to emit metrics.
type Emitter interface {
	Name() string
	Emit([]Metric) error
}

// TelemetryEmitter emits metrics using the go-telemetry-sdk.
type TelemetryEmitter struct {
	name            string
	percentiles     []float64
	harvester       *telemetry.Harvester
	deltaCalculator *cumulative.DeltaCalculator
}

type TelemetryEmitterConfig struct {
	// Percentile values to calculate for every Prometheus metrics of histogram type.
	Percentiles []float64

	// HarvesterOpts configuration functions for the telemetry Harvester.
	HarvesterOpts []TelemetryHarvesterOpt

	// DeltaExpirationAge sets the cumulative DeltaCalculator expiration age
	// which determines how old an entry must be before it is considered for
	// expiration. Defaults to 30s.
	DeltaExpirationAge time.Duration
	// DeltaExpirationCheckInternval sets the cumulative DeltaCalculator
	// duration between checking for expirations. Defaults to 30s.
	DeltaExpirationCheckInternval time.Duration
}

// TelemetryHarvesterOpt sets configuration options for the
// `TelemetryEmitter`'s `telemetry.Harvester`.
type TelemetryHarvesterOpt = func(*telemetry.Config)

// TelemetryHarvesterWithMetricsURL sets the url to use for the metrics endpoint.
func TelemetryHarvesterWithMetricsURL(url string) TelemetryHarvesterOpt {
	return func(config *telemetry.Config) {
		config.MetricsURLOverride = url
	}
}

// TelemetryHarvesterWithHarvestPeriod sets harvest period.
func TelemetryHarvesterWithHarvestPeriod(t time.Duration) TelemetryHarvesterOpt {
	return func(config *telemetry.Config) {
		config.HarvestPeriod = t
	}
}

// TelemetryHarvesterWithInfraTransport wraps the `telemetry.Harvester`
// `Transport` so that it uses the `licenseKey` instead of the `apiKey`.
func TelemetryHarvesterWithInfraTransport(licenseKey string) TelemetryHarvesterOpt {
	return func(cfg *telemetry.Config) {
		cfg.Client.Transport = newInfraTransport(cfg.Client.Transport, licenseKey)
	}
}

// NewTelemetryEmitter returns a new TelemetryEmitter.
func NewTelemetryEmitter(cfg TelemetryEmitterConfig) *TelemetryEmitter {
	dc := cumulative.NewDeltaCalculator()

	if cfg.DeltaExpirationAge != 0 {
		dc.SetExpirationAge(cfg.DeltaExpirationAge)
	} else {
		dc.SetExpirationAge(defaultDeltaExpirationAge)
	}
	logrus.Debugf(
		"telemetry emitter configured with delta counter expiration age: %s",
		cfg.DeltaExpirationAge,
	)

	if cfg.DeltaExpirationCheckInternval != 0 {
		dc.SetExpirationCheckInterval(cfg.DeltaExpirationCheckInternval)
	} else {
		dc.SetExpirationCheckInterval(defaultDeltaExpirationCheckInterval)
	}
	logrus.Debugf(
		"telemetry emitter configured with delta counter expiration check interval: %s",
		cfg.DeltaExpirationAge,
	)

	return &TelemetryEmitter{
		name:            "telemetry",
		harvester:       telemetry.NewHarvester(cfg.HarvesterOpts...),
		percentiles:     cfg.Percentiles,
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
