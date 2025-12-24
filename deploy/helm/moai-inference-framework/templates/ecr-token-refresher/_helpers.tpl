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

{{- define "mif.ecrTokenRefresher.jobTemplate" -}}
metadata:
  labels:
    {{- include "mif.ecrTokenRefresher.labels" . | nindent 8 }}
spec:
  restartPolicy: OnFailure
  serviceAccountName: {{ include "mif.ecrTokenRefresher.serviceAccountName" . }}
  {{- include "common.images.pullSecrets" (dict "images" (list .Values.ecrTokenRefresher.image) "global" .Values.global) | nindent 6 }}
  containers:
    - name: main
      image: {{ include "common.images.image" (dict "imageRoot" .Values.ecrTokenRefresher.image "global" .Values.global) }}
      imagePullPolicy: {{ .Values.ecrTokenRefresher.image.pullPolicy }}
      command:
        - bash
        - -c
      args:
        - |
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
      env:
        - name: NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: AWS_REGION
          value: {{ .Values.ecrTokenRefresher.aws.region | quote }}
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: {{ include "mif.ecrTokenRefresher.fullname" . }}
              key: AWS_ACCESS_KEY_ID
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: {{ include "mif.ecrTokenRefresher.fullname" . }}
              key: AWS_SECRET_ACCESS_KEY
{{- end }}
