{{/* vim: set filetype=mustache: */}}

{{/*
Returns mergeTransformations
Manually concatenates custom transformations with lowDataMode defaults.
This ensures both arrays are properly merged in the rendered YAML output.
*/}}
{{- define "nri-prometheus.mergeTransformations" -}}
    {{/* Remove current `transformations` from config. */}}
    {{- omit .Values.config "transformations" | toYaml | nindent 4 -}}
    {{/* Create new `transformations` yaml section with merged configs from .Values.config.transformations and lowDataMode. */}}
    {{/* Properly merge the two transformation arrays */}}
    {{ $lowDataDefault := .Files.Get "static/lowdatamodedefaults.yaml" | fromYaml }}
    {{ $merged := concat .Values.config.transformations $lowDataDefault.transformations }}
    transformations:
    {{- $merged | toYaml | nindent 4 -}}
{{- end -}}
