//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/goccy/go-yaml"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/moreh-dev/mif/test/utils"
)


func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting MIF E2E test suite\n")
	stopInterruptHandler := setupInterruptHandler()
	defer stopInterruptHandler()
	RunSpecs(t, "MIF E2E Suite")
}

func setupInterruptHandler() func() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})

	go func() {
		select {
		case sig := <-sigChan:
			_, _ = fmt.Fprintf(GinkgoWriter, "\nReceived signal: %v. Initiating immediate cleanup...\n", sig)
			cleanupKindCluster()
		case <-done:
			return
		}
	}()

	return func() {
		signal.Stop(sigChan)
		close(done)
	}
}

func cleanupKindCluster() {
	if cfg.skipKind {
		return
	}

	if !utils.IsKindClusterExists(cfg.kindClusterName) {
		_, _ = fmt.Fprintf(GinkgoWriter, "Kind cluster %s does not exist, skipping deletion\n", cfg.kindClusterName)
		return
	}

	By("deleting kind cluster (always cleanup)")
	_, _ = fmt.Fprintf(GinkgoWriter, "Deleting kind cluster %s...\n", cfg.kindClusterName)

	if err := utils.DeleteKindCluster(cfg.kindClusterName); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to delete kind cluster %s: %v\n", cfg.kindClusterName, err)
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Successfully deleted kind cluster %s\n", cfg.kindClusterName)
	}
}

var _ = BeforeSuite(func() {
	By("checking prerequisites")
	checkPrerequisites()

	if !cfg.skipKind {
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
	} else {
		cfg.isUsingKindCluster = false
		_, _ = fmt.Fprintf(GinkgoWriter, "Using existing cluster (kubeconfig). Resource cleanup will be skipped for safety.\n")
	}

	if !cfg.skipPrerequisite {
		By("checking if cert manager is installed already")
		cfg.isCertManagerAlreadyInstalled = utils.IsCertManagerCRDsInstalled()
		if !cfg.isCertManagerAlreadyInstalled {
			_, _ = fmt.Fprintf(GinkgoWriter, "Installing CertManager...\n")
			Expect(utils.InstallCertManager()).To(Succeed(), "Failed to install CertManager")
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: CertManager is already installed. Skipping installation...\n")
		}
	}

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
	} else {
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

	By("checking if moai-inference-preset is already installed")
	cfg.isPresetAlreadyInstalled = utils.IsPresetInstalled(cfg.mifNamespace)
	if cfg.isPresetAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "moai-inference-preset is already installed in namespace %s. Skipping installation...\n", cfg.mifNamespace)
	} else {
		By("deploying moai-inference-preset")
		Expect(utils.DeployMIFPreset(cfg.mifNamespace, cfg.presetChartPath)).To(Succeed(), "Failed to deploy moai-inference-preset")
	}

	if !cfg.skipPrerequisite {
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
		case gatewayClassKgateway:
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
		default:
			Fail(fmt.Sprintf("Unsupported %s=%s. Supported values are: %s, %s", envGatewayClassName, cfg.gatewayClass, gatewayClassIstio, gatewayClassKgateway))
		}
	}
})

var _ = AfterSuite(func() {
	// Always clean up temporary value/manifest files, regardless of cluster type or SKIP_CLEANUP.
	cleanupE2ETempFiles()

	if !cfg.isUsingKindCluster {
		_, _ = fmt.Fprintf(GinkgoWriter, "Using existing cluster (kubeconfig). Skipping resource cleanup for safety.\n")
		return
	}

	if cfg.skipCleanup {
		_, _ = fmt.Fprintf(GinkgoWriter, "%s=true: skipping test namespace, resources, and kind cluster deletion.\n", envSkipCleanup)
		return
	}

	By("deleting InferenceService")
	cmd := exec.Command("kubectl", "delete", "inferenceservice", testInferenceServiceName,
		"-n", cfg.workloadNamespace, "--ignore-not-found=true")
	_, _ = utils.Run(cmd)

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

	cleanupKindCluster()
})

func cleanupMIFNamespace() {
	namespace := cfg.mifNamespace

	By(fmt.Sprintf("deleting MIF namespace %s", namespace))
	cmd := exec.Command("kubectl", "delete", "ns", namespace, "--timeout=60s", "--ignore-not-found=true")
	output, err := utils.Run(cmd)
	if err != nil {
		By("attempting to force delete namespace by removing finalizers")
		cmd = exec.Command("kubectl", "patch", "namespace", namespace,
			"--type=json", "-p", `[{"op": "replace", "path": "/spec/finalizers", "value": []}]`, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		cmd = exec.Command("kubectl", "delete", "ns", namespace, "--timeout=30s", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)
	} else if output != "" {
		_, _ = fmt.Fprintf(GinkgoWriter, "Namespace deletion output: %s\n", output)
	}
}

func cleanupE2ETempFiles() {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "warning: failed to get project dir for temp file cleanup: %v\n", err)
		return
	}

	tempFiles := []string{
		tempFileMIFValues,
		tempFileHeimdallValues,
		tempFileISValues,
		tempFileIstiodValues,
		tempFileKgatewayValues,
	}

	for _, rel := range tempFiles {
		path := filepath.Join(projectDir, rel)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			_, _ = fmt.Fprintf(GinkgoWriter, "warning: failed to remove temp file %s: %v\n", path, err)
		}
	}
}

func checkPrerequisites() {
	requiredTools := []string{"kubectl", "helm"}
	if !cfg.skipKind {
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

	if !cfg.skipKind {
		return
	}

	cmd := exec.Command("kubectl", "cluster-info")
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Cannot connect to Kubernetes cluster. Please check your kubeconfig.")
}

// writeValuesFile is a helper function to write Helm values files.
func writeValuesFile(relativePath, content string, mode os.FileMode) (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	valuesPath := filepath.Join(projectDir, relativePath)
	if err := os.WriteFile(valuesPath, []byte(content), mode); err != nil {
		return "", fmt.Errorf("failed to write values file %s: %w", relativePath, err)
	}

	return valuesPath, nil
}

func createMIFValuesFile(awsAccessKeyID, awsSecretAccessKey string) (string, error) {
	valuesContent := fmt.Sprintf(`ecrTokenRefresher:
  aws:
    accessKeyId: %s
    secretAccessKey: %s
`, awsAccessKeyID, awsSecretAccessKey)

	return writeValuesFile(tempFileMIFValues, valuesContent, 0600)
}

func createKgatewayValuesFile() (string, error) {
	valuesContent := `inferenceExtension:
  enabled: true
`

	return writeValuesFile(tempFileKgatewayValues, valuesContent, 0644)
}

func createIstiodValuesFile() (string, error) {
	valuesContent := `pilot:
  env:
    PILOT_ENABLE_ALPHA_GATEWAY_API: "true"
    ENABLE_GATEWAY_API_INFERENCE_EXTENSION: "true"
`

	return writeValuesFile(tempFileIstiodValues, valuesContent, 0644)
}

func ensureECRTokenRefresherSecret(namespace string) {
	cronJobName := cronJobNameECRRefresher
	jobName := jobNameECRRefresher
	secretName := secretNameMorehRegistry
	ecrCredsSecretName := secretNameECRCreds

	By(fmt.Sprintf("waiting for secret %s to be created", ecrCredsSecretName))
	Eventually(func() bool {
		cmd := exec.Command("kubectl", "get", "secret", ecrCredsSecretName, "-n", namespace)
		_, err := utils.Run(cmd)
		return err == nil
	}, timeoutShort, intervalMedium).Should(BeTrue(), fmt.Sprintf("Secret %s should be created", ecrCredsSecretName))

	cmd := exec.Command("kubectl", "get", "secret", secretName, "-n", namespace)
	_, err := utils.Run(cmd)
	if err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Secret %s already exists. Skipping ecrTokenRefresher job execution.\n", secretName)
		return
	}

	By(fmt.Sprintf("waiting for CronJob %s to be created", cronJobName))
	Eventually(func() bool {
		cmd = exec.Command("kubectl", "get", "cronjob", cronJobName, "-n", namespace)
		_, err = utils.Run(cmd)
		return err == nil
	}, timeoutShort, intervalMedium).Should(BeTrue(), fmt.Sprintf("CronJob %s should be created", cronJobName))

	cmd = exec.Command("kubectl", "delete", "job", jobName, "-n", namespace, "--ignore-not-found=true")
	_, _ = utils.Run(cmd)

	By(fmt.Sprintf("creating manual job from CronJob %s", cronJobName))
	cmd = exec.Command("kubectl", "create", "job", "--from=cronjob/"+cronJobName, jobName, "-n", namespace)
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create job from CronJob %s", cronJobName))

	By(fmt.Sprintf("waiting for job %s to complete", jobName))
	Eventually(func() bool {
		cmd := exec.Command("kubectl", "get", "job", jobName, "-n", namespace, "-o", "jsonpath={.status.conditions[?(@.type==\"Complete\")].status}")
		output, err := utils.Run(cmd)
		return err == nil && strings.TrimSpace(output) == "True"
	}, timeoutMedium, intervalMedium).Should(BeTrue(), fmt.Sprintf("Job %s should complete successfully", jobName))

	cmd = exec.Command("kubectl", "get", "job", jobName, "-n", namespace, "-o", "jsonpath={.status.conditions[?(@.type==\"Failed\")].status}")
	output, _ := utils.Run(cmd)
	if strings.TrimSpace(output) == "True" {
		cmd = exec.Command("kubectl", "logs", "-n", namespace, "job/"+jobName)
		logs, _ := utils.Run(cmd)
		Fail(fmt.Sprintf("Job %s failed. Logs: %s", jobName, logs))
	}

	By(fmt.Sprintf("waiting for secret %s to be created", secretName))
	Eventually(func() bool {
		cmd := exec.Command("kubectl", "get", "secret", secretName, "-n", namespace)
		_, err := utils.Run(cmd)
		return err == nil
	}, timeoutShort, intervalShort).Should(BeTrue(), fmt.Sprintf("Secret %s should be created", secretName))

	_, _ = fmt.Fprintf(GinkgoWriter, "Successfully created secret %s via ecrTokenRefresher\n", secretName)
}

func createHeimdallValuesFile() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	baseYAML := `global:
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
  gatewayClassName: istio
`

	var values map[string]interface{}
	if err := yaml.Unmarshal([]byte(baseYAML), &values); err != nil {
		return "", fmt.Errorf("failed to parse base Heimdall values YAML: %w", err)
	}

	// Set dynamic values after parsing
	if global, ok := values["global"].(map[string]interface{}); ok {
		if imagePullSecrets, ok := global["imagePullSecrets"].([]interface{}); ok && len(imagePullSecrets) > 0 {
			if secret, ok := imagePullSecrets[0].(map[string]interface{}); ok {
				secret["name"] = secretNameMorehRegistry
			}
		}
	}

	if gateway, ok := values["gateway"].(map[string]interface{}); ok {
		gateway["name"] = gatewayName
		gateway["gatewayClassName"] = cfg.gatewayClass
	}

	if cfg.prometheusStackEnabled {
		values["serviceMonitor"] = map[string]interface{}{
			"enabled": true,
			"labels": map[string]interface{}{
				"release": "prometheus-stack",
			},
		}
	} else {
		values["serviceMonitor"] = map[string]interface{}{
			"enabled": false,
		}
	}

	valuesYAML, err := yaml.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Heimdall values YAML: %w", err)
	}

	valuesPath := filepath.Join(projectDir, tempFileHeimdallValues)
	err = os.WriteFile(valuesPath, valuesYAML, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write Heimdall values file: %w", err)
	}

	return valuesPath, nil
}

func waitForMIFComponents() {
	By("waiting for Odin controller deployment")
	cmd := exec.Command("kubectl", "get", "deployment",
		"-n", cfg.mifNamespace,
		"-l", "app.kubernetes.io/name=odin",
		"-o", "jsonpath={.items[0].metadata.name}")
	output, err := utils.Run(cmd)
	if err != nil || strings.TrimSpace(output) == "" {
		cmd = exec.Command("kubectl", "get", "deployment",
			"-n", cfg.mifNamespace,
			"-o", "jsonpath={.items[?(@.metadata.name=~\"odin.*\")].metadata.name}")
		output, err = utils.Run(cmd)
		if err != nil || strings.TrimSpace(output) == "" {
			_, _ = fmt.Fprintf(GinkgoWriter, "Warning: Could not find Odin deployment by label, trying common name pattern\n")
			cmd = exec.Command("kubectl", "wait", "--for=condition=Available",
				fmt.Sprintf("deployment/%s-odin", helmReleaseMIF),
				"-n", cfg.mifNamespace,
				fmt.Sprintf("--timeout=%v", timeoutMedium))
			_, _ = utils.Run(cmd)
			return
		}
	}
	odinDeploymentName := strings.TrimSpace(output)
	_, _ = fmt.Fprintf(GinkgoWriter, "Found Odin deployment: %s\n", odinDeploymentName)

	cmd = exec.Command("kubectl", "wait", "--for=condition=Available",
		fmt.Sprintf("deployment/%s", odinDeploymentName),
		"-n", cfg.mifNamespace,
		fmt.Sprintf("--timeout=%v", timeoutLong))
	output, err = utils.Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Warning: Odin deployment wait completed with error (may already be ready): %v\n", err)
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Odin deployment is available\n")
	}

	By("waiting for all running pods to be ready")
	cmd = exec.Command("kubectl", "wait", "pod",
		"--for=condition=Ready",
		"--all",
		"--field-selector=status.phase!=Succeeded",
		"-n", cfg.mifNamespace,
		fmt.Sprintf("--timeout=%v", timeoutLong))
	output, err = utils.Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Warning: Some pods may not be ready yet: %v\n", err)
		cmd = exec.Command("kubectl", "get", "pods",
			"-n", cfg.mifNamespace,
			"--field-selector=status.phase!=Succeeded",
			"-o", "wide")
		statusOutput, _ := utils.Run(cmd)
		_, _ = fmt.Fprintf(GinkgoWriter, "Current pod status:\n%s\n", statusOutput)
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "All pods are ready\n")
	}
}
