//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	// minMMLUScore is the minimum acceptable MMLU score threshold
	minMMLUScore = 0.37
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
	err := runQualityBenchmarkJob(serviceName, cfg.testModel, cfg.qualityBenchmarks, cfg.qualityBenchmarkLimit)
	Expect(err).NotTo(HaveOccurred(), "quality benchmark job should complete successfully")
}

func createQualityBenchmarkJob(serviceName string, modelName string, benchmarks string, limit string) (string, error) {
	type jobTemplateData struct {
		Namespace             string
		ModelName             string
		GatewayHost           string
		GatewayPort           string
		HFToken               string
		HFEndpoint            string
		Benchmarks            string
		Limit                 string
		ImagePullSecret       string
		QualityBenchmarkImage string
	}

	data := jobTemplateData{
		Namespace:             cfg.workloadNamespace,
		ModelName:             modelName,
		GatewayHost:           serviceName,
		GatewayPort:           "80",
		HFToken:               cfg.hfToken,
		HFEndpoint:            cfg.hfEndpoint,
		Benchmarks:            benchmarks,
		Limit:                 limit,
		ImagePullSecret:       secretNameMorehRegistry,
		QualityBenchmarkImage: cfg.qualityBenchmarkImage,
	}

	jobYAML, err := renderTemplateFile("quality-benchmark-job.yaml.tmpl", data)
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

func runQualityBenchmarkJob(serviceName string, modelName string, benchmarks string, limit string) error {
	By("creating quality benchmark Job")
	jobName, err := createQualityBenchmarkJob(serviceName, modelName, benchmarks, limit)
	if err != nil {
		return fmt.Errorf("failed to create Job: %w", err)
	}

	if !cfg.skipCleanup {
		defer func() {
			cmd := exec.Command("kubectl", "delete", "job", jobName,
				"-n", cfg.workloadNamespace, "--ignore-not-found=true")
			_, _ = Run(cmd)
		}()
	}

	if err := waitForQualityBenchmarkJob(jobName); err != nil {
		return err
	}

	// Collect and validate results
	logs, err := getQualityBenchmarkJobLogs(jobName)
	if err != nil {
		return fmt.Errorf("failed to get job logs: %w", err)
	}

	// Print logs for debugging
	_, _ = fmt.Fprintf(GinkgoWriter, "Quality benchmark job logs:\n%s\n", logs)

	// Validate benchmark results
	if err := validateQualityBenchmarkResults(logs, benchmarks); err != nil {
		return fmt.Errorf("quality benchmark validation failed: %w", err)
	}

	return nil
}

func waitForQualityBenchmarkJob(jobName string) error {
	By("waiting for quality benchmark job to complete")
	cmd := exec.Command("kubectl", "wait", "job", jobName,
		"--for=condition=complete",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", timeoutVeryLong))
	_, err := Run(cmd)
	if err != nil {
		By("collecting logs from failed quality benchmark job")
		logCmd := exec.Command("kubectl", "logs", "-l", "app=quality-benchmark",
			"-n", cfg.workloadNamespace,
			"--tail=100")
		if logs, logErr := Run(logCmd); logErr == nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "Quality benchmark job logs:\n%s\n", logs)
		}
		return fmt.Errorf("quality benchmark job did not complete within timeout: %w", err)
	}

	return nil
}

func getQualityBenchmarkJobLogs(jobName string) (string, error) {
	By("collecting logs from quality benchmark job")
	// Use job-name label selector to get logs from pods created by this specific job
	logCmd := exec.Command("kubectl", "logs", "-l", fmt.Sprintf("job-name=%s", jobName),
		"-n", cfg.workloadNamespace, "--tail=100")
	logs, err := Run(logCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get job logs: %w", err)
	}
	return logs, nil
}

func extractMMLUScore(logs string) (float64, error) {
	// Only parse the summary table that starts with the Groups header.
	inGroups := false
	for _, line := range strings.Split(logs, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if inGroups {
				// End of summary block
				break
			}
			continue
		}

		if strings.HasPrefix(trimmed, "|") && strings.Contains(trimmed, "Groups") {
			inGroups = true
			continue
		}

		if !inGroups {
			continue
		}

		if !(strings.HasPrefix(trimmed, "|") && strings.Contains(trimmed, "mmlu")) {
			continue
		}
		parts := strings.Split(trimmed, "|")
		// Expected columns:
		// |mmlu              |      2|none  |      |acc   |↑  |0.2295|±  |0.0035|
		if len(parts) < 9 {
			continue
		}
		value := strings.TrimSpace(parts[7])
		score, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse MMLU score %q: %w", value, err)
		}
		return score, nil
	}

	return 0, fmt.Errorf("MMLU score not found in summary table logs. Expected Groups summary table with |mmlu| row")

}

func validateQualityBenchmarkResults(logs string, benchmark string) error {
	By("validating quality benchmark results")

	switch benchmark {
	case "mmlu":
		return validateMMLUResults(logs)
	default:
		// For other benchmarks, just verify logs exist
		if logs == "" {
			return fmt.Errorf("no logs found for benchmark %q", benchmark)
		}
		return nil
	}
}

func validateMMLUResults(logs string) error {
	// Check if the expected table header is present
	requiredHeaders := []string{"Groups", "Version", "Metric"}
	for _, header := range requiredHeaders {
		if !strings.Contains(logs, header) {
			return fmt.Errorf("expected table header %q not found in logs", header)
		}
	}

	// Check if mmlu row exists
	if !strings.Contains(logs, "|mmlu") {
		return fmt.Errorf("MMLU result row not found in logs")
	}

	// Extract MMLU score
	score, err := extractMMLUScore(logs)
	if err != nil {
		return fmt.Errorf("failed to extract MMLU score: %w", err)
	}

	// Skip threshold validation for kind clusters (no GPU support)
	if cfg.isUsingKindCluster {
		By(fmt.Sprintf("MMLU score %.4f extracted (threshold validation skipped for kind cluster)", score))
		return nil
	}

	// Validate score meets minimum threshold
	if score < minMMLUScore {
		return fmt.Errorf("MMLU score %.4f is below minimum threshold %.2f (expected >= %.2f)", score, minMMLUScore, minMMLUScore)
	}

	By(fmt.Sprintf("MMLU score %.4f meets minimum threshold %.2f", score, minMMLUScore))
	return nil
}
