// Package integration ...
// Copyright 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"strings"

	sdk_metadata "github.com/newrelic/infra-integrations-sdk/v4/data/metadata"
	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
)

// Synthesizer group of rules to synthesis entities
type Synthesizer struct {
	rulesByConditions map[Condition]EntityRule
}

// NewSynthesizer initialize and return a Synthesizer from a set of EntityRules
func NewSynthesizer(entitySynthesisDefinitions []SynthesisDefinition) Synthesizer {
	s := Synthesizer{
		rulesByConditions: make(map[Condition]EntityRule),
	}
	for _, ed := range entitySynthesisDefinitions {
		ed.tagRules = ed.Tags.getTagRules()
		s.addConditions(ed.EntityRule)

		for _, er := range ed.Rules {
			er.EntityType = ed.EntityType
			er.tagRules = er.Tags.getTagRules()
			er.tagRules = append(er.tagRules, ed.EntityRule.tagRules...)
			s.addConditions(er)
		}
	}
	return s
}
func (s *Synthesizer) addConditions(rule EntityRule) {
	if rule.Identifier == "" || rule.Name == "" || rule.EntityType == "" {
		return
	}
	for _, c := range rule.Conditions {
		s.rulesByConditions[c] = rule
	}
}

// SynthesisDefinition contains rules to synthesis entities from metrics
type SynthesisDefinition struct {
	EntityRule `mapstructure:",squash"`
	Rules      []EntityRule `mapstructure:"rules"`
}

// EntityRule contains rules to synthesis an entity
type EntityRule struct {
	EntityType string      `mapstructure:"type"`
	Identifier string      `mapstructure:"identifier"`
	Name       string      `mapstructure:"name"`
	Conditions []Condition `mapstructure:"conditions"`
	Tags       Tags        `mapstructure:"tags"`
	tagRules   []tagRule
}

// Condition contains parameters used to match entities from metrics
type Condition struct {
	Attribute string `mapstructure:"attribute"`
	Prefix    string `mapstructure:"prefix"`
	Value     string `mapstructure:"value"`
}

func (c Condition) match(attribute string) bool {
	if c.Value != "" {
		return c.Value == attribute
	}
	if c.Prefix != "" {
		return strings.HasPrefix(attribute, c.Prefix)
	}
	// if Value and Prefix are empty there is a match since the attribute exists
	return true
}

// Tags key value attributes
type Tags map[string]map[string]interface{}

type tagRule struct {
	name          string
	entityTagName string
}

func (t Tags) getTagRules() []tagRule {
	var tr []tagRule
	for k, v := range t {
		if v == nil {
			tr = append(tr, tagRule{name: k, entityTagName: k})
			continue
		}
		if newName, ok := v["entityTagName"]; ok {
			newNameS, _ := newName.(string)
			tr = append(tr, tagRule{name: k, entityTagName: newNameS})
		}
	}
	return tr
}

// GetEntityMetadata lookup for entity synthesis conditions and generates an entity
// based on the metric attributes.
func (s Synthesizer) GetEntityMetadata(m Metric) (sdk_metadata.Metadata, bool) {
	rule, found := s.getMatchingRule(m)
	if !found {
		return sdk_metadata.Metadata{}, false
	}

	entityName := getEntityAttribute(m.attributes, rule.Identifier)
	entityDisplayName := getEntityAttribute(m.attributes, rule.Name)
	if entityName == "" || entityDisplayName == "" {
		return sdk_metadata.Metadata{}, false
	}
	// entity name needs to be unique per customer account. We concatenate the entity type
	// to add uniqueness for entities with same name but different type.
	entityName = rule.EntityType + ":" + entityName

	md := sdk_metadata.New(entityName, rule.EntityType, entityDisplayName)

	// Adds attributes as entity tag, sdk adds the prefix "tags." to the key.
	for _, t := range rule.tagRules {
		if tagVal, ok := m.attributes[t.name]; ok {
			md.AddTag(t.entityTagName, tagVal)
		}
	}

	return *md, true
}

// getMatchingRule iterates over all conditions to check if m satisfy returning the associated rule.
func (s Synthesizer) getMatchingRule(m Metric) (rule EntityRule, found bool) {
	var match *Condition
	for c := range s.rulesByConditions {
		// special case since metricName is not a metric attribute.
		value := m.name
		if c.Attribute != "metricName" {
			val, ok := m.attributes[c.Attribute]
			if !ok {
				continue
			}
			value, _ = val.(string)
		}
		// longer prefix matches take precedences over shorter ones.
		// this allows to discriminate "foo_bar_" from "foo_" kind of metrics.
		if c.match(value) && (match == nil || len(c.Prefix) > len(match.Prefix)) {
			condition := c
			match = &condition
		}
	}
	if match != nil {
		rule, found = s.rulesByConditions[*match]
	}
	return
}

func getEntityAttribute(attributes labels.Set, key string) string {
	att, ok := attributes[key]
	if !ok {
		return ""
	}
	attString, ok := att.(string)
	if !ok {
		return ""
	}
	return attString

}
