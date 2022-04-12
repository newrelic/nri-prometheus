{{/* vim: set filetype=mustache: */}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "nri-prometheus.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "nri-prometheus.labels" -}}
app.kubernetes.io/name: {{ include "common.naming.name" . }}
helm.sh/chart: {{ include "nri-prometheus.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Returns mergeTransformations
Helm can't merge maps of different types. Need to manually create a `transformations` section.
*/}}
{{- define "nri-prometheus.mergeTransformations" -}}
    {{/* Remove current `transformations` from config. */}}
    {{- omit .Values.config "transformations" | toYaml | nindent 4 -}}
    {{/* Create new `transformations` yaml section with merged configs from .Values.config.transformations and lowDataMode. */}}
    transformations:
    {{- .Values.config.transformations | toYaml | nindent 4 -}}
    {{ $lowDataDefault := .Files.Get "static/lowdatamodedefaults.yaml" | fromYaml }}
    {{- $lowDataDefault.transformations | toYaml | nindent 4 -}}
{{- end -}}

{{/*
Returns nrStaging
*/}}
{{- define "newrelic.nrStaging" -}}
{{- if .Values.global }}
  {{- if .Values.global.nrStaging }}
    {{- .Values.global.nrStaging -}}
  {{- end -}}
{{- else if .Values.nrStaging }}
  {{- .Values.nrStaging -}}
{{- end -}}
{{- end -}}
