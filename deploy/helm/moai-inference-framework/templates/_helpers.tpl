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

{{/*
Resolve the MinIO secret key (password) for a bucket consumer, in priority order:
  1. an explicit override (<consumer>Bucket.secretKey), when set;
  2. the value already stored in the <name>-bucket Secret, so it is preserved
     across upgrades (read via lookup);
  3. a freshly generated random string on first install.
This avoids shipping a well-known static password by default while keeping the
credential stable across upgrades.
Usage: include "mif.bucketSecretKey" (dict "ctx" $ "secretName" "loki-bucket" "override" .Values.lokiBucket.secretKey)
*/}}
{{- define "mif.bucketSecretKey" -}}
{{- if .override -}}
{{- .override -}}
{{- else -}}
{{- $existing := lookup "v1" "Secret" (include "common.names.namespace" .ctx) .secretName -}}
{{- if and $existing $existing.data (index $existing.data "AWS_SECRET_ACCESS_KEY") -}}
{{- index $existing.data "AWS_SECRET_ACCESS_KEY" | b64dec -}}
{{- else -}}
{{- randAlphaNum 24 -}}
{{- end -}}
{{- end -}}
{{- end -}}
