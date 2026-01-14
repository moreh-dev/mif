//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/moreh-dev/mif/test/utils"
)

func runInferencePerfBenchmark() {
	By("getting Gateway service name")
	serviceName := getGatewayServiceName(timeoutMedium, intervalMedium)

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

	return nil
}
