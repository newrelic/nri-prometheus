// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package endpoints ...
package endpoints

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
)

// A different regex is needed for replacing because `localhostRE` matches
// IPV6 by using extra `:` that don't belong to the IP but are separators.
var localhostReplaceRE = regexp.MustCompile(`(localhost|LOCALHOST|127(?:\.[0-9]+){0,2}\.[0-9]+|::1)`)

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
func endpointToTarget(tc TargetConfig, hostID string) ([]Target, error) {
	targets := make([]Target, 0, len(tc.URLs))
	for _, URL := range tc.URLs {
		t, err := urlToTarget(URL, tc.TLSConfig, hostID)
		if err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

func urlToTarget(URL string, TLSConfig TLSConfig, hostID string) (Target, error) {
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

	targetName := u.Host
	if hostID != "" {
		targetName = replaceLocalhost(u.Host, hostID)
	}

	return Target{
		Name: targetName,
		Object: Object{
			Name:   targetName,
			Kind:   "user_provided",
			Labels: make(labels.Set),
		},
		TLSConfig: TLSConfig,
		URL:       *u,
	}, nil
}

// ReplaceLocalhost replaces the occurrence of a localhost address with
// the given hostname.
func replaceLocalhost(source, with string) string {
	return localhostReplaceRE.ReplaceAllString(source, with)
}
