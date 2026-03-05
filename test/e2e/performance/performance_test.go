//go:build e2e && !printenv
// +build e2e,!printenv

package performance

import (
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"time"

	envs "github.com/moreh-dev/mif/test/e2e/envs"
	"github.com/moreh-dev/mif/test/utils"
	"github.com/moreh-dev/mif/test/utils/settings"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const heimdallVersion = "v0.7.1"

const heimdallValuesYAML = `
global:
  imagePullSecrets:
    - name: moreh-registry

config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    - type: pd-profile-handler
    - type: prefill-filter
    - type: decode-filter
    - type: queue-scorer
    - type: max-score-picker
  schedulingProfiles:
    - name: prefill
      plugins:
        - pluginRef: prefill-filter
        - pluginRef: queue-scorer
        - pluginRef: max-score-picker
    - name: decode
      plugins:
        - pluginRef: decode-filter
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

extraVolumes:
  - name: models
    persistentVolumeClaim:
      claimName: models

extraVolumeMounts:
  - name: models
    mountPath: /mnt/models
`

const inferenceServicePrefillYAML = `
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: prefill
  namespace: {{ .Namespace }}
spec:
  replicas: 3
  inferencePoolRefs:
    - name: heimdall
  templateRefs:
    - name: vllm-prefill
    - name: quickstart-vllm-meta-llama-llama-3.2-1b-instruct-prefill-amd-mi250-tp2
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

const inferenceServiceDecodeYAML = `
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: decode
  namespace: {{ .Namespace }}
spec:
  replicas: 5
  inferencePoolRefs:
    - name: heimdall
  templateRefs:
    - name: vllm-decode
    - name: quickstart-vllm-meta-llama-llama-3.2-1b-instruct-decode-amd-mi250-tp2
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

const inferencePerfJobYAML = `
apiVersion: batch/v1
kind: Job
metadata:
  generateName: inference-perf-
  labels:
    app: inference-perf
  namespace: {{ .Namespace }}
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
                model_name: meta-llama/Llama-3.2-1B-Instruct
                base_url: {{ .BaseURL }}

              report:
                request_lifecycle:
                  summary: false
                  per_stage: true
                  per_request: false

              storage:
                local_storage:
                  path: reports
                {{- if and .S3AccessKeyID .S3SecretAccessKey }}
                simple_storage_service:
                  bucket_name: "moreh-benchmark"
                  path: {{ .S3Prefix }}
                  report_file_prefix: null
                {{- end }}
          EOF

          /workspace/.venv/bin/inference-perf \
            -c /tmp/config.yaml \
            --log-level INFO

          cat reports*/*.json
        env:
          {{- if and .S3AccessKeyID .S3SecretAccessKey }}
          - name: AWS_ACCESS_KEY_ID
            value: "{{ .S3AccessKeyID }}"
          - name: AWS_SECRET_ACCESS_KEY
            value: "{{ .S3SecretAccessKey }}"
          - name: AWS_DEFAULT_REGION
            value: "ap-northeast-2"
          {{- end }}
          - name: HF_HOME
            value: /mnt/models
          - name: HF_HUB_OFFLINE
            value: "1"
        volumeMounts:
          - name: models
            mountPath: /mnt/models
      volumes:
        - name: models
          persistentVolumeClaim:
            claimName: models
`

var (
	prefillServiceName string
	decodeServiceName  string

	pvName  string
	pvcName string
)

var _ = Describe("Inference Performance", Label("performance"), Ordered, func() {
	SetDefaultEventuallyTimeout(settings.TimeoutShort)
	SetDefaultEventuallyPollingInterval(settings.IntervalShort)

	BeforeAll(func() {
		By("creating workload namespace")
		Expect(utils.CreateWorkloadNamespace(envs.WorkloadNamespace, envs.MIFNamespace)).To(Succeed())

		By("creating Gateway resources")
		Expect(utils.CreateGatewayResource(envs.WorkloadNamespace, envs.GatewayClassName, envs.IstioRev)).To(Succeed())

		var err error
		By("creating model PV")
		pvName, err = utils.CreateModelPV(envs.WorkloadNamespace)
		Expect(err).NotTo(HaveOccurred(), "failed to create model PV")

		By("creating model PVC")
		pvcName, err = utils.CreateModelPVC(envs.WorkloadNamespace)
		Expect(err).NotTo(HaveOccurred(), "failed to create model PVC")

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

		By("creating InferenceServices")
		prefillData := utils.InferenceServiceData{
			Namespace: envs.WorkloadNamespace,
		}
		decodeData := utils.InferenceServiceData{
			Namespace: envs.WorkloadNamespace,
		}

		prefillServiceName, err = utils.CreateInferenceService(envs.WorkloadNamespace, inferenceServicePrefillYAML, prefillData)
		Expect(err).NotTo(HaveOccurred(), "failed to create prefill InferenceService")
		decodeServiceName, err = utils.CreateInferenceService(envs.WorkloadNamespace, inferenceServiceDecodeYAML, decodeData)
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

		By("deleting Heimdall")
		utils.UninstallHeimdall(envs.WorkloadNamespace)

		By("deleting model PVC")
		utils.DeleteModelPVC(envs.WorkloadNamespace, pvcName)

		By("deleting model PV")
		utils.DeleteModelPV(pvName)

		By("deleting Gateway resources")
		utils.DeleteGatewayResource(envs.WorkloadNamespace, envs.GatewayClassName)

		By("deleting workload namespace")
		utils.DeleteNamespace(envs.WorkloadNamespace)
	})

	It("should run inference-perf performance benchmark", func() {
		By("getting Gateway service name")
		serviceName, err := utils.GetGatewayServiceName(envs.WorkloadNamespace)
		Expect(err).NotTo(HaveOccurred(), "failed to get Gateway service name")

		By("getting InferenceService container image")
		image, err := utils.GetInferenceServiceContainerImage(envs.WorkloadNamespace, prefillServiceName)
		Expect(err).NotTo(HaveOccurred(), "failed to get InferenceService container image")

		By("creating inference-perf job")
		jobName, s3Prefix, err := createInferencePerfJob(envs.WorkloadNamespace, fmt.Sprintf("http://%s", serviceName), image)
		Expect(err).NotTo(HaveOccurred(), "failed to create inference-perf job")
		defer deleteInferencePerfJob(envs.WorkloadNamespace, jobName)

		By("waiting for inference-perf job to complete")
		Expect(waitForInferencePerfJob(envs.WorkloadNamespace, jobName)).To(Succeed())

		if envs.S3AccessKeyID != "" && envs.S3SecretAccessKey != "" {
			AddReportEntry(fmt.Sprintf("inference-perf results uploaded to s3 path: %s", s3Prefix))
		} else {
			AddReportEntry("inference-perf results are not uploaded to s3 because S3 credentials are not set")
		}
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

func createInferencePerfJob(namespace string, baseURL string, image string) (string, string, error) {
	_, imageTag, err := utils.ParseImage(image)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse image: %w", err)
	}

	base := "vllm"
	preset := "vllm-pd"
	expType := "performance"
	expName := "synthetic_random_i1024_o1024_c64"
	timestamp := time.Now().Format("20060102150405") + fmt.Sprintf("%04d", rand.Intn(10000))
	s3Prefix := fmt.Sprintf("%s/%s/%s/%s/%s/%s", base, imageTag, preset, expType, expName, timestamp)

	type jobTemplateData struct {
		Namespace         string
		BaseURL           string
		S3Prefix          string
		S3AccessKeyID     string
		S3SecretAccessKey string
	}

	data := jobTemplateData{
		Namespace:         namespace,
		BaseURL:           baseURL,
		S3Prefix:          s3Prefix,
		S3AccessKeyID:     envs.S3AccessKeyID,
		S3SecretAccessKey: envs.S3SecretAccessKey,
	}

	jobYAML, err := utils.RenderTemplate(inferencePerfJobYAML, data)
	if err != nil {
		return "", "", fmt.Errorf("failed to render job template: %w", err)
	}

	cmd := exec.Command("kubectl", "create", "-f", "-", "-n", namespace, "-o", "name")
	cmd.Stdin = strings.NewReader(jobYAML)
	output, err := utils.Run(cmd)
	if err != nil {
		return "", "", fmt.Errorf("failed to create job: %w", err)
	}
	return utils.ParseResourceName(output), s3Prefix, nil
}

func deleteInferencePerfJob(namespace string, jobName string) {
	cmd := exec.Command("kubectl", "delete", "job", jobName,
		"-n", namespace, "--ignore-not-found=true")
	_, _ = utils.Run(cmd)
}

func waitForInferencePerfJob(namespace string, jobName string) error {
	cmd := exec.Command("kubectl", "wait", "job", jobName,
		"--for=condition=complete",
		"-n", namespace,
		fmt.Sprintf("--timeout=%v", settings.TimeoutVeryLong))
	_, err := utils.Run(cmd)
	if err != nil {
		return fmt.Errorf("inference-perf job did not complete within timeout: %w", err)
	}

	return nil
}
