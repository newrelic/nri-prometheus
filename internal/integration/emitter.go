// Package integration ..
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/newrelic/go-telemetry-sdk/cumulative"
	"github.com/newrelic/go-telemetry-sdk/telemetry"
	"github.com/newrelic/nri-prometheus/internal/histogram"
	mpb "github.com/prometheus/client_model/go"
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

// TelemetryEmitterConfig is the configuration required for the
// `TelemetryEmitter`
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
	var results error

	// Record metrics at a uniform time so processing is not reflected in
	// the measurement that already took place.
	now := time.Now()
	for _, metric := range metrics {
		switch metric.metricType {
		case metricType_GAUGE:
			te.harvester.RecordMetric(telemetry.Gauge{
				Name:       metric.name,
				Attributes: metric.attributes,
				Value:      metric.value.(float64),
				Timestamp:  now,
			})
		case metricType_COUNTER:
			m, ok := te.deltaCalculator.CountMetric(
				metric.name,
				metric.attributes,
				metric.value.(float64),
				now,
			)
			if ok {
				te.harvester.RecordMetric(m)
			}
		case metricType_SUMMARY:
			if err := te.emitSummary(metric, now); err != nil {
				if results == nil {
					results = err
				} else {
					results = fmt.Errorf("%v: %w", err, results)
				}
			}
		case metricType_HISTOGRAM:
			if err := te.emitHistogram(metric, now); err != nil {
				if results == nil {
					results = err
				} else {
					results = fmt.Errorf("%v: %w", err, results)
				}
			}
		default:
			if err := fmt.Errorf("unknown metric type %q", metric.metricType); err != nil {
				if results == nil {
					results = err
				} else {
					results = fmt.Errorf("%v: %w", err, results)
				}
			}
		}
	}
	return results
}

// emitSummary sends all quantiles included with the summary as percentiles to New Relic.
//
// Related specification:
// https://github.com/newrelic/newrelic-exporter-specs/blob/master/Guidelines.md#percentiles
func (te *TelemetryEmitter) emitSummary(metric Metric, timestamp time.Time) error {
	summary, ok := metric.value.(*mpb.Summary)
	if !ok {
		return fmt.Errorf("unknown summary metric type for %q: %T", metric.name, metric.value)
	}

	var results error
	metricName := metric.name + ".percentiles"
	quantiles := summary.GetQuantile()
	for _, q := range quantiles {
		// translate to percentiles
		p := q.GetQuantile() * 100.0
		if p < 0.0 || p > 100.0 {
			err := fmt.Errorf("invalid percentile `%g` for %s: must be in range [0.0, 100.0]", p, metric.name)
			if results == nil {
				results = err
			} else {
				results = fmt.Errorf("%v: %w", err, results)
			}
			continue
		}

		v := q.GetValue()
		if !validNRValue(v) {
			err := fmt.Errorf("invalid percentile value for %s: %g", metric.name, v)
			if results == nil {
				results = err
			} else {
				results = fmt.Errorf("%v: %w", err, results)
			}
			continue
		}

		percentileAttrs := copyAttrs(metric.attributes)
		percentileAttrs["percentile"] = p
		te.harvester.RecordMetric(telemetry.Gauge{
			Name:       metricName,
			Attributes: percentileAttrs,
			Value:      v,
			Timestamp:  timestamp,
		})
	}
	return results
}

// emitHistogram sends histogram data and curated percentiles to New Relic.
//
// Related specification:
// https://github.com/newrelic/newrelic-exporter-specs/blob/master/Guidelines.md#histograms
func (te *TelemetryEmitter) emitHistogram(metric Metric, timestamp time.Time) error {
	hist, ok := metric.value.(*mpb.Histogram)
	if !ok {
		return fmt.Errorf("unknown histogram metric type for %q: %T", metric.name, metric.value)
	}

	if validNRValue(hist.GetSampleSum()) {
		if m, ok := te.deltaCalculator.CountMetric(metric.name+".sum", metric.attributes, hist.GetSampleSum(), timestamp); ok {
			te.harvester.RecordMetric(m)
		}
	}

	metricName := metric.name + ".buckets"
	buckets := make(histogram.Buckets, 0, len(hist.Bucket))
	for _, b := range hist.GetBucket() {
		upperBound := b.GetUpperBound()
		count := float64(b.GetCumulativeCount())
		if !math.IsInf(upperBound, 1) && validNRValue(count) {
			bucketAttrs := copyAttrs(metric.attributes)
			bucketAttrs["histogram.bucket.upperBound"] = upperBound
			if m, ok := te.deltaCalculator.CountMetric(metricName, bucketAttrs, count, timestamp); ok {
				te.harvester.RecordMetric(m)
			}
		}
		buckets = append(
			buckets,
			histogram.Bucket{
				UpperBound: upperBound,
				Count:      count,
			},
		)
	}

	var results error
	metricName = metric.name + ".percentiles"
	for _, p := range te.percentiles {
		v, err := histogram.Percentile(p, buckets)
		if err != nil {
			if results == nil {
				results = err
			} else {
				results = fmt.Errorf("%v: %w", err, results)
			}
			continue
		}

		if !validNRValue(v) {
			err := fmt.Errorf("invalid percentile value for %s: %g", metric.name, v)
			if results == nil {
				results = err
			} else {
				results = fmt.Errorf("%v: %w", err, results)
			}
			continue
		}

		percentileAttrs := copyAttrs(metric.attributes)
		percentileAttrs["percentile"] = p
		te.harvester.RecordMetric(telemetry.Gauge{
			Name:       metricName,
			Attributes: percentileAttrs,
			Value:      v,
			Timestamp:  timestamp,
		})
	}

	return results
}

// copyAttrs returns a (shallow) copy of the passed attrs.
func copyAttrs(attrs map[string]interface{}) map[string]interface{} {
	duplicate := make(map[string]interface{}, len(attrs))
	for k, v := range attrs {
		duplicate[k] = v
	}
	return duplicate
}

// validNRValue returns if v is a New Relic metric supported float64.
func validNRValue(v float64) bool {
	return !math.IsInf(v, 0) && !math.IsNaN(v)
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
