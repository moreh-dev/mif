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

var _ = Describe("Prefill-Decode Disaggregation", Ordered, func() {
	var inferenceServiceName = "pd-disaggregation-test"

	cleanupTestResources := func() {
		if skipKindCreate {
			_, _ = fmt.Fprintf(GinkgoWriter, "Using existing cluster (kubeconfig). Skipping resource cleanup for safety.\n")
			return
		}

		if skipCleanup {
			By("skipping cleanup (SKIP_CLEANUP=true)")
			return
		}

		By("cleaning up test resources")
		cmd := exec.Command("kubectl", "delete", "inferenceservice", inferenceServiceName,
			"-n", testNamespace, "--ignore-not-found=true")
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

	SetDefaultEventuallyTimeout(15 * time.Minute)
	SetDefaultEventuallyPollingInterval(5 * time.Second)

	Context("MIF Infrastructure", func() {
		It("should deploy MIF components successfully", func() {
			By("validating that Odin controller is running")
			verifyOdinController := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment",
					"-n", testNamespace,
					"-l", "app.kubernetes.io/name=odin",
					"-o", "jsonpath={.items[0].status.conditions[?(@.type=='Available')].status}")
				output, err := utils.Run(cmd)
				if err != nil || strings.TrimSpace(output) == "" {
					cmd = exec.Command("kubectl", "get", "deployment",
						"-n", testNamespace,
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
					"-n", testNamespace,
					"--field-selector=status.phase!=Succeeded",
					"-o", "jsonpath={.items[*].status.conditions[?(@.type=='Ready')].status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				statuses := strings.Fields(output)
				for _, status := range statuses {
					g.Expect(status).To(Equal("True"), "Some pods are not ready")
				}
			}
			Eventually(verifyAllPodsReady, 15*time.Minute).Should(Succeed())
		})
	})

	Context("Gateway and Helm-based inference-service integration", func() {
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
	repo = os.Getenv("INFERENCE_IMAGE_REPO")
	tag = os.Getenv("INFERENCE_IMAGE_TAG")

	if repo == "" {
		if isUsingKindCluster {
			repo = "ghcr.io/llm-d/llm-d-inference-sim"
		} else {
			repo = "255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/quickstart/moreh-vllm"
		}
	}

	if tag == "" {
		if isUsingKindCluster {
			tag = "v0.6.1"
		} else {
			tag = "20250915.1"
		}
	}

	return repo, tag
}

func getGPUResources(requests, limits string) map[string]interface{} {
	if isUsingKindCluster {
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
		"-n", testNamespace,
		"-o", "jsonpath={.items[*].metadata.name}")
	output, err := utils.Run(cmd)
	if err == nil {
		podNames := strings.Fields(output)
		for _, podName := range podNames {
			cmd = exec.Command("kubectl", "logs", podName, "-n", testNamespace, "--all-containers=true")
			logs, logErr := utils.Run(cmd)
			if logErr == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Pod %s logs:\n%s\n", podName, logs)
			}
		}
	}

	By("fetching Kubernetes events")
	cmd = exec.Command("kubectl", "get", "events", "-n", testNamespace, "--sort-by=.lastTimestamp")
	eventsOutput, err := utils.Run(cmd)
	if err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s\n", eventsOutput)
	}

	By("fetching resource status")
	cmd = exec.Command("kubectl", "get", "all", "-n", testNamespace)
	allOutput, err := utils.Run(cmd)
	if err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "All resources:\n%s\n", allOutput)
	}
}

func applyGatewayResource() {
	By("applying Gateway resource and infrastructure parameters")

	var baseYAML string

	if gatewayClass == "istio" {
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
	} else if gatewayClass == "kgateway" {
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
		Fail(fmt.Sprintf("Unsupported gatewayClassName: %s", gatewayClass))
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
			metadata["namespace"] = testNamespace
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
			"-n", testNamespace,
			"-l", "gateway.networking.k8s.io/gateway-name=mif",
			"-o", "jsonpath={.items[*].status.phase}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		phases := strings.Fields(output)
		g.Expect(len(phases)).To(BeNumerically(">=", 1), "No Gateway pods found")
		for _, p := range phases {
			g.Expect(p).To(Equal("Running"), "Gateway pod is not running")
		}
	}, 10*time.Minute, 10*time.Second).Should(Succeed())
}

func installHeimdallForTest() {
	By("creating Heimdall values file for test")
	heimdallValuesPath, err := createHeimdallValuesFile()
	Expect(err).NotTo(HaveOccurred(), "Failed to create Heimdall values file for test")

	By("installing Heimdall for test")
	Expect(utils.InstallHeimdall(testNamespace, heimdallValuesPath)).To(Succeed(), "Failed to install Heimdall for test")

	By("waiting for Heimdall deployment to be ready")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "deployment",
			"-n", testNamespace,
			"-l", "app.kubernetes.io/instance=heimdall",
			"-o", "jsonpath={.items[0].status.conditions[?(@.type=='Available')].status}")
		output, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(strings.TrimSpace(output)).To(Equal("True"), "Heimdall deployment not available")
	}, 10*time.Minute, 10*time.Second).Should(Succeed())
}

func createInferenceServiceValuesFile() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	imageRepo, imageTag := getInferenceImageInfo()
	hfToken := os.Getenv("HF_TOKEN")
	hfEndpoint := os.Getenv("HF_ENDPOINT")

	baseYAML := `global:
  imagePullSecrets:
    - name: moreh-registry

extraArgs:
  - meta-llama/Llama-3.2-1B-Instruct
  - --quantization
  - "None"
  - --tensor-parallel-size
  - "2"
  - --max-num-batched-tokens
  - "8192"
  - --no-enable-prefix-caching
  - --no-enable-log-requests
  - --disable-uvicorn-access-log

extraEnvVars:
  - name: HF_TOKEN
    value: "<huggingfaceToken>"

_common:
  image:
    repository: 255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/quickstart/moreh-vllm
    tag: "20250915.1"

  resources:
    requests:
      amd.com/gpu: "2"
    limits:
      amd.com/gpu: "2"

  podMonitor:
    labels:
      release: prometheus-stack

decode:
  replicas: 2

prefill:
  enabled: false
`

	var values map[string]interface{}
	err = yaml.Unmarshal([]byte(baseYAML), &values)
	if err != nil {
		return "", fmt.Errorf("failed to parse base YAML: %w", err)
	}

	_common := values["_common"].(map[string]interface{})
	_common["image"].(map[string]interface{})["repository"] = imageRepo
	_common["image"].(map[string]interface{})["tag"] = imageTag

	if isUsingKindCluster {
		// Override command and extraArgs for llm-d-inference-sim (kind cluster) to match official CLI usage.
		_common["command"] = []interface{}{
			"/app/llm-d-inference-sim",
		}
		values["extraArgs"] = []interface{}{
			"--model",
			testModel,
		}
	}

	if resources := getGPUResources("2", "2"); resources != nil {
		_common["resources"] = resources
	} else {
		delete(_common, "resources")
	}

	common := values["_common"].(map[string]interface{})
	for k, v := range common {
		if k == "resources" && getGPUResources("2", "2") == nil {
			continue
		}
		values["decode"].(map[string]interface{})[k] = v
		values["prefill"].(map[string]interface{})[k] = v
	}

	if hfToken != "" || hfEndpoint != "" {
		var extraEnv []map[string]interface{}
		if hfToken != "" {
			extraEnv = append(extraEnv, map[string]interface{}{
				"name":  "HF_TOKEN",
				"value": hfToken,
			})
		}
		if hfEndpoint != "" {
			extraEnv = append(extraEnv, map[string]interface{}{
				"name":  "HF_ENDPOINT",
				"value": hfEndpoint,
			})
		}
		values["extraEnvVars"] = extraEnv
	} else {
		delete(values, "extraEnvVars")
	}

	valuesYAML, err := yaml.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("failed to marshal inference-service values: %w", err)
	}

	valuesPath := filepath.Join(projectDir, "test/e2e/inference-service-values.yaml")
	if err := os.WriteFile(valuesPath, valuesYAML, 0600); err != nil {
		return "", fmt.Errorf("failed to write inference-service values file: %w", err)
	}

	return valuesPath, nil
}

func installInferenceServiceForTest() {
	By("creating inference-service values file")
	valuesPath, err := createInferenceServiceValuesFile()
	Expect(err).NotTo(HaveOccurred(), "Failed to create inference-service values file")

	By("installing inference-service via Helm")
	Expect(utils.InstallInferenceService(testNamespace, valuesPath)).To(Succeed(), "Failed to install inference-service")

	By("waiting for inference-service decode deployment to be ready")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "deployment",
			"-n", testNamespace,
			"-l", "app.kubernetes.io/instance=inference-service,heimdall/role=decode",
			"-o", "jsonpath={.items[0].status.conditions[?(@.type=='Available')].status}")
		output, err := utils.Run(cmd)
		if err != nil || strings.TrimSpace(output) == "" {
			// Fallback: try by deployment name directly
			cmd = exec.Command("kubectl", "get", "deployment",
				"inference-service-decode",
				"-n", testNamespace,
				"-o", "jsonpath={.status.conditions[?(@.type=='Available')].status}")
			output, err = utils.Run(cmd)
		}
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(strings.TrimSpace(output)).To(Equal("True"), "inference-service decode deployment not available")
	}, 15*time.Minute, 15*time.Second).Should(Succeed())
}

func verifyInferenceEndpoint() {
	By("verifying Gateway service exists")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "service",
			"-n", testNamespace,
			"-l", "gateway.networking.k8s.io/gateway-name=mif",
			"-o", "jsonpath={.items[0].metadata.name}")
		output, err := utils.Run(cmd)
		if err != nil || strings.TrimSpace(output) == "" {
			cmd = exec.Command("kubectl", "get", "service",
				"mif",
				"-n", testNamespace,
				"-o", "jsonpath={.metadata.name}")
			output, err = utils.Run(cmd)
		}
		if err != nil || strings.TrimSpace(output) == "" {
			cmd = exec.Command("kubectl", "get", "service",
				"gateway-mif",
				"-n", testNamespace,
				"-o", "jsonpath={.metadata.name}")
			output, err = utils.Run(cmd)
		}
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(strings.TrimSpace(output)).NotTo(BeEmpty(), "Gateway service not found")
	}, 10*time.Minute, 10*time.Second).Should(Succeed())

	By("verifying inference-service decode pods are running")
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "pods",
			"-n", testNamespace,
			"-l", "app.kubernetes.io/instance=inference-service,heimdall/role=decode",
			"--no-headers", "-o", "custom-columns=:metadata.name")
		output, err := utils.Run(cmd)
		if err != nil || strings.TrimSpace(output) == "" {
			cmd = exec.Command("kubectl", "get", "pods",
				"-n", testNamespace,
				"-l", "app.kubernetes.io/instance=inference-service",
				"--field-selector=status.phase=Running",
				"--no-headers", "-o", "custom-columns=:metadata.name")
			output, err = utils.Run(cmd)
		}
		g.Expect(err).NotTo(HaveOccurred())
		podNames := utils.GetNonEmptyLines(output)
		g.Expect(len(podNames)).To(BeNumerically(">=", 1), "No inference-service decode pods found")
	}, 15*time.Minute, 15*time.Second).Should(Succeed())
}

func testInferenceAPI() {
	By("getting Gateway service name for port-forward")
	var serviceName string
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "service",
			"-n", testNamespace,
			"-l", "gateway.networking.k8s.io/gateway-name=mif",
			"-o", "jsonpath={.items[0].metadata.name}")
		output, err := utils.Run(cmd)
		if err != nil || strings.TrimSpace(output) == "" {
			cmd = exec.Command("kubectl", "get", "service",
				"mif",
				"-n", testNamespace,
				"-o", "jsonpath={.metadata.name}")
			output, err = utils.Run(cmd)
		}
		if err != nil || strings.TrimSpace(output) == "" {
			cmd = exec.Command("kubectl", "get", "service",
				"gateway-mif",
				"-n", testNamespace,
				"-o", "jsonpath={.metadata.name}")
			output, err = utils.Run(cmd)
		}
		g.Expect(err).NotTo(HaveOccurred())
		serviceName = strings.TrimSpace(output)
		g.Expect(serviceName).NotTo(BeEmpty(), "Gateway service not found")
	}, 5*time.Minute, 5*time.Second).Should(Succeed())

	By("setting up port-forward to Gateway service")
	portForwardPort := "8000"
	cmd := exec.Command("kubectl", "port-forward",
		"-n", testNamespace,
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
		"model": testModel,
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
	}, 5*time.Minute, 10*time.Second).Should(Succeed())

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
