// +build loadtests

package main

import (
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
	"io"
	"log"
	"os"
	"testing"
)

const timeLimit = 40
const targetExpected = 800
const memoryLimit = 2 * 1e9
const filename = "load_test.results"

func TestLoad(t *testing.T) {
	mfs := parsePrometheusFile(t, filename)

	require.LessOrEqual(t, *mfs["nr_stats_integration_process_duration_seconds"].Metric[0].Gauge.Value, float64(timeLimit), "taking too much time to process metrics")
	log.Printf("nr_stats_integration_process_duration_seconds: %f", *mfs["nr_stats_integration_process_duration_seconds"].Metric[0].Gauge.Value)

	require.LessOrEqual(t, *mfs["process_resident_memory_bytes"].Metric[0].Gauge.Value, float64(memoryLimit), "taking too much time to process metrics")
	log.Printf("memory consumption (process_resident_memory_bytes): %fMB", *mfs["process_resident_memory_bytes"].Metric[0].Gauge.Value/1e6)

	for _, m := range mfs["nr_stats_targets"].Metric {
		if *m.Label[0].Value == "kubernetes" {
			require.GreaterOrEqual(t, *m.Gauge.Value, float64(targetExpected), "missing targets")
			log.Printf("Number of targets scraped: %f", *m.Gauge.Value)
		}
	}

}

// MetricFamiliesByName is a map of Prometheus metrics family names and their representation.
type MetricFamiliesByName map[string]dto.MetricFamily

func parsePrometheusFile(t *testing.T, filename string) MetricFamiliesByName {
	mfs := MetricFamiliesByName{}

	file, err := os.Open(filename)
	defer file.Close()

	require.NoError(t, err, "No error expected")

	d := expfmt.NewDecoder(file, expfmt.TextVersion)
	for {
		var mf dto.MetricFamily
		if err := d.Decode(&mf); err != nil {
			if err == io.EOF {
				break
			}
			require.NoError(t, err, "The only accepted error is EOF")
		}
		mfs[mf.GetName()] = mf
	}
	return mfs
}
