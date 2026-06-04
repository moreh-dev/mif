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

{{/*
Fully-qualified name of a bundled subchart release, matching the service names
those subcharts (minio, loki, and the tempo-distributed dependency aliased to
"tempo") generate for themselves. Delegates to the bundled common library's
common.names.dependency.fullname, which applies the same fullname collapse:
when the release name already contains the subchart name the duplicate suffix
is dropped, otherwise the name is appended. This avoids hardcoding
"<release>-<name>-..." service references, which break for release names that
contain the subchart name.

"name" is the subchart's fullname default, i.e. its dependency alias -- "tempo"
(not "tempo-distributed"), "minio", or "loki".

chartValues is intentionally empty, so the result derives from .Release.Name
alone and is identical in every render context -- including values that a
subchart tpl-renders in its own context (e.g. vector.customConfig), where the
parent's .Values.<subchart> is not visible. As a consequence it does not honor
<subchart>.nameOverride / fullnameOverride (not set by this chart).

Usage: include "mif.subchartFullname" (dict "name" "loki" "ctx" .)
*/}}
{{- define "mif.subchartFullname" -}}
{{- include "common.names.dependency.fullname" (dict "chartName" .name "chartValues" (dict) "context" .ctx) -}}
{{- end -}}
