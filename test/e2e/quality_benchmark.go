//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/moreh-dev/mif/test/utils"
)

func runQualityBenchmark() {
	if !cfg.qualityBenchmarkEnabled {
		Skip("Quality benchmark is disabled (QUALITY_BENCHMARK_ENABLED=false)")
	}

	if cfg.qualityBenchmarks == "" {
		Skip("No quality benchmarks specified (QUALITY_BENCHMARKS is empty)")
	}

	By("getting Gateway service name")
	serviceName := getGatewayServiceName(timeoutMedium, intervalMedium)

	By("running quality benchmarks as Kubernetes Job")
	gatewayServiceURL := fmt.Sprintf("http://%s:80", serviceName)
	err := runQualityBenchmarkJob(gatewayServiceURL, cfg.testModel, cfg.qualityBenchmarks, cfg.qualityBenchmarkLimit)
	Expect(err).NotTo(HaveOccurred(), "quality benchmark job should complete successfully")
}

func createQualityBenchmarkJob(baseURL string, modelName string, benchmarks string, limit string) (string, error) {
	type jobTemplateData struct {
		Namespace         string
		ModelName         string
		GatewayHost       string
		GatewayPort       string
		HFToken           string
		HFEndpoint        string
		GithubToken       string
		Benchmarks        string
		Limit             string
		QualityEvalRepo   string
		QualityEvalBranch string
	}

	gatewayHost := baseURL
	gatewayPort := "80"

	parsedURL, err := url.Parse(baseURL)
	if err == nil && parsedURL.Host != "" {
		host := parsedURL.Host
		// Try to split host and port safely (supports IPv4, IPv6, hostnames)
		if h, p, splitErr := net.SplitHostPort(host); splitErr == nil {
			gatewayHost = h
			gatewayPort = p
		} else {
			// No explicit port in host, keep default port and use host as-is
			gatewayHost = host
		}
	}

	data := jobTemplateData{
		Namespace:         cfg.workloadNamespace,
		ModelName:         modelName,
		GatewayHost:       gatewayHost,
		GatewayPort:       gatewayPort,
		HFToken:           cfg.hfToken,
		HFEndpoint:        cfg.hfEndpoint,
		GithubToken:       cfg.githubToken,
		Benchmarks:        benchmarks,
		Limit:             limit,
		QualityEvalRepo:   "https://github.com/moreh-dev/moreh-llm-eval.git",
		QualityEvalBranch: "main",
	}

	jobYAML, err := renderTemplateFile("quality-benchmark-job.yaml.tmpl", data)
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

func waitForQualityBenchmarkJob(jobName string) error {
	By("waiting for quality benchmark job to complete")
	cmd := exec.Command("kubectl", "wait", "job", jobName,
		"--for=condition=complete",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", timeoutVeryLong))
	_, err := utils.Run(cmd)
	if err != nil {
		By("collecting logs from failed quality benchmark job")
		logCmd := exec.Command("kubectl", "logs", "-l", "app=quality-benchmark",
			"-n", cfg.workloadNamespace,
			"--tail=100")
		if logs, logErr := utils.Run(logCmd); logErr == nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "Quality benchmark job logs:\n%s\n", logs)
		}
		return fmt.Errorf("quality benchmark job did not complete within timeout: %w", err)
	}

	return nil
}

func runQualityBenchmarkJob(baseURL string, modelName string, benchmarks string, limit string) error {
	By("creating quality benchmark Job")
	jobName, err := createQualityBenchmarkJob(baseURL, modelName, benchmarks, limit)
	if err != nil {
		return fmt.Errorf("failed to create Job: %w", err)
	}

	if !cfg.skipCleanup {
		defer func() {
			cmd := exec.Command("kubectl", "delete", "job", jobName,
				"-n", cfg.workloadNamespace, "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		}()
	}

	if err := waitForQualityBenchmarkJob(jobName); err != nil {
		return err
	}

	return nil
}
