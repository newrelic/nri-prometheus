// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"testing"

	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	"github.com/stretchr/testify/assert"
)

func Test_EmitterCanEmit(t *testing.T) {
	t.Parallel()

	summary, err := newSummary(3, 10, []*quantile{{0.5, 10}, {0.999, 100}})
	if err != nil {
		t.Fatal(err)
	}

	metrics := []Metric{
		{
			name:       "common-name",
			metricType: metricType_COUNTER,
			value:      float64(1),
			attributes: labels.Set{
				"name":           "common-name",
				"targetName":     "target-a",
				"nrMetricType":   "count",
				"promMetricType": "counter",
			},
		},
		{
			name:       "common-name2",
			metricType: metricType_COUNTER,
			value:      float64(1),
			attributes: labels.Set{
				"name":           "common-name2",
				"targetName":     "target-b",
				"nrMetricType":   "count",
				"promMetricType": "counter",
			},
		},
		{
			name:       "common-name3",
			metricType: metricType_GAUGE,
			value:      float64(1),
			attributes: labels.Set{
				"name":           "common-name3",
				"targetName":     "target-c",
				"nrMetricType":   "gauge",
				"promMetricType": "gauge",
			},
		},
		{
			name:       "summary-1",
			metricType: metricType_SUMMARY,
			value:      summary,
			attributes: labels.Set{},
		},
	}

	e := NewStdoutEmitter()
	assert.NotNil(t, e)

	err = e.Emit(metrics)
	assert.NoError(t, err)
}
