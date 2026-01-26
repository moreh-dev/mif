//go:build e2e
// +build e2e

package e2e

import (
	utils "github.com/moreh-dev/mif/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Prefill-Decode Disaggregation", Ordered, func() {

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("collecting logs and events for debugging")
			utils.CollectDebugInfo()
		}
	})

	SetDefaultEventuallyTimeout(utils.TimeoutShort)
	SetDefaultEventuallyPollingInterval(utils.IntervalShort)

	Context("MIF Infrastructure", func() {
		BeforeEach(func() {
			if utils.Cfg.SkipPrerequisite {
				Skip("MIF infrastructure is expected to be pre-installed when SKIP_PREREQUISITE=true")
			}
		})

		It("should deploy MIF components successfully", func() {
			By("validating that Odin controller is running")
			Eventually(utils.VerifyOdinController).Should(Succeed())
		})

		It("should have all pods ready", func() {
			By("waiting for all pods to be ready")
			Eventually(utils.VerifyAllPodsReady, utils.TimeoutVeryLong).Should(Succeed())
		})
	})

	Context("Gateway and InferenceService CR integration", func() {
		BeforeAll(func() {
			By("setting up test resources")
			utils.CreateWorkloadNamespace()

			By("applying Gateway resources")
			utils.ApplyGatewayResource()
			By("installing Heimdall")
			utils.InstallHeimdallForTest()
			By("installing Prefill-Decode InferenceServices (creating InferenceServiceTemplates and then InferenceServices)")
			utils.InstallPrefillDecodeInferenceServicesForTest()
		})

		AfterAll(func() {
			if utils.Cfg.SkipCleanup {
				return
			}
			By("cleaning up test workload namespace")
			err := utils.CleanupWorkloadNamespace()
			Expect(err).NotTo(HaveOccurred(), "failed to clean up test workload namespace")
		})

		It("should have inference-service decode pods reachable behind Gateway", func() {
			utils.VerifyInferenceEndpoint()
		})

		It("should run inference-perf performance benchmark", func() {
			if !utils.Cfg.InferencePerfEnabled {
				Skip("Inference performance benchmark is disabled (INFERENCE_PERF_ENABLED=false)")
			}
			utils.RunInferencePerfBenchmark()
		})

		It("should run quality benchmarks", func() {
			if !utils.Cfg.QualityBenchmarkEnabled {
				Skip("Quality benchmark is disabled (QUALITY_BENCHMARK_ENABLED=false)")
			}
			utils.RunQualityBenchmark()
		})
	})

})
