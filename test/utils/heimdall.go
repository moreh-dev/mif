//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// createHeimdallValuesFile creates a values file for Heimdall.
func createHeimdallValuesFile() (string, error) {
	serviceMonitorSection := "serviceMonitor:\n  enabled: false\n"
	if cfg.prometheusStackEnabled {
		serviceMonitorSection = fmt.Sprintf(`serviceMonitor:
  enabled: true
  labels:
    release: %s
`, helmReleaseMIF)
	}

	baseYAML := fmt.Sprintf(`global:
  imagePullSecrets:
    - name: %s

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
  name: %s
  gatewayClassName: %s

inferencePool:
  targetPorts:
    - number: 8000

%s`, secretNameMorehRegistry, gatewayName, cfg.gatewayClass, serviceMonitorSection)

	return writeValuesFile(tempFileHeimdallValues, baseYAML, 0644)
}

// InstallHeimdallForTest installs Heimdall for test runs.
func InstallHeimdallForTest() {
	By("creating Heimdall values file")
	heimdallValuesPath, err := createHeimdallValuesFile()
	Expect(err).NotTo(HaveOccurred(), "Failed to create Heimdall values file")

	By("installing Heimdall")
	Expect(InstallHeimdall(cfg.workloadNamespace, heimdallValuesPath)).To(Succeed(), "Failed to install Heimdall")

	By("waiting for Heimdall deployment to be ready")
	cmd := exec.Command("kubectl", "wait", "deployment",
		"-l", "app.kubernetes.io/instance=heimdall",
		"--for=condition=Available",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", TimeoutLong))
	_, err = Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Heimdall deployment not available")
}

// InstallHeimdall installs Heimdall in the given namespace.
func InstallHeimdall(namespace string, valuesPath string) error {
	By("installing Heimdall")
	helmArgs := []string{
		"upgrade", "--install", "heimdall",
		"moreh/heimdall",
		"--version", "v0.6.0",
		"--namespace", namespace,
		"--create-namespace",
	}
	if valuesPath != "" {
		helmArgs = append(helmArgs, "-f", valuesPath)
	}
	cmd := exec.Command("helm", helmArgs...)
	_, err := Run(cmd)
	return err
}

// UninstallHeimdall uninstalls Heimdall from the given namespace.
func UninstallHeimdall(namespace string) error {
	cmd := exec.Command("helm", "uninstall", "heimdall", "-n", namespace, "--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}
