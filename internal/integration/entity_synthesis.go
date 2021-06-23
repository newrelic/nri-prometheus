// Package integration ...
// Copyright 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"strings"

	sdk_metadata "github.com/newrelic/infra-integrations-sdk/v4/data/metadata"
)

type Synthesizer struct {
	EntityRules []EntityRule
}

type EntityRule struct {
	EntityType string       `mapstructure:"type"`
	Identifier string       `mapstructure:"identifier"`
	Name       string       `mapstructure:"name"`
	Conditions []Conditions `mapstructure:"conditions"`
	Tags       Tags         `mapstructure:"tags"`
}

type Conditions struct {
	Attribute string `mapstructure:"attribute"`
	Prefix    string `mapstructure:"prefix"`
}

type Tags map[string]interface{}

type matcher struct {
	rule            *EntityRule
	maxConditionLen int
}

func (m *matcher) match(attribute string, condition string, er EntityRule) bool {
	if strings.HasPrefix(attribute, condition) {
		// multiple matches can happen if prefix collide on differnet er i.e: "foo_" and "foo_bar".
		// the longest prefix will take precedence.
		if len(condition) > m.maxConditionLen {
			m.rule = &er
			m.maxConditionLen = len(condition)
			return true
		}
	}
	return false
}

func (s *Synthesizer) GetEntityMetadata(m Metric) *sdk_metadata.Metadata {
	var matcher matcher
	for i, er := range s.EntityRules {
		for _, c := range er.Conditions {
			// special case since metricName is not a metric attribute.
			if c.Attribute == "metricName" {
				if matcher.match(m.name, c.Prefix, s.EntityRules[i]) {
					continue
				}
			}
			if val, ok := m.attributes[c.Attribute]; ok {
				att, _ := val.(string)
				if matcher.match(att, c.Prefix, s.EntityRules[i]) {
					continue
				}
			}
		}
	}

	if matcher.rule == nil {
		return nil
	}

	var ok bool
	var identifier interface{}
	var name interface{}

	if identifier, ok = m.attributes[matcher.rule.Identifier]; !ok {
		return nil
	}
	entityName, ok := identifier.(string)
	if !ok {
		return nil
	}

	if name, ok = m.attributes[matcher.rule.Name]; !ok {
		return nil
	}
	entityDisplayName, ok := name.(string)
	if !ok {
		return nil
	}

	// entity name needs to be unique per customer account. We concatenate the entity type
	// to add uniqueness for entities with same name but different type.
	entityName = matcher.rule.EntityType + ":" + entityName

	md := sdk_metadata.New(entityName, matcher.rule.EntityType, entityDisplayName)

	// Adds attributes as entity tag, sdk adds the prefix "tags." to the key.
	for tagKey := range matcher.rule.Tags {
		if tagVal, ok := m.attributes[tagKey]; ok {
			md.AddTag(tagKey, tagVal)
		}
	}

	return md
}
