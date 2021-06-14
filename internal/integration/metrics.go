// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package integration

import "github.com/prometheus/client_golang/prometheus"

var (
	totalTargetsMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nr_stats",
		Name:      "targets",
		Help:      "Discovered targets",
	},
		[]string{
			"retriever",
		},
	)
	totalDiscoveriesMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nr_stats",
		Name:      "discoveries_total",
		Help:      "Attempted discoveries",
	},
		[]string{
			"retriever",
		},
	)
	totalErrorsDiscoveryMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nr_stats",
		Name:      "discovery_errors_total",
		Help:      "Attempted discoveries that resulted in an error",
	},
		[]string{
			"retriever",
		},
	)
	fetchesTotalMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nr_stats",
		Name:      "fetches_total",
		Help:      "Fetches attempted",
	},
		[]string{
			"target",
		},
	)
	fetchErrorsTotalMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nr_stats",
		Name:      "fetch_errors_total",
		Help:      "Fetches attempted that resulted in an error",
	},
		[]string{
			"target",
		},
	)
	totalTimeseriesByTargetAndTypeMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nr_stats",
		Subsystem: "metrics",
		Name:      "total_timeseries_by_target_type",
		Help:      "Total number of metrics by type and target",
	},
		[]string{
			"type",
			"target",
		},
	)
	totalTimeseriesByTypeMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nr_stats",
		Subsystem: "metrics",
		Name:      "total_timeseries_by_type",
		Help:      "Total number of metrics by type",
	},
		[]string{
			"type",
		},
	)
	totalTimeseriesMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "nr_stats",
		Subsystem: "metrics",
		Name:      "total_timeseries",
		Help:      "Total number of timeseries",
	})
	totalTimeseriesByTargetMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nr_stats",
		Subsystem: "metrics",
		Name:      "total_timeseries_by_target",
		Help:      "Total number of timeseries by target",
	},
		[]string{
			"target",
		})
	fetchTargetDurationMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nr_stats",
		Subsystem: "integration",
		Name:      "fetch_target_duration_seconds",
		Help:      "The total time in seconds to fetch the metrics of a target",
	},
		[]string{
			"target",
		},
	)
	processDurationMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "nr_stats",
		Subsystem: "integration",
		Name:      "process_duration_seconds",
		Help:      "The total time in seconds to process all the steps of the integration",
	})
	totalExecutionsMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "nr_stats",
		Subsystem: "integration",
		Name:      "total_executions",
		Help:      "The number of times the integration is executed",
	})
)

func init() {
	prometheus.MustRegister(totalTargetsMetric)
	prometheus.MustRegister(totalDiscoveriesMetric)
	prometheus.MustRegister(totalErrorsDiscoveryMetric)
	prometheus.MustRegister(fetchesTotalMetric)
	prometheus.MustRegister(totalTimeseriesByTypeMetric)
	prometheus.MustRegister(fetchErrorsTotalMetric)
	prometheus.MustRegister(totalTimeseriesByTargetAndTypeMetric)
	prometheus.MustRegister(totalTimeseriesMetric)
	prometheus.MustRegister(totalTimeseriesByTargetMetric)
	prometheus.MustRegister(fetchTargetDurationMetric)
	prometheus.MustRegister(processDurationMetric)
	prometheus.MustRegister(totalExecutionsMetric)
}
