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
	conditionList []conditionGroup // List of conditions used find metrics that match.
}

// conditionGroup contains conditions and it parent entityRule.
type conditionGroup struct {
	condition Condition
	rule      EntityRule
}

// NewSynthesizer initialize and return a Synthesizer from a set of EntityRules
func NewSynthesizer(entitySynthesisDefinitions []SynthesisDefinition) Synthesizer {
	s := Synthesizer{}
	for _, ed := range entitySynthesisDefinitions {
		ed.tagRules = ed.Tags.getTagRules()
		s.addConditions(ed.EntityRule)

		for _, er := range ed.Rules {
			er.EntityType = ed.EntityType
			er.tagRules = append(er.Tags.getTagRules(), ed.EntityRule.tagRules...)
			s.addConditions(er)
		}
	}
	return s
}
func (s *Synthesizer) addConditions(rule EntityRule) {
	if !rule.isValid() {
		return
	}
	for _, c := range rule.Conditions {
		s.conditionList = append(s.conditionList, conditionGroup{c, rule})
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

func (rule EntityRule) isValid() bool {
	return rule.Identifier != "" && rule.Name != "" && rule.EntityType != ""
}

// Condition contains parameters used to match entities from metrics
type Condition struct {
	Attribute string `mapstructure:"attribute"`
	Prefix    string `mapstructure:"prefix"`
	Value     string `mapstructure:"value"`
}

// match evaluates the condition for a particular existing attribute value.
func (c Condition) match(attribute string) bool {
	if c.Value != "" {
		return c.Value == attribute
	}
	// this returns true if c.Prefix is "" and is ok since the attribute exists
	return strings.HasPrefix(attribute, c.Prefix)
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
	var match *conditionGroup
	for i, cg := range s.conditionList {
		// special case since metricName is not a metric attribute.
		value := m.name
		if cg.condition.Attribute != "metricName" {
			val, ok := m.attributes[cg.condition.Attribute]
			if !ok {
				continue
			}
			value, _ = val.(string)
		}
		// longer prefix matches take precedences over shorter ones.
		// this allows to discriminate "foo_bar_" from "foo_" kind of metrics.
		if cg.condition.match(value) && (match == nil || len(cg.condition.Prefix) > len(match.condition.Prefix)) { // nosemgrep: bad-nil-guard
			match = &s.conditionList[i]
		}
	}
	if match != nil {
		return match.rule, true
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
