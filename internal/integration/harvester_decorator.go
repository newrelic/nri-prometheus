package integration

import (
	"context"
	"math"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/sirupsen/logrus"
)

// harvesterDecorator is a layer on top of another harvester that filters out NaN and Infinite float values.
type harvesterDecorator struct {
	innerHarvester harvester
}

func (ha harvesterDecorator) RecordMetric(m telemetry.Metric) {
	switch a := m.(type) {
	case telemetry.Count:
		ha.processMetric(a.Value, m)
	case telemetry.Summary:
		ha.processMetric(a.Sum, m)
	case telemetry.Gauge:
		ha.processMetric(a.Value, m)
	default:
		logrus.Debugf("Unexpected metric in harvesterDecorator: #%v", m)
		ha.innerHarvester.RecordMetric(m)
	}
}

func (ha harvesterDecorator) HarvestNow(ctx context.Context) {
	ha.innerHarvester.HarvestNow(ctx)
}

func (ha harvesterDecorator) processMetric(f float64, m telemetry.Metric) {
	if math.IsNaN(f) {
		logrus.Debugf("Ignoring NaN float value for metric: %v", m)
		return
	}

	if math.IsInf(f, 0) {
		logrus.Debugf("Ignoring Infinite float value for metric: %v", m)
		return
	}

	ha.innerHarvester.RecordMetric(m)
}
