//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/moreh-dev/mif/test/utils"
)

type inferenceServiceTemplateData struct {
	Name            string
	Namespace       string
	ImagePullSecret string
	Image           string
	Model           string
	HFToken         string
	HFEndpoint      string
	IsKind          bool
}

func createPrefillInferenceServiceManifest() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	imageRepo, imageTag := getInferenceImageInfo()
	image := fmt.Sprintf("%s:%s", imageRepo, imageTag)

	const prefillTemplate = `apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: {{ .Name }}-prefill
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  inferencePoolRefs:
    - name: heimdall
  templateRefs:
    - name: workertemplate-vllm-common
    - name: workertemplate-pd-prefill-meta
  parallelism:
    tensor: 4
    data: 1
  workerTemplate:
    spec:
      imagePullSecrets:
        - name: {{ .ImagePullSecret }}
      containers:
        - name: main
          image: {{ .Image }}
{{- if not .IsKind }}
          securityContext:
            capabilities:
              add:
              - IPC_LOCK
{{- end }}
          command:
{{- if .IsKind }}
            - /app/llm-d-inference-sim
{{- else }}
            - vllm
            - serve
{{- end }}
          args:
{{- if .IsKind }}
            - --model
            - {{ .Model }}
            - --port
            - "8000"
{{- else }}
            - {{ .Model }}
            - --port
            - "8000"
            - --quantization
            - "None"
            - --tensor-parallel-size
            - "4"
            - --max-num-batched-tokens
            - "8192"
            - --no-enable-prefix-caching
            - --no-enable-log-requests
            - --disable-uvicorn-access-log
            - --trust-remote-code
            - --kv-transfer-config
            - {"kv_connector":"NixlConnector","kv_role":"kv_both"}
{{- end }}
{{- if and (not .IsKind) (or .HFToken .HFEndpoint) }}
          env:
{{- if .HFToken }}
            - name: HF_TOKEN
              value: "{{ .HFToken }}"
{{- end }}
{{- if .HFEndpoint }}
            - name: HF_ENDPOINT
              value: "{{ .HFEndpoint }}"
{{- end }}
            - name: ISVC_MODEL_NAME
              value: "{{ .Model }}"
{{- end }}
{{- if not .IsKind }}
          resources:
            requests:
              amd.com/gpu: "4"
            limits:
              amd.com/gpu: "4"
{{- end }}
          ports:
            - containerPort: 8000
              name: http
              protocol: TCP
          readinessProbe:
            httpGet:
              path: /health
              port: 8000
              scheme: HTTP
            initialDelaySeconds: {{ if .IsKind }}10{{ else }}120{{ end }}
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 5
          volumeMounts:
            - mountPath: /dev/shm
              name: shm
      volumes:
        - name: shm
          emptyDir:
            medium: Memory
            sizeLimit: "16Gi"
{{- if not .IsKind }}
      tolerations:
        - key: "amd.com/gpu"
          operator: "Exists"
          effect: "NoSchedule"
{{- end }}
`

	data := inferenceServiceTemplateData{
		Name:            inferenceServiceName,
		Namespace:       cfg.workloadNamespace,
		ImagePullSecret: secretNameMorehRegistry,
		Image:           image,
		Model:           cfg.testModel,
		HFToken:         cfg.hfToken,
		HFEndpoint:      cfg.hfEndpoint,
		IsKind:          cfg.isUsingKindCluster,
	}

	rendered, err := renderTextTemplate(prefillTemplate, data)
	if err != nil {
		return "", fmt.Errorf("failed to render prefill InferenceService manifest: %w", err)
	}

	manifestPath := filepath.Join(projectDir, tempFileInferenceServicePrefill)
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write prefill InferenceService manifest file: %w", err)
	}

	return manifestPath, nil
}

func createDecodeInferenceServiceManifest() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	imageRepo, imageTag := getInferenceImageInfo()
	image := fmt.Sprintf("%s:%s", imageRepo, imageTag)

	const decodeTemplate = `apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: {{ .Name }}-decode
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  inferencePoolRefs:
    - name: heimdall
  templateRefs:
    - name: workertemplate-vllm-common
    - name: workertemplate-pd-decode-meta
    - name: workertemplate-decode-proxy
  parallelism:
    tensor: 4
    data: 1
  workerTemplate:
    spec:
      imagePullSecrets:
        - name: {{ .ImagePullSecret }}
      containers:
        - name: main
          image: {{ .Image }}
{{- if not .IsKind }}
          securityContext:
            capabilities:
              add:
              - IPC_LOCK
{{- end }}
          command:
{{- if .IsKind }}
            - /app/llm-d-inference-sim
{{- else }}
            - vllm
            - serve
{{- end }}
          args:
{{- if .IsKind }}
            - --model
            - {{ .Model }}
            - --port
            - "8200"
{{- else }}
            - {{ .Model }}
            - --port
            - "8200"
            - --quantization
            - "None"
            - --tensor-parallel-size
            - "4"
            - --max-num-batched-tokens
            - "8192"
            - --no-enable-prefix-caching
            - --no-enable-log-requests
            - --disable-uvicorn-access-log
            - --trust-remote-code
            - --kv-transfer-config
            - {"kv_connector":"NixlConnector","kv_role":"kv_both"}
{{- end }}
{{- if and (not .IsKind) (or .HFToken .HFEndpoint) }}
          env:
{{- if .HFToken }}
            - name: HF_TOKEN
              value: "{{ .HFToken }}"
{{- end }}
{{- if .HFEndpoint }}
            - name: HF_ENDPOINT
              value: "{{ .HFEndpoint }}"
{{- end }}
            - name: ISVC_MODEL_NAME
              value: "{{ .Model }}"
            - name: ISVC_PORT
              value: "8200"
{{- end }}
{{- if not .IsKind }}
          resources:
            requests:
              amd.com/gpu: "4"
            limits:
              amd.com/gpu: "4"
{{- end }}
          ports:
            - containerPort: 8200
              name: http-decode
              protocol: TCP
          readinessProbe:
            httpGet:
              path: /health
              port: 8200
              scheme: HTTP
            initialDelaySeconds: {{ if .IsKind }}10{{ else }}120{{ end }}
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 5
          volumeMounts:
            - mountPath: /dev/shm
              name: shm
      volumes:
        - name: shm
          emptyDir:
            medium: Memory
            sizeLimit: "16Gi"
{{- if not .IsKind }}
      tolerations:
        - key: "amd.com/gpu"
          operator: "Exists"
          effect: "NoSchedule"
{{- end }}
`

	data := inferenceServiceTemplateData{
		Name:            inferenceServiceName,
		Namespace:       cfg.workloadNamespace,
		ImagePullSecret: secretNameMorehRegistry,
		Image:           image,
		Model:           cfg.testModel,
		HFToken:         cfg.hfToken,
		HFEndpoint:      cfg.hfEndpoint,
		IsKind:          cfg.isUsingKindCluster,
	}

	rendered, err := renderTextTemplate(decodeTemplate, data)
	if err != nil {
		return "", fmt.Errorf("failed to render decode InferenceService manifest: %w", err)
	}

	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceDecode)
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write decode InferenceService manifest file: %w", err)
	}

	return manifestPath, nil
}

func installPrefillDecodeInferenceServicesForTest() {
	By("creating common InferenceServiceTemplate")
	commonTemplatePath, err := createCommonTemplate()
	Expect(err).NotTo(HaveOccurred(), "Failed to create common template")

	By("creating prefill meta InferenceServiceTemplate")
	prefillMetaTemplatePath, err := createPrefillMetaTemplate()
	Expect(err).NotTo(HaveOccurred(), "Failed to create prefill meta template")

	By("creating decode meta InferenceServiceTemplate")
	decodeMetaTemplatePath, err := createDecodeMetaTemplate()
	Expect(err).NotTo(HaveOccurred(), "Failed to create decode meta template")

	By("creating decode proxy InferenceServiceTemplate")
	decodeProxyTemplatePath, err := createDecodeProxyTemplate()
	Expect(err).NotTo(HaveOccurred(), "Failed to create decode proxy template")

	By("applying InferenceServiceTemplates")
	Expect(utils.CreateInferenceServiceTemplate(cfg.workloadNamespace, commonTemplatePath)).To(Succeed(), "Failed to apply common template")
	Expect(utils.CreateInferenceServiceTemplate(cfg.workloadNamespace, prefillMetaTemplatePath)).To(Succeed(), "Failed to apply prefill meta template")
	Expect(utils.CreateInferenceServiceTemplate(cfg.workloadNamespace, decodeMetaTemplatePath)).To(Succeed(), "Failed to apply decode meta template")
	Expect(utils.CreateInferenceServiceTemplate(cfg.workloadNamespace, decodeProxyTemplatePath)).To(Succeed(), "Failed to apply decode proxy template")

	By("creating prefill InferenceService manifest file")
	prefillManifestPath, err := createPrefillInferenceServiceManifest()
	Expect(err).NotTo(HaveOccurred(), "Failed to create prefill InferenceService manifest file")

	By("creating decode InferenceService manifest file")
	decodeManifestPath, err := createDecodeInferenceServiceManifest()
	Expect(err).NotTo(HaveOccurred(), "Failed to create decode InferenceService manifest file")

	By("creating prefill InferenceService")
	Expect(utils.CreateInferenceService(cfg.workloadNamespace, prefillManifestPath)).To(Succeed(), "Failed to create prefill InferenceService")

	By("creating decode InferenceService")
	Expect(utils.CreateInferenceService(cfg.workloadNamespace, decodeManifestPath)).To(Succeed(), "Failed to create decode InferenceService")

	By("waiting for prefill InferenceService pods to be ready")
	cmd := exec.Command("kubectl", "wait", "pod",
		"-l", fmt.Sprintf("app.kubernetes.io/name=%s-prefill", inferenceServiceName),
		"--for=condition=Ready",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", timeoutVeryLong))
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Prefill InferenceService pods not ready")

	By("waiting for decode InferenceService pods to be ready")
	cmd = exec.Command("kubectl", "wait", "pod",
		"-l", fmt.Sprintf("app.kubernetes.io/name=%s-decode", inferenceServiceName),
		"--for=condition=Ready",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", timeoutVeryLong))
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Decode InferenceService pods not ready")
}

func verifyInferenceEndpoint() {
	By("verifying Gateway service exists")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "service",
			"-n", cfg.workloadNamespace,
			"-l", "gateway.networking.k8s.io/gateway-name=mif",
			"-o", "jsonpath={.items[0].metadata.name}")
		output, err := utils.Run(cmd)
		if err != nil || strings.TrimSpace(output) == "" {
			cmd = exec.Command("kubectl", "get", "service",
				"mif",
				"-n", cfg.workloadNamespace,
				"-o", "jsonpath={.metadata.name}")
			output, err = utils.Run(cmd)
		}
		if err != nil || strings.TrimSpace(output) == "" {
			cmd = exec.Command("kubectl", "get", "service",
				"gateway-mif",
				"-n", cfg.workloadNamespace,
				"-o", "jsonpath={.metadata.name}")
			output, err = utils.Run(cmd)
		}
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(strings.TrimSpace(output)).NotTo(BeEmpty(), "Gateway service not found")
	}, timeoutLong, intervalLong).Should(Succeed())

	By("waiting for inference-service decode pods to be ready")
	cmd := exec.Command("kubectl", "wait", "pod",
		"-l", fmt.Sprintf("app.kubernetes.io/name=%s-decode", inferenceServiceName),
		"--for=condition=Ready",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", timeoutVeryLong))
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "InferenceService decode pods not ready")
}
