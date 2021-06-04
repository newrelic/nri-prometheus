package integration

import (
	"context"
	"testing"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
)

type mockHarvester struct {
	metrics  int
	harvests int
}

func (h *mockHarvester) RecordMetric(m telemetry.Metric) {
	h.metrics++
}

func (h *mockHarvester) HarvestNow(ctx context.Context) {
	h.harvests++
}

// Checks that bindHarvester returns a harvester, correctly overriding settings
// Also check the async routine is spawned
func TestBindHarvester(t *testing.T) {
	t.Parallel()

	cfg := BoundedHarvesterCfg{
		MetricCap:                0,
		HarvestPeriod:            1,
		MinReportInterval:        10,
		DisablePeriodicReporting: true,
	}

	mock := &mockHarvester{}
	h := bindHarvester(mock, cfg)

	bh, ok := h.(*boundedHarvester)
	if !ok {
		t.Fatalf("returned harvester is not a boundedHarvester")
	}
	defer bh.Stop()

	if bh.MinReportInterval != BoundedHarvesterDefaultMinReportInterval {
		t.Fatalf("MinReportInterval was not overridden")
	}

	if bh.HarvestPeriod != BoundedHarvesterDefaultHarvestPeriod {
		t.Fatalf("HarvestPeriod was not overridden")
	}

	if bh.MetricCap != BoundedHarvesterDefaultMetricsCap {
		t.Fatalf("MetricCap was not overridden")
	}

	time.Sleep(time.Second)
	if mock.harvests != 0 {
		t.Fatalf("Periodic routine was called despite being disabled")
	}
}

func TestHarvestRoutine(t *testing.T) {
	t.Parallel()

	cfg := BoundedHarvesterCfg{
		HarvestPeriod:     300 * time.Millisecond,
		MinReportInterval: BoundedHarvesterDefaultMinReportInterval,
	}

	mock := &mockHarvester{}
	h := bindHarvester(mock, cfg)

	bh, ok := h.(*boundedHarvester)
	if !ok {
		t.Fatalf("returned harvester is not a boundedHarvester")
	}
	defer bh.Stop()

	time.Sleep(time.Second)
	if mock.harvests < 1 {
		t.Fatalf("harvest routine was not called within 1s")
	}
}

func TestRoutineStopChannel(t *testing.T) {
	t.Parallel()

	cfg := BoundedHarvesterCfg{
		HarvestPeriod:     300 * time.Millisecond,
		MinReportInterval: BoundedHarvesterDefaultMinReportInterval,
	}

	mock := &mockHarvester{}
	h := bindHarvester(mock, cfg)

	bh, ok := h.(*boundedHarvester)
	if !ok {
		t.Fatalf("returned harvester is not a boundedHarvester")
	}
	defer bh.Stop()

	time.Sleep(time.Second)
	bh.Stop()
	time.Sleep(time.Second)
	harvests := mock.harvests
	time.Sleep(time.Second)
	if mock.harvests != harvests {
		t.Fatalf("Stop() did not stop the harvest routine")
	}
}

func TestRoutineStopFlag(t *testing.T) {
	t.Parallel()

	cfg := BoundedHarvesterCfg{
		HarvestPeriod:     300 * time.Millisecond,
		MinReportInterval: BoundedHarvesterDefaultMinReportInterval,
	}

	mock := &mockHarvester{}
	h := bindHarvester(mock, cfg)

	bh, ok := h.(*boundedHarvester)
	if !ok {
		t.Fatalf("returned harvester is not a boundedHarvester")
	}
	defer bh.Stop()

	time.Sleep(time.Second)
	bh.DisablePeriodicReporting = true
	time.Sleep(time.Second)
	harvests := mock.harvests
	time.Sleep(time.Second)
	if mock.harvests != harvests {
		t.Fatalf("DisablePeriodicReporting = true did not stop the harvest routine")
	}
}

func TestHarvestNow(t *testing.T) {
	t.Parallel()

	cfg := BoundedHarvesterCfg{
		DisablePeriodicReporting: true,
	}

	mock := &mockHarvester{}
	h := bindHarvester(mock, cfg)

	bh, ok := h.(*boundedHarvester)
	if !ok {
		t.Fatalf("returned harvester is not a boundedHarvester")
	}
	defer bh.Stop()

	h.HarvestNow(context.Background())
	time.Sleep(100 * time.Millisecond) // Inner HarvestNow is asynchronous
	if mock.harvests < 1 {
		t.Fatalf("HarvestNow did not trigger a harvest")
	}
}

func TestMetricCap(t *testing.T) {
	t.Parallel()

	cfg := BoundedHarvesterCfg{
		HarvestPeriod: time.Hour,
		MetricCap:     3,
	}

	mock := &mockHarvester{}
	h := bindHarvester(mock, cfg)

	bh, ok := h.(*boundedHarvester)
	if !ok {
		t.Fatalf("returned harvester is not a boundedHarvester")
	}
	defer bh.Stop()

	h.RecordMetric(telemetry.Count{})
	h.RecordMetric(telemetry.Count{})
	h.RecordMetric(telemetry.Count{})
	h.RecordMetric(telemetry.Count{})
	time.Sleep(time.Second) // Wait for MinReportInterval

	if mock.harvests < 1 {
		t.Fatalf("Stacking metrics did not trigger a harvest")
	}
}
