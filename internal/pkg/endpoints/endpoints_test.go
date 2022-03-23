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
		hostID       string
	}{
		{
			testName:     "default schema and path",
			input:        "somehost",
			expectedName: "somehost",
			expectedURL:  "http://somehost/metrics",
			hostID:       "a-host-id",
		},
		{
			testName:     "default schema and path, provided port",
			input:        "somehost:8080",
			expectedName: "somehost:8080",
			expectedURL:  "http://somehost:8080/metrics",
			hostID:       "a-host-id",
		},
		{
			testName:     "default path, provided port and schema",
			input:        "https://somehost:8080",
			expectedName: "somehost:8080",
			expectedURL:  "https://somehost:8080/metrics",
			hostID:       "a-host-id",
		},
		{
			testName:     "default schema",
			input:        "somehost:8080/path",
			expectedName: "somehost:8080",
			expectedURL:  "http://somehost:8080/path",
			hostID:       "a-host-id",
		},
		{
			testName:     "with URL params",
			input:        "somehost:8080/path/with/params?format=prometheus(123)",
			expectedName: "somehost:8080",
			expectedURL:  "http://somehost:8080/path/with/params?format=prometheus(123)",
			hostID:       "a-host-id",
		},
		{
			testName:     "provided all",
			input:        "https://somehost:8080/path",
			expectedName: "somehost:8080",
			expectedURL:  "https://somehost:8080/path",
			hostID:       "a-host-id",
		},
		{
			testName:     "provided all with IP 128.0.0.1",
			input:        "https://128.0.0.1:8080/path",
			expectedName: "128.0.0.1:8080",
			expectedURL:  "https://128.0.0.1:8080/path",
			hostID:       "a-host-id",
		},
		{
			testName:     "provided host id modifying name if localhost",
			input:        "https://localhost:8080/path",
			expectedName: "a-host-id:8080",
			expectedURL:  "https://localhost:8080/path",
			hostID:       "a-host-id",
		},
		{
			testName:     "empty host id modifying name if LOCALHOST",
			input:        "https://LOCALHOST:8080/path",
			expectedName: "a-host-id:8080",
			expectedURL:  "https://LOCALHOST:8080/path",
			hostID:       "a-host-id",
		},
		{
			testName:     "empty host id modifying name if 127.0.0.1",
			input:        "https://127.0.0.1:8080/path",
			expectedName: "a-host-id:8080",
			expectedURL:  "https://127.0.0.1:8080/path",
			hostID:       "a-host-id",
		},
		{
			testName:     "empty host id not modifying if empty",
			input:        "https://localhost:8080/path",
			expectedName: "localhost:8080",
			expectedURL:  "https://localhost:8080/path",
			hostID:       "",
		},
	}
	for _, c := range cases {
		c := c

		t.Run(c.testName, func(t *testing.T) {
			t.Parallel()

			targets, err := endpointToTarget(TargetConfig{URLs: []string{c.input}}, c.hostID)
			assert.NoError(t, err)
			assert.Len(t, targets, 1)
			assert.Equal(t, c.expectedName, targets[0].Name)
			assert.Equal(t, c.expectedName, targets[0].Object.Name)
			assert.Equal(t, c.expectedURL, targets[0].URL.String())
		})
	}
}
