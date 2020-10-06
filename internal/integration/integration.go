// Package integration ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/newrelic/nri-prometheus/internal/pkg/endpoints"
)

const (
	// Name of the integration
	Name = "nri-prometheus"
)

var (
	// Version of the integration
	Version = "dev"
)

var ilog = logrus.WithField("component", "integration.Execute")

// Execute the integration loop. It sets the retrievers to start watching for
// new targets and starts the processing pipeline. The pipeline fetches
// metrics from the registered targets, transforms them according to a set
// of rules and emits them.
//
// with first-class functions
func Execute(
	scrapeDuration time.Duration,
	selfRetriever endpoints.TargetRetriever,
	retrievers []endpoints.TargetRetriever,
	fetcher Fetcher,
	processor Processor,
	emitters []Emitter,
) {
	for _, retriever := range retrievers {
		err := retriever.Watch()
		if err != nil {
			ilog.WithError(err).WithField("retriever", retriever.Name()).Error("while getting the initial list of targets")
		}
	}

	for {
		totalTimeseriesMetric.Set(0)
		totalTimeseriesByTargetMetric.Reset()
		totalTimeseriesByTargetAndTypeMetric.Reset()
		totalTimeseriesByTypeMetric.Reset()

		startTime := time.Now()
		process(retrievers, fetcher, processor, emitters)
		totalExecutionsMetric.Inc()
		if duration := time.Since(startTime); duration < scrapeDuration {
			time.Sleep(scrapeDuration - duration)
		}
		processWithoutTelemetry(selfRetriever, fetcher, processor, emitters)
	}
}

// ExecuteOnce executes the integration once. The pipeline fetches
// metrics from the registered targets, transforms them according to a set
// of rules and emits them.
func ExecuteOnce(retrievers []endpoints.TargetRetriever, fetcher Fetcher, processor Processor, emitters []Emitter) {
	for _, retriever := range retrievers {
		err := retriever.Watch()
		if err != nil {
			ilog.WithError(err).WithField("retriever", retriever.Name()).Error("while getting the initial list of targets")
		}
	}

	for _, retriever := range retrievers {
		processWithoutTelemetry(retriever, fetcher, processor, emitters)
	}
}

// processWithoutTelemetry processes a target retriever without doing any
// kind of telemetry calculation.
func processWithoutTelemetry(
	retriever endpoints.TargetRetriever,
	fetcher Fetcher,
	processor Processor,
	emitters []Emitter,
) {
	targets, err := retriever.GetTargets()
	if err != nil {
		ilog.WithError(err).Error("error getting targets")
		return
	}
	pairs := fetcher.Fetch(targets)
	processed := processor(pairs)
	for pair := range processed {
		for _, e := range emitters {
			err := e.Emit(pair.Metrics)
			if err != nil {
				ilog.WithField("emitter", e.Name()).WithError(err).Warn("error emitting metrics")
			}
		}
	}
}

func process(retrievers []endpoints.TargetRetriever, fetcher Fetcher, processor Processor, emitters []Emitter) {
	ptimer := prometheus.NewTimer(prometheus.ObserverFunc(processDurationMetric.Set))

	targets := make([]endpoints.Target, 0)
	for _, retriever := range retrievers {
		totalDiscoveriesMetric.WithLabelValues(retriever.Name()).Set(1)
		t, err := retriever.GetTargets()
		if err != nil {
			ilog.WithError(err).Error("error getting targets")
			totalErrorsDiscoveryMetric.WithLabelValues(retriever.Name()).Set(1)
			return
		}
		totalTargetsMetric.WithLabelValues(retriever.Name()).Set(float64(len(t)))
		targets = append(targets, t...)
	}
	pairs := fetcher.Fetch(targets) // fetch metrics from /metrics endpoints
	processed := processor(pairs)   // apply processing

	emittedMetrics := 0
	for pair := range processed {
		emittedMetrics += len(pair.Metrics)

		for _, e := range emitters {
			err := e.Emit(pair.Metrics)
			if err != nil {
				ilog.WithField("emitter", e.Name()).WithError(err).Warn("error emitting metrics")
			}
		}
	}

	duration := ptimer.ObserveDuration()

	logrus.WithFields(logrus.Fields{
		"duration":            duration.Round(time.Second),
		"targetCount":         len(targets),
		"emitterCount":        len(emitters),
		"emittedMetricsCount": emittedMetrics,
	}).Debug("Processing metrics finished.")
}
