package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	var buf bytes.Buffer
	tee := io.TeeReader(os.Stdin, &buf)

	metadata := Metadata{}
	decoder := json.NewDecoder(tee)
	if err := decoder.Decode(&metadata); err != nil {
		log.Fatal(err)
	}

	configs, err := metadata.MetricTypeRelabelConfigs()
	if err != nil {
		log.Fatal(err)
	}

	output, err := yaml.Marshal(configs)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s", output)
}

func hasCounterSuffix(metricName string) bool {
	if strings.HasSuffix(metricName, "_total") {
		return true
	}
	if strings.HasSuffix(metricName, "_count") {
		return true
	}
	if strings.HasSuffix(metricName, "_sum") {
		return true
	}
	if strings.HasSuffix(metricName, "_bucket") {
		return true
	}
	return false
}

type Metadata struct {
	Status string                      `json:"status"`
	Data   map[string][]MetricMetadata `json:"data"`
}

type MetricMetadata struct {
	Type string `json:"type"`
	Help string `json:"help"`
	Unit string `json:"unit"`
}

// MetricTypeRelabelConfigs generates relabel configs needed for the New Relic Prometheus Remote Write
// to use the real metric type instead of the inferred type based on the metric name.
func (m Metadata) MetricTypeRelabelConfigs() (*WriteRelabelConfigs, error) {
	rc := []RelabelConfig{}

	if m.Status != "success" {
		return nil, fmt.Errorf("checking metadata: status not successful")
	}

	for metricName, metricsMetadata := range m.Data {
		if len(metricsMetadata) != 1 {
			log.Printf("Metric %s skipped since contains more than 1 metadata", metricName)
			continue
		}

		switch metricsMetadata[0].Type {
		case "counter":
			if !hasCounterSuffix(metricName) {
				rc = append(rc, RelabelConfig{
					SourceLabels: "[__name__]",
					Regex:        fmt.Sprintf("^%s$", metricName),
					TargetLabel:  "newrelic_metric_type",
					Replacement:  "counter",
					Action:       "replace",
				})
			}

		case "gauge":
			if hasCounterSuffix(metricName) {
				rc = append(rc, RelabelConfig{
					SourceLabels: "[__name__]",
					Regex:        fmt.Sprintf("^%s$", metricName),
					TargetLabel:  "newrelic_metric_type",
					Replacement:  "gauge",
					Action:       "replace",
				})
			}

			// TODO check behaivor of NR with summary. Prometheus adds the _sum and _count suffix automatically to these metrics
			// so by default they will be consider as count. According to doc _sum should be treated as summary.
			// https://docs.newrelic.com/docs/infrastructure/prometheus-integrations/view-query-data/translate-promql-queries-nrql#compare
			//
			// case "summary":
			// 	rc = append(rc, WriteRelabelConfigs{
			// 		SourceLabels: "[__name__]",
			//	    // metricName is the baseName of the metric.
			// 		Regex:        fmt.Sprintf("^%s_sum$", metricName),
			// 		TargetLabel:  "newrelic_metric_type",
			// 		Replacement:  "summary",
			// 		Action:       "replace",
			// 	})

		}
	}

	return &WriteRelabelConfigs{rc}, nil
}

type WriteRelabelConfigs struct {
	WriteRelabelConfigs []RelabelConfig `yaml:"write_relabel_configs"`
}

type RelabelConfig struct {
	SourceLabels string `yaml:"source_labels"`
	Regex        string `yaml:"regex"`
	TargetLabel  string `yaml:"target_label"`
	Replacement  string `yaml:"replacement"`
	Action       string `yaml:"action"`
}
