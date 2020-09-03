// Package integration ..
// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	defaultDeltaExpirationAge           = 5 * time.Minute
	defaultDeltaExpirationCheckInterval = 5 * time.Minute
)

// Emitter is an interface representing the ability to emit metrics.
type Emitter interface {
	Name() string
	Emit([]Metric) error
}

// copyAttrs returns a (shallow) copy of the passed attrs.
func copyAttrs(attrs map[string]interface{}) map[string]interface{} {
	duplicate := make(map[string]interface{}, len(attrs))
	for k, v := range attrs {
		duplicate[k] = v
	}
	return duplicate
}

// StdoutEmitter emits metrics to stdout.
type StdoutEmitter struct {
	name string
}

// NewStdoutEmitter returns a NewStdoutEmitter.
func NewStdoutEmitter() *StdoutEmitter {
	return &StdoutEmitter{
		name: "stdout",
	}
}

// Name is the StdoutEmitter name.
func (se *StdoutEmitter) Name() string {
	return se.name
}

// Emit prints the metrics into stdout.
func (se *StdoutEmitter) Emit(metrics []Metric) error {
	b, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
