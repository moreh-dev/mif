//go:build e2e && !printenv
// +build e2e,!printenv

package functional

import (
	"fmt"
	"os/exec"

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
		cmd := exec.Command("kubectl", "wait", "inferenceService", serviceName,
			"--for=condition=Ready",
			"-n", envs.WorkloadNamespace,
			fmt.Sprintf("--timeout=%v", settings.TimeoutVeryLong))
		output, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "failed to wait for InferenceService to be ready")
		Expect(output).To(Equal("True"), "InferenceService should be in Ready state")
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

		By("sending inference request via curl")
		cmd := exec.Command("kubectl", "exec", "-n", envs.WorkloadNamespace,
			"deploy/heimdall", "--",
			"curl", "-sf", "--max-time", "30",
			"-X", "POST",
			fmt.Sprintf("http://%s/v1/completions", gwServiceName),
			"-H", "Content-Type: application/json",
			"-d", fmt.Sprintf(`{"model":"%s","prompt":"Hello","max_tokens":1}`, "meta-llama/Llama-3.2-1B-Instruct"))
		output, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "inference request failed")
		Expect(output).NotTo(BeEmpty(), "inference response should not be empty")
	})
})
