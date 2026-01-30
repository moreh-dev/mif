{{- define "mif.labels" -}}
{{ include "common.labels.standard" (dict "customLabels" .Values.commonLabels "context" $) -}}
{{- end -}}
