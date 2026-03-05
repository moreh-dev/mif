//go:build e2e && !printenv
// +build e2e,!printenv

package smoke

import (
	"fmt"
	"os/exec"
	"strings"

	envs "github.com/moreh-dev/mif/test/e2e/envs"
	"github.com/moreh-dev/mif/test/utils"
	"github.com/moreh-dev/mif/test/utils/settings"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const pcHeimdallVersion = "v0.7.1"

const pcHeimdallValuesYAML = `
global:
  imagePullSecrets:
    - name: moreh-registry

config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    - type: single-profile-handler
    - type: precise-prefix-cache-scorer
      parameters:
        indexerConfig:
          prefixStoreConfig:
            blockSize: 16
          tokenProcessorConfig:
            blockSize: 16
            hashSeed: "42"
          kvBlockIndexConfig:
            enableMetrics: true
          tokenizersPoolConfig:
            modelName: meta-llama/Llama-3.2-1B-Instruct
            hf:
              tokenizersCacheDir: "/mnt/models"
        kvEventsConfig:
          zmqEndpoint: "tcp://*:5557"
          topicFilter: "kv@"
    - type: max-score-picker
  schedulingProfiles:
    - name: default
      plugins:
        - pluginRef: precise-prefix-cache-scorer
          weight: 10
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

extraEnvVars:
  - name: PYTHONHASHSEED
    value: "42"
  - name: HF_HOME
    value: /mnt/models
  - name: HF_HUB_OFFLINE
    value: "1"

extraVolumes:
  - name: models
    persistentVolumeClaim:
      claimName: models

extraVolumeMounts:
  - name: models
    mountPath: /mnt/models
`

const pcInferenceServiceYAML = `
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: prefix-cache-test
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
          env:
            - name: ISVC_USE_KV_EVENTS
              value: "true"
            - name: ISVC_EXTRA_ARGS
              value: >-
                --block-size 16
                --max-num-seqs 128
            - name: PYTHONHASHSEED
              value: "42"
          resources:
            requests:
              mellanox/hca: "1"
            limits:
              mellanox/hca: "1"
`

const pcCurlJobYAML = `
apiVersion: batch/v1
kind: Job
metadata:
  generateName: pc-curl-
  namespace: {{ .Namespace }}
spec:
  backoffLimit: 0
  template:
    metadata:
      labels:
        sidecar.istio.io/inject: "false"
    spec:
      restartPolicy: Never
      containers:
      - name: curl
        image: curlimages/curl:8.12.1
        command:
        - /bin/sh
        - -c
        args:
        - |
          BODY='{"model":"meta-llama/Llama-3.2-1B-Instruct","prompt":"You are an expert assistant specializing in computer science and artificial intelligence. Provide detailed and technically accurate responses to all questions. Explain the evolution of transformer architectures from the original Attention Is All You Need paper to modern large language models.","max_tokens":32}'
          METRICS_URL="{{ .MetricsURL }}"

          # First request to populate KV cache on one of the pods
          echo "Sending first request to populate KV cache..."
          curl -sf --max-time 60 -X POST {{ .BaseURL }} \
            -H "Content-Type: application/json" \
            -d "$BODY" \
            -o /dev/null

          echo "First request completed"

          # Wait for ZMQ KV cache events to propagate to EPP
          sleep 5

          # Capture baseline lookup hits from Heimdall metrics
          HITS_BEFORE=$(curl -sf --max-time 10 "$METRICS_URL" \
            | grep "^kvcache_index_lookup_hits_total " \
            | awk '{print int($2)}')
          HITS_BEFORE=${HITS_BEFORE:-0}
          echo "Lookup hits before second request: $HITS_BEFORE"

          # Second request with the same prompt triggers a prefix cache lookup
          echo "Sending second request with same prompt..."
          curl -sf --max-time 60 -X POST {{ .BaseURL }} \
            -H "Content-Type: application/json" \
            -d "$BODY" \
            -o /dev/null

          echo "Second request completed"
          sleep 2

          # Check that lookup hits increased
          HITS_AFTER=$(curl -sf --max-time 10 "$METRICS_URL" \
            | grep "^kvcache_index_lookup_hits_total " \
            | awk '{print int($2)}')
          HITS_AFTER=${HITS_AFTER:-0}
          echo "Lookup hits after second request: $HITS_AFTER"

          # Check admissions to verify ZMQ event flow
          ADMISSIONS=$(curl -sf --max-time 10 "$METRICS_URL" \
            | grep "^kvcache_index_admissions_total " \
            | awk '{print int($2)}')
          ADMISSIONS=${ADMISSIONS:-0}
          echo "Admissions after second request: $ADMISSIONS"

          if [ "$HITS_AFTER" -gt "$HITS_BEFORE" ]; then
            echo "SUCCESS: kvcache_index_lookup_hits_total increased ($HITS_BEFORE -> $HITS_AFTER)"
            exit 0
          else
            echo "FAILURE: kvcache_index_lookup_hits_total did not increase ($HITS_BEFORE -> $HITS_AFTER)"
            echo "Admissions total: $ADMISSIONS (0 means sim is not sending ZMQ events)"
            echo "Dumping all kvcache metrics for debugging:"
            curl -sf --max-time 10 "$METRICS_URL" | grep "^kvcache_" || echo "(no kvcache metrics found)"
            exit 1
          fi
`

var (
	pcServiceName string

	pvName  string
	pvcName string
)

var _ = Describe("Prefix Cache Smoke", Label("smoke"), Ordered, func() {
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

		By("installing Heimdall with precise-prefix-cache configuration")
		data := struct {
			GatewayClassName string
			IstioRev         string
		}{
			GatewayClassName: envs.GatewayClassName,
			IstioRev:         envs.IstioRev,
		}

		values, err := utils.RenderTemplate(pcHeimdallValuesYAML, data)
		Expect(err).NotTo(HaveOccurred(), "failed to render Heimdall values template")
		Expect(utils.InstallHeimdall(envs.WorkloadNamespace, pcHeimdallVersion, values)).To(Succeed())

		By("creating InferenceService with KV cache enabled")
		svcData := utils.InferenceServiceData{
			Namespace: envs.WorkloadNamespace,
		}
		pcServiceName, err = utils.CreateInferenceService(envs.WorkloadNamespace, pcInferenceServiceYAML, svcData)
		Expect(err).NotTo(HaveOccurred(), "failed to create InferenceService")

		By("waiting for InferenceService to be ready")
		Expect(waitForInferenceService(envs.WorkloadNamespace, pcServiceName)).To(Succeed())
	})

	AfterAll(func() {
		if envs.SkipCleanup {
			return
		}

		By("deleting InferenceService")
		utils.DeleteInferenceService(envs.WorkloadNamespace, pcServiceName)

		By("deleting Heimdall")
		utils.UninstallHeimdall(envs.WorkloadNamespace)

		By("deleting Gateway resources")
		utils.DeleteGatewayResource(envs.WorkloadNamespace, envs.GatewayClassName)

		By("deleting workload namespace")
		utils.DeleteNamespace(envs.WorkloadNamespace)
	})

	It("should increase kvcache lookup hits when same prompt is sent repeatedly", func() {
		By("getting Gateway service name")
		gwServiceName, err := utils.GetGatewayServiceName(envs.WorkloadNamespace)
		Expect(err).NotTo(HaveOccurred(), "failed to get Gateway service name")

		By("creating curl job to verify prefix cache lookup hits")
		metricsURL := fmt.Sprintf("http://heimdall.%s.svc.cluster.local:9090/metrics", envs.WorkloadNamespace)
		jobName, err := createPCCurlJob(envs.WorkloadNamespace, fmt.Sprintf("http://%s/v1/completions", gwServiceName), metricsURL)
		Expect(err).NotTo(HaveOccurred(), "failed to create curl job")
		defer deletePCCurlJob(envs.WorkloadNamespace, jobName)

		By("waiting for curl job to complete")
		Expect(waitForPCCurlJob(envs.WorkloadNamespace, jobName)).To(Succeed())
	})
})

func createPCCurlJob(namespace string, baseURL string, metricsURL string) (string, error) {
	type jobTemplateData struct {
		Namespace  string
		BaseURL    string
		MetricsURL string
	}

	data := jobTemplateData{
		Namespace:  namespace,
		BaseURL:    baseURL,
		MetricsURL: metricsURL,
	}

	jobYAML, err := utils.RenderTemplate(pcCurlJobYAML, data)
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

func deletePCCurlJob(namespace string, jobName string) {
	cmd := exec.Command("kubectl", "delete", "job", jobName,
		"-n", namespace, "--ignore-not-found=true")
	_, _ = utils.Run(cmd)
}

func waitForPCCurlJob(namespace string, jobName string) error {
	type result struct {
		condition string
		err       error
	}

	ch := make(chan result, 2)
	for _, cond := range []string{"complete", "failed"} {
		go func(c string) {
			cmd := exec.Command("kubectl", "wait", "job", jobName,
				"--for=condition="+c,
				"-n", namespace,
				fmt.Sprintf("--timeout=%v", settings.TimeoutVeryLong))
			_, err := utils.Run(cmd)
			ch <- result{condition: c, err: err}
		}(cond)
	}

	res := <-ch
	if res.condition == "failed" && res.err == nil {
		return fmt.Errorf("prefix cache curl job failed")
	}
	if res.err != nil {
		return fmt.Errorf("prefix cache curl job did not complete within timeout: %w", res.err)
	}
	return nil
}
