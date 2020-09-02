// Package integration ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	yaml "gopkg.in/yaml.v2"
)

const fileNameMatcher = `^prometheus_.*\.ya?ml$`

// Specs contains all the services specs mapped with the service name
type Specs struct {
	SpecsByName map[string]SpecDef
}

// SpecDef contains the rules to group metrics into entities
type SpecDef struct {
	Service  string      `yaml:"service"`
	Entities []EntityDef `yaml:"entities"`
}

// EntityDef has info related to each entity
type EntityDef struct {
	Type       string        `yaml:"name"`
	Properties PropertiesDef `yaml:"properties"`
	Metrics    []MetricDef   `yaml:"metrics"`
}

// PropertiesDef defines the dimension used to get entity names
type PropertiesDef struct {
	Dimensions []string `yaml:"dimensions"`
}

// MetricDef contains metrics definitions
type MetricDef struct {
	Name string `yaml:"provider_name"`
}

// LoadSpecFiles loads all service spec files named like "prometheus_*.yml" that are in the filesPath
func LoadSpecFiles(filesPath string) (Specs, error) {
	specs := Specs{SpecsByName: make(map[string]SpecDef)}
	var files []string

	filesInPath, err := ioutil.ReadDir(filesPath)
	if err != nil {
		return specs, err
	}
	for _, f := range filesInPath {
		if ok, _ := regexp.MatchString(fileNameMatcher, f.Name()); ok {
			files = append(files, path.Join(filesPath, f.Name()))
		}
	}

	for _, file := range files {
		f, err := ioutil.ReadFile(file)
		if err != nil {
			logrus.Errorf("fail to read service spec file %s: %s ", file, err)
			continue
		}

		var sd SpecDef
		err = yaml.Unmarshal(f, &sd)
		if err != nil {
			logrus.Errorf("fail parse service spec file %s: %s", file, err)
			continue
		}
		logrus.Debugf("spec file loaded for service:%s", sd.Service)
		specs.SpecsByName[sd.Service] = sd
	}

	return specs, nil
}

// getEntity returns entity name and type of the metric based on the spec configuration defined for the service.
// conditions for the metric:
//   - metric.name has to start a prefix that matches with a service from the spec files
//   - metric.name has to be defined in one of the entities of the spec file
//   - if dimension has been specified for the entity, the metric need to have all of them.
//   - metrics that belongs to entities with no dimension specified will share the same name
func (s *Specs) getEntity(m Metric) (entityName string, entityType string, err error) {
	spec, err := s.findSpec(m.name)
	if err != nil {
		return "", "", err
	}

	e, ok := spec.findEntity(m.name)
	if !ok {
		return "", "", fmt.Errorf("metric: %s is not defined in service:%s", m.name, spec.Service)
	}

	entityType = strings.Title(spec.Service) + strings.Title(e.Type)

	entityName = e.Type

	for _, d := range e.Properties.Dimensions {
		var val interface{}
		var ok bool
		// the metric needs all the dimensions defined to avoid entity name collision
		if val, ok = m.attributes[d]; !ok {
			return "", "", fmt.Errorf("dimension %s not found in metric %s", d, m.name)
		}
		// entity name will be composed by the value of the dimensions defined for the entity in order
		entityName = entityName + ":" + fmt.Sprintf("%v", val)
	}

	return entityName, entityType, nil
}

// findSpec parses the metric name to extract the service and resturns the spec definition that matches
func (s *Specs) findSpec(metricName string) (SpecDef, error) {
	var spec SpecDef

	res := strings.SplitN(metricName, "_", 2)
	if len(res) < 2 {
		return spec, fmt.Errorf("metric: %s has no suffix to identify the entity", metricName)
	}
	serviceName := res[0]

	var ok bool
	if spec, ok = s.SpecsByName[serviceName]; !ok {
		return spec, fmt.Errorf("no spec files for service: %s", serviceName)
	}

	return spec, nil
}

// findEntity returns the entity where metricName is defined
func (s *SpecDef) findEntity(metricName string) (EntityDef, bool) {
	for _, e := range s.Entities {
		for _, em := range e.Metrics {
			if metricName == em.Name {
				return e, true
			}
		}
	}
	return EntityDef{}, false
}
