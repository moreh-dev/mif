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
	heimdallVersion    = "v0.7.1"
	MMLUScoreThreshold = 0.37
)

const heimdallValuesYAML = `
global:
  imagePullSecrets:
    - name: moreh-registry

config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    - type: single-profile-handler
    - type: queue-scorer
    - type: max-score-picker
  schedulingProfiles:
    - name: default
      plugins:
        - pluginRef: queue-scorer
        - pluginRef: max-score-picker

gateway:
  name: mif
  gatewayClassName: {{ .GatewayClassName }}
  {{- if .IstioRev }}
  labels:
    istio.io/rev: {{ .IstioRev }}
  {{- end }}

inferencePool:
  targetPorts:
    - number: 8000
`

const inferenceServiceYAML = `
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: vllm
  namespace: {{ .Namespace }}
spec:
  replicas: 2
  inferencePoolRefs:
    - name: heimdall
  templateRefs:
    - name: vllm
    - name: quickstart-vllm-meta-llama-llama-3.2-1b-instruct-amd-mi250-tp2
    - name: vllm-hf-hub-offline
  template:
    spec:
      containers:
        - name: main
          resources:
            requests:
              mellanox/hca: "1"
            limits:
              mellanox/hca: "1"
`

const qualityBenchmarkJobYAML = `
apiVersion: batch/v1
kind: Job
metadata:
  generateName: quality-benchmark-
  labels:
    app: quality-benchmark
  namespace: {{ .Namespace }}
spec:
  template:
    metadata:
      labels:
        app: quality-benchmark
        sidecar.istio.io/inject: "false"
    spec:
      restartPolicy: Never
      imagePullSecrets:
        - name: moreh-registry
      containers:
      - name: quality-benchmark
        image: 255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/moreh-llm-eval:v0.0.1
        command: ["/bin/bash", "-c"]
        args:
          - |
            set -e
            echo "Setting up writable cache layer..."
            mkdir -p /tmp/hf_cache
            cp -as /mnt/models/. /tmp/hf_cache/
            
            exec /usr/local/bin/run \
              --eval "mmlu" \
              --model "meta-llama/Llama-3.2-1B-Instruct" \
              --host "{{.GatewayHost}}" \
              --port "80" \
        volumeMounts:
          - name: model
            mountPath: /mnt/models
          - name: writable-cache
            mountPath: /tmp/hf_cache
        env:
          - name: HF_HUB_OFFLINE
            value: "1"
          - name: HF_DATASETS_OFFLINE
            value: "1"
          - name: HF_HOME
            value: /tmp/hf_cache
      volumes:
        - name: model
          persistentVolumeClaim:
            claimName: models
        - name: writable-cache
          emptyDir: {}
`

var (
	serviceName string

	pvName  string
	pvcName string
)

var _ = Describe("Quality Benchmark", Label("quality"), Ordered, func() {
	SetDefaultEventuallyTimeout(settings.TimeoutShort)
	SetDefaultEventuallyPollingInterval(settings.IntervalShort)

	BeforeAll(func() {
		By("creating workload namespace")
		Expect(utils.CreateWorkloadNamespace(envs.WorkloadNamespace, envs.MIFNamespace)).To(Succeed())

		By("creating Gateway resources")
		Expect(utils.CreateGatewayResource(envs.WorkloadNamespace, envs.GatewayClassName, envs.IstioRev)).To(Succeed())

		By("installing Heimdall")
		data := struct {
			GatewayClassName string
			IstioRev         string
		}{
			GatewayClassName: envs.GatewayClassName,
			IstioRev:         envs.IstioRev,
		}

		values, err := utils.RenderTemplate(heimdallValuesYAML, data)
		Expect(err).NotTo(HaveOccurred(), "failed to render Heimdall values template")
		Expect(utils.InstallHeimdall(envs.WorkloadNamespace, heimdallVersion, values)).To(Succeed())

		By("creating model PV")
		pvName, err = utils.CreateModelPV(envs.WorkloadNamespace)
		Expect(err).NotTo(HaveOccurred(), "failed to create model PV")

		By("creating model PVC")
		pvcName, err = utils.CreateModelPVC(envs.WorkloadNamespace)
		Expect(err).NotTo(HaveOccurred(), "failed to create model PVC")

		By("creating InferenceServices")
		vllmData := utils.InferenceServiceData{
			Namespace: envs.WorkloadNamespace,
		}
		serviceName, err = utils.CreateInferenceService(envs.WorkloadNamespace, inferenceServiceYAML, vllmData)
		Expect(err).NotTo(HaveOccurred(), "failed to create vllm InferenceService")

		By("waiting for vllm InferenceService to be ready")
		Expect(waitForInferenceService(envs.WorkloadNamespace, serviceName)).To(Succeed())
	})

	AfterAll(func() {
		if envs.SkipCleanup {
			return
		}

		By("deleting InferenceServices")
		utils.DeleteInferenceService(envs.WorkloadNamespace, serviceName)

		By("deleting model PVC")
		utils.DeleteModelPVC(envs.WorkloadNamespace, pvcName)

		By("deleting model PV")
		utils.DeleteModelPV(pvName)

		By("deleting Heimdall")
		utils.UninstallHeimdall(envs.WorkloadNamespace)

		By("deleting Gateway resources")
		utils.DeleteGatewayResource(envs.WorkloadNamespace, envs.GatewayClassName)

		By("deleting workload namespace")
		utils.DeleteNamespace(envs.WorkloadNamespace)
	})

	It("should run quality benchmarks (mmlu)", func() {
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
		var logs string
		Eventually(func() bool {
			var err error
			logs, err = getQualityBenchmarkJobLogs(envs.WorkloadNamespace, jobName)
			if err != nil {
				return false
			}
			return checkQualityBenchmarkJobLogs("mmlu", logs) == nil
		}, settings.TimeoutShort, settings.IntervalShort).Should(BeTrue(), "failed to find job logs")

		By("validating quality benchmark results")
		Expect(validateQualityBenchmarkResults("mmlu", logs)).To(Succeed())
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
		Namespace   string
		GatewayHost string
	}

	data := jobTemplateData{
		Namespace:   namespace,
		GatewayHost: serviceName,
	}

	jobYAML, err := utils.RenderTemplate(qualityBenchmarkJobYAML, data)
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

	if !strings.Contains(logs, "|") || !strings.Contains(logs, "mmlu") {
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

	if score < MMLUScoreThreshold {
		return fmt.Errorf("MMLU score %.4f is below minimum threshold %.2f", score, MMLUScoreThreshold)
	}

	_, _ = fmt.Fprintf(GinkgoWriter, "MMLU score %.4f is above minimum threshold %.2f\n", score, MMLUScoreThreshold)

	return nil
}
