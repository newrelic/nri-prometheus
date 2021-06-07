// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package endpoints

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		testName     string
		input        string
		expectedName string
		expectedURL  string
	}{
		{
			testName:     "default schema and path",
			input:        "somehost",
			expectedName: "somehost",
			expectedURL:  "http://somehost/metrics",
		},
		{
			testName:     "default schema and path, provided port",
			input:        "somehost:8080",
			expectedName: "somehost:8080",
			expectedURL:  "http://somehost:8080/metrics",
		},
		{
			testName:     "default path, provided port and schema",
			input:        "https://somehost:8080",
			expectedName: "somehost:8080",
			expectedURL:  "https://somehost:8080/metrics",
		},
		{
			testName:     "default schema",
			input:        "somehost:8080/path",
			expectedName: "somehost:8080",
			expectedURL:  "http://somehost:8080/path",
		},
		{
			testName:     "with URL params",
			input:        "somehost:8080/path/with/params?format=prometheus(123)",
			expectedName: "somehost:8080",
			expectedURL:  "http://somehost:8080/path/with/params?format=prometheus(123)",
		},
		{
			testName:     "provided all",
			input:        "https://somehost:8080/path",
			expectedName: "somehost:8080",
			expectedURL:  "https://somehost:8080/path",
		},
	}
	for _, c := range cases {
		c := c

		t.Run(c.testName, func(t *testing.T) {
			t.Parallel()

			targets, err := EndpointToTarget(TargetConfig{URLs: []string{c.input}})
			assert.NoError(t, err)
			assert.Len(t, targets, 1)
			assert.Equal(t, c.expectedName, targets[0].Name)
			assert.Equal(t, c.expectedURL, targets[0].URL.String())
		})
	}
}
