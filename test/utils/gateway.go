//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// InstallGatewayAPI installs Gateway API.
func InstallGatewayAPI() error {
	cmd := exec.Command("kubectl", "apply", "--server-side",
		"-f", "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml")
	_, err := Run(cmd)
	return err
}

// UninstallGatewayAPI uninstalls Gateway API.
func UninstallGatewayAPI() error {
	cmd := exec.Command("kubectl", "delete",
		"-f", "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml",
		"--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}

// InstallGatewayInferenceExtension installs Gateway Inference Extension.
func InstallGatewayInferenceExtension() error {
	cmd := exec.Command("kubectl", "apply",
		"-f", "https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v1.1.0/manifests.yaml")
	_, err := Run(cmd)
	return err
}

// UninstallGatewayInferenceExtension uninstalls Gateway Inference Extension.
func UninstallGatewayInferenceExtension() error {
	cmd := exec.Command("kubectl", "delete",
		"-f", "https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v1.1.0/manifests.yaml",
		"--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}

// InstallGatewayController installs the gateway controller for the given gateway class.
func InstallGatewayController(gatewayClass string) error {
	projectDir, err := GetProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %w", err)
	}

	switch gatewayClass {
	case "istio":
		cmd := exec.Command("helm", "repo", "add", "istio", "https://istio-release.storage.googleapis.com/charts")
		if _, err := Run(cmd); err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to add istio helm repo: %w", err)
		}

		cmd = exec.Command("helm", "repo", "update", "istio")
		if _, err := Run(cmd); err != nil {
			return fmt.Errorf("failed to update istio helm repo: %w", err)
		}

		cmd = exec.Command("helm", "upgrade", "-i", "istio-base", "istio/base",
			"--version", "1.28.1",
			"-n", "istio-system",
			"--create-namespace")
		if _, err := Run(cmd); err != nil {
			return err
		}

		cmd = exec.Command("helm", "upgrade", "-i", "istiod", "istio/istiod",
			"--version", "1.28.1",
			"-n", "istio-system",
			"-f", filepath.Join(projectDir, "test/scripts/base/istiod-values.yaml"))
		_, err := Run(cmd)
		return err

	case "kgateway":
		cmd := exec.Command("helm", "upgrade", "-i", "kgateway-crds",
			"oci://cr.kgateway.dev/kgateway-dev/charts/kgateway-crds",
			"--version", "v2.1.1",
			"-n", "kgateway-system",
			"--create-namespace")
		if _, err := Run(cmd); err != nil {
			return err
		}

		cmd = exec.Command("helm", "upgrade", "-i", "kgateway",
			"oci://cr.kgateway.dev/kgateway-dev/charts/kgateway",
			"--version", "v2.1.1",
			"-n", "kgateway-system",
			"-f", filepath.Join(projectDir, "test/scripts/base/kgateway-values.yaml"))
		_, err = Run(cmd)
		return err

	default:
		return fmt.Errorf("unsupported gateway class: %s", gatewayClass)
	}
}

// UninstallGatewayController uninstalls the gateway controller for the given gateway class.
func UninstallGatewayController(gatewayClass string) error {
	switch gatewayClass {
	case "istio":
		cmd := exec.Command("helm", "uninstall", "istiod", "-n", "istio-system", "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			return err
		}

		cmd = exec.Command("helm", "uninstall", "istio-base", "-n", "istio-system", "--ignore-not-found=true")
		_, err := Run(cmd)
		return err
	case "kgateway":
		cmd := exec.Command("helm", "uninstall", "kgateway", "-n", "kgateway-system", "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			return err
		}

		cmd = exec.Command("helm", "uninstall", "kgateway-crds", "-n", "kgateway-system", "--ignore-not-found=true")
		_, err := Run(cmd)
		return err
	default:
		return fmt.Errorf("unsupported gateway class: %s", gatewayClass)
	}
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

// ApplyGatewayResource applies Gateway resources in the workload namespace.
func ApplyGatewayResource() {
	By("applying Gateway resource and infrastructure parameters")

	var baseYAML string

	switch cfg.gatewayClass {
	case "istio":
		baseYAML = `apiVersion: v1
kind: ConfigMap
metadata:
  name: mif-gateway-infrastructure
data:
  service: |
    spec:
      type: ClusterIP
  deployment: |
    spec:
      template:
        metadata:
          annotations:
            proxy.istio.io/config: |
              accessLogFile: /dev/stdout
              accessLogEncoding: JSON
        spec:
          containers:
            - name: istio-proxy
              resources:
                limits: null

---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: mif
spec:
  gatewayClassName: istio
  infrastructure:
    parametersRef:
      group: ""
      kind: ConfigMap
      name: mif-gateway-infrastructure
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: All
`
	case "kgateway":
		baseYAML = `apiVersion: gateway.kgateway.dev/v1alpha1
kind: GatewayParameters
metadata:
  name: mif-gateway-infrastructure
spec:
  kube:
    service:
      type: ClusterIP

---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: mif
spec:
  gatewayClassName: kgateway
  infrastructure:
    parametersRef:
      group: gateway.kgateway.dev
      kind: GatewayParameters
      name: mif-gateway-infrastructure
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: All
`
	default:
		Fail(fmt.Sprintf("Unsupported gatewayClassName: %s", cfg.gatewayClass))
	}

	cmd := exec.Command("kubectl", "apply", "-f", "-", "-n", cfg.workloadNamespace, "--request-timeout=60s")
	cmd.Stdin = strings.NewReader(baseYAML)
	_, err := Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to apply Gateway resources")

	By("waiting for Gateway to be accepted")
	cmd = exec.Command("kubectl", "wait", "gateway", "mif",
		"--for=condition=Accepted",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", TimeoutLong))
	_, err = Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Gateway not accepted")

	By("waiting for Gateway pods to be created")
	Eventually(func() (string, error) {
		checkCmd := exec.Command("kubectl", "get", "pods",
			"-l", "gateway.networking.k8s.io/gateway-name=mif",
			"-n", cfg.workloadNamespace,
			"-o", "name")
		return Run(checkCmd)
	}, TimeoutLong, IntervalShort).ShouldNot(BeEmpty())

	By("waiting for Gateway pods to be ready")
	cmd = exec.Command("kubectl", "wait", "pod",
		"-l", "gateway.networking.k8s.io/gateway-name=mif",
		"--for=condition=Ready",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", TimeoutLong))
	_, err = Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Gateway pods not ready")
}

// DeleteGatewayResources deletes Gateway resources (Gateway, ConfigMap, or GatewayParameters) from the given namespace.
func DeleteGatewayResources(workloadNamespace, gatewayClass string) error {
	By("deleting Gateway resource")
	cmd := exec.Command("kubectl", "delete", "gateway", "mif",
		"-n", workloadNamespace, "--ignore-not-found=true")
	if _, err := Run(cmd); err != nil {
		warnError(fmt.Errorf("failed to delete Gateway: %w", err))
	}

	switch gatewayClass {
	case "istio":
		By("deleting Gateway infrastructure ConfigMap")
		cmd := exec.Command("kubectl", "delete", "configmap", "mif-gateway-infrastructure",
			"-n", workloadNamespace, "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			warnError(fmt.Errorf("failed to delete Gateway ConfigMap: %w", err))
		}
	case "kgateway":
		By("deleting Gateway infrastructure GatewayParameters")
		cmd := exec.Command("kubectl", "delete", "gatewayparameters", "mif-gateway-infrastructure",
			"-n", workloadNamespace, "--ignore-not-found=true")
		if _, err := Run(cmd); err != nil {
			warnError(fmt.Errorf("failed to delete Gateway GatewayParameters: %w", err))
		}
	}

	return nil
}
