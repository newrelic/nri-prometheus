// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package endpoints ...
package endpoints

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
)

// TargetRetriever is implemented by any type that can return the URL of a set of Prometheus metrics providers
type TargetRetriever interface {
	GetTargets() ([]Target, error)
	Watch() error
	Name() string
}

// Object represents a kubernetes object like a pod or a service or an endpoint.
type Object struct {
	Name   string
	Kind   string
	Labels labels.Set
}

// Target is a prometheus endpoint which is exposed by an Object.
type Target struct {
	Name      string
	Object    Object
	URL       url.URL
	metadata  labels.Set
	TLSConfig TLSConfig
	// UseBearer tells nri-prometheus whether it should send the Kubernetes Service Account token as a Bearer token in
	// the HTTP request.
	UseBearer bool
}

// Metadata returns the Target's metadata, if the current metadata is nil,
// it's constructed from the Target's attributes, saved and returned.
// Subsequent calls will returned the already saved value.
func (t *Target) Metadata() labels.Set {
	if t.metadata == nil {
		metadata := labels.Set{}
		if targetURL := redactedURLString(&t.URL); targetURL != "" {
			metadata["scrapedTargetURL"] = targetURL
		}
		if t.Object.Name != "" {
			metadata["scrapedTargetName"] = t.Object.Name
			metadata["scrapedTargetKind"] = t.Object.Kind
		}
		labels.Accumulate(metadata, t.Object.Labels)

		t.metadata = metadata
	}
	return t.metadata
}

// redactedURLString returns the string representation of the URL object while redacting the password that could be present.
// This code is copied from this commit https://github.com/golang/go/commit/e3323f57df1f4a44093a2d25fee33513325cbb86.
// The feature is supposed to be added to the net/url.URL type in Golang 1.15.
func redactedURLString(u *url.URL) string {
	if u == nil {
		return ""
	}
	ru := *u
	if _, has := ru.User.Password(); has {
		ru.User = url.UserPassword(ru.User.Username(), "xxxxx")
	}
	return ru.String()
}

// endpointToTarget returns a list of Targets from the provided TargetConfig struct.
// The URL processing for every Target follows the next conventions:
// - if no schema is provided, it assumes http
// - if no path is provided, it assumes /metrics
// For example, hostname:8080 will be interpreted as http://hostname:8080/metrics
func endpointToTarget(tc TargetConfig) ([]Target, error) {
	targets := make([]Target, 0, len(tc.URLs))
	for _, URL := range tc.URLs {
		t, err := urlToTarget(URL, tc.TLSConfig)
		if err != nil {
			return nil, err
		}
		t.UseBearer = tc.UseBearer
		targets = append(targets, t)
	}
	return targets, nil
}

func urlToTarget(URL string, TLSConfig TLSConfig) (Target, error) {
	if !strings.Contains(URL, "://") {
		URL = fmt.Sprint("http://", URL)
	}

	u, err := url.Parse(URL)
	if err != nil {
		return Target{}, err
	}
	if u.Path == "" {
		u.Path = "/metrics"
	}

	return Target{
		Name: u.Host,
		Object: Object{
			Name:   u.Host,
			Kind:   "user_provided",
			Labels: make(labels.Set),
		},
		TLSConfig: TLSConfig,
		URL:       *u,
	}, nil
}
