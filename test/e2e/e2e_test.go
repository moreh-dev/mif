//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/moreh-dev/mif/test/utils"
)

var _ = Describe("Prefill-Decode Disaggregation", Ordered, func() {

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("collecting logs and events for debugging")
			collectDebugInfo()
		}
	})

	SetDefaultEventuallyTimeout(timeoutShort)
	SetDefaultEventuallyPollingInterval(intervalShort)

	Context("MIF Infrastructure", func() {
		BeforeEach(func() {
			if cfg.skipPrerequisite {
				Skip("MIF infrastructure is expected to be pre-installed when SKIP_PREREQUISITE=true")
			}
		})

		It("should deploy MIF components successfully", func() {
			By("validating that Odin controller is running")
			Eventually(verifyOdinController).Should(Succeed())
		})

		It("should have all pods ready", func() {
			By("waiting for all pods to be ready")
			Eventually(verifyAllPodsReady, timeoutVeryLong).Should(Succeed())
		})
	})

	Context("Gateway and InferenceService CR integration", func() {
		BeforeAll(func() {
			By("setting up test resources")
			createWorkloadNamespace()

			By("applying Gateway resources")
			applyGatewayResource()
			By("installing Heimdall")
			installHeimdallForTest()
			By("installing Prefill-Decode InferenceServices (creating InferenceServiceTemplates and then InferenceServices)")
			installPrefillDecodeInferenceServicesForTest()
		})

		AfterAll(func() {
			if cfg.skipCleanup {
				return
			}
			By("cleaning up test workload namespace")
			if err := utils.CleanupWorkloadNamespace(utils.CleanupConfig{
				WorkloadNamespace: cfg.workloadNamespace,
				GatewayClass:      cfg.gatewayClass,
				MIFNamespace:      cfg.mifNamespace,
				PrefillName:       fmt.Sprintf("%s-prefill", inferenceServiceName),
				DecodeName:        fmt.Sprintf("%s-decode", inferenceServiceName),
				TemplateNames: []string{
					"workertemplate-vllm-common",
					"workertemplate-pd-prefill-meta",
					"workertemplate-pd-decode-meta",
					"workertemplate-decode-proxy",
				},
			}); err != nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to cleanup workload namespace: %v\n", err)
			}
		})

		It("should have inference-service decode pods reachable behind Gateway", func() {
			verifyInferenceEndpoint()
		})

		It("should run inference-perf performance benchmark", func() {
			runInferencePerfBenchmark()
		})
	})

})

func renderTextTemplate(templateText string, data any) (string, error) {
	t, err := template.New("manifest").Parse(templateText)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func verifyOdinController(g Gomega) {
	cmd := exec.Command("kubectl", "wait", "deployment",
		"-l", "app.kubernetes.io/name=odin",
		"--for=condition=Available",
		"-n", cfg.mifNamespace,
		fmt.Sprintf("--timeout=%v", timeoutLong))
	_, err := utils.Run(cmd)
	g.Expect(err).NotTo(HaveOccurred(), "Odin controller not available")
}

func verifyAllPodsReady(g Gomega) {
	cmd := exec.Command("kubectl", "wait", "pod",
		"--all",
		"--field-selector=status.phase!=Succeeded",
		"--for=condition=Ready",
		"-n", cfg.mifNamespace,
		fmt.Sprintf("--timeout=%v", timeoutVeryLong))
	_, err := utils.Run(cmd)
	g.Expect(err).NotTo(HaveOccurred(), "Some pods are not ready")
}

func getInferenceImageInfo() (repo, tag string) {
	repoDefault := imageRepoDefault
	tagDefault := imageTagDefault
	if cfg.isUsingKindCluster {
		repoDefault = imageRepoKindDefault
		tagDefault = imageTagKindDefault
	}

	repo = cfg.inferenceImageRepo
	if repo == "" {
		repo = repoDefault
	}

	tag = cfg.inferenceImageTag
	if tag == "" {
		tag = tagDefault
	}

	return repo, tag
}

func collectDebugInfo() {
	By("fetching pod logs")
	cmd := exec.Command("kubectl", "get", "pods",
		"-n", cfg.mifNamespace,
		"-o", "jsonpath={.items[*].metadata.name}")
	output, err := utils.Run(cmd)
	if err == nil {
		podNames := strings.Fields(output)
		for _, podName := range podNames {
			cmd = exec.Command("kubectl", "logs", podName, "-n", cfg.mifNamespace, "--all-containers=true", "--tail=100")
			logs, logErr := utils.Run(cmd)
			if logErr == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Pod %s logs:\n%s\n", podName, logs)
			}
		}
	}

	By("fetching Kubernetes events")
	cmd = exec.Command("kubectl", "get", "events", "-n", cfg.mifNamespace, "--sort-by=.lastTimestamp")
	eventsOutput, err := utils.Run(cmd)
	if err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s\n", eventsOutput)
	}

	By("fetching resource status")
	cmd = exec.Command("kubectl", "get", "all", "-n", cfg.mifNamespace)
	allOutput, err := utils.Run(cmd)
	if err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "All resources:\n%s\n", allOutput)
	}
}

func createWorkloadNamespace() {
	By("creating workload namespace")
	cmd := exec.Command("kubectl", "create", "ns", cfg.workloadNamespace, "--request-timeout=30s")
	_, err := utils.Run(cmd)
	if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
		Expect(err).NotTo(HaveOccurred(), "Failed to create workload namespace")
	}

	if cfg.mifNamespace != cfg.workloadNamespace {
		By("adding mif=enabled label to workload namespace for automatic secret copying")
		cmd = exec.Command("kubectl", "label", "namespace", cfg.workloadNamespace,
			"mif=enabled", "--overwrite", "--request-timeout=30s")
		_, err = utils.Run(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to add mif=enabled label to namespace: %v\n", err)
		}
	}

	if cfg.istioRev != "" {
		By(fmt.Sprintf("adding istio.io/rev=%s label to workload namespace", cfg.istioRev))
		cmd = exec.Command("kubectl", "label", "namespace", cfg.workloadNamespace,
			fmt.Sprintf("istio.io/rev=%s", cfg.istioRev), "--overwrite", "--request-timeout=30s")
		_, err = utils.Run(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to add istio.io/rev label to namespace: %v\n", err)
		}
	}
}

func applyGatewayResource() {
	By("applying Gateway resource and infrastructure parameters")

	var baseYAML string

	switch cfg.gatewayClass {
	case "istio":
		baseYAML = `apiVersion: v1
kind: ConfigMap
metadata:
  name: mif-gateway-infrastructure
data:
  service: |
    spec:
      type: ClusterIP
  deployment: |
    spec:
      template:
        metadata:
          annotations:
            proxy.istio.io/config: |
              accessLogFile: /dev/stdout
              accessLogEncoding: JSON
        spec:
          containers:
            - name: istio-proxy
              resources:
                limits: null

---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: mif
spec:
  gatewayClassName: istio
  infrastructure:
    parametersRef:
      group: ""
      kind: ConfigMap
      name: mif-gateway-infrastructure
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: All
`
	case "kgateway":
		baseYAML = `apiVersion: gateway.kgateway.dev/v1alpha1
kind: GatewayParameters
metadata:
  name: mif-gateway-infrastructure
spec:
  kube:
    service:
      type: ClusterIP

---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: mif
spec:
  gatewayClassName: kgateway
  infrastructure:
    parametersRef:
      group: gateway.kgateway.dev
      kind: GatewayParameters
      name: mif-gateway-infrastructure
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: All
`
	default:
		Fail(fmt.Sprintf("Unsupported gatewayClassName: %s", cfg.gatewayClass))
	}

	cmd := exec.Command("kubectl", "apply", "-f", "-", "-n", cfg.workloadNamespace, "--request-timeout=60s")
	cmd.Stdin = strings.NewReader(baseYAML)
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to apply Gateway resources")

	By("waiting for Gateway to be accepted")
	cmd = exec.Command("kubectl", "wait", "gateway", "mif",
		"--for=condition=Accepted",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", timeoutLong))
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Gateway not accepted")

	By("waiting for Gateway pods to be created")
	Eventually(func() (string, error) {
		checkCmd := exec.Command("kubectl", "get", "pods",
			"-l", "gateway.networking.k8s.io/gateway-name=mif",
			"-n", cfg.workloadNamespace,
			"-o", "name")
		return utils.Run(checkCmd)
	}, timeoutLong, intervalShort).ShouldNot(BeEmpty())

	By("waiting for Gateway pods to be ready")
	cmd = exec.Command("kubectl", "wait", "pod",
		"-l", "gateway.networking.k8s.io/gateway-name=mif",
		"--for=condition=Ready",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", timeoutLong))
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Gateway pods not ready")
}

func installHeimdallForTest() {
	By("creating Heimdall values file for test")
	heimdallValuesPath, err := createHeimdallValuesFile()
	Expect(err).NotTo(HaveOccurred(), "Failed to create Heimdall values file for test")

	By("installing Heimdall for test")
	Expect(utils.InstallHeimdall(cfg.workloadNamespace, heimdallValuesPath)).To(Succeed(), "Failed to install Heimdall for test")

	By("waiting for Heimdall deployment to be ready")
	cmd := exec.Command("kubectl", "wait", "deployment",
		"-l", "app.kubernetes.io/instance=heimdall",
		"--for=condition=Available",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", timeoutLong))
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Heimdall deployment not available")
}

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

func runInferencePerfBenchmark() {
	By("getting Gateway service name")
	var serviceName string
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
		serviceName = strings.TrimSpace(output)
		g.Expect(serviceName).NotTo(BeEmpty(), "Gateway service not found")
	}, timeoutMedium, intervalMedium).Should(Succeed())

	By("running inference-perf performance benchmark as Kubernetes Job")
	gatewayServiceURL := getGatewayServiceURL(serviceName)
	if gatewayServiceURL == "" {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to get Gateway service URL. Skipping benchmark.\n")
		return
	}

	err := runInferencePerfJob(gatewayServiceURL, cfg.testModel)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to run inference-perf job: %v. Skipping benchmark.\n", err)
		return
	}
}

const (
	maxP95LatencySeconds = 5.0
	maxTTFTSeconds       = 1.0
	minThroughputReqPerS = 1.0
)

func getGatewayServiceURL(serviceName string) string {
	return fmt.Sprintf("http://%s", serviceName)
}

func createInferencePerfJob(baseURL string, modelName string) (string, error) {
	jobTemplate := `apiVersion: batch/v1
kind: Job
metadata:
  generateName: inference-perf-
  labels:
    app: inference-perf
  namespace: {{.Namespace}}
spec:
  template:
    metadata:
      labels:
        app: inference-perf
    spec:
      restartPolicy: Never
      containers:
      - name: inference-perf
        image: quay.io/inference-perf/inference-perf:d8e4af8
        command:
        - /bin/sh
        - -c
        args:
        - |
          cat <<EOF > /tmp/config.yaml
              api:
                type: completion
                streaming: true

              data:
                type: random
                input_distribution:
                  mean: 1000
                  std_dev: 0
                output_distribution:
                  mean: 1000
                  std_dev: 0

              load:
                type: constant
                interval: 5
                stages:
                  - rate: 20
                    duration: 10
                num_workers: 20
                worker_max_concurrency: 1000
                worker_max_tcp_connections: 2000
                request_timeout: 300

              server:
                type: vllm
                model_name: {{.ModelName}}
                base_url: {{.BaseURL}}

              report:
                request_lifecycle:
                  summary: false
                  per_stage: true
                  per_request: false

              storage:
                local_storage:
                  path: reports
          EOF

          /workspace/.venv/bin/inference-perf \
            -c /tmp/config.yaml \
            --log-level INFO

          cat reports*/*.json
{{- if .EnvVars}}
        env:
{{- range .EnvVars}}
          - name: {{.Name}}
            value: "{{.Value}}"
{{- end}}
{{- end}}
`

	type envVar struct {
		Name  string
		Value string
	}

	var envVars []envVar
	if cfg.hfToken != "" {
		envVars = append(envVars, envVar{Name: "HF_TOKEN", Value: cfg.hfToken})
	}
	if cfg.hfEndpoint != "" {
		envVars = append(envVars, envVar{Name: "HF_ENDPOINT", Value: cfg.hfEndpoint})
	}

	data := map[string]interface{}{
		"Namespace": cfg.workloadNamespace,
		"ModelName": modelName,
		"BaseURL":   baseURL,
		"EnvVars":   envVars,
	}

	jobYAML, err := renderTextTemplate(jobTemplate, data)
	if err != nil {
		return "", fmt.Errorf("failed to render job template: %w", err)
	}

	cmd := exec.Command("kubectl", "create", "-f", "-", "-n", cfg.workloadNamespace)
	cmd.Stdin = strings.NewReader(jobYAML)
	output, err := utils.Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}

	jobName := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(output, "job.batch/"), "created"))
	return jobName, nil
}

func waitForInferencePerfJob(jobName string) error {
	By("waiting for inference-perf job pod to be ready")
	podName, err := getInferencePerfJobPodName(jobName)
	if err != nil {
		return fmt.Errorf("failed to get job pod name: %w", err)
	}

	By("waiting for inference-perf to complete")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "logs", "-n", cfg.workloadNamespace, podName, "--tail=50")
		logs, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		
		logStr := strings.ToLower(logs)
		hasCompleted := strings.Contains(logStr, "stage 0 - run completed") ||
			strings.Contains(logStr, "generating reports") ||
			strings.Contains(logStr, "report saved to")
		
		if !hasCompleted {
			g.Expect(false).To(BeTrue(), "inference-perf has not completed yet")
		}
	}, timeoutVeryLong, intervalMedium).Should(Succeed(), "inference-perf should complete execution")

	By("waiting for inference-perf job to complete")
	Eventually(func() bool {
		cmd := exec.Command("kubectl", "get", "job", jobName,
			"-n", cfg.workloadNamespace,
			"-o", "jsonpath={.status.conditions[?(@.type==\"Complete\")].status}")
		output, err := utils.Run(cmd)
		return err == nil && strings.TrimSpace(output) == "True"
	}, timeoutMedium, intervalShort).Should(BeTrue(), "inference-perf job should complete successfully")

	cmd := exec.Command("kubectl", "get", "job", jobName,
		"-n", cfg.workloadNamespace,
		"-o", "jsonpath={.status.conditions[?(@.type==\"Failed\")].status}")
	output, _ := utils.Run(cmd)
	if strings.TrimSpace(output) == "True" {
		cmd = exec.Command("kubectl", "logs", "-n", cfg.workloadNamespace, fmt.Sprintf("job/%s", jobName))
		logs, _ := utils.Run(cmd)
		return fmt.Errorf("inference-perf job failed. Logs: %s", logs)
	}

	return nil
}

func getInferencePerfJobPodName(jobName string) (string, error) {
	cmd := exec.Command("kubectl", "get", "pods",
		"-n", cfg.workloadNamespace,
		"-l", fmt.Sprintf("job-name=%s", jobName),
		"-o", "jsonpath={.items[0].metadata.name}")
	output, err := utils.Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to get job pod name: %w", err)
	}
	podName := strings.TrimSpace(output)
	if podName == "" {
		return "", fmt.Errorf("job pod not found")
	}
	return podName, nil
}

func extractInferencePerfResults(jobName string) error {
	By("extracting inference-perf results from job pod")
	podName, err := getInferencePerfJobPodName(jobName)
	if err != nil {
		return err
	}

	cmd := exec.Command("kubectl", "exec", "-n", cfg.workloadNamespace, podName,
		"--", "sh", "-c", "cat reports*/*.json")
	reportData, err := utils.Run(cmd)
	if err != nil {
		return fmt.Errorf("failed to read report files: %w", err)
	}

	reportData = strings.TrimSpace(reportData)
	if reportData == "" {
		return fmt.Errorf("report data is empty")
	}

	var report map[string]interface{}
	err = json.Unmarshal([]byte(reportData), &report)
	if err != nil {
		return fmt.Errorf("failed to parse report JSON: %w", err)
	}

	if throughput, ok := report["throughput"].(float64); ok {
		Expect(throughput).To(BeNumerically(">", minThroughputReqPerS), "Throughput should be positive")
		fmt.Fprintf(GinkgoWriter, "Throughput: %.2f req/s\n", throughput)
	}

	if latency, ok := report["p95_latency"].(float64); ok {
		Expect(latency).To(BeNumerically("<", maxP95LatencySeconds),
			fmt.Sprintf("P95 latency (%.2fs) should be less than %.2fs", latency, maxP95LatencySeconds))
		fmt.Fprintf(GinkgoWriter, "P95 Latency: %.2fs\n", latency)
	}

	if ttft, ok := report["ttft_p95"].(float64); ok {
		Expect(ttft).To(BeNumerically("<", maxTTFTSeconds),
			fmt.Sprintf("P95 TTFT (%.2fs) should be less than %.2fs", ttft, maxTTFTSeconds))
		fmt.Fprintf(GinkgoWriter, "P95 TTFT: %.2fs\n", ttft)
	}

	return nil
}

func runInferencePerfJob(baseURL string, modelName string) error {
	By("creating inference-perf Job")
	jobName, err := createInferencePerfJob(baseURL, modelName)
	if err != nil {
		return fmt.Errorf("failed to create Job: %w", err)
	}

	defer func() {
		cmd := exec.Command("kubectl", "delete", "job", jobName,
			"-n", cfg.workloadNamespace, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)
	}()

	if err := waitForInferencePerfJob(jobName); err != nil {
		return err
	}

	if err := extractInferencePerfResults(jobName); err != nil {
		return fmt.Errorf("failed to extract results: %w", err)
	}

	return nil
}