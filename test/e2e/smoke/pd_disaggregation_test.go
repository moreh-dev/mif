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

const pdHeimdallVersion = "v0.7.1"

const pdHeimdallValuesYAML = `
global:
  imagePullSecrets:
    - name: moreh-registry

config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    - type: pd-profile-handler
    - type: prefill-header-handler
    - type: response-header-handler
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
`

const pdPrefillServiceYAML = `
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: pd-prefill
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  inferencePoolRefs:
    - name: heimdall
  templateRefs:
    - name: sim-prefill
`

const pdDecodeServiceYAML = `
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: pd-decode
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  inferencePoolRefs:
    - name: heimdall
  templateRefs:
    - name: sim-decode
`

const pdCurlJobYAML = `
apiVersion: batch/v1
kind: Job
metadata:
  generateName: pd-curl-
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
          BODY='{"model":"meta-llama/Llama-3.2-1B-Instruct","messages":[{"role":"user","content":"Hello"}]}'

          HEADERS=$(curl -sf --max-time 60 -X POST {{ .BaseURL }} \
            -H "Content-Type: application/json" \
            -d "$BODY" \
            -o /dev/null -D -)

          echo "Response headers:"
          echo "$HEADERS"

          DECODER=$(echo "$HEADERS" | grep -i "x-decoder-host-port:" | sed 's/[^:]*: //' | tr -d '\r\n')
          PREFILLER=$(echo "$HEADERS" | grep -i "x-prefiller-host-port:" | sed 's/[^:]*: //' | tr -d '\r\n')

          echo "Decoder endpoint:  $DECODER"
          echo "Prefiller endpoint: $PREFILLER"

          if [ -z "$DECODER" ]; then
            echo "FAILURE: x-decoder-host-port header not found"
            exit 1
          fi

          if [ -z "$PREFILLER" ]; then
            echo "FAILURE: x-prefiller-host-port header not found (PD not active)"
            exit 1
          fi

          if [ "$DECODER" != "$PREFILLER" ]; then
            echo "SUCCESS: Decoder and Prefiller handled by different pods"
            exit 0
          else
            echo "FAILURE: Decoder and Prefiller on same endpoint ($DECODER)"
            exit 1
          fi
`

var (
	pdPrefillName string
	pdDecodeName  string
)

var _ = Describe("PD Disaggregation Smoke", Label("smoke"), Ordered, func() {
	SetDefaultEventuallyTimeout(settings.TimeoutShort)
	SetDefaultEventuallyPollingInterval(settings.IntervalShort)

	BeforeAll(func() {
		By("creating workload namespace")
		Expect(utils.CreateWorkloadNamespace(envs.WorkloadNamespace, envs.MIFNamespace)).To(Succeed())

		By("creating Gateway resources")
		Expect(utils.CreateGatewayResource(envs.WorkloadNamespace, envs.GatewayClassName, envs.IstioRev)).To(Succeed())

		By("installing Heimdall with PD configuration")
		data := struct {
			GatewayClassName string
			IstioRev         string
		}{
			GatewayClassName: envs.GatewayClassName,
			IstioRev:         envs.IstioRev,
		}

		values, err := utils.RenderTemplate(pdHeimdallValuesYAML, data)
		Expect(err).NotTo(HaveOccurred(), "failed to render Heimdall values template")
		Expect(utils.InstallHeimdall(envs.WorkloadNamespace, pdHeimdallVersion, values)).To(Succeed())

		By("creating prefill InferenceService")
		svcData := utils.InferenceServiceData{Namespace: envs.WorkloadNamespace}
		pdPrefillName, err = utils.CreateInferenceService(envs.WorkloadNamespace, pdPrefillServiceYAML, svcData)
		Expect(err).NotTo(HaveOccurred(), "failed to create prefill InferenceService")

		By("creating decode InferenceService")
		pdDecodeName, err = utils.CreateInferenceService(envs.WorkloadNamespace, pdDecodeServiceYAML, svcData)
		Expect(err).NotTo(HaveOccurred(), "failed to create decode InferenceService")

		By("waiting for prefill InferenceService to be ready")
		Expect(waitForInferenceService(envs.WorkloadNamespace, pdPrefillName)).To(Succeed())

		By("waiting for decode InferenceService to be ready")
		Expect(waitForInferenceService(envs.WorkloadNamespace, pdDecodeName)).To(Succeed())
	})

	AfterAll(func() {
		if envs.SkipCleanup {
			return
		}

		By("deleting InferenceServices")
		utils.DeleteInferenceService(envs.WorkloadNamespace, pdPrefillName)
		utils.DeleteInferenceService(envs.WorkloadNamespace, pdDecodeName)

		By("deleting Heimdall")
		utils.UninstallHeimdall(envs.WorkloadNamespace)

		By("deleting Gateway resources")
		utils.DeleteGatewayResource(envs.WorkloadNamespace, envs.GatewayClassName)

		By("deleting workload namespace")
		utils.DeleteNamespace(envs.WorkloadNamespace)
	})

	It("should route requests through separate prefill and decode pods", func() {
		By("getting Gateway service name")
		gwServiceName, err := utils.GetGatewayServiceName(envs.WorkloadNamespace)
		Expect(err).NotTo(HaveOccurred(), "failed to get Gateway service name")

		By("creating curl job to verify PD routing")
		jobName, err := createPDCurlJob(envs.WorkloadNamespace, fmt.Sprintf("http://%s/v1/chat/completions", gwServiceName))
		Expect(err).NotTo(HaveOccurred(), "failed to create curl job")
		defer deletePDCurlJob(envs.WorkloadNamespace, jobName)

		By("waiting for curl job to complete")
		Expect(waitForPDCurlJob(envs.WorkloadNamespace, jobName)).To(Succeed())
	})
})

func createPDCurlJob(namespace string, baseURL string) (string, error) {
	type jobTemplateData struct {
		Namespace string
		BaseURL   string
	}

	data := jobTemplateData{
		Namespace: namespace,
		BaseURL:   baseURL,
	}

	jobYAML, err := utils.RenderTemplate(pdCurlJobYAML, data)
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

func deletePDCurlJob(namespace string, jobName string) {
	cmd := exec.Command("kubectl", "delete", "job", jobName,
		"-n", namespace, "--ignore-not-found=true")
	_, _ = utils.Run(cmd)
}

func waitForPDCurlJob(namespace string, jobName string) error {
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
		return fmt.Errorf("PD curl job failed")
	}
	if res.err != nil {
		return fmt.Errorf("PD curl job did not complete within timeout: %w", res.err)
	}
	return nil
}
