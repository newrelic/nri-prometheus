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
	SpecsByName map[string]Spec
}

// Spec contains the rules to group metrics into entities
type Spec struct {
	Provider string      `yaml:"provider"`
	Service  string      `yaml:"service"`
	Entities []EntityDef `yaml:"entities"`
}

// EntityDef has info related to each entity
type EntityDef struct {
	Type       string        `yaml:"name"`
	Properties PropertiesDef `yaml:"properties"`
}

// PropertiesDef defines the dimension used to get entity names
type PropertiesDef struct {
	Dimensions []string `yaml:"dimensions"`
}

// LoadSpecFiles loads all service spec files named like "prometheus_*.yml" that are in the filesPath
func LoadSpecFiles(filesPath string) (Specs, error) {
	specs := Specs{SpecsByName: make(map[string]Spec)}
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

		var sd Spec
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
// metric example: serviceName_entityName_metricName{dimension="dim"} 0
// serviceName, entityName and all dimensions defined in the spec should match to get an entity
func (s *Specs) getEntity(m Metric) (entityName string, entityType string, err error) {
	res := strings.Split(m.name, "_")
	// We assume that minimun metric name is composed by "serviceName_entityType"
	if len(res) < 2 {
		return "", "", fmt.Errorf("metric: %s has no suffix to identify the entity", m.name)
	}
	serviceName := res[0]
	metricType := res[1]

	var spec Spec
	var ok bool
	if spec, ok = s.SpecsByName[serviceName]; !ok {
		return "", "", fmt.Errorf("no spec files for service: %s", serviceName)
	}
	for _, e := range spec.Entities {
		if metricType == e.Type {
			entityType = metricType
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
			break
		}
	}
	entityName = strings.TrimPrefix(entityName, ":")
	return entityName, entityType, nil
}
