// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	infra "github.com/newrelic/infra-integrations-sdk/v4/data/metric"
	sdk "github.com/newrelic/infra-integrations-sdk/v4/integration"
	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
)

// A different regex is needed for replacing because `localhostRE` matches
// IPV6 by using extra `:` that don't belong to the IP but are separators.
var localhostReplaceRE = regexp.MustCompile(`(localhost|LOCALHOST|127(?:\.[0-9]+){0,2}\.[0-9]+|::1)`)

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
	integrationMetadata Metadata
	hostID              string
}

// Metadata contains the name and version of the exporter that is being scraped.
// The Infra-Agent use the metadata to populate instrumentation.name and instrumentation.value
type Metadata struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

func (im *Metadata) isValid() bool {
	return im.Name != "" && im.Version != ""
}

// NewInfraSdkEmitter creates a new Infra SDK emitter
func NewInfraSdkEmitter(hostID string) *InfraSdkEmitter {
	return &InfraSdkEmitter{
		// By default it uses the nri-prometheus and it version.
		integrationMetadata: Metadata{
			Name:    Name,
			Version: Version,
		},
		hostID: hostID,
	}
}

// SetIntegrationMetadata overrides integrationMetadata.
func (e *InfraSdkEmitter) SetIntegrationMetadata(integrationMetadata Metadata) error {
	if !integrationMetadata.isValid() {
		return fmt.Errorf("invalid integration metadata")
	}
	e.integrationMetadata = integrationMetadata
	return nil
}

// Name is the InfraSdkEmitter name.
func (e *InfraSdkEmitter) Name() string {
	return "infra-sdk"
}

// Emit emits the metrics using the infra sdk
func (e *InfraSdkEmitter) Emit(metrics []Metric) error {
	// create new Infra sdk Integration
	i, err := sdk.New(e.integrationMetadata.Name, e.integrationMetadata.Version)
	if err != nil {
		return err
	}
	// We want the agent to not send metrics attached to any entity in order to make the entity synthesis to take place
	// completely in the backend. Since V4 SDK still needs an entity (Dataset) to attach the metrics to, we are using
	// the default hostEntity to attach all the metrics to it but setting this flag, IgnoreEntity: true that
	// will cause the agent to send them unattached to any entity
	i.HostEntity.SetIgnoreEntity(true)

	now := time.Now()
	for _, me := range metrics {
		switch me.metricType {
		case metricType_GAUGE:
			err = e.emitGauge(i, me, now)
			break
		case metricType_COUNTER:
			err = e.emitCumulativeCounter(i, me, now)
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

// emitCumulativeCounter calls CumulativeCount that instead of Count, in this way in the agent the delta will be
// computed and reported instead of the absolute value
func (e *InfraSdkEmitter) emitCumulativeCounter(i *sdk.Integration, metric Metric, timestamp time.Time) error {
	m, err := sdk.CumulativeCount(timestamp, metric.name, metric.value.(float64))
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
	e.addDimensions(m, metric.attributes, i.HostEntity)
	i.HostEntity.AddMetric(m)
	return nil
}

func (e *InfraSdkEmitter) addDimensions(m infra.Metric, attributes labels.Set, entity *sdk.Entity) {
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
			if k == "scrapedTargetName" || k == "targetName" {
				value = replaceLocalhost(value, e.hostID)
			}
			entity.AddCommonDimension(k, value)
			continue
		}
		err := m.AddDimension(k, value)
		if err != nil {
			logrus.WithError(err).Warnf("failed to add attribute %v(%v) as dimension to metric", k, v)
		}
	}
}

// resizeToLimit makes sure that the entity name is less than the limit of 500
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

// ReplaceLocalhost replaces the occurrence of a localhost address with
// the given hostname
func replaceLocalhost(originalHost, hostID string) string {
	if hostID != "" {
		return localhostReplaceRE.ReplaceAllString(originalHost, hostID)
	}
	return originalHost
}
