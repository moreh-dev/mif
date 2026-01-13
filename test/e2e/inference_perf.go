//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/moreh-dev/mif/test/utils"
)

const (
	maxP95LatencySeconds = 5.0
	maxTTFTSeconds       = 1.0
	minThroughputReqPerS = 1.0
)

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
	err := runInferencePerfJob(gatewayServiceURL, cfg.testModel)
	Expect(err).NotTo(HaveOccurred(), "inference-perf job should complete successfully")
}

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
				sidecar.istio.io/inject: "false"
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

	cmd := exec.Command("kubectl", "create", "-f", "-", "-n", cfg.workloadNamespace, "-o", "name")
	cmd.Stdin = strings.NewReader(jobYAML)
	output, err := utils.Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}

	jobName := strings.TrimPrefix(strings.TrimSpace(output), "job.batch/")
	return jobName, nil
}

func waitForInferencePerfJob(jobName string) error {
	By("waiting for inference-perf job to complete")
	cmd := exec.Command("kubectl", "wait", "job", jobName,
		"--for=condition=complete",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", timeoutVeryLong))
	_, err := utils.Run(cmd)
	if err != nil {
		return fmt.Errorf("inference-perf job did not complete within timeout: %w", err)
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
