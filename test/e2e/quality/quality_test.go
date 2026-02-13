//go:build e2e && !printenv
// +build e2e,!printenv

package quality

import (
	"bufio"
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

const (
	HeimdallValues       = "test/e2e/quality/config/heimdall-values.yaml.tmpl"
	InferenceServicePath = "test/e2e/quality/config/inference-service.yaml.tmpl"
	QualityBenchmarkJob  = "test/e2e/quality/config/quality-benchmark-job.yaml.tmpl"

	QualityBenchmarkImage = "255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/moreh-llm-eval:v0.0.1"
	MinMMLUScore          = 0.37
)

var (
	vllmServiceName string

	pvName  string
	pvcName string
)

var _ = Describe("Quality Benchmark", Label("quality"), Ordered, func() {
	SetDefaultEventuallyTimeout(settings.TimeoutShort)
	SetDefaultEventuallyPollingInterval(settings.IntervalShort)

	BeforeAll(func() {
		isKind := !envs.SkipKind

		By("creating workload namespace")
		Expect(utils.CreateWorkloadNamespace(envs.WorkloadNamespace, envs.MIFNamespace)).To(Succeed())

		By("creating Gateway resources")
		Expect(utils.CreateGatewayResource(envs.WorkloadNamespace, envs.GatewayClassName, envs.IstioRev)).To(Succeed())

		By("installing Heimdall")
		data := struct {
			MorehRegistrySecretName string
			GatewayName             string
			GatewayClass            string
			IstioRev                string
		}{
			MorehRegistrySecretName: settings.MorehRegistrySecretName,
			GatewayName:             settings.GatewayName,
			GatewayClass:            envs.GatewayClassName,
			IstioRev:                envs.IstioRev,
		}

		values, err := utils.RenderTemplate(HeimdallValues, data)
		Expect(err).NotTo(HaveOccurred(), "failed to render Heimdall values template")
		Expect(utils.InstallHeimdall(envs.WorkloadNamespace, values)).To(Succeed())

		if envs.SkipKind {
			By("creating model PV")
			pvName, err = utils.CreateModelPV(envs.WorkloadNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create model PV")

			By("creating model PVC")
			pvcName, err = utils.CreateModelPVC(envs.WorkloadNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create model PVC")
		}

		By("creating InferenceServices")
		// PD disaggregation environment cannot run tests normally, so we test in aggregate environment
		var vllmData utils.InferenceServiceData
		if isKind {
			vllmData = utils.InferenceServiceData{
				Name:         "vllm",
				Namespace:    envs.WorkloadNamespace,
				Replicas:     2,
				TemplateRefs: []string{"sim"},
				HFToken:      envs.HFToken,
				HFEndpoint:   envs.HFEndpoint,
				IsKind:       isKind,
			}
		} else {
			vllmData = utils.InferenceServiceData{
				Name:         "vllm",
				Namespace:    envs.WorkloadNamespace,
				Replicas:     2,
				TemplateRefs: []string{"vllm", envs.TestTemplateDecode, "vllm-hf-hub-offline"},
				HFToken:      envs.HFToken,
				HFEndpoint:   envs.HFEndpoint,
				IsKind:       isKind,
			}
		}
		vllmServiceName, err = utils.CreateInferenceService(envs.WorkloadNamespace, InferenceServicePath, vllmData)
		Expect(err).NotTo(HaveOccurred(), "failed to create vllm InferenceService")

		By("waiting for vllm InferenceService to be ready")
		Expect(waitForInferenceService(envs.WorkloadNamespace, vllmServiceName)).To(Succeed())
	})

	AfterAll(func() {
		if envs.SkipCleanup {
			return
		}

		By("deleting InferenceServices")
		utils.DeleteInferenceService(envs.WorkloadNamespace, vllmServiceName)

		if envs.SkipKind {
			By("deleting model PVC")
			utils.DeleteModelPVC(envs.WorkloadNamespace, pvcName)

			By("deleting model PV")
			utils.DeleteModelPV(pvName)
		}

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
		jobName, err := createQualityBenchmarkJob(envs.WorkloadNamespace, serviceName, pvcName)
		Expect(err).NotTo(HaveOccurred(), "failed to create quality benchmark job")
		defer deleteQualityBenchmarkJob(envs.WorkloadNamespace, jobName)

		By("waiting for quality benchmark job to complete")
		Expect(waitForQualityBenchmarkJob(envs.WorkloadNamespace, jobName)).To(Succeed())

		By("getting quality benchmark job logs")
		var logs string
		Eventually(func() bool {
			var err error
			logs, err = getQualityBenchmarkJobLogs(envs.WorkloadNamespace, jobName)
			if err != nil {
				return false
			}
			return checkQualityBenchmarkJobLogs(envs.QualityBenchmarks, logs) == nil
		}, settings.TimeoutShort, settings.IntervalShort).Should(BeTrue(), "failed to find job logs")

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

func createQualityBenchmarkJob(namespace string, serviceName string, pvcName string) (string, error) {
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
		IsKind                bool
		QualityBenchmarkImage string
		PVCName               string
	}

	isKind := !envs.SkipKind
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
		IsKind:                isKind,
		QualityBenchmarkImage: QualityBenchmarkImage,
		PVCName:               pvcName,
	}

	jobYAML, err := utils.RenderTemplate(QualityBenchmarkJob, data)
	if err != nil {
		return "", fmt.Errorf("failed to render job template: %w", err)
	}

	cmd := exec.Command("kubectl", "create", "-f", "-", "-n", namespace, "-o", "name")
	cmd.Stdin = strings.NewReader(jobYAML)
	output, err := utils.Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}
	return utils.ParseResourceName(output), nil
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
		fmt.Sprintf("--timeout=%v", settings.Timeout30Min))
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
		"-n", namespace, "--tail=20")
	logs, err := utils.Run(logCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get job logs: %w", err)
	}
	return logs, nil
}

func checkQualityBenchmarkJobLogs(benchmark string, logs string) error {
	switch benchmark {
	case "mmlu":
		return checkMMLULogs(logs)
	default:
		return nil
	}
}

func checkMMLULogs(logs string) error {
	requiredHeaders := []string{"Groups", "Version", "Metric"}
	for _, header := range requiredHeaders {
		if !strings.Contains(logs, header) {
			return fmt.Errorf("expected table header %q not found in logs", header)
		}
	}

	if !(strings.Contains(logs, "|") && strings.Contains(logs, "mmlu")) {
		return fmt.Errorf("MMLU result row not found in logs")
	}

	return nil
}

// extractMMLUScore extracts the MMLU score from the logs.
// MMLU score table example:
// ```
// |      Groups      |Version|Filter|n-shot|Metric|   |Value |   |Stderr|
// |------------------|------:|------|------|------|---|-----:|---|-----:|
// |mmlu              |      2|none  |      |acc   |↑  |0.2295|±  |0.0035|
// | - humanities     |      2|none  |      |acc   |↑  |0.2421|±  |0.0062|
// | - other          |      2|none  |      |acc   |↑  |0.2398|±  |0.0076|
// | - social sciences|      2|none  |      |acc   |↑  |0.2171|±  |0.0074|
// | - stem           |      2|none  |      |acc   |↑  |0.2125|±  |0.0073|
// ```
// The MMLU score is in the 8th column (index 7) of the "mmlu" row.
func extractMMLUScore(logs string) (float64, error) {
	scanner := bufio.NewScanner(strings.NewReader(logs))
	inGroups := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			if inGroups {
				break
			}
			continue
		}

		if strings.HasPrefix(line, "|") && strings.Contains(line, "Groups") {
			inGroups = true
			continue
		}

		if inGroups && strings.Contains(line, "|") && strings.Contains(line, "mmlu") {
			parts := strings.Split(line, "|")
			if len(parts) >= 8 {
				value := strings.TrimSpace(parts[7])
				score, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse MMLU score %q: %w", value, err)
				}
				return score, nil
			}
		}
	}

	return 0, fmt.Errorf("MMLU score not found in summary table logs. Expected Groups summary table with |mmlu row:\n%s", logs)
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
	score, err := extractMMLUScore(logs)
	if err != nil {
		return fmt.Errorf("failed to extract MMLU score: %w", err)
	}

	if score < MinMMLUScore {
		return fmt.Errorf("MMLU score %.4f is below minimum threshold %.2f", score, MinMMLUScore)
	}

	_, _ = fmt.Fprintf(GinkgoWriter, "MMLU score %.4f is above minimum threshold %.2f\n", score, MinMMLUScore)

	return nil
}
