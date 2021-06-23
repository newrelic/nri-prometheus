// Package integration ...
// Copyright 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"testing"

	sdk_metadata "github.com/newrelic/infra-integrations-sdk/v4/data/metadata"
	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	"github.com/stretchr/testify/assert"
)

var redisRule = EntityRule{
	EntityType: "REDIS",
	Identifier: "targetName",
	Name:       "targetName",
	Conditions: []Conditions{
		{
			Attribute: "metricName",
			Prefix:    "redis_",
		},
	},
	Tags: Tags{
		"foo": nil,
	},
}
var redisSecondRule = EntityRule{
	EntityType: "REDIS_SECOND",
	Identifier: "targetName",
	Name:       "targetName",
	Conditions: []Conditions{
		{
			Attribute: "metricName",
			Prefix:    "redis_second",
		},
	},
	Tags: Tags{
		"foo": nil,
	},
}
var redisEntityMetadata = sdk_metadata.Metadata{
	Name:        "REDIS:localhost:9999",
	DisplayName: "localhost:9999",
	EntityType:  "REDIS",
	Metadata: sdk_metadata.Map{
		"tags.foo": "bar",
	},
}
var redisSecondEntityMetadata = sdk_metadata.Metadata{
	Name:        "REDIS_SECOND:localhost:9999",
	DisplayName: "localhost:9999",
	EntityType:  "REDIS_SECOND",
	Metadata: sdk_metadata.Map{
		"tags.foo": "bar",
	},
}

var metricAttributes = labels.Set{
	"targetName": "localhost:9999",
	"foo":        "bar",
}

func Test_synthesis_GetEntityMetadata(t *testing.T) {
	tests := []struct {
		name        string
		entityRules []EntityRule
		metric      Metric
		want        *sdk_metadata.Metadata
	}{
		{
			name:        "happy",
			entityRules: []EntityRule{redisRule, redisSecondRule},
			metric: Metric{
				name:       "redis_foo_bar",
				value:      1.0,
				metricType: "gauge",
				attributes: metricAttributes,
			},
			want: &redisEntityMetadata,
		},
		{
			name:        "longer prefix match takes precedence",
			entityRules: []EntityRule{redisRule, redisSecondRule},
			metric: Metric{
				name:       "redis_second_foo_bar",
				value:      1.0,
				metricType: "gauge",
				attributes: metricAttributes,
			},
			want: &redisSecondEntityMetadata,
		},
		{
			name:        "metric has not matches",
			entityRules: []EntityRule{redisRule},
			metric: Metric{
				name:       "go_goroutines",
				value:      1.0,
				metricType: "gauge",
				attributes: metricAttributes,
			},
			want: nil,
		},
		{
			name: "rule based on metric attribute",
			entityRules: []EntityRule{
				{
					EntityType: "FOO",
					Identifier: "identifier",
					Name:       "displayName",
					Conditions: []Conditions{
						{
							Attribute: "metricAttribute",
							Prefix:    "attribute_prefix_",
						},
					},
				},
			},
			metric: Metric{
				name:       "go_goroutines",
				value:      1.0,
				metricType: "gauge",
				attributes: labels.Set{
					"identifier":      "GUID",
					"displayName":     "NiceName",
					"metricAttribute": "attribute_prefix_bar",
				},
			},
			want: &sdk_metadata.Metadata{
				Name:        "FOO:GUID",
				DisplayName: "NiceName",
				EntityType:  "FOO",
				Metadata:    sdk_metadata.Map{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Synthesizer{
				EntityRules: tt.entityRules,
			}
			assert.EqualValues(t, tt.want, s.GetEntityMetadata(tt.metric))
		})
	}
}
