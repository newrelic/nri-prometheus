package integration

import (
	"context"
	"sync"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/sirupsen/logrus"
)

const boundHarvesterDefaultPeriod = 3 * time.Second
const boundHarvesterDefaultMetricCap = 1e5
const boundHarvesterDefaultMinReportInterval = 100 * time.Millisecond

// bindHarvester creates a boundedHarvester from an existing harvester.
// It also returns a cancel channel to stop the periodic harvest goroutine.
func bindHarvester(inner harvester, cfg BoundedHarvesterCfg) (harvester, chan struct{}) {
	if _, ok := inner.(*telemetry.Harvester); ok {
		logrus.Debug("using telemetry.Harvester as underlying harvester, make sure to set HarvestPeriod to 0")
	}

	if cfg.HarvestPeriod == 0 {
		cfg.HarvestPeriod = boundHarvesterDefaultPeriod
	}

	if cfg.MetricCap == 0 {
		cfg.MetricCap = boundHarvesterDefaultMetricCap
	}

	if cfg.MinReportInterval == 0 {
		cfg.MinReportInterval = boundHarvesterDefaultMinReportInterval
	}

	h := &boundedHarvester{
		BoundedHarvesterCfg: cfg,
		mtx:                 sync.Mutex{},
		inner:               inner,
	}

	cancel := make(chan struct{})
	go h.periodicHarvest(cancel)
	return h, cancel
}

// BoundedHarvesterCfg stores the configurable values for boundedHarvester
type BoundedHarvesterCfg struct {
	// Never report more often than MinReportInterval
	MinReportInterval time.Duration

	// Report when the number of stored metrics is greater than MetricCap
	MetricCap int
	// Also report at least once every HarvestPeriod
	HarvestPeriod time.Duration
}

// boundedHarvester wraps another harvester and triggers its HarvestNow operation when a number of metrics have been
// collected, or periodically every HarvestPeriod.
// Additionally, it ensures that reports do not happen more often than MinReportInterval
type boundedHarvester struct {
	BoundedHarvesterCfg

	mtx sync.Mutex

	reportedMetrics int
	lastReport      time.Time

	inner harvester
}

// RecordMetric records the metric in the underlying harvester and reports all of them if needed
func (h *boundedHarvester) RecordMetric(m telemetry.Metric) {
	h.inner.RecordMetric(m)
	h.reportIfNeeded(context.Background(), 1, false)
}

// HarvestNow forces a new report
func (h *boundedHarvester) HarvestNow(ctx context.Context) {
	h.reportIfNeeded(ctx, 0, true)
}

// reportIfNeeded carries the logic to report metrics.
// A report is triggered if:
// - Force is set to true, or
// - Last report occurred earlier than Now() - HarvestPeriod, or
// - The number of metrics is above MetricCap and MinReportInterval has passed since last report
func (h *boundedHarvester) reportIfNeeded(ctx context.Context, newMetrics int, force bool) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	h.reportedMetrics += newMetrics
	if force ||
		time.Since(h.lastReport) >= h.HarvestPeriod ||
		(h.reportedMetrics > h.MetricCap && time.Since(h.lastReport) > h.MinReportInterval) {

		h.lastReport = time.Now()
		h.reportedMetrics = 0
		h.inner.HarvestNow(ctx)
	}
}

// periodicHarvest can be run in a separate goroutine to periodically call reportIfNeeded every HarvestPeriod
func (h *boundedHarvester) periodicHarvest(cancel chan struct{}) {
	t := time.NewTicker(h.HarvestPeriod)
	for {
		select {
		case <-cancel:
			t.Stop()
			return
		case <-t.C:
			h.reportIfNeeded(context.Background(), 0, false)
		}
	}
}
