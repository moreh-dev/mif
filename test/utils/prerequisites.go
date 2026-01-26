//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// CheckPrerequisites verifies that required tools are available.
func CheckPrerequisites() {
	requiredTools := []string{"kubectl", "helm"}
	if !cfg.SkipKind {
		requiredTools = append(requiredTools, "kind")
		cmd := exec.Command("which", "kind")
		if err := cmd.Run(); err != nil {
			Fail(fmt.Sprintf("kind is required but not available. Install kind or set %s=true to use existing cluster", envSkipKind))
		}
	}
	for _, tool := range requiredTools {
		cmd := exec.Command("which", tool)
		if err := cmd.Run(); err != nil {
			Fail(fmt.Sprintf("Required tool %s is not available", tool))
		}
	}

	if !cfg.SkipKind {
		return
	}

	By("verifying Kubernetes cluster connectivity")
	cmd := exec.Command("kubectl", "cluster-info", "--request-timeout=30s")
	_, err := Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Cannot connect to Kubernetes cluster. Please check your kubeconfig.")
}

// SetupPrerequisites installs prerequisite components if needed.
func SetupPrerequisites() {
	if cfg.SkipPrerequisite {
		return
	}

	By("installing CertManager")
	Expect(InstallCertManager()).To(Succeed(), "Failed to install CertManager")

	setupMIF()
	setupPreset()
	setupGateway()
}

// cleanupPrerequisites uninstalls prerequisite components.
func cleanupPrerequisites() {
	switch cfg.gatewayClass {
	case gatewayClassIstio:
		By("uninstalling Istio")
		if err := UninstallGatewayController(cfg.gatewayClass); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to uninstall Gateway controller: %v\n", err)
		}
	case gatewayClassKgateway:
		By("uninstalling Kgateway")
		if err := UninstallGatewayController(cfg.gatewayClass); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to uninstall Gateway controller: %v\n", err)
		}
	}

	By("uninstalling Gateway API Inference Extension")
	if err := UninstallGatewayInferenceExtension(); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to uninstall Gateway API Inference Extension: %v\n", err)
	}

	By("uninstalling Gateway API")
	if err := UninstallGatewayAPI(); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to uninstall Gateway API: %v\n", err)
	}

	By("Uninstalling CertManager...")
	UninstallCertManager()
}

// setupPreset installs moai-inference-preset.
func setupPreset() {
	By("deploying moai-inference-preset")
	Expect(DeployMIFPreset(cfg.mifNamespace, cfg.presetChartPath)).To(Succeed(), "Failed to deploy moai-inference-preset")
}
