//go:build e2e
// +build e2e

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/moreh-dev/mif/test/utils"
)

var (
	skipCertManagerInstall        = os.Getenv("SKIP_CERT_MANAGER") == "true"
	isCertManagerAlreadyInstalled = false
	kedaEnabled                   = "true"
	lwsEnabled                    = "true"
	odinCRDEnabled                = "true"
	prometheusStackEnabled        = "true"

	testNamespace   = getEnvOrDefault("NAMESPACE", "mif")
	mifChartPath    = getEnvOrDefault("MIF_CHART_PATH", "deploy/helm/moai-inference-framework")
	presetChartPath = getEnvOrDefault("PRESET_CHART_PATH", "deploy/helm/moai-inference-preset")
	testModel       = getEnvOrDefault("TEST_MODEL", "meta-llama/Llama-3.2-1B-Instruct")
	gatewayClass    = getEnvOrDefault("GATEWAY_CLASS_NAME", "istio")
	skipCleanup     = os.Getenv("SKIP_CLEANUP") == "true"

	kindClusterName    = getEnvOrDefault("KIND_CLUSTER_NAME", "mif-e2e")
	skipKindCreate     = os.Getenv("SKIP_KIND_CREATE") == "true"
	skipKindDelete     = os.Getenv("SKIP_KIND_DELETE") == "true"
	skipMIFDeploy      = os.Getenv("SKIP_MIF_DEPLOY") == "true"
	skipPresetDeploy   = os.Getenv("SKIP_PRESET_DEPLOY") == "true"
	isUsingKindCluster = false
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting MIF E2E test suite\n")
	setupInterruptHandler()
	RunSpecs(t, "MIF E2E Suite")
}

func setupInterruptHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		_, _ = fmt.Fprintf(GinkgoWriter, "\nReceived signal: %v. Initiating immediate cleanup...\n", sig)
		cleanupKindClusterOnInterrupt()
	}()
}

func cleanupKindCluster() {
	if skipKindDelete {
		_, _ = fmt.Fprintf(GinkgoWriter, "Skipping kind cluster deletion (SKIP_KIND_DELETE=true)\n")
		return
	}

	if skipKindCreate {
		return
	}

	if !utils.IsKindClusterExists(kindClusterName) {
		_, _ = fmt.Fprintf(GinkgoWriter, "Kind cluster %s does not exist, skipping deletion\n", kindClusterName)
		return
	}

	By("deleting kind cluster (always cleanup)")
	_, _ = fmt.Fprintf(GinkgoWriter, "Deleting kind cluster %s...\n", kindClusterName)

	if err := utils.DeleteKindCluster(kindClusterName); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to delete kind cluster %s: %v\n", kindClusterName, err)
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Successfully deleted kind cluster %s\n", kindClusterName)
	}
}

func cleanupKindClusterOnInterrupt() {
	cleanupKindCluster()
}

var _ = BeforeSuite(func() {
	By("checking prerequisites")
	checkPrerequisites()

	if !skipKindCreate {
		By("creating kind cluster")
		if utils.IsKindClusterExists(kindClusterName) {
			_, _ = fmt.Fprintf(GinkgoWriter, "Kind cluster %s already exists. Skipping creation...\n", kindClusterName)
			By("exporting kubeconfig for existing kind cluster")
			cmd := exec.Command("kind", "export", "kubeconfig", "--name", kindClusterName)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to export kubeconfig for existing kind cluster")
			isUsingKindCluster = true
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "Creating kind cluster %s...\n", kindClusterName)
			if err := utils.CreateKindCluster(kindClusterName); err != nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kind cluster creation failed. Attempting to clean up partially created cluster...\n")
				if utils.IsKindClusterExists(kindClusterName) {
					_ = utils.DeleteKindCluster(kindClusterName)
				}
				Expect(err).NotTo(HaveOccurred(), "Failed to create kind cluster")
			}
			isUsingKindCluster = true

			By("verifying kubectl access to kind cluster")
			contextName := fmt.Sprintf("kind-%s", kindClusterName)
			cmd := exec.Command("kubectl", "cluster-info", "--context", contextName)
			_, err := utils.Run(cmd)
			if err != nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to verify kind cluster. Attempting to clean up...\n")
				if utils.IsKindClusterExists(kindClusterName) {
					_ = utils.DeleteKindCluster(kindClusterName)
				}
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to access kind cluster via kubectl context %s", contextName))
			}
		}
	} else {
		isUsingKindCluster = false
		_, _ = fmt.Fprintf(GinkgoWriter, "Using existing cluster (kubeconfig). Resource cleanup will be skipped for safety.\n")
	}

	if !skipCertManagerInstall {
		By("checking if cert manager is installed already")
		isCertManagerAlreadyInstalled = utils.IsCertManagerCRDsInstalled()
		if !isCertManagerAlreadyInstalled {
			_, _ = fmt.Fprintf(GinkgoWriter, "Installing CertManager...\n")
			Expect(utils.InstallCertManager()).To(Succeed(), "Failed to install CertManager")
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: CertManager is already installed. Skipping installation...\n")
		}
	}

	if !skipMIFDeploy {
		By("creating test namespace")
		cmd := exec.Command("kubectl", "create", "ns", testNamespace)
		_, err := utils.Run(cmd)
		if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
			Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")
		}

		By("deploying MIF infrastructure via Helm")

		awsAccessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
		awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

		if awsAccessKeyID != "" && awsSecretAccessKey != "" {
			By("scheduling asynchronous ecrTokenRefresher execution to create moreh-registry secret")
			go func(ns string) {
				defer GinkgoRecover()
				ensureECRTokenRefresherSecret(ns)
			}(testNamespace)
		} else {
			_, _ = fmt.Fprintf(
				GinkgoWriter,
				"AWS_ACCESS_KEY_ID or AWS_SECRET_ACCESS_KEY is not set; skipping ecrTokenRefresher configuration for MIF chart.\n",
			)
		}

		helmArgs := []string{
			"upgrade", "--install", "moai-inference-framework",
			mifChartPath,
			"--namespace", testNamespace,
			"--wait",
			"--timeout", "15m",
			"--set", "odin.enabled=true",
			"--set", fmt.Sprintf("odin-crd.enabled=%s", odinCRDEnabled),
			"--set", fmt.Sprintf("prometheus-stack.enabled=%s", prometheusStackEnabled),
			"--set", fmt.Sprintf("keda.enabled=%s", kedaEnabled),
			"--set", fmt.Sprintf("lws.enabled=%s", lwsEnabled),
			"--set", "replicator.enabled=true",
		}

		if awsAccessKeyID != "" && awsSecretAccessKey != "" {
			By("creating moai-inference-framework values file for ECR token refresher")
			mifValuesPath, err := createMIFValuesFile(awsAccessKeyID, awsSecretAccessKey)
			Expect(err).NotTo(HaveOccurred(), "Failed to create moai-inference-framework values file")
			helmArgs = append(helmArgs, "-f", mifValuesPath)
		}

		cmd = exec.Command("helm", helmArgs...)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy MIF via Helm")

		By("waiting for MIF components to be ready")
		waitForMIFComponents()
	}

	if !skipPresetDeploy {
		By("deploying moai-inference-preset")
		Expect(utils.DeployMIFPreset(testNamespace, presetChartPath)).To(Succeed(), "Failed to deploy moai-inference-preset")
	}

	By("installing Gateway API standard CRDs")
	Expect(utils.InstallGatewayAPI()).To(Succeed(), "Failed to install Gateway API standard CRDs")

	By("installing Gateway API Inference Extension CRDs")
	Expect(utils.InstallGatewayInferenceExtension()).To(Succeed(), "Failed to install Gateway API Inference Extension CRDs")

	switch gatewayClass {
	case "istio":
		By("installing Istio base")
		Expect(utils.InstallIstioBase()).To(Succeed(), "Failed to install Istio base")

		By("creating istiod values file")
		istiodValuesPath, err := createIstiodValuesFile()
		Expect(err).NotTo(HaveOccurred(), "Failed to create istiod values file")

		By("installing Istiod control plane")
		Expect(utils.InstallIstiod(istiodValuesPath)).To(Succeed(), "Failed to install Istiod control plane")
	case "kgateway":
		By("creating Kgateway values file")
		kgatewayValuesPath, err := createKgatewayValuesFile()
		Expect(err).NotTo(HaveOccurred(), "Failed to create Kgateway values file")

		By("installing Kgateway CRDs")
		Expect(utils.InstallKgatewayCRDs()).To(Succeed(), "Failed to install Kgateway CRDs")

		By("installing Kgateway controller")
		Expect(utils.InstallKgateway(kgatewayValuesPath)).To(Succeed(), "Failed to install Kgateway controller")
	default:
		Fail(fmt.Sprintf("Unsupported GATEWAY_CLASS_NAME=%s. Supported values are: istio, kgateway", gatewayClass))
	}

	if envValue := os.Getenv("KEDA_ENABLED"); envValue != "" {
		kedaEnabled = envValue
		_, _ = fmt.Fprintf(GinkgoWriter, "KEDA_ENABLED set to %s via environment variable\n", kedaEnabled)
	} else {
		By("checking if KEDA is already installed")
		if utils.IsKEDAInstalled() {
			kedaEnabled = "false"
			_, _ = fmt.Fprintf(GinkgoWriter, "KEDA is already installed in the cluster. Disabling KEDA in MIF chart to avoid conflicts.\n")
		} else {
			kedaEnabled = "true"
			_, _ = fmt.Fprintf(GinkgoWriter, "KEDA is not installed. Enabling KEDA in MIF chart.\n")
		}
	}

	if envValue := os.Getenv("LWS_ENABLED"); envValue != "" {
		lwsEnabled = envValue
		_, _ = fmt.Fprintf(GinkgoWriter, "LWS_ENABLED set to %s via environment variable\n", lwsEnabled)
	} else {
		By("checking if LWS is already installed")
		if utils.IsLWSInstalled() {
			lwsEnabled = "false"
			_, _ = fmt.Fprintf(GinkgoWriter, "LWS is already installed in the cluster. Disabling LWS in MIF chart to avoid conflicts.\n")
		} else {
			lwsEnabled = "true"
			_, _ = fmt.Fprintf(GinkgoWriter, "LWS is not installed. Enabling LWS in MIF chart.\n")
		}
	}

	if envValue := os.Getenv("ODIN_CRD_ENABLED"); envValue != "" {
		odinCRDEnabled = envValue
		_, _ = fmt.Fprintf(GinkgoWriter, "ODIN_CRD_ENABLED set to %s via environment variable\n", odinCRDEnabled)
	} else {
		By("checking if Odin CRD is already installed")
		if utils.IsOdinCRDInstalled() {
			odinCRDEnabled = "false"
			_, _ = fmt.Fprintf(GinkgoWriter, "Odin CRD is already installed in the cluster. Disabling Odin CRD in MIF chart to avoid conflicts.\n")
		} else {
			odinCRDEnabled = "true"
			_, _ = fmt.Fprintf(GinkgoWriter, "Odin CRD is not installed. Enabling Odin CRD in MIF chart.\n")
		}
	}

	if envValue := os.Getenv("PROMETHEUS_STACK_ENABLED"); envValue != "" {
		prometheusStackEnabled = envValue
		_, _ = fmt.Fprintf(GinkgoWriter, "PROMETHEUS_STACK_ENABLED set to %s via environment variable\n", prometheusStackEnabled)
	} else {
		By("checking if Prometheus is already installed")
		if utils.IsPrometheusInstalled() {
			prometheusStackEnabled = "false"
			_, _ = fmt.Fprintf(GinkgoWriter, "Prometheus is already installed in the cluster. Disabling Prometheus Stack in MIF chart to avoid conflicts.\n")
		} else {
			prometheusStackEnabled = "false"
			_, _ = fmt.Fprintf(GinkgoWriter, "Prometheus Stack disabled by default in E2E tests to avoid resource issues. Set PROMETHEUS_STACK_ENABLED=true to enable.\n")
		}
	}
})

var _ = AfterSuite(func() {
	cleanupKindCluster()

	if !isUsingKindCluster {
		_, _ = fmt.Fprintf(GinkgoWriter, "Using existing cluster (kubeconfig). Skipping resource cleanup for safety.\n")
		return
	}

	if !skipCleanup {
		By("ensuring test namespace is cleaned up")
		cleanupTestNamespace()

		if !skipPresetDeploy {
			By("uninstalling moai-inference-preset")
			utils.UninstallMIFPreset(testNamespace)
		}

		if !skipMIFDeploy {
			By("uninstalling MIF")
			cmd := exec.Command("helm", "uninstall", "moai-inference-framework", "-n", testNamespace, "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		}
	}

	if !skipCertManagerInstall && !isCertManagerAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Uninstalling CertManager...\n")
		utils.UninstallCertManager()
	}
})

func cleanupTestNamespace() {
	namespace := getEnvOrDefault("NAMESPACE", "mif")
	inferenceServiceName := "pd-disaggregation-test"
	helmReleaseName := "moai-inference-framework"

	By(fmt.Sprintf("cleaning up test resources in namespace %s", namespace))

	By("deleting InferenceService")
	cmd := exec.Command("kubectl", "delete", "inferenceservice", inferenceServiceName,
		"-n", namespace, "--ignore-not-found=true")
	_, _ = utils.Run(cmd)

	By("uninstalling MIF Helm release")
	cmd = exec.Command("helm", "uninstall", helmReleaseName, "-n", namespace, "--ignore-not-found=true")
	_, _ = utils.Run(cmd)

	By("deleting test namespace")
	cmd = exec.Command("kubectl", "delete", "ns", namespace, "--timeout=60s", "--ignore-not-found=true")
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

func checkPrerequisites() {
	requiredTools := []string{"kubectl", "helm"}
	if !skipKindCreate {
		requiredTools = append(requiredTools, "kind")
		cmd := exec.Command("which", "kind")
		if err := cmd.Run(); err != nil {
			Fail(fmt.Sprintf("kind is required but not available. Install kind or set SKIP_KIND_CREATE=true to use existing cluster"))
		}
	}
	for _, tool := range requiredTools {
		cmd := exec.Command("which", tool)
		if err := cmd.Run(); err != nil {
			Fail(fmt.Sprintf("Required tool %s is not available", tool))
		}
	}

	if !skipKindCreate {
		return
	}

	cmd := exec.Command("kubectl", "cluster-info")
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Cannot connect to Kubernetes cluster. Please check your kubeconfig.")
}

func createMIFValuesFile(awsAccessKeyID, awsSecretAccessKey string) (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	valuesContent := fmt.Sprintf(`ecrTokenRefresher:
  aws:
    accessKeyId: %s
    secretAccessKey: %s
`, awsAccessKeyID, awsSecretAccessKey)

	valuesPath := filepath.Join(projectDir, "test/e2e/moai-inference-framework-values.yaml")
	err = os.WriteFile(valuesPath, []byte(valuesContent), 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write moai-inference-framework values file: %w", err)
	}

	return valuesPath, nil
}

func createKgatewayValuesFile() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	valuesContent := `inferenceExtension:
  enabled: true
`

	valuesPath := filepath.Join(projectDir, "test/e2e/kgateway-values.yaml")
	err = os.WriteFile(valuesPath, []byte(valuesContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write Kgateway values file: %w", err)
	}

	return valuesPath, nil
}

func createIstiodValuesFile() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	valuesContent := `pilot:
  env:
    PILOT_ENABLE_ALPHA_GATEWAY_API: "true"
    ENABLE_GATEWAY_API_INFERENCE_EXTENSION: "true"
`

	valuesPath := filepath.Join(projectDir, "test/e2e/istiod-values.yaml")
	err = os.WriteFile(valuesPath, []byte(valuesContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write istiod values file: %w", err)
	}

	return valuesPath, nil
}

func ensureECRTokenRefresherSecret(namespace string) {
	cronJobName := "moai-inference-framework-ecr-token-refresher"
	jobName := "ecr-token-refresher-init-manual"
	secretName := "moreh-registry"
	ecrCredsSecretName := "moai-inference-framework-ecr-token-refresher"

	By(fmt.Sprintf("waiting for secret %s to be created", ecrCredsSecretName))
	Eventually(func() bool {
		cmd := exec.Command("kubectl", "get", "secret", ecrCredsSecretName, "-n", namespace)
		_, err := utils.Run(cmd)
		return err == nil
	}, "2m", "5s").Should(BeTrue(), fmt.Sprintf("Secret %s should be created", ecrCredsSecretName))

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
	}, "2m", "5s").Should(BeTrue(), fmt.Sprintf("CronJob %s should be created", cronJobName))

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
	}, "3m", "5s").Should(BeTrue(), fmt.Sprintf("Job %s should complete successfully", jobName))

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
	}, "1m", "2s").Should(BeTrue(), fmt.Sprintf("Secret %s should be created", secretName))

	_, _ = fmt.Fprintf(GinkgoWriter, "Successfully created secret %s via ecrTokenRefresher\n", secretName)
}

func createHeimdallValuesFile() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	valuesContent := fmt.Sprintf(`global:
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
  gatewayClassName: %s

serviceMonitor:
  labels:
    release: prometheus-stack
`, gatewayClass)

	valuesPath := filepath.Join(projectDir, "test/e2e/heimdall-values.yaml")
	err = os.WriteFile(valuesPath, []byte(valuesContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write Heimdall values file: %w", err)
	}

	return valuesPath, nil
}

func waitForMIFComponents() {
	By("waiting for Odin controller deployment")
	cmd := exec.Command("kubectl", "get", "deployment",
		"-n", testNamespace,
		"-l", "app.kubernetes.io/name=odin",
		"-o", "jsonpath={.items[0].metadata.name}")
	output, err := utils.Run(cmd)
	if err != nil || strings.TrimSpace(output) == "" {
		cmd = exec.Command("kubectl", "get", "deployment",
			"-n", testNamespace,
			"-o", "jsonpath={.items[?(@.metadata.name=~\"odin.*\")].metadata.name}")
		output, err = utils.Run(cmd)
		if err != nil || strings.TrimSpace(output) == "" {
			_, _ = fmt.Fprintf(GinkgoWriter, "Warning: Could not find Odin deployment by label, trying common name pattern\n")
			cmd = exec.Command("kubectl", "wait", "--for=condition=Available",
				"deployment/moai-inference-framework-odin",
				"-n", testNamespace,
				"--timeout=5m")
			_, _ = utils.Run(cmd)
			return
		}
	}
	odinDeploymentName := strings.TrimSpace(output)
	_, _ = fmt.Fprintf(GinkgoWriter, "Found Odin deployment: %s\n", odinDeploymentName)

	cmd = exec.Command("kubectl", "wait", "--for=condition=Available",
		fmt.Sprintf("deployment/%s", odinDeploymentName),
		"-n", testNamespace,
		"--timeout=10m")
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
		"-n", testNamespace,
		"--timeout=10m")
	output, err = utils.Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Warning: Some pods may not be ready yet: %v\n", err)
		cmd = exec.Command("kubectl", "get", "pods",
			"-n", testNamespace,
			"--field-selector=status.phase!=Succeeded",
			"-o", "wide")
		statusOutput, _ := utils.Run(cmd)
		_, _ = fmt.Fprintf(GinkgoWriter, "Current pod status:\n%s\n", statusOutput)
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "All pods are ready\n")
	}
}
