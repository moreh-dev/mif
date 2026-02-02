{{- define "mif.runtimeBase.labels" -}}
{{ include "mif.labels" . }}
mif.moreh.io/template.type: runtime-base
{{- end -}}
