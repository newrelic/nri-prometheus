package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/cumulative"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
)

// Harvester aggregates and reports metrics and spans
type harvester interface {
	RecordMetric(m telemetry.Metric)
	HarvestNow(ct context.Context)
}

// TelemetryEmitter emits metrics using the go-telemetry-sdk.
type TelemetryEmitter struct {
	name            string
	harvester       harvester
	deltaCalculator *cumulative.DeltaCalculator
}

// TelemetryEmitterConfig is the configuration required for the
// `TelemetryEmitter`
type TelemetryEmitterConfig struct {
	// HarvesterOpts configuration functions for the telemetry Harvester.
	HarvesterOpts []TelemetryHarvesterOpt

	// DeltaExpirationAge sets the cumulative DeltaCalculator expiration age
	// which determines how old an entry must be before it is considered for
	// expiration. Defaults to 30s.
	DeltaExpirationAge time.Duration
	// DeltaExpirationCheckInternval sets the cumulative DeltaCalculator
	// duration between checking for expirations. Defaults to 30s.
	DeltaExpirationCheckInternval time.Duration

	// boundedHarvester configuration
	DisableBoundedHarvester bool
	BoundedHarvesterCfg
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
func telemetryHarvesterZeroPeriod(config *telemetry.Config) {
	config.HarvestPeriod = 0
}

// TelemetryHarvesterWithLicenseKeyRoundTripper wraps the emitter
// client Transport to use the `licenseKey` instead of the `apiKey`.
//
// Other options that modify the underlying Client.Transport should be
// set before this one, because this will change the Transport type
// to licenseKeyRoundTripper.
func TelemetryHarvesterWithLicenseKeyRoundTripper(licenseKey string) TelemetryHarvesterOpt {
	return func(cfg *telemetry.Config) {
		cfg.Client.Transport = newLicenseKeyRoundTripper(
			cfg.Client.Transport,
			licenseKey,
		)
	}
}

// TelemetryHarvesterWithTLSConfig sets the TLS configuration to the
// emitter client transport.
func TelemetryHarvesterWithTLSConfig(tlsConfig *tls.Config) TelemetryHarvesterOpt {

	return func(cfg *telemetry.Config) {
		rt := cfg.Client.Transport
		if rt == nil {
			rt = http.DefaultTransport
		}

		t, ok := rt.(*http.Transport)
		if !ok {
			logrus.Warning(
				"telemetry emitter TLS configuration couldn't be set, ",
				"client transport is not an http.Transport.",
			)
			return
		}

		t = t.Clone()
		t.TLSClientConfig = tlsConfig
		cfg.Client.Transport = http.RoundTripper(t)
		return
	}
}

// TelemetryHarvesterWithProxy sets proxy configuration to the emitter
// client transport.
func TelemetryHarvesterWithProxy(proxyURL *url.URL) TelemetryHarvesterOpt {
	return func(cfg *telemetry.Config) {
		rt := cfg.Client.Transport
		if rt == nil {
			rt = http.DefaultTransport
		}

		t, ok := rt.(*http.Transport)
		if !ok {
			logrus.Warning(
				"telemetry emitter couldn't be configured with proxy, ",
				"client transport is not an http.Transport, ",
				"continuing without proxy support",
			)
			return
		}

		t = t.Clone()
		t.Proxy = http.ProxyURL(proxyURL)
		cfg.Client.Transport = http.RoundTripper(t)
		return
	}
}

// NewTelemetryEmitter returns a new TelemetryEmitter.
func NewTelemetryEmitter(cfg TelemetryEmitterConfig) (*TelemetryEmitter, error) {
	dc := cumulative.NewDeltaCalculator()

	deltaExpirationAge := defaultDeltaExpirationAge
	if cfg.DeltaExpirationAge != 0 {
		deltaExpirationAge = cfg.DeltaExpirationAge
	}
	dc.SetExpirationAge(deltaExpirationAge)
	logrus.Debugf(
		"telemetry emitter configured with delta counter expiration age: %s",
		deltaExpirationAge,
	)

	deltaExpirationCheckInterval := defaultDeltaExpirationCheckInterval
	if cfg.DeltaExpirationCheckInternval != 0 {
		deltaExpirationCheckInterval = cfg.DeltaExpirationCheckInternval
	}
	dc.SetExpirationCheckInterval(deltaExpirationCheckInterval)
	logrus.Debugf(
		"telemetry emitter configured with delta counter expiration check interval: %s",
		deltaExpirationCheckInterval,
	)

	var h harvester
	h, err := telemetry.NewHarvester(append(cfg.HarvesterOpts, telemetryHarvesterZeroPeriod)...)
	if err != nil {
		return nil, errors.Wrap(err, "could not create new Harvester")
	}

	if !cfg.DisableBoundedHarvester {
		// Create a bound harvester based on passed configuration if going to run in a loop
		h, _ = bindHarvester(h, cfg.BoundedHarvesterCfg)
	}

	// Wrap the harvester so we can filter out invalid float values: NaN and Infinity.
	// If we do send them, the harvester will always output these as errors
	h = harvesterDecorator{h}

	return &TelemetryEmitter{
		name:            "telemetry",
		harvester:       h,
		deltaCalculator: dc,
	}, nil
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

func (te *TelemetryEmitter) emitSummary(metric Metric, timestamp time.Time) error {
	summary, ok := metric.value.(*dto.Summary)
	if !ok {
		return fmt.Errorf("unknown summary metric type for %q: %T", metric.name, metric.value)
	}

	if sumCount, ok := te.deltaCalculator.CountMetric(metric.name+"_sum", metric.attributes, float64(summary.GetSampleSum()), timestamp); ok {
		te.harvester.RecordMetric(telemetry.Summary{
			Name:       metric.name + "_sum",
			Attributes: metric.attributes,
			Count:      1,
			Sum:        sumCount.Value,
			Min:        math.NaN(),
			Max:        math.NaN(),
			Timestamp:  timestamp,
		})
	}

	if count, ok := te.deltaCalculator.CountMetric(metric.name+"_count", metric.attributes, float64(summary.GetSampleCount()), timestamp); ok {
		te.harvester.RecordMetric(count)
	}

	quantiles := summary.GetQuantile()
	for _, q := range quantiles {
		quantileAttrs := copyAttrs(metric.attributes)
		quantileAttrs["quantile"] = fmt.Sprintf("%g", q.GetQuantile())
		te.harvester.RecordMetric(telemetry.Gauge{
			Name:       metric.name,
			Attributes: quantileAttrs,
			Value:      q.GetValue(),
			Timestamp:  timestamp,
		})
	}
	return nil
}

func (te *TelemetryEmitter) emitHistogram(metric Metric, timestamp time.Time) error {
	hist, ok := metric.value.(*dto.Histogram)
	if !ok {
		return fmt.Errorf("unknown histogram metric type for %q: %T", metric.name, metric.value)
	}

	if sumCount, ok := te.deltaCalculator.CountMetric(metric.name+"_sum", metric.attributes, float64(hist.GetSampleSum()), timestamp); ok {
		te.harvester.RecordMetric(telemetry.Summary{
			Name:       metric.name + "_sum",
			Attributes: metric.attributes,
			Count:      1,
			Sum:        sumCount.Value,
			Min:        math.NaN(),
			Max:        math.NaN(),
			Timestamp:  timestamp,
		})
	}

	if count, ok := te.deltaCalculator.CountMetric(metric.name+"_count", metric.attributes, float64(hist.GetSampleCount()), timestamp); ok {
		te.harvester.RecordMetric(count)
	}

	metricName := metric.name + "_bucket"
	for _, b := range hist.GetBucket() {
		bucketAttrs := copyAttrs(metric.attributes)
		bucketAttrs["le"] = fmt.Sprintf("%g", b.GetUpperBound())

		bucketCount, ok := te.deltaCalculator.CountMetric(
			metricName,
			bucketAttrs,
			float64(b.GetCumulativeCount()),
			timestamp,
		)
		if ok {
			te.harvester.RecordMetric(bucketCount)
		}
	}

	return nil
}
