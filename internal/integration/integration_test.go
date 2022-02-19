// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/newrelic/nri-prometheus/internal/pkg/endpoints"
	"github.com/newrelic/nri-prometheus/internal/synthesis"
	"github.com/stretchr/testify/assert"
)

type nilEmit struct{}

func (*nilEmit) Name() string {
	return "nil-emitter"
}

func (*nilEmit) Emit([]Metric) error {
	return nil
}

func BenchmarkIntegration(b *testing.B) {
	cachedFile, err := ioutil.ReadFile("test/cadvisor.txt")
	assert.NoError(b, err)
	contentLength := strconv.Itoa(len(cachedFile))
	b.Log("payload size", contentLength)
	server := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Length", contentLength)
		_, err := resp.Write(cachedFile)
		assert.NoError(b, err)
	}))
	defer server.Close()

	fr, err := endpoints.FixedRetriever(endpoints.TargetConfig{URLs: []string{server.URL}})
	assert.NoError(b, err)
	var retrievers []endpoints.TargetRetriever
	for i := 0; i < 20; i++ {
		retrievers = append(retrievers, fr)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		do(b, retrievers)
	}
}

func do(b *testing.B, retrievers []endpoints.TargetRetriever) {
	b.ReportAllocs()
	process(
		retrievers,
		NewFetcher(30*time.Second, 5000000000, 4, "", "", false, queueLength),
		RuleProcessor([]ProcessingRule{}, queueLength),
		AnnotationRulesProcessor,
		[]Emitter{&nilEmit{}},
	)
}

func BenchmarkIntegrationInfraSDKEmitter(b *testing.B) {
	cachedFile, err := ioutil.ReadFile("test/cadvisor.txt")
	assert.NoError(b, err)
	contentLength := strconv.Itoa(len(cachedFile))
	b.Log("payload size", contentLength)
	server := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Length", contentLength)
		_, err := resp.Write(cachedFile)
		assert.NoError(b, err)
	}))
	defer server.Close()

	fr, err := endpoints.FixedRetriever(endpoints.TargetConfig{URLs: []string{server.URL}})
	assert.NoError(b, err)
	var retrievers []endpoints.TargetRetriever
	for i := 0; i < 20; i++ {
		retrievers = append(retrievers, fr)
	}

	sd := []synthesis.Definition{
		{
			EntityRule: synthesis.EntityRule{
				EntityType: "CONTAINER",
				Identifier: "id",
				Name:       "container_name",
				Conditions: []synthesis.Condition{
					{
						Attribute: "metricName",
						Prefix:    "container_",
					},
				},
				Tags: synthesis.Tags{
					"namespace":  nil,
					"targetName": nil,
				},
			},
		},
	}

	s := synthesis.NewSynthesizer(sd)
	emitter := NewInfraSdkEmitter(s)
	emitters := []Emitter{emitter}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ExecuteOnce(
			retrievers,
			NewFetcher(30*time.Second, 5000000000, 4, "", "", false, queueLength),
			RuleProcessor([]ProcessingRule{}, queueLength),
			AnnotationRulesProcessor,
			emitters)
	}
}
