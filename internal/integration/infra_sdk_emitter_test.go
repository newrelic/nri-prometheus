package integration

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfraSdkEmitter_Name(t *testing.T) {
	// given
	e := NewInfraSdkEmitter(Specs{})
	assert.NotNil(t, e)

	// when
	actual := e.Name()

	// then
	expected := "infra-sdk"

	assert.Equal(t, expected, actual)
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
			e := NewInfraSdkEmitter(Specs{})

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
			assert.NoError(t, err)

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
					assert.NotEmpty(t, m.Dimensions)
					assert.Contains(t, m.Dimensions, "hostname")
					assert.Contains(t, m.Dimensions, "env")
				}
			}
		})
	}
}

func TestInfraSdkEmitter_HistogramEmitsCorrectValue(t *testing.T) {
	e := NewInfraSdkEmitter(Specs{})

	//TODO find way to emit with different values so we can test the delta calculation on the hist sum
	metrics := getHistogram(t)

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// when
	err := e.Emit(metrics)
	_ = w.Close()

	//then
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
			assert.NotEmpty(t, m.Dimensions)
			assert.Contains(t, m.Dimensions, "hostname")
			assert.Contains(t, m.Dimensions, "env")
			// in "prod" we do not include +Inf so it would have been 5
			assert.Len(t, m.Buckets, 5)
			assert.Equal(t, float64(6), m.SampleSum, "sampleSum")
			assert.Equal(t, uint64(3), m.SampleCount, "sampleCount")
		}
	}
}

func TestInfraSdkEmitter_SummaryEmitsCorrectValue(t *testing.T) {
	e := NewInfraSdkEmitter(Specs{})

	//TODO find way to emit with different values so we can test the delta calculation on the hist sum
	metrics := getSummary(t)

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// when
	err := e.Emit(metrics)
	_ = w.Close()

	//then
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
			assert.NotEmpty(t, m.Dimensions)
			assert.Contains(t, m.Dimensions, "hostname")
			assert.Contains(t, m.Dimensions, "env")
			assert.Equal(t, 0.0009405, m.SampleSum, "sampleSum")
			assert.Equal(t, uint64(7), m.SampleCount, "sampleCount")
			assert.Len(t, m.Quantiles, 5)
			for _, q := range m.Quantiles {
				assert.NotNil(t, q.Value)
				assert.NotNil(t, q.Quantile)
			}
		}
	}
}

func Test_Emitter_EmitsCorrectEntity(t *testing.T) {

	specs := Specs{
		SpecsByName: map[string]SpecDef{
			"redis": {
				Service: "redis",
				Entities: []EntityDef{
					{
						Type:       "instance",
						Properties: PropertiesDef{},
						Metrics: []MetricDef{
							{Name: "metric1"},
							{Name: "metric2"},
						},
					},
					{
						Type:       "database",
						Properties: PropertiesDef{},
						Metrics: []MetricDef{
							{Name: "redis_database_metric3"},
						},
					},
				},
				DefaultEntity: "instance",
			},
		},
	}

	emitter := NewInfraSdkEmitter(specs)

	gauges := getGauges(t)
	counters := getCounters(t)
	metrics := append(gauges, counters...)
	metrics = append(metrics, Metric{
		name:       "redis_database_metric3",
		value:      0.0,
		metricType: "gauge",
		attributes: nil,
	})

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// when
	err := emitter.Emit(metrics)
	_ = w.Close()

	//then
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
	// 2 entities: instance, database, "host"
	assert.Len(t, result.Entities, 3)
	for _, e := range result.Entities {
		assert.NotNil(t, e.Entity)
		// we cannot assert on the entity name and type being present
		// some metrics may be associated with the "host" entity because there is no service declared for their prefix.
		// for example: go_*
		assert.NotEmpty(t, e.Metrics)
	}
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
redis_exporter_scrapes_total{hostname="localhost",env="dev"} 3
`
	// when they are scraped
	metrics := scrapeString(t, input)
	return metrics.Metrics
}

//---- simplified structs mimicking the real Infra SDK output structure
type metadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type entityMetadata struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	EntityType  string                 `json:"type"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type common struct{}

type quant struct {
	Quantile *float64 `json:"quantile,omitempty"`
	Value    *float64 `json:"value,omitempty"`
}

type bucket struct {
	CumulativeCount *uint64  `json:"cumulative_count,omitempty"`
	UpperBound      *float64 `json:"upper_bound,omitempty"`
}

type metricData struct {
	Timestamp   int64             `json:"timestamp"`
	Name        string            `json:"name"`
	Dimensions  map[string]string `json:"attributes"`
	SampleCount uint64            `json:"sample_count,omitempty"`
	SampleSum   float64           `json:"sample_sum,omitempty"`
	Quantiles   []quant           `json:"quantiles,omitempty"`
	Buckets     []bucket          `json:"buckets,omitempty"`
}

type entityDef struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Metadata entityMetadata `json:"metadata,omitempty"`
}
type entity struct {
	Common  common       `json:"common"`
	Entity  entityDef    `json:"entity,omitempty"`
	Metrics []metricData `json:"metrics"`
}

type Result struct {
	ProtocolVersion string   `json:"protocol_version"`
	Metadata        metadata `json:"integration"`
	Entities        []entity `json:"data"`
}
