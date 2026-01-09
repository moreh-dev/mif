//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/moreh-dev/mif/test/utils"
)

// setupKindCluster creates or reuses a kind cluster for testing.
func setupKindCluster() {
	By("creating kind cluster")
	if utils.IsKindClusterExists(cfg.kindClusterName) {
		_, _ = fmt.Fprintf(GinkgoWriter, "Kind cluster %s already exists. Skipping creation...\n", cfg.kindClusterName)
		By("exporting kubeconfig for existing kind cluster")
		cmd := exec.Command("kind", "export", "kubeconfig", "--name", cfg.kindClusterName)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to export kubeconfig for existing kind cluster")
		cfg.isUsingKindCluster = true
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Creating kind cluster %s...\n", cfg.kindClusterName)
		if err := utils.CreateKindCluster(cfg.kindClusterName); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "Kind cluster creation failed. Attempting to clean up partially created cluster...\n")
			if utils.IsKindClusterExists(cfg.kindClusterName) {
				_ = utils.DeleteKindCluster(cfg.kindClusterName)
			}
			Expect(err).NotTo(HaveOccurred(), "Failed to create kind cluster")
		}
		cfg.isUsingKindCluster = true

		By("verifying kubectl access to kind cluster")
		contextName := fmt.Sprintf("kind-%s", cfg.kindClusterName)
		cmd := exec.Command("kubectl", "cluster-info", "--context", contextName)
		_, err := utils.Run(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "Failed to verify kind cluster. Attempting to clean up...\n")
			if utils.IsKindClusterExists(cfg.kindClusterName) {
				_ = utils.DeleteKindCluster(cfg.kindClusterName)
			}
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to access kind cluster via kubectl context %s", contextName))
		}
	}

	By("adding moreh Helm repository")
	cmd := exec.Command("helm", "repo", "add", helmRepoName, helmRepoURL)
	if _, err := utils.Run(cmd); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to add moreh helm repo: %v\n", err)
		}
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Successfully added moreh Helm repository\n")
	}
}

// setupPrerequisites installs prerequisite components if needed.
func setupPrerequisites() {
	if cfg.skipPrerequisite {
		return
	}

	By("checking if cert manager is installed already")
	cfg.isCertManagerAlreadyInstalled = utils.IsCertManagerCRDsInstalled()
	if !cfg.isCertManagerAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Installing CertManager...\n")
		Expect(utils.InstallCertManager()).To(Succeed(), "Failed to install CertManager")
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: CertManager is already installed. Skipping installation...\n")
	}

	detectComponentState()
	setupMIF()
	setupPreset()
	setupGateway()
}

// detectComponentState auto-detects cluster state and adjusts component enable flags.
func detectComponentState() {
	By("auto-detecting cluster state and adjusting component enable flags")
	if os.Getenv(envKEDAEnabled) == "" {
		By("checking if KEDA is already installed")
		cfg.kedaEnabled = !utils.IsKEDAInstalled()
		if cfg.kedaEnabled {
			_, _ = fmt.Fprintf(GinkgoWriter, "KEDA is not installed. Enabling KEDA in MIF chart.\n")
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "KEDA is already installed in the cluster. Disabling KEDA in MIF chart to avoid conflicts.\n")
		}
	}

	if os.Getenv(envLWSEnabled) == "" {
		By("checking if LWS is already installed")
		cfg.lwsEnabled = !utils.IsLWSInstalled()
		if cfg.lwsEnabled {
			_, _ = fmt.Fprintf(GinkgoWriter, "LWS is not installed. Enabling LWS in MIF chart.\n")
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "LWS is already installed in the cluster. Disabling LWS in MIF chart to avoid conflicts.\n")
		}
	}

	if os.Getenv(envOdinCRDEnabled) == "" {
		By("checking if Odin CRD is already installed")
		cfg.odinCRDEnabled = !utils.IsOdinCRDInstalled()
		if cfg.odinCRDEnabled {
			_, _ = fmt.Fprintf(GinkgoWriter, "Odin CRD is not installed. Enabling Odin CRD in MIF chart.\n")
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "Odin CRD is already installed in the cluster. Disabling Odin CRD in MIF chart to avoid conflicts.\n")
		}
	}

	if os.Getenv(envPrometheusStackEnabled) == "" {
		By("checking if Prometheus is already installed")
		if utils.IsPrometheusInstalled() {
			cfg.prometheusStackEnabled = false
			_, _ = fmt.Fprintf(GinkgoWriter, "Prometheus is already installed in the cluster. Disabling Prometheus Stack in MIF chart to avoid conflicts.\n")
		} else {
			cfg.prometheusStackEnabled = false
			_, _ = fmt.Fprintf(GinkgoWriter, "Prometheus Stack disabled by default in E2E tests to avoid resource issues. Set %s=true to enable.\n", envPrometheusStackEnabled)
		}
	}
}

// setupMIF installs MIF infrastructure if not already installed.
func setupMIF() {
	By("creating MIF namespace")
	cmd := exec.Command("kubectl", "create", "ns", cfg.mifNamespace)
	_, err := utils.Run(cmd)
	if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
		Expect(err).NotTo(HaveOccurred(), "Failed to create MIF namespace")
	}

	By("checking if MIF is already installed")
	cfg.isMIFAlreadyInstalled = utils.IsMIFInstalled(cfg.mifNamespace)
	if cfg.isMIFAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "MIF is already installed in namespace %s. Skipping installation...\n", cfg.mifNamespace)
		return
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
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to deploy MIF via Helm")

	By("waiting for MIF components to be ready")
	waitForMIFComponents()
}

// setupPreset installs moai-inference-preset if not already installed.
func setupPreset() {
	By("checking if moai-inference-preset is already installed")
	cfg.isPresetAlreadyInstalled = utils.IsPresetInstalled(cfg.mifNamespace)
	if cfg.isPresetAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "moai-inference-preset is already installed in namespace %s. Skipping installation...\n", cfg.mifNamespace)
	} else {
		By("deploying moai-inference-preset")
		Expect(utils.DeployMIFPreset(cfg.mifNamespace, cfg.presetChartPath)).To(Succeed(), "Failed to deploy moai-inference-preset")
	}
}

// setupGateway installs Gateway API and controller if needed.
func setupGateway() {
	if cfg.skipPrerequisite {
		return
	}

	By("checking if Gateway API is already installed")
	cfg.isGatewayAPIAlreadyInstalled = utils.IsGatewayAPIInstalled()
	if cfg.isGatewayAPIAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Gateway API is already installed. Skipping installation...\n")
	} else {
		By("installing Gateway API standard CRDs")
		Expect(utils.InstallGatewayAPI()).To(Succeed(), "Failed to install Gateway API standard CRDs")
	}

	By("checking if Gateway API Inference Extension is already installed")
	cfg.isGatewayInferenceExtensionInstalled = utils.IsGatewayInferenceExtensionInstalled()
	if cfg.isGatewayInferenceExtensionInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Gateway API Inference Extension is already installed. Skipping installation...\n")
	} else {
		By("installing Gateway API Inference Extension CRDs")
		Expect(utils.InstallGatewayInferenceExtension()).To(Succeed(), "Failed to install Gateway API Inference Extension CRDs")
	}

	switch cfg.gatewayClass {
	case gatewayClassIstio:
		setupIstio()
	case gatewayClassKgateway:
		setupKgateway()
	default:
		Fail(fmt.Sprintf("Unsupported %s=%s. Supported values are: %s, %s", envGatewayClassName, cfg.gatewayClass, gatewayClassIstio, gatewayClassKgateway))
	}
}

// setupIstio installs Istio if not already installed.
func setupIstio() {
	By("checking if Istio is already installed")
	cfg.isIstioAlreadyInstalled = utils.IsIstioInstalled()
	if cfg.isIstioAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Istio is already installed. Skipping installation...\n")
	} else {
		By("installing Istio base")
		Expect(utils.InstallIstioBase()).To(Succeed(), "Failed to install Istio base")

		By("creating istiod values file")
		istiodValuesPath, err := createIstiodValuesFile()
		Expect(err).NotTo(HaveOccurred(), "Failed to create istiod values file")

		By("installing Istiod control plane")
		Expect(utils.InstallIstiod(istiodValuesPath)).To(Succeed(), "Failed to install Istiod control plane")
	}
}

// setupKgateway installs KGateway if not already installed.
func setupKgateway() {
	By("checking if Kgateway is already installed")
	cfg.isKgatewayAlreadyInstalled = utils.IsKgatewayInstalled()
	if cfg.isKgatewayAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Kgateway is already installed. Skipping installation...\n")
	} else {
		By("creating Kgateway values file")
		kgatewayValuesPath, err := createKgatewayValuesFile()
		Expect(err).NotTo(HaveOccurred(), "Failed to create Kgateway values file")

		By("installing Kgateway CRDs")
		Expect(utils.InstallKgatewayCRDs()).To(Succeed(), "Failed to install Kgateway CRDs")

		By("installing Kgateway controller")
		Expect(utils.InstallKgateway(kgatewayValuesPath)).To(Succeed(), "Failed to install Kgateway controller")
	}
}

// cleanupKindResources cleans up resources specific to kind cluster.
func cleanupKindResources() {
	if !cfg.isPresetAlreadyInstalled {
		By("uninstalling moai-inference-preset")
		utils.UninstallMIFPreset(cfg.mifNamespace)
	}

	if !cfg.isMIFAlreadyInstalled {
		By("uninstalling MIF")
		cmd := exec.Command("helm", "uninstall", helmReleaseMIF, "-n", cfg.mifNamespace, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)
	}

	By("deleting MIF namespace")
	cleanupMIFNamespace()

	if !cfg.skipPrerequisite {
		cleanupPrerequisites()
	}

	cleanupKindCluster()
}

// cleanupPrerequisites uninstalls prerequisite components if they were installed during tests.
func cleanupPrerequisites() {
	if cfg.gatewayClass == gatewayClassIstio && !cfg.isIstioAlreadyInstalled {
		By("uninstalling Istio")
		if err := utils.UninstallGatewayController(cfg.gatewayClass); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to uninstall Gateway controller: %v\n", err)
		}
	} else if cfg.gatewayClass == gatewayClassKgateway && !cfg.isKgatewayAlreadyInstalled {
		By("uninstalling Kgateway")
		if err := utils.UninstallGatewayController(cfg.gatewayClass); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to uninstall Gateway controller: %v\n", err)
		}
	}

	if !cfg.isGatewayInferenceExtensionInstalled {
		By("uninstalling Gateway API Inference Extension")
		if err := utils.UninstallGatewayInferenceExtension(); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to uninstall Gateway API Inference Extension: %v\n", err)
		}
	}

	if !cfg.isGatewayAPIAlreadyInstalled {
		By("uninstalling Gateway API")
		if err := utils.UninstallGatewayAPI(); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to uninstall Gateway API: %v\n", err)
		}
	}

	if !cfg.isCertManagerAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Uninstalling CertManager...\n")
		utils.UninstallCertManager()
	}
}