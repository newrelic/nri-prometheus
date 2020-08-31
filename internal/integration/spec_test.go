// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"testing"

	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	"github.com/stretchr/testify/assert"
)

func TestLoadSpecFilesOk(t *testing.T) {
	specs, err := LoadSpecFiles("./test/")
	assert.NoError(t, err)
	assert.Contains(t, specs.SpecsByName, "ibmmq")
	assert.Contains(t, specs.SpecsByName, "ravendb")
}
func TestLoadSpecFilesNoFiles(t *testing.T) {
	specs, err := LoadSpecFiles(".")
	assert.NoError(t, err)
	assert.Len(t, specs.SpecsByName, 0)
}

func TestSpecs_getEntity(t *testing.T) {

	specs, err := LoadSpecFiles("./test/")
	assert.NoError(t, err)
	assert.Contains(t, specs.SpecsByName, "ravendb")

	type fields struct {
		SpecsByName map[string]SpecDef
	}
	type args struct {
		m Metric
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantEntityName string
		wantEntityType string
		wantErr        bool
	}{
		{
			name:   "matchEntity",
			fields: fields{specs.SpecsByName},
			args: args{
				Metric{
					name: "ravendb_database_document_put_bytes_total",
					attributes: labels.Set{
						"database": "test",
					},
				}},
			wantEntityName: "database:test",
			wantEntityType: "PrometheusRavendbDatabase",
			wantErr:        false,
		},
		{
			name:   "matchEntityWithoutDimensions",
			fields: fields{specs.SpecsByName},
			args: args{
				Metric{
					name:       "ravendb_document_put_bytes_total",
					attributes: labels.Set{},
				}},
			wantEntityName: "node",
			wantEntityType: "PrometheusRavendbNode",
			wantErr:        false,
		},
		{
			name:   "matchEntityConcatenatedName",
			fields: fields{specs.SpecsByName},
			args: args{
				Metric{
					name: "ravendb_testentity_document_put_bytes_total",
					attributes: labels.Set{
						"dim1": "first",
						"dim2": "second",
					},
				}},
			wantEntityName: "testentity:first:second",
			wantEntityType: "PrometheusRavendbTestentity",
			wantErr:        false,
		},
		{
			name:    "missingDimentions",
			fields:  fields{specs.SpecsByName},
			args:    args{Metric{name: "ravendb_database_document_put_bytes_total"}},
			wantErr: true,
		},
		{
			name:    "serviceNotDefined",
			fields:  fields{specs.SpecsByName},
			args:    args{Metric{name: "service_metric_undefined"}},
			wantErr: true,
		},
		{
			name:    "metricNotDefined",
			fields:  fields{specs.SpecsByName},
			args:    args{Metric{name: "ravendb_metric_undefined"}},
			wantErr: true,
		},
		{
			name:    "shortMetricName",
			fields:  fields{specs.SpecsByName},
			args:    args{Metric{name: "shortname"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Specs{
				SpecsByName: tt.fields.SpecsByName,
			}
			gotEntityName, gotEntityType, err := s.getEntity(tt.args.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("Specs.getEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotEntityName != tt.wantEntityName {
				t.Errorf("Specs.getEntity() gotEntityName = %v, want %v", gotEntityName, tt.wantEntityName)
			}
			if gotEntityType != tt.wantEntityType {
				t.Errorf("Specs.getEntity() gotEntityType = %v, want %v", gotEntityType, tt.wantEntityType)
			}
		})
	}
}
