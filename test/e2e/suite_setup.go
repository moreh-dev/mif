//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// setupKindCluster creates or reuses a kind cluster for testing.
func setupKindCluster() {
	By("creating kind cluster")
	if IsKindClusterExists(cfg.kindClusterName) {
		_, _ = fmt.Fprintf(GinkgoWriter, "Kind cluster %s already exists. Skipping creation...\n", cfg.kindClusterName)
		By("exporting kubeconfig for existing kind cluster")
		cmd := exec.Command("kind", "export", "kubeconfig", "--name", cfg.kindClusterName)
		_, err := Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to export kubeconfig for existing kind cluster")
		cfg.isUsingKindCluster = true
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Creating kind cluster %s...\n", cfg.kindClusterName)
		if err := CreateKindCluster(cfg.kindClusterName); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "Kind cluster creation failed. Attempting to clean up partially created cluster...\n")
			if IsKindClusterExists(cfg.kindClusterName) {
				_ = DeleteKindCluster(cfg.kindClusterName)
			}
			Expect(err).NotTo(HaveOccurred(), "Failed to create kind cluster")
		}
		cfg.isUsingKindCluster = true

		By("verifying kubectl access to kind cluster")
		contextName := fmt.Sprintf("kind-%s", cfg.kindClusterName)
		cmd := exec.Command("kubectl", "cluster-info", "--context", contextName)
		_, err := Run(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "Failed to verify kind cluster. Attempting to clean up...\n")
			if IsKindClusterExists(cfg.kindClusterName) {
				_ = DeleteKindCluster(cfg.kindClusterName)
			}
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to access kind cluster via kubectl context %s", contextName))
		}
	}

	By("adding moreh Helm repository")
	cmd := exec.Command("helm", "repo", "add", helmRepoName, helmRepoURL)
	if _, err := Run(cmd); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to add moreh helm repo: %v\n", err)
		}
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Successfully added moreh Helm repository\n")
	}

	By("updating moreh Helm repository")
	cmd = exec.Command("helm", "repo", "update", helmRepoName)
	if _, err := Run(cmd); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to update moreh helm repo: %v\n", err)
	}
}

// setupPrerequisites installs prerequisite components if needed.
func setupPrerequisites() {
	if cfg.skipPrerequisite {
		return
	}

	By("installing CertManager")
	Expect(InstallCertManager()).To(Succeed(), "Failed to install CertManager")

	setupMIF()
	setupPreset()
	setupGateway()
}



// setupMIF installs MIF infrastructure if not already installed.
func setupMIF() {
	By("creating MIF namespace")
	cmd := exec.Command("kubectl", "create", "ns", cfg.mifNamespace)
	_, err := Run(cmd)
	if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
		Expect(err).NotTo(HaveOccurred(), "Failed to create MIF namespace")
	}

	By("deploying MIF infrastructure via Helm")

	if cfg.awsAccessKeyID != "" && cfg.awsSecretAccessKey != "" {
		By("scheduling asynchronous ecrTokenRefresher execution to create moreh-registry secret")
		go func(ns string) {
			defer GinkgoRecover()
			ensureECRTokenRefresherSecret(ns)
		}(cfg.mifNamespace)
	} else {
		_, _ = fmt.Fprintf(
			GinkgoWriter,
			"%s or %s is not set; skipping ecrTokenRefresher configuration for MIF chart.\n",
			envAWSAccessKeyID, envAWSSecretAccessKey,
		)
	}

	helmArgs := []string{
		"upgrade", "--install", helmReleaseMIF,
		cfg.mifChartPath,
		"--namespace", cfg.mifNamespace,
		"--wait",
		fmt.Sprintf("--timeout=%v", timeoutVeryLong),
		"--set", "odin.enabled=true",
		"--set", fmt.Sprintf("odin-crd.enabled=%t", cfg.odinCRDEnabled),
		"--set", fmt.Sprintf("prometheus-stack.enabled=%t", cfg.prometheusStackEnabled),
		"--set", fmt.Sprintf("keda.enabled=%t", cfg.kedaEnabled),
		"--set", fmt.Sprintf("lws.enabled=%t", cfg.lwsEnabled),
		"--set", "replicator.enabled=true",
	}

	if cfg.awsAccessKeyID != "" && cfg.awsSecretAccessKey != "" {
		By("creating moai-inference-framework values file for ECR token refresher")
		mifValuesPath, err := createMIFValuesFile(cfg.awsAccessKeyID, cfg.awsSecretAccessKey)
		Expect(err).NotTo(HaveOccurred(), "Failed to create moai-inference-framework values file")
		helmArgs = append(helmArgs, "-f", mifValuesPath)
	}

	cmd = exec.Command("helm", helmArgs...)
	_, err = Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to deploy MIF via Helm")

	By("waiting for MIF components to be ready")
	waitForMIFComponents()
}

// setupPreset installs moai-inference-preset.
func setupPreset() {
	By("deploying moai-inference-preset")
	Expect(DeployMIFPreset(cfg.mifNamespace, cfg.presetChartPath)).To(Succeed(), "Failed to deploy moai-inference-preset")
}

// setupGateway installs Gateway API and controller.
func setupGateway() {
	By("installing Gateway API standard CRDs")
	Expect(InstallGatewayAPI()).To(Succeed(), "Failed to install Gateway API standard CRDs")

	By("installing Gateway API Inference Extension CRDs")
	Expect(InstallGatewayInferenceExtension()).To(Succeed(), "Failed to install Gateway API Inference Extension CRDs")

	By(fmt.Sprintf("installing %s controller", cfg.gatewayClass))
	Expect(InstallGatewayController(cfg.gatewayClass)).To(Succeed(), fmt.Sprintf("Failed to install %s controller", cfg.gatewayClass))
}

// cleanupKindResources cleans up resources specific to kind cluster.
func cleanupKindResources() {
	By("uninstalling moai-inference-preset")
	UninstallMIFPreset(cfg.mifNamespace)

	By("uninstalling MIF")
	cmd := exec.Command("helm", "uninstall", helmReleaseMIF, "-n", cfg.mifNamespace, "--ignore-not-found=true")
	_, _ = Run(cmd)

	By("deleting MIF namespace")
	cleanupMIFNamespace()

	if !cfg.skipPrerequisite {
		cleanupPrerequisites()
	}

	cleanupKindCluster()
}

// cleanupPrerequisites uninstalls prerequisite components.
func cleanupPrerequisites() {
	if cfg.gatewayClass == gatewayClassIstio {
		By("uninstalling Istio")
		if err := UninstallGatewayController(cfg.gatewayClass); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to uninstall Gateway controller: %v\n", err)
		}
	} else if cfg.gatewayClass == gatewayClassKgateway {
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
