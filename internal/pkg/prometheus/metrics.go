// Package prometheus ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package prometheus

import prom "github.com/prometheus/client_golang/prometheus"

var (
	targetSize = prom.NewGaugeVec(prom.GaugeOpts{
		Namespace: "nr_stats",
		Subsystem: "integration",
		Name:      "payload_size",
		Help:      "Size of target's payload",
	},
		[]string{
			"target",
		},
	)
	totalScrapedPayload = prom.NewGauge(prom.GaugeOpts{
		Namespace: "nr_stats",
		Subsystem: "integration",
		Name:      "total_payload_size",
		Help:      "Total size of the payloads scraped",
	})
)

func init() {
	prom.MustRegister(targetSize)
	prom.MustRegister(totalScrapedPayload)
}
