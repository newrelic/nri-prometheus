// Package endpoints ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package endpoints

import "github.com/prometheus/client_golang/prometheus"

var (
	listTargetsDurationByKind = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "nr_stats",
		Subsystem: "integration",
		Name:      "list_targets_duration_by_kind",
		Help:      "The total time in seconds to get the list of targets for a resource kind",
	},
		[]string{
			"retriever",
			"kind",
		},
	)
)

func init() {
	prometheus.MustRegister(listTargetsDurationByKind)
}
