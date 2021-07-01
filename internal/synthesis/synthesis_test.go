// Package entity_synthesis ...
// Copyright 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package synthesis

import (
	"testing"

	sdk_metadata "github.com/newrelic/infra-integrations-sdk/v4/data/metadata"
	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	"github.com/stretchr/testify/assert"
)

var redisRule = Definition{
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
			"realName": map[string]interface{}{
				"entityTagName": "preferredName",
			},
		},
	},
}

var redisSecondRule = Definition{
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

var fooRule = Definition{
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

var fooValueRule = Definition{
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

var fooAttributeRule = Definition{
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

var fooMultiRule = Definition{
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
		"tags.foo":           "bar",
		"tags.preferredName": "renamedTagValue",
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

	type metric struct {
		name       string
		attributes labels.Set
	}

	tests := []struct {
		name        string
		definitions []Definition
		metric      metric
		want        want
	}{
		{
			name:        "happy",
			definitions: []Definition{redisRule, redisSecondRule},
			metric: metric{
				name: "redis_foo_bar",
				attributes: labels.Set{
					"targetName": "localhost:9999",
					"foo":        "bar",
					"realName":   "renamedTagValue",
				},
			},
			want: want{redisEntityMetadata, true},
		},
		{
			name:        "longer prefix match takes precedence",
			definitions: []Definition{redisRule, redisSecondRule},
			metric: metric{
				name:       "redis_second_foo_bar",
				attributes: metricAttributes,
			},
			want: want{redisSecondEntityMetadata, true},
		},
		{
			name:        "metric has not matches",
			definitions: []Definition{redisRule},
			metric: metric{
				name:       "go_goroutines",
				attributes: metricAttributes,
			},
			want: want{sdk_metadata.Metadata{}, false},
		},
		{
			name:        "rule based on metric attribute",
			definitions: []Definition{fooRule},
			metric: metric{
				name: "go_goroutines",
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
			definitions: []Definition{fooValueRule},
			metric: metric{
				name: "go_goroutines",
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
			definitions: []Definition{fooAttributeRule},
			metric: metric{
				name: "go_goroutines",
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
			definitions: []Definition{},
			metric: metric{
				name: "go_goroutines",
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
			definitions: []Definition{redisRule, redisSecondRule},
			metric: metric{
				name:       "redis_foo_bar",
				attributes: nil,
			},
			want: want{sdk_metadata.Metadata{}, false},
		},
		{
			name:        "metric attribute for identifier is not a string",
			definitions: []Definition{redisRule, redisSecondRule},
			metric: metric{
				name: "redis_foo_bar",
				attributes: labels.Set{
					"targetName": 123,
				},
			},
			want: want{sdk_metadata.Metadata{}, false},
		},
		{
			name:        "metric attribute used for entity display name is missing",
			definitions: []Definition{redisRule, redisSecondRule},
			metric: metric{
				name: "redis_foo_bar",
				attributes: labels.Set{
					"targetName": 123,
				},
			},
			want: want{sdk_metadata.Metadata{}, false},
		},
		{
			name:        "rule based on metric attribute",
			definitions: []Definition{fooRule},
			metric: metric{
				name: "go_goroutines",
				attributes: labels.Set{
					"identifier":      "GUID",
					"metricAttribute": "attribute_prefix_bar",
				},
			},
			want: want{sdk_metadata.Metadata{}, false},
		},
		{
			name:        "synthesis definition with multiple rules, match rule 1",
			definitions: []Definition{fooMultiRule},
			metric: metric{
				name: "test.metric",
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
			definitions: []Definition{fooMultiRule},
			metric: metric{
				name: "test_metric",
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
			definitions: []Definition{fooMultiRule},
			metric: metric{
				name: "metric",
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
			em, found := s.GetEntityMetadata(tt.metric.name, tt.metric.attributes)
			assert.EqualValues(t, tt.want.metadata, em)
			assert.EqualValues(t, tt.want.found, found)
		})
	}
}
