package integration

import (
	"fmt"
	"strings"
	"time"

	infra "github.com/newrelic/infra-integrations-sdk/v4/data/metric"
	sdk "github.com/newrelic/infra-integrations-sdk/v4/integration"
	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	"github.com/newrelic/nri-prometheus/internal/synthesis"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
)

// Metric attributes that are shared by all metrics of an entity.
var commonAttributes = map[string]struct{}{
	"scrapedTargetKind": {},
	"scrapedTargetName": {},
	"scrapedTargetURL":  {},
	"targetName":        {},
}

// Metric attributes not needed for the infra-agent metrics pipeline.
var removedAttributes = map[string]struct{}{
	"nrMetricType":   {},
	"promMetricType": {},
}

// InfraSdkEmitter is the emitter using the infra sdk to output metrics to stdout
type InfraSdkEmitter struct {
	synthesisRules synthesis.Synthesizer
}

// NewInfraSdkEmitter creates a new Infra SDK emitter
func NewInfraSdkEmitter(synthesisRules synthesis.Synthesizer) *InfraSdkEmitter {
	return &InfraSdkEmitter{synthesisRules: synthesisRules}
}

// Name is the InfraSdkEmitter name.
func (e *InfraSdkEmitter) Name() string {
	return "infra-sdk"
}

// Emit emits the metrics using the infra sdk
func (e *InfraSdkEmitter) Emit(metrics []Metric) error {
	// create new Infra sdk Integration
	i, err := sdk.New(Name, Version)
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
	logrus.Debugf("%d metrics processed", len(metrics))

	return i.Publish()
}

func (e *InfraSdkEmitter) emitGauge(i *sdk.Integration, metric Metric, timestamp time.Time) error {
	m, err := sdk.Gauge(timestamp, metric.name, metric.value.(float64))
	if err != nil {
		return err
	}
	return e.addMetricToEntity(i, metric, m)
}

func (e *InfraSdkEmitter) emitCounter(i *sdk.Integration, metric Metric, timestamp time.Time) error {
	m, err := sdk.Count(timestamp, metric.name, metric.value.(float64))
	if err != nil {
		return err
	}
	return e.addMetricToEntity(i, metric, m)
}

func (e *InfraSdkEmitter) emitHistogram(i *sdk.Integration, metric Metric, timestamp time.Time) error {
	hist, ok := metric.value.(*dto.Histogram)
	if !ok {
		return fmt.Errorf("unknown histogram metric type for %q: %T", metric.name, metric.value)
	}

	ph, err := infra.NewPrometheusHistogram(timestamp, metric.name, *hist.SampleCount, *hist.SampleSum)
	if err != nil {
		return fmt.Errorf("failed to create histogram metric for %q", metric.name)
	}

	buckets := hist.Bucket
	for _, b := range buckets {
		ph.AddBucket(*b.CumulativeCount, *b.UpperBound)
	}

	return e.addMetricToEntity(i, metric, ph)
}

func (e *InfraSdkEmitter) emitSummary(i *sdk.Integration, metric Metric, timestamp time.Time) error {
	summary, ok := metric.value.(*dto.Summary)
	if !ok {
		return fmt.Errorf("unknown summary metric type for %q: %T", metric.name, metric.value)
	}

	ps, err := infra.NewPrometheusSummary(timestamp, metric.name, *summary.SampleCount, *summary.SampleSum)
	if err != nil {
		return fmt.Errorf("failed to create summary metric for %q", metric.name)
	}

	quantiles := summary.GetQuantile()
	for _, q := range quantiles {
		ps.AddQuantile(*q.Quantile, *q.Value)
	}

	return e.addMetricToEntity(i, metric, ps)
}

func (e *InfraSdkEmitter) addMetricToEntity(i *sdk.Integration, metric Metric, m infra.Metric) error {
	entityMetadata, found := e.synthesisRules.GetEntityMetadata(metric.name, metric.attributes)
	// if we can't find an entity for the metric, add it to the "host" entity
	if !found {
		addDimensions(m, metric.attributes, i.HostEntity)
		i.HostEntity.AddMetric(m)
		return nil
	}

	// try to find the entity and add the metric to it
	// if there's no entity with the same name yet, create it and add it to the integration
	entity, ok := i.FindEntity(entityMetadata.Name)
	if !ok {
		var err error
		entity, err = i.NewEntity(entityMetadata.Name, entityMetadata.EntityType, entityMetadata.DisplayName)
		if err != nil {
			logrus.WithError(err).Errorf("failed to create entity name:%s type:%s displayName:%s", entityMetadata.Name, entityMetadata.EntityType, entityMetadata.DisplayName)
			return err
		}
		i.AddEntity(entity)
	}
	// entity metadata could be dispersed on different metrics so we add found tags from each entity.
	for k, v := range entityMetadata.Metadata {
		if err := entity.AddMetadata(k, v); err != nil {
			logrus.WithError(err).Debugf("fail to add metadata k:%s v:%v ", k, v)
		}
	}
	addDimensions(m, metric.attributes, entity)

	entity.AddMetric(m)
	return nil
}

// resizeToLimit makes sure that the entity name is lee than the limit of 500
// it removed "full tokens" from the string so we don't get partial values in the name
func resizeToLimit(sb *strings.Builder) (resized bool) {
	if sb.Len() < 500 {
		return false
	}

	tokens := strings.Split(sb.String(), ":")
	sb.Reset()

	// add tokens until we get to the limit
	sb.WriteString(tokens[0])
	for _, t := range tokens[1:] {
		if sb.Len()+len(t)+1 >= 500 {
			resized = true
			break
		}
		sb.WriteRune(':')
		sb.WriteString(t)
	}
	return
}

func addDimensions(m infra.Metric, attributes labels.Set, entity *sdk.Entity) {
	var value string
	var ok bool
	for k, v := range attributes {
		if _, ok = removedAttributes[k]; ok {
			continue
		}
		if value, ok = v.(string); !ok {
			logrus.Debugf("the value (%v) of %s attribute should be a string", k, v)
			continue
		}
		if _, ok = commonAttributes[k]; ok {
			entity.AddCommonDimension(k, value)
			continue
		}
		err := m.AddDimension(k, value)
		if err != nil {
			logrus.WithError(err).Warnf("failed to add attribute %v(%v) as dimension to metric", k, v)
		}
	}
}
