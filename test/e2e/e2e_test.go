//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/moreh-dev/mif/test/utils"
)

var _ = Describe("Prefill-Decode Disaggregation", Ordered, func() {

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("collecting logs and events for debugging")
			collectDebugInfo()
		}
	})

	SetDefaultEventuallyTimeout(timeoutShort)
	SetDefaultEventuallyPollingInterval(intervalShort)

	Context("MIF Infrastructure", func() {
		BeforeEach(func() {
			if cfg.skipPrerequisite {
				Skip("MIF infrastructure is expected to be pre-installed when SKIP_PREREQUISITE=true")
			}
		})

		It("should deploy MIF components successfully", func() {
			By("validating that Odin controller is running")
			Eventually(verifyOdinController).Should(Succeed())
		})

		It("should have all pods ready", func() {
			By("waiting for all pods to be ready")
			Eventually(verifyAllPodsReady, timeoutVeryLong).Should(Succeed())
		})
	})

	Context("Gateway and InferenceService CR integration", func() {
		BeforeAll(func() {
			By("setting up test resources")
			createWorkloadNamespace()

			By("applying Gateway resources")
			applyGatewayResource()
			By("installing Heimdall")
			installHeimdallForTest()
			By("installing Prefill-Decode InferenceServices (creating InferenceServiceTemplates and then InferenceServices)")
			installPrefillDecodeInferenceServicesForTest()
		})

		AfterAll(func() {
			if cfg.skipCleanup {
				return
			}
			By("cleaning up test workload namespace")
			if err := utils.CleanupWorkloadNamespace(utils.CleanupConfig{
				WorkloadNamespace: cfg.workloadNamespace,
				GatewayClass:      cfg.gatewayClass,
				MIFNamespace:      cfg.mifNamespace,
				PrefillName:       fmt.Sprintf("%s-prefill", inferenceServiceName),
				DecodeName:        fmt.Sprintf("%s-decode", inferenceServiceName),
				TemplateNames: []string{
					"workertemplate-vllm-common",
					"workertemplate-pd-prefill-meta",
					"workertemplate-pd-decode-meta",
					"workertemplate-decode-proxy",
				},
			}); err != nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to cleanup workload namespace: %v\n", err)
			}
		})

		It("should have inference-service decode pods reachable behind Gateway", func() {
			verifyInferenceEndpoint()
		})

		It("should run inference-perf performance benchmark", func() {
			runInferencePerfBenchmark()
		})
	})

})