//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	cleanupTestResources := func() {
		if cfg.skipKind {
			_, _ = fmt.Fprintf(GinkgoWriter, "Using existing cluster (kubeconfig). Skipping resource cleanup for safety.\n")
			return
		}

		if cfg.skipCleanup {
			By("skipping cleanup (SKIP_CLEANUP=true)")
			return
		}

		By("cleaning up test resources")
		cmd := exec.Command("kubectl", "delete", "inferenceservice", inferenceServiceName,
			"-n", cfg.testNamespace, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)
	}

	AfterAll(func() {
		cleanupTestResources()
	})

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
		It("should deploy MIF components successfully", func() {
			By("validating that Odin controller is running")
			verifyOdinController := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment",
					"-n", cfg.testNamespace,
					"-l", "app.kubernetes.io/name=odin",
					"-o", "jsonpath={.items[0].status.conditions[?(@.type=='Available')].status}")
				output, err := utils.Run(cmd)
				if err != nil || strings.TrimSpace(output) == "" {
					cmd = exec.Command("kubectl", "get", "deployment",
						"-n", cfg.testNamespace,
						"-o", "jsonpath={.items[?(@.metadata.name=~\"odin.*\")].status.conditions[?(@.type=='Available')].status}")
					output, err = utils.Run(cmd)
				}
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(strings.TrimSpace(output)).To(Equal("True"), "Odin controller not available")
			}
			Eventually(verifyOdinController).Should(Succeed())
		})

		It("should have all pods ready", func() {
			By("waiting for all pods to be ready")
			verifyAllPodsReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods",
					"-n", cfg.testNamespace,
					"--field-selector=status.phase!=Succeeded",
					"-o", "jsonpath={.items[*].status.conditions[?(@.type=='Ready')].status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				statuses := strings.Fields(output)
				for _, status := range statuses {
					g.Expect(status).To(Equal("True"), "Some pods are not ready")
				}
			}
			Eventually(verifyAllPodsReady, timeoutVeryLong).Should(Succeed())
		})
	})

	Context("Gateway and InferenceService CR integration", func() {
		BeforeAll(func() {
			applyGatewayResource()
			installHeimdallForTest()
			installInferenceServiceForTest()
		})

		It("should have inference-service decode pods reachable behind Gateway", func() {
			verifyInferenceEndpoint()
		})

		It("should respond to inference requests correctly", func() {
			testInferenceAPI()
		})
	})

})

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
		"-n", cfg.testNamespace,
		"-o", "jsonpath={.items[*].metadata.name}")
	output, err := utils.Run(cmd)
	if err == nil {
		podNames := strings.Fields(output)
		for _, podName := range podNames {
			cmd = exec.Command("kubectl", "logs", podName, "-n", cfg.testNamespace, "--all-containers=true")
			logs, logErr := utils.Run(cmd)
			if logErr == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Pod %s logs:\n%s\n", podName, logs)
			}
		}
	}

	By("fetching Kubernetes events")
	cmd = exec.Command("kubectl", "get", "events", "-n", cfg.testNamespace, "--sort-by=.lastTimestamp")
	eventsOutput, err := utils.Run(cmd)
	if err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s\n", eventsOutput)
	}

	By("fetching resource status")
	cmd = exec.Command("kubectl", "get", "all", "-n", cfg.testNamespace)
	allOutput, err := utils.Run(cmd)
	if err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "All resources:\n%s\n", allOutput)
	}
}

func applyGatewayResource() {
	By("applying Gateway resource and infrastructure parameters")

	var baseYAML string

	if cfg.gatewayClass == "istio" {
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
	} else if cfg.gatewayClass == "kgateway" {
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
	} else {
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
		}
		if metadata, ok := doc["metadata"].(map[string]interface{}); ok {
			metadata["namespace"] = cfg.testNamespace
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
	encoder.Close()

	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(yamlContent.String())
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to apply Gateway resources")

	By("waiting for Gateway pods to be ready")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "pods",
			"-n", cfg.testNamespace,
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
	Expect(utils.InstallHeimdall(cfg.testNamespace, heimdallValuesPath)).To(Succeed(), "Failed to install Heimdall for test")

	By("waiting for Heimdall deployment to be ready")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "deployment",
			"-n", cfg.testNamespace,
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
	if imageRepo == "" && imageTag == "" {
		image = ""
	} else {
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
		md["namespace"] = cfg.testNamespace
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

	// Set imagePullSecrets
	if imagePullSecrets, ok := podSpec["imagePullSecrets"].([]interface{}); ok && len(imagePullSecrets) > 0 {
		if secret, ok := imagePullSecrets[0].(map[string]interface{}); ok {
			secret["name"] = secretNameMorehRegistry
		}
	}

	// Set image only if provided, otherwise use YAML default
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

		// Set initialDelaySeconds to 10 for kind cluster
		if readinessProbe, ok := mainContainer["readinessProbe"].(map[string]interface{}); ok {
			readinessProbe["initialDelaySeconds"] = 10
		}
	}

	if resources := getGPUResources("2", "2"); resources != nil {
		mainContainer["resources"] = resources
	} else {
		delete(mainContainer, "resources")
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
	} else {
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

	By("installing Odin InferenceService for test")
	Expect(utils.InstallInferenceService(cfg.testNamespace, manifestPath)).To(Succeed(), "Failed to install Odin InferenceService for test")

	By("waiting for Odin InferenceService deployment to be ready")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "deployment",
			inferenceServiceName,
			"-n", cfg.testNamespace,
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
			"-n", cfg.testNamespace,
			"-l", "gateway.networking.k8s.io/gateway-name=mif",
			"-o", "jsonpath={.items[0].metadata.name}")
		output, err := utils.Run(cmd)
		if err != nil || strings.TrimSpace(output) == "" {
			cmd = exec.Command("kubectl", "get", "service",
				"mif",
				"-n", cfg.testNamespace,
				"-o", "jsonpath={.metadata.name}")
			output, err = utils.Run(cmd)
		}
		if err != nil || strings.TrimSpace(output) == "" {
			cmd = exec.Command("kubectl", "get", "service",
				"gateway-mif",
				"-n", cfg.testNamespace,
				"-o", "jsonpath={.metadata.name}")
			output, err = utils.Run(cmd)
		}
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(strings.TrimSpace(output)).NotTo(BeEmpty(), "Gateway service not found")
	}, timeoutLong, intervalLong).Should(Succeed())

	By("verifying inference-service decode pods are running")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "pods",
			"-n", cfg.testNamespace,
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
			"-n", cfg.testNamespace,
			"-l", "gateway.networking.k8s.io/gateway-name=mif",
			"-o", "jsonpath={.items[0].metadata.name}")
		output, err := utils.Run(cmd)
		if err != nil || strings.TrimSpace(output) == "" {
			cmd = exec.Command("kubectl", "get", "service",
				"mif",
				"-n", cfg.testNamespace,
				"-o", "jsonpath={.metadata.name}")
			output, err = utils.Run(cmd)
		}
		if err != nil || strings.TrimSpace(output) == "" {
			cmd = exec.Command("kubectl", "get", "service",
				"gateway-mif",
				"-n", cfg.testNamespace,
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
		"-n", cfg.testNamespace,
		fmt.Sprintf("service/%s", serviceName),
		fmt.Sprintf("%s:80", portForwardPort))
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter

	err := cmd.Start()
	Expect(err).NotTo(HaveOccurred(), "Failed to start port-forward")
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	time.Sleep(2 * time.Second)

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
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	Expect(err).NotTo(HaveOccurred(), "Failed to create HTTP request")

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 2 * time.Minute,
	}

	var resp *http.Response
	Eventually(func(g Gomega) {
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
