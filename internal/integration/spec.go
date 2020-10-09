// Package integration ...
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"errors"
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
	Service       string      `yaml:"service"`
	Entities      []EntityDef `yaml:"entities"`
	DefaultEntity string      `yaml:"default_entity"`
}

// EntityDef has info related to each entity
type EntityDef struct {
	Name        string        `yaml:"name"`
	DisplayName string        `yaml:"display_name"`
	Properties  PropertiesDef `yaml:"properties"`
	Metrics     []MetricDef   `yaml:"metrics"`
}

// PropertiesDef defines the dimension used to get entity names
type PropertiesDef struct {
	Labels []string `yaml:"labels"`
}

// MetricDef contains metrics definitions
type MetricDef struct {
	Name string `yaml:"provider_name"`
}

// entityNameProps contains entity properties required to build the entity name
type entityNameProps struct {
	Name        string
	DisplayName string
	Type        string
	Service     string
	Labels      []keyValue
}

type keyValue struct {
	Key   string
	Value string
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
func (s *Specs) getEntity(m Metric) (props entityNameProps, err error) {
	spec, err := s.findSpec(m.name)
	if err != nil {
		return entityNameProps{}, err
	}

	e, ok := spec.findEntity(m.name)
	if !ok {
		if spec.DefaultEntity != "" {
			e, ok = spec.findEntityByName(spec.DefaultEntity)
			if !ok {
				msg := fmt.Sprintf("could not find default entity '%v' for metric '%v'", spec.DefaultEntity, m.name)
				return entityNameProps{}, errors.New(msg)
			}
			if len(e.Properties.Labels) > 0 {
				return entityNameProps{}, errors.New("default entity must not have labels")
			}
		} else {
			return entityNameProps{}, fmt.Errorf("metric: %s is not defined in service:%s and no default entity is defined", m.name, spec.Service)
		}
	}

	props.Name = e.Name
	props.DisplayName = e.DisplayName
	props.Type = strings.ToUpper(spec.Service) + "_" + strings.ToUpper(e.Name)
	props.Service = spec.Service
	props.Labels = []keyValue{}

	for _, d := range e.Properties.Labels {
		var val interface{}
		var ok bool
		// the metric needs all the labels defined to avoid entity name collision
		if val, ok = m.attributes[d]; !ok {
			return entityNameProps{}, fmt.Errorf("label %s not found in metric %s", d, m.name)
		}
		props.Labels = append(props.Labels, keyValue{fmt.Sprintf("%v", d), fmt.Sprintf("%v", val)})
	}

	return props, nil
}

// findSpec parses the metric name to extract the service and returns the spec definition that matches
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

func (s *SpecDef) findEntityByName(entityName string) (EntityDef, bool) {
	for _, e := range s.Entities {
		if e.Name == entityName {
			return e, true
		}
	}
	return EntityDef{}, false
}
