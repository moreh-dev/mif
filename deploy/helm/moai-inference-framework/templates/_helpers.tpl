{{- define "mif.labels" -}}
{{- range $name, $value := .Values.commonLabels -}}
{{ $name }}: {{ tpl $value $ }}
{{ end -}}
helm.sh/chart: {{ include "common.names.chart" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | replace "+" "_" | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}
