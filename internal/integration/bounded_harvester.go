package integration

import (
	"context"
	"sync"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	log "github.com/sirupsen/logrus"
)

const BoundedHarvesterDefaultPeriod = 5 * time.Second
const BoundedHarvesterDefaultMetricCap = 10000
const BoundedHarvesterDefaultMinReportInterval = 200 * time.Millisecond

// bindHarvester creates a boundedHarvester from an existing harvester.
// It also returns a cancel channel to stop the periodic harvest goroutine.
func bindHarvester(inner harvester, cfg BoundedHarvesterCfg) (harvester, chan struct{}) {
	if _, ok := inner.(*telemetry.Harvester); ok {
		log.Debug("using telemetry.Harvester as underlying harvester, make sure to set HarvestPeriod to 0")
	}

	if cfg.HarvestPeriod == 0 {
		cfg.HarvestPeriod = BoundedHarvesterDefaultPeriod
	}

	if cfg.MetricCap == 0 {
		cfg.MetricCap = BoundedHarvesterDefaultMetricCap
	}

	if cfg.MinReportInterval == 0 {
		cfg.MinReportInterval = BoundedHarvesterDefaultMinReportInterval
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
	// MetricCap is the number of metrics to store in memory before triggering a HarvestNow action regardless of
	// HarvestPeriod. It will directly influence the amount of memory that nri-prometheus allocates.
	// A value of 10000 is rougly equivalent to 500M in RAM in the tested scenarios
	MetricCap int

	// HarvestPeriod specifies the period that will trigger a HarvestNow action for the inner harvester.
	// It is not necessary to decrease this value further, as other conditions (namely the MetricCap) will also trigger a
	// harvest action.
	HarvestPeriod time.Duration

	// MinReportInterval Specifies the minimum amount of time to wait before reports.
	// This will be always enforced, regardless of HarvestPeriod and MetricCap
	MinReportInterval time.Duration
}

// boundedHarvester is a harvester implementation and wrapper that keeps count of the number of metrics that are waiting
// to be harvested. Every small period of time (BoundedHarvesterCfg.MinReportInterval), if the number of accumulated
// metrics is above a given threshold (BoundedHarvesterCfg.MetricCap), a harvest is triggered.
// A harvest is also triggered in periodic time intervals (BoundedHarvesterCfg.HarvestPeriod)
// boundedHarvester will never trigger harvests more often than specified in BoundedHarvesterCfg.MinReportInterval.
type boundedHarvester struct {
	BoundedHarvesterCfg

	mtx sync.Mutex

	storedMetrics int
	lastReport    time.Time

	inner harvester
}

// RecordMetric records the metric in the underlying harvester and reports all of them if needed
func (h *boundedHarvester) RecordMetric(m telemetry.Metric) {
	h.mtx.Lock()
	h.storedMetrics++
	h.mtx.Unlock()

	h.inner.RecordMetric(m)
}

// HarvestNow forces a new report
func (h *boundedHarvester) HarvestNow(ctx context.Context) {
	h.reportIfNeeded(ctx, true)
}

// reportIfNeeded carries the logic to report metrics.
// A report is triggered if:
// - Force is set to true, or
// - Last report occurred earlier than Now() - HarvestPeriod, or
// - The number of metrics is above MetricCap and MinReportInterval has passed since last report
// A report will not be triggered in any case if time since last harvest is less than MinReportInterval
func (h *boundedHarvester) reportIfNeeded(ctx context.Context, force bool) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if force ||
		time.Since(h.lastReport) >= h.HarvestPeriod ||
		(h.storedMetrics > h.MetricCap && time.Since(h.lastReport) > h.MinReportInterval) {

		log.Tracef("triggering harvest, last harvest: %v ago", time.Since(h.lastReport))

		h.lastReport = time.Now()
		h.storedMetrics = 0

		go h.inner.HarvestNow(ctx)
	}
}

// periodicHarvest is run in a separate goroutine to periodically call reportIfNeeded every MinReportInterval
func (h *boundedHarvester) periodicHarvest(cancel chan struct{}) {
	t := time.NewTicker(h.MinReportInterval)
	for {
		select {
		case <-cancel:
			t.Stop()
			return
		case <-t.C:
			h.reportIfNeeded(context.Background(), false)
		}
	}
}
