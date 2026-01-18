{{- define "mif.ecrTokenRefresher.fullname" -}}
{{ include "common.names.fullname" . }}-ecr-token-refresher
{{- end }}

{{- define "mif.ecrTokenRefresher.matchLabels" -}}
app.kubernetes.io/name: {{ include "common.names.name" . }}-ecr-token-refresher
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "mif.ecrTokenRefresher.labels" -}}
{{- include "mif.labels" . }}
{{ include "mif.ecrTokenRefresher.matchLabels" . }}
{{- end }}

{{- define "mif.ecrTokenRefresher.serviceAccountName" -}}
{{- if .Values.ecrTokenRefresher.serviceAccount.create }}
{{- default (include "mif.ecrTokenRefresher.fullname" .) .Values.ecrTokenRefresher.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.ecrTokenRefresher.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "mif.ecrTokenRefresher.script" -}}
kubectl create secret -n ${NAMESPACE} docker-registry {{ .Values.ecrTokenRefresher.pullSecret.name }} \
  --docker-server={{ .Values.ecrTokenRefresher.pullSecret.server }} \
  --docker-username={{ .Values.ecrTokenRefresher.pullSecret.username }} \
  --docker-password=$(aws ecr get-login-password --region ${AWS_REGION}) \
  --dry-run=client -o yaml | \
  kubectl apply -f -

{{- range $key, $value := .Values.ecrTokenRefresher.pullSecret.annotations }}
kubectl annotate secret -n ${NAMESPACE} {{ $.Values.ecrTokenRefresher.pullSecret.name }} \
  "{{ $key }}={{ $value }}" \
  --overwrite
{{- end }}

echo "ECR token refreshed at $(date)"
{{- end }}
