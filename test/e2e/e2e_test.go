//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/moreh-dev/mif/test/utils"
)

const inferenceServiceName = testInferenceServiceName

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
			By("installing InferenceService")
			installInferenceServiceForTest()
		})

		AfterAll(func() {
			if cfg.skipCleanup {
				return
			}
			if !cfg.isUsingKindCluster {
				return
			}
			By("cleaning up test workload namespace")
			if err := utils.CleanupWorkloadNamespace(cfg.workloadNamespace, testInferenceServiceName, cfg.gatewayClass); err != nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to cleanup workload namespace: %v\n", err)
			}
		})

		It("should have inference-service decode pods reachable behind Gateway", func() {
			verifyInferenceEndpoint()
		})

		It("should respond to inference requests correctly", func() {
			testInferenceAPI()
		})
	})

})

func verifyOdinController(g Gomega) {
	cmd := exec.Command("kubectl", "get", "deployment",
		"-n", cfg.mifNamespace,
		"-l", "app.kubernetes.io/name=odin",
		"-o", "jsonpath={.items[0].status.conditions[?(@.type=='Available')].status}")
	output, err := utils.Run(cmd)
	if err != nil || strings.TrimSpace(output) == "" {
		cmd = exec.Command("kubectl", "get", "deployment",
			"-n", cfg.mifNamespace,
			"-o", "jsonpath={.items[?(@.metadata.name=~\"odin.*\")].status.conditions[?(@.type=='Available')].status}")
		output, err = utils.Run(cmd)
	}
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strings.TrimSpace(output)).To(Equal("True"), "Odin controller not available")
}

func verifyAllPodsReady(g Gomega) {
	cmd := exec.Command("kubectl", "get", "pods",
		"-n", cfg.mifNamespace,
		"--field-selector=status.phase!=Succeeded",
		"-o", "jsonpath={.items[*].status.conditions[?(@.type=='Ready')].status}")
	output, err := utils.Run(cmd)
	g.Expect(err).NotTo(HaveOccurred())
	statuses := strings.Fields(output)
	for _, status := range statuses {
		g.Expect(status).To(Equal("True"), "Some pods are not ready")
	}
}

func getInferenceImageInfo() (repo, tag string) {
	repo = cfg.inferenceImageRepo
	tag = cfg.inferenceImageTag

	if repo == "" {
		if cfg.isUsingKindCluster {
			repo = imageRepoKindDefault
		} else {
			// Use default from YAML template
			repo = ""
		}
	}

	if tag == "" {
		if cfg.isUsingKindCluster {
			tag = imageTagKindDefault
		} else {
			// Use default from YAML template
			tag = ""
		}
	}

	return repo, tag
}

func getGPUResources(requests, limits string) map[string]interface{} {
	if cfg.isUsingKindCluster {
		return nil
	}
	return map[string]interface{}{
		"requests": map[string]interface{}{
			"amd.com/gpu": requests,
		},
		"limits": map[string]interface{}{
			"amd.com/gpu": limits,
		},
	}
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
			cmd = exec.Command("kubectl", "logs", podName, "-n", cfg.mifNamespace, "--all-containers=true")
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
  namespace: mif
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
  namespace: mif
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
  namespace: mif
spec:
  kube:
    service:
      type: ClusterIP

---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: mif
  namespace: mif
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

	var yamlDocs []map[string]interface{}
	decoder := yaml.NewDecoder(strings.NewReader(baseYAML))
	for {
		var doc map[string]interface{}
		if err := decoder.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			Expect(err).NotTo(HaveOccurred(), "Failed to parse base YAML")
			return
		}
		if metadata, ok := doc["metadata"].(map[string]interface{}); ok {
			metadata["namespace"] = cfg.workloadNamespace
		}
		yamlDocs = append(yamlDocs, doc)
	}

	var yamlContent strings.Builder
	encoder := yaml.NewEncoder(&yamlContent)
	for i, doc := range yamlDocs {
		if i > 0 {
			yamlContent.WriteString("---\n")
		}
		Expect(encoder.Encode(doc)).To(Succeed())
	}
	Expect(encoder.Close()).To(Succeed())

	cmd := exec.Command("kubectl", "apply", "-f", "-", "--request-timeout=60s")
	cmd.Stdin = strings.NewReader(yamlContent.String())
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to apply Gateway resources")

	By("waiting for Gateway pods to be ready")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "pods",
			"-n", cfg.workloadNamespace,
			"-l", "gateway.networking.k8s.io/gateway-name=mif",
			"-o", "jsonpath={.items[*].status.phase}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		phases := strings.Fields(output)
		g.Expect(len(phases)).To(BeNumerically(">=", 1), "No Gateway pods found")
		for _, p := range phases {
			g.Expect(p).To(Equal("Running"), "Gateway pod is not running")
		}
	}, timeoutLong, intervalLong).Should(Succeed())
}

func installHeimdallForTest() {
	By("creating Heimdall values file for test")
	heimdallValuesPath, err := createHeimdallValuesFile()
	Expect(err).NotTo(HaveOccurred(), "Failed to create Heimdall values file for test")

	By("installing Heimdall for test")
	Expect(utils.InstallHeimdall(cfg.workloadNamespace, heimdallValuesPath)).To(Succeed(), "Failed to install Heimdall for test")

	By("waiting for Heimdall deployment to be ready")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "deployment",
			"-n", cfg.workloadNamespace,
			"-l", "app.kubernetes.io/instance=heimdall",
			"-o", "jsonpath={.items[0].status.conditions[?(@.type=='Available')].status}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(strings.TrimSpace(output)).To(Equal("True"), "Heimdall deployment not available")
	}, timeoutLong, intervalLong).Should(Succeed())
}

func createInferenceServiceValuesFile() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	imageRepo, imageTag := getInferenceImageInfo()
	var image string
	if imageRepo != "" && imageTag != "" {
		image = fmt.Sprintf("%s:%s", imageRepo, imageTag)
	}

	baseYAML := `apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: vllm-llama3-1b-instruct-tp2
  namespace: quickstart
spec:
  replicas: 2
  inferencePoolRefs:
    - name: heimdall
  template:
    spec:
      imagePullSecrets:
        - name: moreh-registry
      containers:
        - name: main
          image: 255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/quickstart/moreh-vllm:20250915.1
          securityContext:
            capabilities:
              add:
              - IPC_LOCK
          command:
            - vllm
            - serve
          args:
            - meta-llama/Llama-3.2-1B-Instruct
            - --port
            - "8000"
            - --quantization
            - "None"
            - --tensor-parallel-size
            - "2"
            - --max-num-batched-tokens
            - "8192"
            - --no-enable-prefix-caching
            - --no-enable-log-requests
            - --disable-uvicorn-access-log
          env:
            - name: HF_TOKEN
              value: "<huggingfaceToken>"
          resources:
            requests:
              amd.com/gpu: "2"
            limits:
              amd.com/gpu: "2"
          ports:
            - containerPort: 8000
              name: http
              protocol: TCP
          readinessProbe:
            httpGet:
              path: /health
              port: 8000
              scheme: HTTP
            initialDelaySeconds: 120
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
            sizeLimit: "2Gi"
      tolerations:
        - key: "amd.com/gpu"
          operator: "Exists"
          effect: "NoSchedule"
`

	var manifest map[string]interface{}
	if err := yaml.Unmarshal([]byte(baseYAML), &manifest); err != nil {
		return "", fmt.Errorf("failed to parse base Odin InferenceService YAML: %w", err)
	}

	if md, ok := manifest["metadata"].(map[string]interface{}); ok {
		md["name"] = inferenceServiceName
		md["namespace"] = cfg.workloadNamespace
	}

	spec, ok := manifest["spec"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid Odin InferenceService manifest: missing spec")
	}

	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid Odin InferenceService manifest: missing spec.template")
	}

	podSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid Odin InferenceService manifest: missing spec.template.spec")
	}

	containers, ok := podSpec["containers"].([]interface{})
	if !ok || len(containers) == 0 {
		return "", fmt.Errorf("invalid Odin InferenceService manifest: missing containers")
	}

	mainContainer, ok := containers[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid Odin InferenceService manifest: malformed main container")
	}

	if imagePullSecrets, ok := podSpec["imagePullSecrets"].([]interface{}); ok && len(imagePullSecrets) > 0 {
		if secret, ok := imagePullSecrets[0].(map[string]interface{}); ok {
			secret["name"] = secretNameMorehRegistry
		}
	}

	if image != "" {
		mainContainer["image"] = image
	}

	if cfg.isUsingKindCluster {
		mainContainer["command"] = []interface{}{
			"/app/llm-d-inference-sim",
		}
		mainContainer["args"] = []interface{}{
			"--model",
			cfg.testModel,
			"--port",
			"8000",
		}

		if readinessProbe, ok := mainContainer["readinessProbe"].(map[string]interface{}); ok {
			readinessProbe["initialDelaySeconds"] = 10
		}
	} else {
		if args, ok := mainContainer["args"].([]interface{}); ok && len(args) > 0 {
			args[0] = cfg.testModel
			mainContainer["args"] = args
		}
	}

	if resources := getGPUResources("2", "2"); resources != nil {
		mainContainer["resources"] = resources
	} else {
		delete(mainContainer, "resources")
	}

	if env, ok := mainContainer["env"].([]interface{}); ok {
		var filteredEnv []interface{}
		for _, envVar := range env {
			if envMap, ok := envVar.(map[string]interface{}); ok {
				if value, ok := envMap["value"].(string); ok && value == "<huggingfaceToken>" {
					continue
				}
				filteredEnv = append(filteredEnv, envVar)
			}
		}
		if len(filteredEnv) > 0 {
			mainContainer["env"] = filteredEnv
		} else {
			delete(mainContainer, "env")
		}
	}

	if cfg.hfToken != "" || cfg.hfEndpoint != "" {
		var envList []interface{}
		if cfg.hfToken != "" {
			envList = append(envList, map[string]interface{}{
				"name":  "HF_TOKEN",
				"value": cfg.hfToken,
			})
		}
		if cfg.hfEndpoint != "" {
			envList = append(envList, map[string]interface{}{
				"name":  "HF_ENDPOINT",
				"value": cfg.hfEndpoint,
			})
		}
		if len(envList) > 0 {
			mainContainer["env"] = envList
		}
	} else if _, exists := mainContainer["env"]; !exists {
		delete(mainContainer, "env")
	}

	manifestYAML, err := yaml.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Odin InferenceService manifest: %w", err)
	}

	manifestPath := filepath.Join(projectDir, tempFileISValues)
	if err := os.WriteFile(manifestPath, manifestYAML, 0600); err != nil {
		return "", fmt.Errorf("failed to write Odin InferenceService manifest file: %w", err)
	}

	return manifestPath, nil
}

func installInferenceServiceForTest() {
	By("creating Odin InferenceService manifest file for test")
	manifestPath, err := createInferenceServiceValuesFile()
	Expect(err).NotTo(HaveOccurred(), "Failed to create Odin InferenceService manifest file for test")

	By("creating Odin InferenceService for test")
	Expect(utils.CreateInferenceService(cfg.workloadNamespace, manifestPath)).To(Succeed(), "Failed to create Odin InferenceService for test")

	By("waiting for Odin InferenceService deployment to be ready")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "deployment",
			inferenceServiceName,
			"-n", cfg.workloadNamespace,
			"-o", "jsonpath={.status.conditions[?(@.type=='Available')].status}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(strings.TrimSpace(output)).To(Equal("True"), "Odin InferenceService deployment not available")
	}, timeoutVeryLong, intervalLong).Should(Succeed())
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

	By("verifying inference-service decode pods are running")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "pods",
			"-n", cfg.workloadNamespace,
			"-l", fmt.Sprintf("app.kubernetes.io/name=%s", inferenceServiceName),
			"--field-selector=status.phase=Running",
			"--no-headers", "-o", "custom-columns=:metadata.name")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		podNames := utils.GetNonEmptyLines(output)
		g.Expect(len(podNames)).To(BeNumerically(">=", 1), "No Odin InferenceService pods found")
	}, timeoutVeryLong, intervalLong).Should(Succeed())
}

func testInferenceAPI() {
	By("getting Gateway service name for port-forward")
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

	By("setting up port-forward to Gateway service")
	portForwardPort := "8000"
	cmd := exec.Command("kubectl", "port-forward",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("service/%s", serviceName),
		fmt.Sprintf("%s:80", portForwardPort))
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter

	defer func() {
		if cmd.Process == nil {
			return
		}
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			return
		}
		if err := cmd.Process.Kill(); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "failed to kill port-forward process: %v\n", err)
		}
	}()

	err := cmd.Start()
	Expect(err).NotTo(HaveOccurred(), "Failed to start port-forward")

	Eventually(func(g Gomega) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%s", portForwardPort), 1*time.Second)
		if err == nil {
			conn.Close()
		}
		g.Expect(err).NotTo(HaveOccurred(), "port-forward not ready")
	}, 30*time.Second, 1*time.Second).Should(Succeed())

	By("sending chat completion request to inference endpoint")
	requestBody := map[string]interface{}{
		"model": cfg.testModel,
		"messages": []map[string]interface{}{
			{
				"role":    "developer",
				"content": "You are a helpful assistant.",
			},
			{
				"role":    "user",
				"content": "Hello!",
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	Expect(err).NotTo(HaveOccurred(), "Failed to marshal request body")

	url := fmt.Sprintf("http://localhost:%s/v1/chat/completions", portForwardPort)

	client := &http.Client{
		Timeout: 2 * time.Minute,
	}

	var resp *http.Response
	Eventually(func(g Gomega) {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		g.Expect(err).NotTo(HaveOccurred(), "Failed to create HTTP request")
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Do(req)
		g.Expect(err).NotTo(HaveOccurred(), "Failed to send HTTP request")
		g.Expect(resp.StatusCode).To(Equal(http.StatusOK), fmt.Sprintf("Expected status 200, got %d", resp.StatusCode))
	}, timeoutMedium, intervalLong).Should(Succeed())

	defer resp.Body.Close()

	By("verifying response body")
	body, err := io.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred(), "Failed to read response body")

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal response JSON")

	Expect(response).To(HaveKey("id"), "Response should have 'id' field")
	Expect(response).To(HaveKey("choices"), "Response should have 'choices' field")
	Expect(response).To(HaveKey("model"), "Response should have 'model' field")

	choices, ok := response["choices"].([]interface{})
	Expect(ok).To(BeTrue(), "Response 'choices' should be an array")
	Expect(len(choices)).To(BeNumerically(">=", 1), "Response should have at least one choice")

	choice, ok := choices[0].(map[string]interface{})
	Expect(ok).To(BeTrue(), "Choice should be an object")
	Expect(choice).To(HaveKey("message"), "Choice should have 'message' field")

	message, ok := choice["message"].(map[string]interface{})
	Expect(ok).To(BeTrue(), "Message should be an object")
	Expect(message).To(HaveKey("content"), "Message should have 'content' field")
	Expect(message["content"]).NotTo(BeEmpty(), "Message content should not be empty")

	_, _ = fmt.Fprintf(GinkgoWriter, "Successfully received inference response: %s\n", string(body))
}