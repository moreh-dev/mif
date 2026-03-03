//go:build e2e && !printenv
// +build e2e,!printenv

package functional

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

const heimdallVersion = "v0.7.1"

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
  name: functional-test
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  inferencePoolRefs:
    - name: heimdall
  templateRefs:
    - name: sim
`

const curlJobYAML = `
apiVersion: batch/v1
kind: Job
metadata:
  generateName: functional-curl-
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
        - curl
        - -sf
        - --max-time
        - "30"
        - -X
        - POST
        - {{ .BaseURL }}
        - -H
        - "Content-Type: application/json"
        - -d
        - '{"model":"meta-llama/Llama-3.2-1B-Instruct","messages":[{"role":"user","content":"Hello"}]}'
`

var (
	serviceName string
)

var _ = Describe("InferenceService Lifecycle", Label("functional"), Ordered, func() {
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

		By("creating InferenceService")
		svcData := utils.InferenceServiceData{
			Namespace: envs.WorkloadNamespace,
		}
		serviceName, err = utils.CreateInferenceService(envs.WorkloadNamespace, inferenceServiceYAML, svcData)
		Expect(err).NotTo(HaveOccurred(), "failed to create InferenceService")

		By("waiting for InferenceService to be ready")
		Expect(waitForInferenceService(envs.WorkloadNamespace, serviceName)).To(Succeed())
	})

	AfterAll(func() {
		if envs.SkipCleanup {
			return
		}

		By("deleting InferenceService")
		utils.DeleteInferenceService(envs.WorkloadNamespace, serviceName)

		By("deleting Heimdall")
		utils.UninstallHeimdall(envs.WorkloadNamespace)

		By("deleting Gateway resources")
		utils.DeleteGatewayResource(envs.WorkloadNamespace, envs.GatewayClassName)

		By("deleting workload namespace")
		utils.DeleteNamespace(envs.WorkloadNamespace)
	})

	It("should create InferenceService and reach Ready state", func() {
		By("verifying InferenceService is in Ready condition")
		cmd := exec.Command("kubectl", "get", "inferenceservice", serviceName,
			"-n", envs.WorkloadNamespace,
			"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
		output, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "failed to get InferenceService status")
		Expect(output).To(Equal("True"), "InferenceService should be in Ready state")
	})

	It("should serve inference requests through Gateway", func() {
		By("getting Gateway service name")
		gwServiceName, err := utils.GetGatewayServiceName(envs.WorkloadNamespace)
		Expect(err).NotTo(HaveOccurred(), "failed to get Gateway service name")

		By("creating curl job")
		jobName, err := createCurlJob(envs.WorkloadNamespace, fmt.Sprintf("http://%s/v1/chat/completions", gwServiceName))
		Expect(err).NotTo(HaveOccurred(), "failed to create curl job")
		defer deleteCurlJob(envs.WorkloadNamespace, jobName)

		By("waiting for curl job to complete")
		Expect(waitForCurlJob(envs.WorkloadNamespace, jobName)).To(Succeed())
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

func createCurlJob(namespace string, baseURL string) (string, error) {
	type jobTemplateData struct {
		Namespace string
		BaseURL   string
	}

	data := jobTemplateData{
		Namespace: namespace,
		BaseURL:   baseURL,
	}

	jobYAML, err := utils.RenderTemplate(curlJobYAML, data)
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

func deleteCurlJob(namespace string, jobName string) {
	cmd := exec.Command("kubectl", "delete", "job", jobName,
		"-n", namespace, "--ignore-not-found=true")
	_, _ = utils.Run(cmd)
}

func waitForCurlJob(namespace string, jobName string) error {
	cmd := exec.Command("kubectl", "wait", "job", jobName,
		"--for=condition=complete",
		"-n", namespace,
		fmt.Sprintf("--timeout=%v", settings.TimeoutVeryLong))
	_, err := utils.Run(cmd)
	if err != nil {
		return fmt.Errorf("curl job did not complete within timeout: %w", err)
	}

	return nil
}
