package integration

import (
	"context"
	"sync"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	log "github.com/sirupsen/logrus"
)

// bindHarvester creates a boundedHarvester from an existing harvester.
// It also returns a cancel channel to stop the periodic harvest goroutine.
// The returned boundedHarvester always runs in a loop.
func bindHarvester(inner harvester, cfg BoundedHarvesterCfg) harvester {
	if _, ok := inner.(*telemetry.Harvester); ok {
		log.Debug("using telemetry.Harvester as underlying harvester, make sure to set HarvestPeriod to 0")
	}

	if cfg.MinReportInterval < BoundedHarvesterDefaultMinReportInterval {
		log.Warnf("Ignoring min_emitter_harvest_period %v < %v", cfg.MinReportInterval, BoundedHarvesterDefaultMinReportInterval)
		cfg.MinReportInterval = BoundedHarvesterDefaultMinReportInterval
	}

	if cfg.HarvestPeriod < cfg.MinReportInterval {
		log.Warnf("Ignoring emitter_harvest_period %v < %v, setting to default %v", cfg.HarvestPeriod, cfg.MinReportInterval, BoundedHarvesterDefaultHarvestPeriod)
		cfg.HarvestPeriod = BoundedHarvesterDefaultHarvestPeriod
	}

	if cfg.MetricCap == 0 {
		cfg.MetricCap = BoundedHarvesterDefaultMetricsCap
	}

	h := &boundedHarvester{
		BoundedHarvesterCfg: cfg,
		mtx:                 sync.Mutex{},
		inner:               inner,
	}

	if !cfg.DisablePeriodicReporting {
		h.stopper = make(chan struct{}, 2)
		go h.periodicHarvest()
	}

	return h
}

// BoundedHarvesterCfg stores the configurable values for boundedHarvester
type BoundedHarvesterCfg struct {
	// MetricCap is the number of metrics to store in memory before triggering a HarvestNow action regardless of
	// HarvestPeriod. It will directly influence the amount of memory that nri-prometheus allocates.
	// A value of 10000 is rougly equivalent to 500M in RAM in the tested scenarios
	MetricCap int

	// HarvestPeriod specifies the period that will trigger a HarvestNow action for the inner harvester.
	// It is not necessary to decrease this value further, as other conditions (namely the MetricCap) will also trigger
	// a harvest action.
	HarvestPeriod time.Duration

	// MinReportInterval Specifies the minimum amount of time to wait before reports.
	// This will be always enforced, regardless of HarvestPeriod and MetricCap.
	MinReportInterval time.Duration

	// DisablePeriodicReporting prevents bindHarvester from spawning the periodic report routine.
	// It also causes an already spawned reporting routine to be stopped on the next interval.
	DisablePeriodicReporting bool
}

// BoundedHarvesterDefaultHarvestPeriod is the default harvest period. Since harvests are also triggered by stacking
// metrics, there is no need for this to be very low
const BoundedHarvesterDefaultHarvestPeriod = 1 * time.Second

// BoundedHarvesterDefaultMetricsCap is the default number of metrics stack before triggering a harvest. 10000 metrics
// require around 500MiB in our testing setup
const BoundedHarvesterDefaultMetricsCap = 10000

// BoundedHarvesterDefaultMinReportInterval is the default and minimum enforced harvest interval time. No harvests will
// be issued if previous harvest was less than this value ago (except for those triggered with HarvestNow)
const BoundedHarvesterDefaultMinReportInterval = 200 * time.Millisecond

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

	stopper chan struct{}
	stopped bool

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

func (h *boundedHarvester) Stop() {
	// We need to nil the channel and flag stopped synchronously to avoid double-stop races
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if h.stopper != nil && !h.stopped {
		h.stopper <- struct{}{}
		h.stopped = true
	}
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
func (h *boundedHarvester) periodicHarvest() {
	t := time.NewTicker(h.MinReportInterval)
	for {
		select {
		case <-h.stopper:
			t.Stop()
			return

		case <-t.C:
			if h.DisablePeriodicReporting {
				h.Stop()
				continue
			}

			h.reportIfNeeded(context.Background(), false)
		}
	}
}
