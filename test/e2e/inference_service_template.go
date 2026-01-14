//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moreh-dev/mif/test/utils"
)

func createCommonTemplate() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	imageRepo, imageTag := getInferenceImageInfo()
	image := fmt.Sprintf("%s:%s", imageRepo, imageTag)

	const templateContent = `apiVersion: odin.moreh.io/v1alpha1
kind: InferenceServiceTemplate
metadata:
  name: workertemplate-vllm-common
  namespace: %s
spec:
  workerTemplate:
    spec:
      imagePullSecrets:
        - name: %s
      containers:
        - name: main
          image: %s
          securityContext:
            capabilities:
              add:
                - IPC_LOCK
          command:
            - /bin/bash
            - -c
          args:
            - |
              set -ex
              
              vllm serve ${ISVC_MODEL_NAME} \
                --port ${ISVC_PORT} \
                --served-model-name ${ISVC_MODEL_NAME} \
                --tensor-parallel-size 4 \
                ${ISVC_EXTRA_ARGS}
          env:
            - name: ISVC_EXTRA_ARGS
              value: >-
                --trust-remote-code
                --no-enable-log-requests
                --disable-uvicorn-access-log
                --quantization None
                --kv-transfer-config {"kv_connector":"NixlConnector","kv_role":"kv_both"}
                --no-enable-prefix-caching
                --max-num-batched-tokens 8192
            - name: VLLM_NIXL_SIDE_CHANNEL_HOST
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: UCX_TLS
              value: rocm_copy,rocm_ipc,self,sm,rc_x
            - name: ISVC_PORT
              value: "8000"
            - name: ISVC_MODEL_NAME
              value: "%s"
          ports:
            - name: http
              containerPort: 8000
            - name: http-decode
              containerPort: 8200
          volumeMounts:
            - name: shm
              mountPath: /dev/shm
      volumes:
        - name: shm
          emptyDir:
            medium: Memory
            sizeLimit: 16Gi
`

	rendered := fmt.Sprintf(templateContent, cfg.workloadNamespace, secretNameMorehRegistry, image, cfg.testModel)
	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceTemplateCommon)
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write common template file: %w", err)
	}

	return manifestPath, nil
}

func createPrefillMetaTemplate() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	const templateContent = `apiVersion: odin.moreh.io/v1alpha1
kind: InferenceServiceTemplate
metadata:
  name: workertemplate-pd-prefill-meta
  namespace: %s
spec:
  workerTemplate:
    metadata:
      labels:
        heimdall.moreh.io/role: prefill
`

	rendered := fmt.Sprintf(templateContent, cfg.workloadNamespace)
	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceTemplatePrefillMeta)
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write prefill meta template file: %w", err)
	}

	return manifestPath, nil
}

func createDecodeMetaTemplate() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	const templateContent = `apiVersion: odin.moreh.io/v1alpha1
kind: InferenceServiceTemplate
metadata:
  name: workertemplate-pd-decode-meta
  namespace: %s
spec:
  workerTemplate:
    metadata:
      labels:
        heimdall.moreh.io/role: decode
`

	rendered := fmt.Sprintf(templateContent, cfg.workloadNamespace)
	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceTemplateDecodeMeta)
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write decode meta template file: %w", err)
	}

	return manifestPath, nil
}

func createDecodeProxyTemplate() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	const templateContent = `apiVersion: odin.moreh.io/v1alpha1
kind: InferenceServiceTemplate
metadata:
  name: workertemplate-decode-proxy
  namespace: %s
spec:
  workerTemplate:
    spec:
      imagePullSecrets:
        - name: %s
      initContainers:
        - name: proxy
          image: 255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/heimdall-proxy:v0.6.0
          restartPolicy: Always
          command:
            - /bin/bash
            - -c
          args:
            - |
              set -ex

              exec /app/proxy \
                --port $(ISVC_PORT) \
                --decoder-ip $(POD_IP) \
                --decoder-port 8200 \
                $(ISVC_EXTRA_ARGS)
          env:
            - name: ISVC_PORT
              value: "8000"
            - name: ISVC_EXTRA_ARGS
              value: >-
                --pd-coordinator vllm/nixl
                --log-format json
                --log-level warn
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
          readinessProbe:
            httpGet:
              path: /health-proxy
              port: 8000
              scheme: HTTP
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 5
            successThreshold: 1
            failureThreshold: 3
      containers:
        - name: main
          env:
            - name: ISVC_PORT
              value: "8200"
            - name: ISVC_KV_PORT_OFFSET
              value: "200"
`

	rendered := fmt.Sprintf(templateContent, cfg.workloadNamespace, secretNameMorehRegistry)
	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceTemplateDecodeProxy)
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write decode proxy template file: %w", err)
	}

	return manifestPath, nil
}
