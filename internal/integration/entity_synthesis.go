// Package integration ...
// Copyright 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"strings"

	sdk_metadata "github.com/newrelic/infra-integrations-sdk/v4/data/metadata"
	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
)

// Synthesizer group of rules to synthesis entities from metrics.
// The entity synthesis mapping logic is based on this project (https://github.com/newrelic-experimental/entity-synthesis-definitions).
// The definition of rules are similar to the ones defined in the definition.yaml files of the mentioned repo
// and are set by using the entity_definitions configuration parameter.
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
	for _, definition := range entitySynthesisDefinitions {
		definition.tagRules = definition.Tags.getTagRules()
		s.addConditions(definition.EntityRule)

		for _, rule := range definition.Rules {
			rule.EntityType = definition.EntityType
			rule.tagRules = append(rule.Tags.getTagRules(), definition.EntityRule.tagRules...)
			s.addConditions(rule)
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
	Identifier string      `mapstructure:"identifier"` // Name of the attribute that will be used to uniquely identify the synthesized entity.
	Name       string      `mapstructure:"name"`       // Name of the attribute that will be used to name the synthesized entity.
	Conditions []Condition `mapstructure:"conditions"` // List of rules used to determining if a metric belongs to this entity.
	Tags       Tags        `mapstructure:"tags"`       // List of attributes that will be added to the entity as tags.
	tagRules   []tagRule   // List of all tags better structured to facilitate synthesis process.
}

func (rule EntityRule) isValid() bool {
	return rule.Identifier != "" && rule.Name != "" && rule.EntityType != ""
}

// Condition decides whether a metric is suitable to be included into an entity based on the metric attributes and metric name.
type Condition struct {
	Attribute string `mapstructure:"attribute"`
	Prefix    string `mapstructure:"prefix"`
	Value     string `mapstructure:"value"`
}

// match evaluates the condition an attribute by checking either its whole name against `Value` or if it starts with `Prefix`.
func (c Condition) match(attribute string) bool {
	if c.Value != "" {
		return c.Value == attribute
	}
	// this returns true if c.Prefix is "" and is ok since the attribute exists
	return strings.HasPrefix(attribute, c.Prefix)
}

// Tags stores a collection of attributes that will be added to the entity as tags as the keys of a map.
// The values of the map contains optional rules that applies to the tag when synthetising, like renaming.
type Tags map[string]map[string]interface{}

// tagRule stores the metric attribute name that will be used for the tag and the tag conversions.
type tagRule struct {
	sourceAttribute string // Name of the attribute from the original metric used for the tag.
	entityTagName   string // Name of the tag that will be used in the entity.
}

// getTagRules generates a list of tagRule from the Tags.
func (t Tags) getTagRules() []tagRule {
	var tagRules []tagRule
	for attributeName, rules := range t {
		if rules == nil {
			tagRules = append(tagRules, tagRule{sourceAttribute: attributeName, entityTagName: attributeName})
			continue
		}
		// entityTagName is the name of the rule for tag renaming defined by the entity synthesis protocol.
		if newName, ok := rules["entityTagName"]; ok {
			newNameS, _ := newName.(string)
			tagRules = append(tagRules, tagRule{sourceAttribute: attributeName, entityTagName: newNameS})
		}
	}
	return tagRules
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
		if tagVal, ok := m.attributes[t.sourceAttribute]; ok {
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
