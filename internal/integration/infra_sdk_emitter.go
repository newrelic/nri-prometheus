package integration

import (
	"fmt"
	"math"
	"time"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
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

	if sumDelta, ok := e.deltaCalculator.CountMetric(metric.name+"_sum", metric.attributes, hist.GetSampleSum(), timestamp); ok {
		m, err := integration.Summary(timestamp, metric.name+"_sum",
			1,
			math.NaN(), // can we calc average here?
			sumDelta.Value,
			math.NaN(),
			math.NaN(),
		)
		if err != nil {
			return err
		}
		addDimensions(m, metric.attributes)
		i.HostEntity.AddMetric(m)
	}

	m, err := integration.Count(timestamp, metric.name+"_count", float64(hist.GetSampleCount()))
	if err != nil {
		return err
	}
	addDimensions(m, metric.attributes)
	i.HostEntity.AddMetric(m)

	metricName := metric.name + "_bucket"
	for _, b := range hist.GetBucket() {
		m, err = integration.Count(timestamp, metricName, float64(b.GetCumulativeCount()))
		if err != nil {
			return err
		}
		addDimensions(m, metric.attributes)
		_ = m.AddDimension("le", fmt.Sprintf("%g", b.GetUpperBound()))

		i.HostEntity.AddMetric(m)
	}
	return nil
}

func (e *InfraSdkEmitter) emitSummary(i *integration.Integration, metric Metric, timestamp time.Time) (err error) {
	summary, ok := metric.value.(*dto.Summary)
	if !ok {
		return fmt.Errorf("unknown summary metric type for %q: %T", metric.name, metric.value)
	}

	m, err := integration.Summary(timestamp, metric.name+"_sum",
		1,
		math.NaN(),             // can we calc average here?
		summary.GetSampleSum(), // not sure the delta gets calculated in the agent for summary metrics
		math.NaN(),
		math.NaN(),
	)
	if err != nil {
		return err
	}
	addDimensions(m, metric.attributes)
	i.HostEntity.AddMetric(m)
	// we use count here to let the agent calculate the delta, so in effect it will be send to NR as a delta
	m, err = integration.Count(timestamp, metric.name+"_count", float64(summary.GetSampleCount()))
	if err != nil {
		return err
	}
	addDimensions(m, metric.attributes)
	i.HostEntity.AddMetric(m)

	quantiles := summary.GetQuantile()
	for _, q := range quantiles {
		m, err := integration.Gauge(timestamp, metric.name, q.GetValue())
		if err != nil {
			return err
		}
		addDimensions(m, metric.attributes)
		_ = m.AddDimension("quantile", fmt.Sprintf("%g", q.GetQuantile()))
		i.HostEntity.AddMetric(m)
	}
	return nil
}

func addDimensions(m metric.Metric, attributes labels.Set) {
	var err error
	for k, v := range attributes {
		err = m.AddDimension(k, v.(string))
		if err != nil {
			logrus.WithError(err).Warnf("failed to add attribute %v(%v) as dimension to metric", k, v)
		}
	}
}
