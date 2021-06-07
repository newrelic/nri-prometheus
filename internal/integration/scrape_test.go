// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
)

func TestScrape(t *testing.T) {
	t.Parallel()

	// Given a set of fetched metrics
	input := `# HELP redis_exporter_build_info redis exporter build_info
# TYPE redis_exporter_build_info gauge
redis_exporter_build_info{build_date="2018-07-03-14:18:56",commit_sha="3e15af27aac37e114b32a07f5e9dc0510f4cbfc4",golang_version="go1.9.4",version="v0.20.2"} 1
# HELP redis_exporter_scrapes_total Current total redis scrapes.
# TYPE redis_exporter_scrapes_total counter
redis_exporter_scrapes_total{cosa="fina"} 42
# HELP redis_instance_info Information about the Redis instance
# TYPE redis_instance_info gauge
redis_instance_info{addr="ohai-playground-redis-master:6379",alias="ohai-playground-redis",os="Linux 4.15.0 x86_64",redis_build_id="c701a4acd98ea64a",redis_mode="standalone",redis_version="4.0.10",role="master"} 1
redis_instance_info{addr="ohai-playground-redis-slave:6379",alias="ohai-playground-redis",os="Linux 4.15.0 x86_64",redis_build_id="c701a4acd98ea64a",redis_mode="standalone",redis_version="4.0.10",role="slave"} 1
# HELP redis_instantaneous_input_kbps instantaneous_input_kbpsmetric
# TYPE redis_instantaneous_input_kbps gauge
redis_instantaneous_input_kbps{addr="ohai-playground-redis-master:6379",alias="ohai-playground-redis"} 0.05
redis_instantaneous_input_kbps{addr="ohai-playground-redis-slave:6379",alias="ohai-playground-redis"} 0
`
	// when they are scraped
	pair := scrapeString(t, input)

	// The returned input contains all the expected metrics
	assert.NotEmpty(t, pair.Target.Name)
	assert.NotEmpty(t, pair.Target.URL)
	assert.Len(t, pair.Metrics, 6)

	for _, metric := range pair.Metrics {
		switch metric.name {
		case "redis_exporter_scrapes_total":
		case "redis_instantaneous_input_kbps":
			switch metric.attributes["addr"] {
			case "ohai-playground-redis-slave:6379":
				expected := labels.Set{
					"addr":  "ohai-playground-redis-slave:6379",
					"alias": "ohai-playground-redis",
				}
				AssertContainsTree(t, metric.attributes, expected)
			case "ohai-playground-redis-master:6379":
				expected := labels.Set{
					"addr":  "ohai-playground-redis-master:6379",
					"alias": "ohai-playground-redis",
				}
				AssertContainsTree(t, metric.attributes, expected)
			default:
				assert.Failf(t, "unexpected addr field:", "%#v", metric.attributes)
			}
		case "redis_exporter_build_info":
			expected := labels.Set{
				"build_date":     "2018-07-03-14:18:56",
				"commit_sha":     "3e15af27aac37e114b32a07f5e9dc0510f4cbfc4",
				"golang_version": "go1.9.4",
				"version":        "v0.20.2",
			}
			AssertContainsTree(t, metric.attributes, expected)
		case "redis_instance_info":
			switch metric.attributes["addr"] {
			case "ohai-playground-redis-slave:6379":
				expected := labels.Set{
					"addr":           "ohai-playground-redis-slave:6379",
					"alias":          "ohai-playground-redis",
					"os":             "Linux 4.15.0 x86_64",
					"redis_build_id": "c701a4acd98ea64a",
					"redis_mode":     "standalone",
					"redis_version":  "4.0.10",
					"role":           "slave",
				}
				AssertContainsTree(t, metric.attributes, expected)
			case "ohai-playground-redis-master:6379":
				expected := labels.Set{
					"addr":           "ohai-playground-redis-master:6379",
					"alias":          "ohai-playground-redis",
					"os":             "Linux 4.15.0 x86_64",
					"redis_build_id": "c701a4acd98ea64a",
					"redis_mode":     "standalone",
					"redis_version":  "4.0.10",
					"role":           "master",
				}
				AssertContainsTree(t, metric.attributes, expected)
			default:
				assert.Failf(t, "unexpected addr field:", "%#v", metric.attributes)
			}
		default:
			assert.True(t, strings.HasSuffix(metric.name, "_info"), "unexpected metric %s", metric.name)
		}
	}
}
