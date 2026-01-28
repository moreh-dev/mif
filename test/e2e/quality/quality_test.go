//go:build e2e && !printenv
// +build e2e,!printenv

package quality

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	envs "github.com/moreh-dev/mif/test/e2e/envs"
	"github.com/moreh-dev/mif/test/utils"
	"github.com/moreh-dev/mif/test/utils/settings"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	commonTemplateName      string
	prefillMetaTemplateName string
	decodeMetaTemplateName  string
	decodeProxyTemplateName string
	prefillServiceName      string
	decodeServiceName       string
)

var _ = Describe("Quality Benchmark", Label("quality"), Ordered, func() {
	SetDefaultEventuallyTimeout(settings.TimeoutShort)
	SetDefaultEventuallyPollingInterval(settings.IntervalShort)

	BeforeAll(func() {
		By("creating workload namespace")
		Expect(utils.CreateWorkloadNamespace(envs.WorkloadNamespace, envs.MIFNamespace, envs.IstioRev)).To(Succeed())

		By("creating Gateway resources")
		Expect(utils.CreateGatewayResource(envs.WorkloadNamespace, envs.GatewayClassName)).To(Succeed())

		By("installing Heimdall")
		Expect(utils.InstallHeimdall(envs.WorkloadNamespace, envs.GatewayClassName)).To(Succeed())

		By("creating InferenceServiceTemplates")
		var err error
		isKind := !envs.SkipKind
		inferenceServiceData := utils.GetInferenceServiceData(envs.WorkloadNamespace, envs.TestModel, envs.HFToken, envs.HFEndpoint, isKind)
		commonTemplateName, err = utils.CreateInferenceServiceTemplate(envs.WorkloadNamespace, settings.InferenceServiceTemplateCommon, inferenceServiceData)
		Expect(err).NotTo(HaveOccurred(), "failed to create common InferenceServiceTemplate")
		prefillMetaTemplateName, err = utils.CreateInferenceServiceTemplate(envs.WorkloadNamespace, settings.InferenceServiceTemplatePrefillMeta, inferenceServiceData)
		Expect(err).NotTo(HaveOccurred(), "failed to create prefill meta InferenceServiceTemplate")
		decodeMetaTemplateName, err = utils.CreateInferenceServiceTemplate(envs.WorkloadNamespace, settings.InferenceServiceTemplateDecodeMeta, inferenceServiceData)
		Expect(err).NotTo(HaveOccurred(), "failed to create decode meta InferenceServiceTemplate")
		decodeProxyTemplateName, err = utils.CreateInferenceServiceTemplate(envs.WorkloadNamespace, settings.InferenceServiceTemplateDecodeProxy, inferenceServiceData)
		Expect(err).NotTo(HaveOccurred(), "failed to create decode proxy InferenceServiceTemplate")

		By("creating InferenceServices")
		prefillServiceName, err = utils.CreateInferenceService(envs.WorkloadNamespace, settings.InferenceServicePrefill, inferenceServiceData)
		Expect(err).NotTo(HaveOccurred(), "failed to create prefill InferenceService")
		decodeServiceName, err = utils.CreateInferenceService(envs.WorkloadNamespace, settings.InferenceServiceDecode, inferenceServiceData)
		Expect(err).NotTo(HaveOccurred(), "failed to create decode InferenceService")

		By("waiting for prefill InferenceService to be ready")
		Expect(waitForInferenceService(envs.WorkloadNamespace, prefillServiceName)).To(Succeed())

		By("waiting for decode InferenceService to be ready")
		Expect(waitForInferenceService(envs.WorkloadNamespace, decodeServiceName)).To(Succeed())
	})

	AfterAll(func() {
		if envs.SkipCleanup {
			return
		}

		By("deleting InferenceServices")
		utils.DeleteInferenceService(envs.WorkloadNamespace, prefillServiceName)
		utils.DeleteInferenceService(envs.WorkloadNamespace, decodeServiceName)

		By("deleting InferenceServiceTemplates")
		utils.DeleteInferenceServiceTemplate(envs.WorkloadNamespace, commonTemplateName)
		utils.DeleteInferenceServiceTemplate(envs.WorkloadNamespace, prefillMetaTemplateName)
		utils.DeleteInferenceServiceTemplate(envs.WorkloadNamespace, decodeMetaTemplateName)
		utils.DeleteInferenceServiceTemplate(envs.WorkloadNamespace, decodeProxyTemplateName)

		By("deleting Heimdall")
		utils.UninstallHeimdall(envs.WorkloadNamespace)

		By("deleting Gateway resources")
		utils.DeleteGatewayResource(envs.WorkloadNamespace, envs.GatewayClassName)

		By("deleting workload namespace")
		utils.DeleteNamespace(envs.WorkloadNamespace)
	})

	It("should run quality benchmarks", func() {
		By("getting Gateway service name")
		serviceName, err := utils.GetGatewayServiceName(envs.WorkloadNamespace)
		Expect(err).NotTo(HaveOccurred(), "failed to get Gateway service name")

		By("creating quality benchmark job")
		jobName, err := createQualityBenchmarkJob(envs.WorkloadNamespace, serviceName)
		Expect(err).NotTo(HaveOccurred(), "failed to create quality benchmark job")
		defer deleteQualityBenchmarkJob(envs.WorkloadNamespace, jobName)

		By("waiting for quality benchmark job to complete")
		Expect(waitForQualityBenchmarkJob(envs.WorkloadNamespace, jobName)).To(Succeed())

		By("getting quality benchmark job logs")
		logs, err := getQualityBenchmarkJobLogs(envs.WorkloadNamespace, jobName)
		Expect(err).NotTo(HaveOccurred(), "failed to get job logs")

		isKind := !envs.SkipKind
		if isKind {
			By("skipping quality benchmark results validation (kind cluster)")
			return
		}

		By("validating quality benchmark results")
		Expect(validateQualityBenchmarkResults(envs.QualityBenchmarks, logs)).To(Succeed())
	})
})

func waitForInferenceService(namespace string, name string) error {
	cmd := exec.Command("kubectl", "wait", "inferenceService", name,
		"--for=condition=Ready",
		"-n", namespace,
		fmt.Sprintf("--timeout=%v", settings.TimeoutVeryLong))
	_, err := utils.Run(cmd)
	return err
}

func createQualityBenchmarkJob(namespace string, serviceName string) (string, error) {
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
		Namespace:             namespace,
		ModelName:             envs.TestModel,
		GatewayHost:           serviceName,
		GatewayPort:           "80",
		HFToken:               envs.HFToken,
		HFEndpoint:            envs.HFEndpoint,
		Benchmarks:            envs.QualityBenchmarks,
		Limit:                 envs.QualityBenchmarkLimit,
		ImagePullSecret:       settings.MorehRegistrySecretName,
		QualityBenchmarkImage: settings.QualityBenchmarkImage,
	}

	jobYAML, err := utils.RenderTemplate(settings.QualityBenchmarkJobTemplate, data)
	if err != nil {
		return "", fmt.Errorf("failed to render job template: %w", err)
	}

	cmd := exec.Command("kubectl", "create", "-f", "-", "-n", namespace, "-o", "name")
	cmd.Stdin = strings.NewReader(jobYAML)
	output, err := utils.Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}

	jobName := strings.TrimPrefix(strings.TrimSpace(output), "job.batch/")
	return jobName, nil
}

func deleteQualityBenchmarkJob(namespace string, jobName string) {
	cmd := exec.Command("kubectl", "delete", "job", jobName,
		"-n", namespace, "--ignore-not-found=true")
	_, _ = utils.Run(cmd)
}

func waitForQualityBenchmarkJob(namespace string, jobName string) error {
	cmd := exec.Command("kubectl", "wait", "job", jobName,
		"--for=condition=complete",
		"-n", namespace,
		fmt.Sprintf("--timeout=%v", settings.TimeoutVeryLong))
	_, err := utils.Run(cmd)
	if err != nil {
		logCmd := exec.Command("kubectl", "logs", "-l", "app=quality-benchmark",
			"-n", namespace,
			"--tail=100")
		_, _ = utils.Run(logCmd)
		return fmt.Errorf("quality benchmark job did not complete within timeout: %w", err)
	}

	return nil
}

func getQualityBenchmarkJobLogs(namespace string, jobName string) (string, error) {
	logCmd := exec.Command("kubectl", "logs", "-l", fmt.Sprintf("job-name=%s", jobName),
		"-n", namespace, "--tail=100")
	logs, err := utils.Run(logCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get job logs: %w", err)
	}
	return logs, nil
}

func extractMMLUScore(logs string) (float64, error) {
	inGroups := false
	for _, line := range strings.Split(logs, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if inGroups {
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

func validateQualityBenchmarkResults(benchmark string, logs string) error {
	switch benchmark {
	case "mmlu":
		return validateMMLUResults(logs)
	default:
		if logs == "" {
			return fmt.Errorf("no logs found for benchmark %q", benchmark)
		}
		return nil
	}
}

func validateMMLUResults(logs string) error {
	requiredHeaders := []string{"Groups", "Version", "Metric"}
	for _, header := range requiredHeaders {
		if !strings.Contains(logs, header) {
			return fmt.Errorf("expected table header %q not found in logs", header)
		}
	}

	if !strings.Contains(logs, "|mmlu") {
		return fmt.Errorf("MMLU result row not found in logs")
	}

	score, err := extractMMLUScore(logs)
	if err != nil {
		return fmt.Errorf("failed to extract MMLU score: %w", err)
	}

	if score < settings.MinMMLUScore {
		return fmt.Errorf("MMLU score %.4f is below minimum threshold %.2f (expected >= %.2f)", score, settings.MinMMLUScore, settings.MinMMLUScore)
	}

	return nil
}
