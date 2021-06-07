// Copyright 2019 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package integration

import (
	"testing"

	"github.com/newrelic/nri-prometheus/internal/pkg/labels"
	"github.com/stretchr/testify/assert"
)

func TestLoadSpecFilesOk(t *testing.T) {
	t.Parallel()

	specs, err := LoadSpecFiles("./test/")
	assert.NoError(t, err)
	assert.Contains(t, specs.SpecsByName, "ibmmq")
	assert.Contains(t, specs.SpecsByName, "ravendb")
	assert.Contains(t, specs.SpecsByName, "redis")
}

func TestLoadSpecFilesNoFiles(t *testing.T) {
	t.Parallel()

	specs, err := LoadSpecFiles(".")
	assert.NoError(t, err)
	assert.Len(t, specs.SpecsByName, 0)
}

func TestSpecs_getEntity(t *testing.T) {
	t.Parallel()

	specs, err := LoadSpecFiles("./test/")
	assert.NoError(t, err)

	type fields struct {
		SpecsByName map[string]SpecDef
	}
	type args struct {
		m Metric
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantProps entityNameProps
		wantErr   bool
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
				},
			},
			wantProps: entityNameProps{
				Service: "ravendb", Name: "database", DisplayName: "Database", Type: "RAVENDB_DATABASE",
				Labels: []keyValue{{"database", "test"}},
			},
			wantErr: false,
		},
		{
			name:   "matchEntityWithoutDimensions",
			fields: fields{specs.SpecsByName},
			args: args{
				Metric{
					name:       "ravendb_document_put_bytes_total",
					attributes: labels.Set{},
				},
			},
			wantProps: entityNameProps{Service: "ravendb", Name: "node", DisplayName: "RavenDb Node", Type: "RAVENDB_NODE", Labels: []keyValue{}},
			wantErr:   false,
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
				},
			},
			wantProps: entityNameProps{
				Service: "ravendb", Name: "testentity", DisplayName: "testEntity", Type: "RAVENDB_TESTENTITY",
				Labels: []keyValue{{"dim1", "first"}, {"dim2", "second"}},
			},
			wantErr: false,
		},
		{
			name:   "redisEntityMetric1",
			fields: fields{specs.SpecsByName},
			args: args{
				Metric{
					name:       "redis_commands_duration_seconds_total",
					attributes: labels.Set{},
				},
			},
			wantProps: entityNameProps{
				Service: "redis", Name: "instance", DisplayName: "Redis Instance", Type: "REDIS_INSTANCE",
				Labels: []keyValue{},
			},
			wantErr: false,
		},
		{
			name:   "redisEntityMetric2",
			fields: fields{specs.SpecsByName},
			args: args{
				Metric{
					name:       "redis_repl_backlog_is_active",
					attributes: labels.Set{},
				},
			},
			wantProps: entityNameProps{
				Service: "redis", Name: "instance", DisplayName: "Redis Instance", Type: "REDIS_INSTANCE",
				Labels: []keyValue{},
			},
			wantErr: false,
		},
		{
			name:   "redisBisMetric",
			fields: fields{specs.SpecsByName},
			args: args{
				Metric{
					name:       "redis_metric_bis",
					attributes: labels.Set{},
				},
			},
			wantProps: entityNameProps{
				Service: "redis", Name: "testbisredis", DisplayName: "Redis test bis", Type: "REDIS_TESTBISREDIS",
				Labels: []keyValue{},
			},
			wantErr: false,
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
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &Specs{
				SpecsByName: tt.fields.SpecsByName,
			}
			props, err := s.getEntity(tt.args.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("Specs.getEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.EqualValues(t, tt.wantProps, props)
		})
	}
}
