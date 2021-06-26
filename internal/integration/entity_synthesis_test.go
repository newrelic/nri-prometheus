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

var redisRule = SynthesisDefinition{
	EntityRule: EntityRule{
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
	},
}
var redisSecondRule = SynthesisDefinition{
	EntityRule: EntityRule{
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
	},
}
var fooRule = SynthesisDefinition{
	EntityRule: EntityRule{
		EntityType: "FOO",
		Identifier: "identifier",
		Name:       "displayName",
		Conditions: []Condition{
			{
				Attribute: "metricAttribute",
				Prefix:    "attribute_prefix_",
			},
		},
	},
}
var fooValueRule = SynthesisDefinition{
	EntityRule: EntityRule{
		EntityType: "FOO",
		Identifier: "identifier",
		Name:       "displayName",
		Conditions: []Condition{
			{
				Attribute: "metricAttribute",
				Value:     "complete_metric_name",
			},
		},
	},
}
var fooAttributeRule = SynthesisDefinition{
	EntityRule: EntityRule{
		EntityType: "FOO",
		Identifier: "identifier",
		Name:       "displayName",
		Conditions: []Condition{
			{
				Attribute: "metricAttribute",
			},
		},
	},
}
var fooMultiRule = SynthesisDefinition{
	EntityRule: EntityRule{
		EntityType: "FOO",
		Tags: Tags{
			"commonAttribute": nil,
		},
	},
	Rules: []EntityRule{
		{
			Identifier: "identifier1",
			Name:       "displayName1",
			Conditions: []Condition{
				{
					Attribute: "metricName",
					Prefix:    "test.",
				},
			},
			Tags: Tags{
				"tag1": nil,
			},
		},
		{
			Identifier: "identifier2",
			Name:       "displayName2",
			Conditions: []Condition{
				{
					Attribute: "metricName",
					Prefix:    "test_",
				},
			},
		},
		{
			Identifier: "identifier3",
			Name:       "displayName3",
			Conditions: []Condition{
				{
					Attribute: "metricAttribute",
					Value:     "matched by attribute only",
				},
			},
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
var fooEntityMetadata = sdk_metadata.Metadata{
	Name:        "FOO:GUID",
	DisplayName: "NiceName",
	EntityType:  "FOO",
	Metadata:    sdk_metadata.Map{},
}
var fooEntityMetadataMultiRule = sdk_metadata.Metadata{
	Name:        "FOO:GUID",
	DisplayName: "NiceName",
	EntityType:  "FOO",
	Metadata: sdk_metadata.Map{
		"tags.commonAttribute": "commonAttributeValue",
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
		definitions []SynthesisDefinition
		metric      Metric
		want        want
	}{
		{
			name:        "happy",
			definitions: []SynthesisDefinition{redisRule, redisSecondRule},
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
			definitions: []SynthesisDefinition{redisRule, redisSecondRule},
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
			definitions: []SynthesisDefinition{redisRule},
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
			definitions: []SynthesisDefinition{fooRule},
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
			want: want{fooEntityMetadata, true},
		},
		{
			name:        "entity matches rule that specify the value",
			definitions: []SynthesisDefinition{fooValueRule},
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
			want: want{fooEntityMetadata, true},
		},
		{
			name:        "entity matches rule that don't have prefix nor value",
			definitions: []SynthesisDefinition{fooAttributeRule},
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
			want: want{fooEntityMetadata, true},
		},
		{
			name:        "empty rules",
			definitions: []SynthesisDefinition{},
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
			definitions: []SynthesisDefinition{redisRule, redisSecondRule},
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
			definitions: []SynthesisDefinition{redisRule, redisSecondRule},
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
			definitions: []SynthesisDefinition{redisRule, redisSecondRule},
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
			definitions: []SynthesisDefinition{fooRule},
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
		{
			name:        "synthesis definition with multiple rules, match rule 1",
			definitions: []SynthesisDefinition{fooMultiRule},
			metric: Metric{
				name:       "test.metric",
				value:      1.0,
				metricType: "gauge",
				attributes: labels.Set{
					"identifier1":     "GUID",
					"displayName1":    "NiceName",
					"commonAttribute": "commonAttributeValue",
					"tag1":            "Foo",
				},
			},
			want: want{sdk_metadata.Metadata{
				Name:        "FOO:GUID",
				DisplayName: "NiceName",
				EntityType:  "FOO",
				Metadata: sdk_metadata.Map{
					"tags.commonAttribute": "commonAttributeValue",
					"tags.tag1":            "Foo",
				},
			}, true},
		},
		{
			name:        "synthesis definition with multiple rules, match rule 2",
			definitions: []SynthesisDefinition{fooMultiRule},
			metric: Metric{
				name:       "test_metric",
				value:      1.0,
				metricType: "gauge",
				attributes: labels.Set{
					"identifier2":     "GUID",
					"displayName2":    "NiceName",
					"commonAttribute": "commonAttributeValue",
				},
			},
			want: want{fooEntityMetadataMultiRule, true},
		},
		{
			name:        "synthesis definition with multiple rules, match rule 3",
			definitions: []SynthesisDefinition{fooMultiRule},
			metric: Metric{
				name:       "metric",
				value:      1.0,
				metricType: "gauge",
				attributes: labels.Set{
					"identifier3":     "GUID",
					"displayName3":    "NiceName",
					"metricAttribute": "matched by attribute only",
					"commonAttribute": "commonAttributeValue",
				},
			},
			want: want{fooEntityMetadataMultiRule, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSynthesizer(tt.definitions)
			em, found := s.GetEntityMetadata(tt.metric)
			assert.EqualValues(t, tt.want.metadata, em)
			assert.EqualValues(t, tt.want.found, found)
		})
	}
}
