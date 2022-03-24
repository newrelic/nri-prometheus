// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/newrelic/nri-prometheus/internal/pkg/endpoints"
	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	"github.com/stretchr/testify/assert"
)

func TestInfraSdkEmitter_Name(t *testing.T) {
	t.Parallel()

	// given
	e := NewInfraSdkEmitter("")
	assert.NotNil(t, e)

	// when
	actual := e.Name()

	// then
	expected := "infra-sdk"

	assert.Equal(t, expected, actual)
}

func TestInfraSdkEmitter_InvalidMetadata(t *testing.T) {
	e := NewInfraSdkEmitter("")
	invalid := Metadata{Name: "test", Version: ""}
	valid := Metadata{Name: "foo", Version: "bar"}
	assert.Error(t, e.SetIntegrationMetadata(invalid))
	assert.NoError(t, e.SetIntegrationMetadata(valid))
}

func TestInfraSdkEmitter_Emit(t *testing.T) {
	type args struct {
		in0 []Metric
	}
	tests := []struct {
		name         string
		args         args
		wantEntities int
		wantMetrics  int
	}{
		{
			name:         "CanEmitGauges",
			args:         args{getGauges(t)},
			wantEntities: 1,
			wantMetrics:  5,
		},
		{
			name:         "CanEmitCounters",
			args:         args{getCounters(t)},
			wantEntities: 1,
			wantMetrics:  5,
		},
		{
			name:         "CanEmitSummary",
			args:         args{getSummary(t)},
			wantEntities: 1,
			wantMetrics:  1,
		},
		{
			name:         "CanEmitHistogram",
			args:         args{getHistogram(t)},
			wantEntities: 1,
			wantMetrics:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			e := NewInfraSdkEmitter("")

			rescueStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// when
			assert.NoError(t, e.Emit(tt.args.in0))

			_ = w.Close()
			bytes, _ := ioutil.ReadAll(r)
			assert.NotEmpty(t, bytes)
			os.Stdout = rescueStdout

			// print json for debug purposes
			t.Log(string(bytes))

			// then
			// convert the json into a similar metric structure so we can assert more easily
			var result Result
			err := json.Unmarshal(bytes, &result)
			// errors from unmarshal not checked since Result struct is a Mock for summary and histogram
			if err != nil {
				t.Log(err)
			}

			assert.NotEmpty(t, result.ProtocolVersion)
			assert.NotNil(t, result.Metadata)
			assert.NotEmpty(t, result.Metadata.Name)
			assert.NotEmpty(t, result.Metadata.Version)
			assert.Len(t, result.Entities, tt.wantEntities)
			for _, e := range result.Entities {
				assert.Len(t, e.Metrics, tt.wantMetrics)
				for _, m := range e.Metrics {
					assert.NotZero(t, m.Timestamp)
					assert.NotEmpty(t, m.Name)
					assert.NotEmpty(t, m.Labels)
					assert.Contains(t, m.Labels, "hostname")
					assert.Contains(t, m.Labels, "env")
				}
			}
		})
	}
}

func TestInfraSdkEmitter_HistogramEmitsCorrectValue(t *testing.T) {
	e := NewInfraSdkEmitter("")

	// TODO find way to emit with different values so we can test the delta calculation on the hist sum
	metrics := getHistogram(t)

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// when
	err := e.Emit(metrics)
	_ = w.Close()

	// then
	assert.NoError(t, err)
	bytes, _ := ioutil.ReadAll(r)
	assert.NotEmpty(t, bytes)
	os.Stdout = rescueStdout

	// print json for debug purposes
	t.Log(string(bytes))

	// convert the json into a similar metric structure so we can assert more easily
	var result Result
	err = json.Unmarshal(bytes, &result)
	assert.NoError(t, err)

	assert.NotEmpty(t, result.ProtocolVersion)
	assert.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.Name)
	assert.NotEmpty(t, result.Metadata.Version)
	assert.Len(t, result.Entities, 1)
	for _, e := range result.Entities {
		assert.Len(t, e.Metrics, 1)
		for _, m := range e.Metrics {
			assert.NotZero(t, m.Timestamp)
			assert.NotEmpty(t, m.Name)
			assert.NotEmpty(t, m.Labels)
			assert.Contains(t, m.Labels, "hostname")
			assert.Contains(t, m.Labels, "env")
			// in "prod" we do not include +Inf so it would have been 5
			assert.Len(t, m.Value.Buckets, 5)
			assert.Equal(t, float64(6), m.Value.SampleSum, "sampleSum")
			assert.Equal(t, uint64(3), m.Value.SampleCount, "sampleCount")
		}
	}
}

func TestInfraSdkEmitter_SummaryEmitsCorrectValue(t *testing.T) {
	t.Parallel()

	e := NewInfraSdkEmitter("a-host-id")

	// TODO find way to emit with different values so we can test the delta calculation on the hist sum
	metrics := getSummary(t)

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// when
	err := e.Emit(metrics)
	_ = w.Close()

	// then
	assert.NoError(t, err)
	bytes, _ := ioutil.ReadAll(r)
	assert.NotEmpty(t, bytes)
	os.Stdout = rescueStdout

	// print json for debug purposes
	t.Log(string(bytes))

	// convert the json into a similar metric structure so we can assert more easily
	var result Result
	err = json.Unmarshal(bytes, &result)
	assert.NoError(t, err)

	assert.NotEmpty(t, result.ProtocolVersion)
	assert.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.Name)
	assert.NotEmpty(t, result.Metadata.Version)
	assert.Len(t, result.Entities, 1)
	for _, e := range result.Entities {
		assert.Len(t, e.Metrics, 1)
		for _, m := range e.Metrics {
			assert.NotZero(t, m.Timestamp)
			assert.NotEmpty(t, m.Name)
			assert.NotEmpty(t, m.Labels)
			assert.Contains(t, m.Labels, "hostname")
			assert.Contains(t, m.Labels, "env")
			assert.Equal(t, 0.0009405, m.Value.SampleSum, "sampleSum")
			assert.Equal(t, uint64(7), m.Value.SampleCount, "sampleCount")
			assert.Len(t, m.Value.Quantiles, 5)
			for _, q := range m.Value.Quantiles {
				assert.NotNil(t, q.Value)
				assert.NotNil(t, q.Quantile)
			}
		}
	}
}

func Test_Emitter_EmitsEntity(t *testing.T) {
	t.Parallel()

	testMetadata := Metadata{
		Name:    "nri-foo",
		Version: "test",
	}

	emitter := NewInfraSdkEmitter("a-host-id")
	assert.NoError(t, emitter.SetIntegrationMetadata(testMetadata))
	// and this exporter input metrics
	input := `
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total{hostname="localhost",env="dev"} 0.04
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines{hostname="localhost",env="dev"} 7
# HELP foo_bar Test metric for multirule.
# TYPE foo_bar gauge
foo_bar{hostname="localhost",env="dev",foo="foo"} 0
# HELP bar_foo Test metric for multirule.
# TYPE bar_foo gauge
bar_foo{hostname="localhost",env="dev",bar="bar"} 1
# HELP redis_exporter_build_info redis exporter build_info
# TYPE redis_exporter_build_info gauge
redis_exporter_build_info{hostname="localhost",env="dev",build_date="2020-08-18-01:07:46",commit_sha="bac1cfead5cdb77dbce3ad567c9786f11424cf02",golang_version="go1.14.7",version="v1.10.0"} 1
# HELP redis_exporter_last_scrape_connect_time_seconds exporter_last_scrape_connect_time_seconds metric
# TYPE redis_exporter_last_scrape_connect_time_seconds gauge
redis_exporter_last_scrape_connect_time_seconds{hostname="localhost",env="dev"} 0.003180941
# HELP redis_exporter_scrapes_total Current total redis scrapes.
# TYPE redis_exporter_scrapes_total counter
redis_exporter_scrapes_total{hostname="localhost",env="dev",uniquelabel="test"} 3
# HELP redis_foo_scrapes_total Test metric.
# TYPE redis_foo_scrapes_total gauge
redis_foo_test{hostname="localhost",env="dev",uniquelabel="test"} 3
`

	// when they are scraped
	metrics := scrapeString(t, input)

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// and processed by the sdk emitter
	err := emitter.Emit(metrics.Metrics)
	_ = w.Close()

	// then
	assert.NoError(t, err)
	bytes, _ := ioutil.ReadAll(r)
	assert.NotEmpty(t, bytes)
	os.Stdout = rescueStdout

	// print json for debug purposes
	t.Log(string(bytes))

	var result Result
	assert.Error(t, json.Unmarshal(bytes, &result))

	assert.Equal(t, testMetadata.Name, result.Metadata.Name)
	assert.Equal(t, testMetadata.Version, result.Metadata.Version)

	assert.Len(t, result.Entities, 1)
	e, ok := result.findEntity("")
	assert.True(t, ok)
	assert.Len(t, e.Metrics, 8)

	// Metrics does not contain common nor removed attributes.
	for _, m := range e.Metrics {
		for k := range commonAttributes {
			_, ok := m.Labels[k]
			assert.False(t, ok)
		}
		for k := range removedAttributes {
			_, ok := m.Labels[k]
			assert.False(t, ok)
		}
	}

	assert.Contains(t, e.Common.Attributes, "targetName")
}

func Test_Emitter_EmitsEntityWithCorrectTargetName(t *testing.T) {
	cases := []struct {
		testName     string
		input        TargetMetrics
		expectedName string
		expectedURL  string
		hostID       string
	}{
		{
			testName: "when hostID provided but host does not match localhostReplaceRE  ",
			input: TargetMetrics{
				Metrics: []Metric{
					{
						name:       "a-metric",
						value:      float64(3),
						metricType: "count",
						attributes: labels.Set{
							"targetName":        "128.0.0.1:8080",
							"scrapedTargetName": "128.0.0.1:8080",
							"scrapedTargetURL":  "https://128.0.0.1:8080",
							"env":               "dev",
						},
					},
				},
				Target: endpoints.Target{},
			},
			expectedName: "128.0.0.1:8080",
			expectedURL:  "https://128.0.0.1:8080",
			hostID:       "a-host-id",
		},
		{
			testName: "provided host id modifying name if localhost",
			input: TargetMetrics{
				Metrics: []Metric{
					{
						name:       "a-metric",
						value:      float64(3),
						metricType: "count",
						attributes: labels.Set{
							"targetName":        "localhost:8080",
							"scrapedTargetName": "localhost:8080",
							"scrapedTargetURL":  "https://localhost:8080",
							"env":               "dev",
						},
					},
				},
				Target: endpoints.Target{},
			},
			expectedName: "a-host-id:8080",
			expectedURL:  "https://localhost:8080",
			hostID:       "a-host-id",
		},
		{
			testName: "empty host id modifying name if LOCALHOST",
			input: TargetMetrics{
				Metrics: []Metric{
					{
						name:       "a-metric",
						value:      float64(3),
						metricType: "count",
						attributes: labels.Set{
							"targetName":        "LOCALHOST:8080",
							"scrapedTargetName": "LOCALHOST:8080",
							"scrapedTargetURL":  "https://LOCALHOST:8080",
							"env":               "dev",
						},
					},
				},
				Target: endpoints.Target{},
			},
			expectedName: "a-host-id:8080",
			expectedURL:  "https://LOCALHOST:8080",
			hostID:       "a-host-id",
		},
		{
			testName: "empty host id modifying name if 127.0.0.1",
			input: TargetMetrics{
				Metrics: []Metric{
					{
						name:       "a-metric",
						value:      float64(3),
						metricType: "count",
						attributes: labels.Set{
							"targetName":        "127.0.0.1:8080",
							"scrapedTargetName": "127.0.0.1:8080",
							"scrapedTargetURL":  "https://127.0.0.1:8080",
							"env":               "dev",
						},
					},
				},
				Target: endpoints.Target{},
			},
			expectedName: "a-host-id:8080",
			expectedURL:  "https://127.0.0.1:8080",
			hostID:       "a-host-id",
		},
		{
			testName: "empty host id not modifying if empty",
			input: TargetMetrics{
				Metrics: []Metric{
					{
						name:       "a-metric",
						value:      float64(3),
						metricType: "count",
						attributes: labels.Set{
							"targetName":        "localhost:8080",
							"scrapedTargetName": "localhost:8080",
							"scrapedTargetURL":  "https://localhost:8080",
							"env":               "dev",
						},
					},
				},
				Target: endpoints.Target{},
			},
			expectedName: "localhost:8080",
			expectedURL:  "https://localhost:8080",
			hostID:       "",
		},
	}

	testMetadata := Metadata{
		Name:    "nri-foo",
		Version: "test",
	}

	for _, c := range cases {
		c := c

		t.Run(c.testName, func(t *testing.T) {

			emitter := NewInfraSdkEmitter(c.hostID)
			assert.NoError(t, emitter.SetIntegrationMetadata(testMetadata))

			rescueStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// when input processed by the sdk emitter
			err := emitter.Emit(c.input.Metrics)
			_ = w.Close()

			assert.NoError(t, err)
			bytes, _ := ioutil.ReadAll(r)
			assert.NotEmpty(t, bytes)
			os.Stdout = rescueStdout

			var result Result
			assert.Error(t, json.Unmarshal(bytes, &result))
			e, ok := result.findEntity("")
			assert.True(t, ok)
			assert.Equal(t, c.expectedName, e.Common.Attributes["targetName"])
			assert.Equal(t, c.expectedName, e.Common.Attributes["scrapedTargetName"])
			assert.Equal(t, c.expectedURL, e.Common.Attributes["scrapedTargetURL"])
		})
	}

}

func Test_ResizeToLimit(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	for i := 0; i < 10; i++ {
		sb.WriteString("token")
		sb.WriteRune(':')
	}
	original := sb.Len()

	resized := resizeToLimit(&sb)
	// no change
	assert.False(t, resized)
	assert.Equal(t, original, sb.Len())

	sb.Reset()

	// going over the limit
	for i := 0; i < 110; i++ {
		sb.WriteString("token")
		sb.WriteRune(':')
	}
	original = sb.Len()

	resized = resizeToLimit(&sb)
	// should have been resized
	assert.True(t, resized)
	assert.Less(t, sb.Len(), original)
}

func getHistogram(t *testing.T) []Metric {
	input := `
# HELP http_request_duration_seconds request duration histogram
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.5",hostname="localhost",env="dev"} 0
http_request_duration_seconds_bucket{le="1",hostname="localhost",env="dev"} 1
http_request_duration_seconds_bucket{le="2",hostname="localhost",env="dev"} 2
http_request_duration_seconds_bucket{le="3",hostname="localhost",env="dev"} 3
http_request_duration_seconds_bucket{le="5",hostname="localhost",env="dev"} 3
http_request_duration_seconds_bucket{le="+Inf",hostname="localhost",env="dev"} 3
http_request_duration_seconds_sum{hostname="localhost",env="dev"} 6
http_request_duration_seconds_count{hostname="localhost",env="dev"} 3
`
	// when they are scraped
	metrics := scrapeString(t, input)
	return metrics.Metrics
}

func getSummary(t *testing.T) []Metric {
	input := `
# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0",hostname="localhost",env="dev"} 8.27e-05
go_gc_duration_seconds{quantile="0.25",hostname="localhost",env="dev"} 8.92e-05
go_gc_duration_seconds{quantile="0.5",hostname="localhost",env="dev"} 0.0001236
go_gc_duration_seconds{quantile="0.75",hostname="localhost",env="dev"} 0.0002019
go_gc_duration_seconds{quantile="1",hostname="localhost",env="dev"} 0.0002212
go_gc_duration_seconds_sum{hostname="localhost",env="dev"} 0.0009405
go_gc_duration_seconds_count{hostname="localhost",env="dev"} 7
`
	// when they are scraped
	metrics := scrapeString(t, input)
	return metrics.Metrics
}

// all gauge metrics
func getGauges(t *testing.T) []Metric {
	input := `
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines{hostname="localhost",env="dev"} 7
# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes{hostname="localhost",env="dev"} 1.163824e+06
# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds{hostname="localhost",env="dev"} 11
# HELP redis_exporter_build_info redis exporter build_info
# TYPE redis_exporter_build_info gauge
redis_exporter_build_info{hostname="localhost",env="dev",build_date="2020-08-18-01:07:46",commit_sha="bac1cfead5cdb77dbce3ad567c9786f11424cf02",golang_version="go1.14.7",version="v1.10.0"} 1
# HELP redis_exporter_last_scrape_connect_time_seconds exporter_last_scrape_connect_time_seconds metric
# TYPE redis_exporter_last_scrape_connect_time_seconds gauge
redis_exporter_last_scrape_connect_time_seconds{hostname="localhost",env="dev"} 0.003180941
`
	// when they are scraped
	metrics := scrapeString(t, input)
	return metrics.Metrics
}

// all counter metrics
func getCounters(t *testing.T) []Metric {
	input := `
# HELP go_memstats_alloc_bytes_total Total number of bytes allocated, even if freed.
# TYPE go_memstats_alloc_bytes_total counter
go_memstats_alloc_bytes_total{hostname="localhost",env="dev"} 967400
# HELP go_memstats_frees_total Total number of frees.
# TYPE go_memstats_frees_total counter
go_memstats_frees_total{hostname="localhost",env="dev"} 242
# HELP go_memstats_mallocs_total Total number of mallocs.
# TYPE go_memstats_mallocs_total counter
go_memstats_mallocs_total{hostname="localhost",env="dev"} 4705
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total{hostname="localhost",env="dev"} 0.04
# HELP redis_exporter_scrapes_total Current total redis scrapes.
# TYPE redis_exporter_scrapes_total counter
redis_exporter_scrapes_total{hostname="localhost",env="dev",uniquelabel="test"} 3
`
	// when they are scraped
	metrics := scrapeString(t, input)
	return metrics.Metrics
}

//---- simplified structs mimicking the real Infra SDK output structure
type entityMetadata struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	EntityType  string                 `json:"type"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type common struct {
	Timestamp  *int64                 `json:"timestamp,omitempty"`
	Interval   *int64                 `json:"interval.ms,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type quant struct {
	Quantile *float64 `json:"quantile,omitempty"`
	Value    *float64 `json:"value,omitempty"`
}

type bucket struct {
	CumulativeCount *uint64  `json:"cumulative_count,omitempty"`
	UpperBound      *float64 `json:"upper_bound,omitempty"`
}

type metricData struct {
	Timestamp int64               `json:"timestamp"`
	Name      string              `json:"name"`
	Labels    map[string]string   `json:"attributes"`
	Value     PrometheusMockValue `json:"value,omitempty"`
}

type PrometheusMockValue struct {
	SampleCount uint64  `json:"sample_count,omitempty"`
	SampleSum   float64 `json:"sample_sum,omitempty"`
	// Buckets defines the buckets into which observations are counted. Each
	// element in the slice is the upper inclusive bound of a bucket. The
	// values must are sorted in strictly increasing order.
	Buckets   []*bucket `json:"buckets,omitempty"`
	Quantiles []quant   `json:"quantiles,omitempty"`
}

type entity struct {
	Common    common         `json:"common"`
	EntityDef entityMetadata `json:"entity,omitempty"`
	Metrics   []metricData   `json:"metrics"`
}

type Result struct {
	ProtocolVersion string   `json:"protocol_version"`
	Metadata        Metadata `json:"integration"`
	Entities        []entity `json:"data"`
}

func (r Result) findEntity(name string) (entity, bool) {
	for _, e := range r.Entities {
		if e.EntityDef.Name == name {
			return e, true
		}
	}

	return entity{}, false
}
