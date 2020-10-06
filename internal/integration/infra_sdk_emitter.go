package integration

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	infra "github.com/newrelic/infra-integrations-sdk/data/metric"
	sdk "github.com/newrelic/infra-integrations-sdk/integration"
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
	logrus.Debugf("%d metrics not found in definition file and added to the Host Entity", len(i.HostEntity.Metrics))

	return i.Publish()
}

func (e *InfraSdkEmitter) emitGauge(i *sdk.Integration, metric Metric, timestamp time.Time) error {
	m, err := sdk.Gauge(timestamp, metric.name, metric.value.(float64))
	if err != nil {
		return err
	}
	addDimensions(m, metric.attributes)

	return e.addMetricToEntity(i, metric, m)
}

func (e *InfraSdkEmitter) emitCounter(i *sdk.Integration, metric Metric, timestamp time.Time) error {
	m, err := sdk.Count(timestamp, metric.name, metric.value.(float64))
	if err != nil {
		return err
	}
	addDimensions(m, metric.attributes)

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
	addDimensions(ph, metric.attributes)

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
	addDimensions(ps, metric.attributes)

	quantiles := summary.GetQuantile()
	for _, q := range quantiles {
		ps.AddQuantile(*q.Quantile, *q.Value)
	}

	return e.addMetricToEntity(i, metric, ps)
}

func (e *InfraSdkEmitter) addMetricToEntity(i *sdk.Integration, metric Metric, m infra.Metric) error {
	entityProps, err := e.definitions.getEntity(metric)
	// if we can't find an entity for the metric, add it to the "host" entity
	if err != nil {
		i.HostEntity.AddMetric(m)
		return nil
	}

	entityName := buildEntityName(entityProps, m)
	// try to find the entity and add the metric to it
	// if there's no entity with the same name yet, create it and add it to the integration
	entity, ok := i.FindEntity(entityName)
	if !ok {
		entity, err = i.NewEntity(entityName, entityProps.Type, entityProps.DisplayName)
		if err != nil {
			logrus.WithError(err).Errorf("failed to create entity %v", entityName)
			return err
		}
		i.AddEntity(entity)
	}

	entity.AddMetric(m)
	return nil
}

// build the entity name based on various properties
// the format should be as follows:
//  serviceName:exporterHost:exporterPort:entityName:dimension1:dimension2..
func buildEntityName(props entityNameProps, m infra.Metric) string {
	var sb strings.Builder

	sb.WriteString(props.Service)

	tn := m.Dimension("scrapedTargetURL")
	if tn != "" {
		u, err := url.Parse(tn)
		if err != nil {
			logrus.WithError(err).Warnf("'scrapedTargetURL' metric dimension is not a proper URL")
		} else {
			sb.WriteRune(':')
			sb.WriteString(u.Host)
		}
	}

	sb.WriteRune(':')
	sb.WriteString(props.Name)

	for _, v := range props.Dimensions {
		sb.WriteRune(':')
		sb.WriteString(v)
	}

	original := sb.String()
	// make sure entity name length is less than 500.
	resized := resizeToLimit(&sb)
	if resized {
		logrus.
			WithField("original", original).
			WithField("resized", sb.String()).
			Warn("entity was over the limit of '500' and has been resized")
	}

	return sb.String()
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

func addDimensions(m infra.Metric, attributes labels.Set) {
	var err error
	for k, v := range attributes {
		err = m.AddDimension(k, v.(string))
		if err != nil {
			logrus.WithError(err).Warnf("failed to add attribute %v(%v) as dimension to metric", k, v)
		}
	}
}
