//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	type jobTemplateData struct {
		Namespace          string
		ModelName          string
		BaseURL            string
		HFToken            string
		HFEndpoint         string
		IsKind             bool
		S3AccessKeyID      string
		S3SecretAccessKey  string
		S3Region           string
		S3Bucket           string
		S3PrefixBase       string
		VLLMTag            string
		Preset             string
		ExpType            string
		ExpName            string
	}

	_, imageTag := getInferenceImageInfo()

	data := jobTemplateData{
		Namespace:         cfg.workloadNamespace,
		ModelName:         modelName,
		BaseURL:           baseURL,
		HFToken:           cfg.hfToken,
		HFEndpoint:        cfg.hfEndpoint,
		IsKind:            cfg.isUsingKindCluster,
		S3AccessKeyID:     cfg.s3AccessKeyID,
		S3SecretAccessKey: cfg.s3SecretAccessKey,
		S3Region:          cfg.s3Region,
		S3Bucket:          cfg.s3Bucket,
		S3PrefixBase:      inferencePerfS3PrefixBase,
		VLLMTag:           imageTag,
		Preset:            inferencePerfPreset,
		ExpType:           inferencePerfExpType,
		ExpName:           inferencePerfExpName,
	}

	jobYAML, err := renderTemplateFile("inference-perf-job.yaml.tmpl", data)
	if err != nil {
		return "", fmt.Errorf("failed to render job template: %w", err)
	}

	cmd := exec.Command("kubectl", "create", "-f", "-", "-n", cfg.workloadNamespace, "-o", "name")
	cmd.Stdin = strings.NewReader(jobYAML)
	output, err := Run(cmd)
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
	_, err := Run(cmd)
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
		_, _ = Run(cmd)
	}()

	if err := waitForInferencePerfJob(jobName); err != nil {
		return err
	}

	return nil
}
