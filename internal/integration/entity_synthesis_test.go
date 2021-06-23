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
	Conditions: []Condition{
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
	Conditions: []Condition{
		{
			Attribute: "metricName",
			Prefix:    "redis_second",
		},
	},
	Tags: Tags{
		"foo": nil,
	},
}
var fooRule = EntityRule{
	EntityType: "FOO",
	Identifier: "identifier",
	Name:       "displayName",
	Conditions: []Condition{
		{
			Attribute: "metricAttribute",
			Prefix:    "attribute_prefix_",
		},
	},
}
var valueRule = EntityRule{
	EntityType: "FOO",
	Identifier: "identifier",
	Name:       "displayName",
	Conditions: []Condition{
		{
			Attribute: "metricAttribute",
			Value:     "complete_metric_name",
		},
	},
}
var attributeRule = EntityRule{
	EntityType: "FOO",
	Identifier: "identifier",
	Name:       "displayName",
	Conditions: []Condition{
		{
			Attribute: "metricAttribute",
		},
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
	type want struct {
		metadata sdk_metadata.Metadata
		found    bool
	}
	tests := []struct {
		name        string
		entityRules []EntityRule
		metric      Metric
		want        want
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
			want: want{redisEntityMetadata, true},
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
			want: want{redisSecondEntityMetadata, true},
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
			want: want{sdk_metadata.Metadata{}, false},
		},
		{
			name:        "rule based on metric attribute",
			entityRules: []EntityRule{fooRule},
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
			want: want{sdk_metadata.Metadata{
				Name:        "FOO:GUID",
				DisplayName: "NiceName",
				EntityType:  "FOO",
				Metadata:    sdk_metadata.Map{},
			}, true},
		},
		{
			name:        "entity matches rule that specify the value",
			entityRules: []EntityRule{valueRule},
			metric: Metric{
				name:       "go_goroutines",
				value:      1.0,
				metricType: "gauge",
				attributes: labels.Set{
					"identifier":      "GUID",
					"displayName":     "NiceName",
					"metricAttribute": "complete_metric_name",
				},
			},
			want: want{sdk_metadata.Metadata{
				Name:        "FOO:GUID",
				DisplayName: "NiceName",
				EntityType:  "FOO",
				Metadata:    sdk_metadata.Map{},
			}, true},
		},
		{
			name:        "entity matches rule that don't have prefix nor value",
			entityRules: []EntityRule{attributeRule},
			metric: Metric{
				name:       "go_goroutines",
				value:      1.0,
				metricType: "gauge",
				attributes: labels.Set{
					"identifier":      "GUID",
					"displayName":     "NiceName",
					"metricAttribute": "doesn't care the content here",
				},
			},
			want: want{sdk_metadata.Metadata{
				Name:        "FOO:GUID",
				DisplayName: "NiceName",
				EntityType:  "FOO",
				Metadata:    sdk_metadata.Map{},
			}, true},
		},
		{
			name:        "empty rules",
			entityRules: []EntityRule{},
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
			want: want{sdk_metadata.Metadata{}, false},
		},
		{
			name:        "metric attribute for identifier is missing",
			entityRules: []EntityRule{redisRule, redisSecondRule},
			metric: Metric{
				name:       "redis_foo_bar",
				value:      1.0,
				metricType: "gauge",
				attributes: nil,
			},
			want: want{sdk_metadata.Metadata{}, false},
		},
		{
			name:        "metric attribute for identifier is not a string",
			entityRules: []EntityRule{redisRule, redisSecondRule},
			metric: Metric{
				name:       "redis_foo_bar",
				value:      1.0,
				metricType: "gauge",
				attributes: labels.Set{
					"targetName": 123,
				},
			},
			want: want{sdk_metadata.Metadata{}, false},
		},
		{
			name:        "metric attribute used for entity display name is missing",
			entityRules: []EntityRule{redisRule, redisSecondRule},
			metric: Metric{
				name:       "redis_foo_bar",
				value:      1.0,
				metricType: "gauge",
				attributes: labels.Set{
					"targetName": 123,
				},
			},
			want: want{sdk_metadata.Metadata{}, false},
		},
		{
			name:        "rule based on metric attribute",
			entityRules: []EntityRule{fooRule},
			metric: Metric{
				name:       "go_goroutines",
				value:      1.0,
				metricType: "gauge",
				attributes: labels.Set{
					"identifier":      "GUID",
					"metricAttribute": "attribute_prefix_bar",
				},
			},
			want: want{sdk_metadata.Metadata{}, false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSynthesizer(tt.entityRules)
			em, found := s.GetEntityMetadata(tt.metric)
			assert.EqualValues(t, tt.want.metadata, em)
			assert.EqualValues(t, tt.want.found, found)
		})
	}
}
