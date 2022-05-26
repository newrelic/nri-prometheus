package main

import (
	"log"
	"os"
	"reflect"
	"testing"
)

func ExampleMain() {
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		log.Fatal(err)
	}

	origStdin := os.Stdin
	os.Stdin = stdinReader

	defer func() {
		os.Stdin = origStdin
	}()

	stdinText := `
	{
		"status": "success",
	    "data": {
	        "count_metric_without_suffix": [
	            {
	                "type": "counter"
	            }
	        ]
		}
	}
	`
	_, err = stdinWriter.Write([]byte(stdinText))

	main()
	// Output:
	// write_relabel_configs:
	//     - source_labels: '[__name__]'
	//       regex: ^count_metric_without_suffix$
	//       target_label: newrelic_metric_type
	//       replacement: counter
	//       action: replace
}

func Test_MetricTypeRelabelConfigs(t *testing.T) {
	type args struct {
		metadata Metadata
	}
	tests := []struct {
		name    string
		args    args
		want    []RelabelConfig
		wantErr bool
	}{
		{
			name: "count metric without prefix",
			args: args{
				Metadata{
					Status: "success",
					Data: map[string][]MetricMetadata{
						"count_metric": {
							{Type: "counter"},
						},
					},
				},
			},
			want: []RelabelConfig{
				{
					SourceLabels: "[__name__]",
					Regex:        "^count_metric$",
					TargetLabel:  "newrelic_metric_type",
					Replacement:  "counter",
					Action:       "replace",
				},
			},
		},
		{
			name: "gauge metrics with count sufix",
			args: args{
				Metadata{
					Status: "success",
					Data: map[string][]MetricMetadata{
						"gauge_metric_total": {
							{Type: "gauge"},
						},
					},
				},
			},
			want: []RelabelConfig{
				{
					SourceLabels: "[__name__]",
					Regex:        "^gauge_metric_total$",
					TargetLabel:  "newrelic_metric_type",
					Replacement:  "gauge",
					Action:       "replace",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.args.metadata.MetricTypeRelabelConfigs()
			if (err != nil) != tt.wantErr {
				t.Errorf("MetricTypeRelabelConfigs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.WriteRelabelConfigs, tt.want) {
				t.Errorf("MetricTypeRelabelConfigs() = %v, want %v", got, tt.want)
			}
		})
	}
}
