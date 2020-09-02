package integration

import (
	"fmt"
	"time"

	metrics "github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/newrelic-telemetry-sdk-go/cumulative"
	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
)

// InfraSdkEmitter is the emitter using the infra sdk to output metrics to stdout
type InfraSdkEmitter struct {
	deltaCalculator *cumulative.DeltaCalculator
}

// NewInfraSdkEmitter creates a new Infra SDK emitter
func NewInfraSdkEmitter() (*InfraSdkEmitter, error) {
	return &InfraSdkEmitter{
		// perhaps allow configuration..
		deltaCalculator: cumulative.NewDeltaCalculator(),
	}, nil
}

// Name is the InfraSdkEmitter name.
func (e *InfraSdkEmitter) Name() string {
	return "infra-sdk"
}

// Emit emits the metrics using the infra sdk
func (e *InfraSdkEmitter) Emit(metrics []Metric) error {
	//TODO this may be moved to the main package. waiting on the ohi-ified version
	i, err := integration.New("com.newrelic.prometheus", "1.0.0")
	if err != nil {
		return err
	}

	now := time.Now()
	for _, me := range metrics {
		switch me.metricType {
		case metricType_GAUGE:
			err = e.emitGauge(i, me, now)
			break
		case metricType_COUNTER:
			err = e.emitCounter(i, me, now)
			break
		case metricType_SUMMARY:
			err = e.emitSummary(i, me, now)
			break
		case metricType_HISTOGRAM:
			err = e.emitHistogram(i, me, now)
			break
		default:
			err = fmt.Errorf("unknown metric type %q", me.metricType)
		}

		if err != nil {
			logrus.WithError(err).Errorf("failed to create metric from '%s'", me.name)
		}
	}

	err = i.Publish()
	return err
}

func (e *InfraSdkEmitter) emitGauge(i *integration.Integration, metric Metric, timestamp time.Time) error {
	m, err := integration.Gauge(timestamp, metric.name, metric.value.(float64))
	if err != nil {
		return err
	}
	addDimensions(m, metric.attributes)
	i.HostEntity.AddMetric(m)

	return nil
}

func (e *InfraSdkEmitter) emitCounter(i *integration.Integration, metric Metric, timestamp time.Time) error {
	m, err := integration.Count(timestamp, metric.name, metric.value.(float64))
	if err != nil {
		return err
	}
	addDimensions(m, metric.attributes)
	i.HostEntity.AddMetric(m)

	return nil
}

func (e *InfraSdkEmitter) emitHistogram(i *integration.Integration, metric Metric, timestamp time.Time) error {
	hist, ok := metric.value.(*dto.Histogram)
	if !ok {
		return fmt.Errorf("unknown histogram metric type for %q: %T", metric.name, metric.value)
	}

	ph, err := metrics.NewPrometheusHistogram(timestamp, metric.name, *hist.SampleCount, *hist.SampleSum)
	if err != nil {
		return fmt.Errorf("failed to create histogram metric for %q", metric.name)
	}
	addDimensions(ph, metric.attributes)

	buckets := hist.Bucket
	for _, b := range buckets {
		ph.AddBucket(*b.CumulativeCount, *b.UpperBound)
	}

	i.HostEntity.AddMetric(ph)

	return nil
}

func (e *InfraSdkEmitter) emitSummary(i *integration.Integration, metric Metric, timestamp time.Time) error {
	summary, ok := metric.value.(*dto.Summary)
	if !ok {
		return fmt.Errorf("unknown summary metric type for %q: %T", metric.name, metric.value)
	}

	ps, err := metrics.NewPrometheusSummary(timestamp, metric.name, *summary.SampleCount, *summary.SampleSum)
	if err != nil {
		return fmt.Errorf("failed to create summary metric for %q", metric.name)
	}
	addDimensions(ps, metric.attributes)

	quantiles := summary.GetQuantile()
	for _, q := range quantiles {
		ps.AddQuantile(*q.Quantile, *q.Value)
	}

	i.HostEntity.AddMetric(ps)

	return nil
}

func addDimensions(m metrics.Metric, attributes labels.Set) {
	var err error
	for k, v := range attributes {
		err = m.AddDimension(k, v.(string))
		if err != nil {
			logrus.WithError(err).Warnf("failed to add attribute %v(%v) as dimension to metric", k, v)
		}
	}
}
