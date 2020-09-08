package integration

import (
	"fmt"
	"net/url"
	"time"

	metrics "github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
)

// InfraSdkEmitter is the emitter using the infra sdk to output metrics to stdout
type InfraSdkEmitter struct {
	definitions Specs
}

// NewInfraSdkEmitter creates a new Infra SDK emitter
func NewInfraSdkEmitter(specs Specs) *InfraSdkEmitter {
	return &InfraSdkEmitter{definitions: specs}
}

// Name is the InfraSdkEmitter name.
func (e *InfraSdkEmitter) Name() string {
	return "infra-sdk"
}

// Emit emits the metrics using the infra sdk
func (e *InfraSdkEmitter) Emit(metrics []Metric) error {
	// instrumentation name and version
	i, err := integration.New(Name, Version)
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

	return i.Publish()
}

func (e *InfraSdkEmitter) emitGauge(i *integration.Integration, metric Metric, timestamp time.Time) error {
	m, err := integration.Gauge(timestamp, metric.name, metric.value.(float64))
	if err != nil {
		return err
	}
	addDimensions(m, metric.attributes)

	return e.addMetricToEntity(i, metric, m)
}

func (e *InfraSdkEmitter) emitCounter(i *integration.Integration, metric Metric, timestamp time.Time) error {
	m, err := integration.Count(timestamp, metric.name, metric.value.(float64))
	if err != nil {
		return err
	}
	addDimensions(m, metric.attributes)

	return e.addMetricToEntity(i, metric, m)
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

	return e.addMetricToEntity(i, metric, ph)
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

	return e.addMetricToEntity(i, metric, ps)
}

func (e *InfraSdkEmitter) addMetricToEntity(i *integration.Integration, metric Metric, m metrics.Metric) error {
	baseEntityName, entityType, err := e.definitions.getEntity(metric)
	// if we can't find an entity for the metric, add it to the "host" entity
	if err != nil {
		logrus.WithError(err).Debugf("failed to map metric to entity. using 'host' entity")
		i.HostEntity.AddMetric(m)
		return nil
	}
	entityName := buildEntityName(baseEntityName, m)
	// try to find the entity and add the metric to it
	// if there's no entity with the same name yet, create it and add it to the integration
	entity, ok := i.FindEntity(entityName)
	if !ok {
		entity, err = i.NewEntity(entityName, entityType, entityName)
		if err != nil {
			logrus.WithError(err).Errorf("failed to create entity %v", entityName)
			return err
		}
		i.AddEntity(entity)
	}

	entity.AddMetric(m)
	return nil
}

// adds target host as part of the name in order to differentiate between instances of the "same" entity
// for example in a cluster scenario, each "node" would have the same metrics but will be a distinct entity so
// so we need the ip/hostname to be able to make them distinct
func buildEntityName(baseEntityName string, m metrics.Metric) string {
	tn := m.Dimension("scrapedTargetURL")
	if tn != "" {
		u, err := url.Parse(tn)
		if err != nil {
			logrus.WithError(err).Warnf("'scrapedTargetURL' metric dimension is not a proper URL")
			return baseEntityName
		}
		return baseEntityName + ":" + u.Host
	}
	return baseEntityName
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
