package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2" // nolint:revive,staticcheck
)

const (
	certmanagerVersion   = "v1.18.4"
	certmanagerChart     = "oci://quay.io/jetstack/charts/cert-manager"
	certmanagerNamespace = "cert-manager"
)

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) (string, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %q\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %q\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%q failed with error %q: %w", command, string(output), err)
	}

	return string(output), nil
}

// UninstallCertManager uninstalls the cert manager using Helm
func UninstallCertManager() {
	cmd := exec.Command("helm", "uninstall", "cert-manager",
		"--namespace", certmanagerNamespace,
		"--ignore-not-found")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}

	kubeSystemLeases := []string{
		"cert-manager-cainjector-leader-election",
		"cert-manager-controller",
	}
	for _, lease := range kubeSystemLeases {
		cmd = exec.Command("kubectl", "delete", "lease", lease,
			"-n", "kube-system", "--ignore-not-found", "--force", "--grace-period=0")
		if _, err := Run(cmd); err != nil {
			warnError(err)
		}
	}
}

// InstallCertManager installs cert-manager using Helm
func InstallCertManager() error {
	helmArgs := []string{
		"upgrade", "--install", "cert-manager",
		certmanagerChart,
		"--version", certmanagerVersion,
		"--namespace", certmanagerNamespace,
		"--create-namespace",
		"--set", "crds.enabled=true",
		"--wait",
		"--timeout", "5m",
	}
	cmd := exec.Command("helm", helmArgs...)
	if _, err := Run(cmd); err != nil {
		return err
	}

	cmd = exec.Command("kubectl", "wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", certmanagerNamespace,
		"--timeout", "5m",
	)

	_, err := Run(cmd)
	return err
}

func IsCertManagerCRDsInstalled() bool {
	certManagerCRDs := []string{
		"certificates.cert-manager.io",
		"issuers.cert-manager.io",
		"clusterissuers.cert-manager.io",
		"certificaterequests.cert-manager.io",
		"orders.acme.cert-manager.io",
		"challenges.acme.cert-manager.io",
	}

	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	crdList := GetNonEmptyLines(output)
	foundCRDs := make(map[string]bool, len(certManagerCRDs))
	for _, crd := range certManagerCRDs {
		foundCRDs[crd] = false
	}

	for _, line := range crdList {
		for _, crd := range certManagerCRDs {
			if strings.Contains(line, crd) {
				foundCRDs[crd] = true
				break
			}
		}
	}

	for _, found := range foundCRDs {
		if !found {
			return false
		}
	}

	return true
}

func IsKEDAInstalled() bool {
	kedaCRDs := []string{
		"scaledobjects.keda.sh",
		"scaledjobs.keda.sh",
		"cloudeventsources.eventing.keda.sh",
		"clustertriggerauthentications.keda.sh",
	}

	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	crdList := GetNonEmptyLines(output)
	foundCRDs := make(map[string]bool, len(kedaCRDs))
	for _, crd := range kedaCRDs {
		foundCRDs[crd] = false
	}

	for _, line := range crdList {
		for _, crd := range kedaCRDs {
			if strings.Contains(line, crd) {
				foundCRDs[crd] = true
				break
			}
		}
	}

	allCRDsPresent := true
	for _, found := range foundCRDs {
		if !found {
			allCRDsPresent = false
			break
		}
	}

	if allCRDsPresent {
		return true
	}

	cmd = exec.Command("helm", "list", "-A", "-q", "-f", "^keda$")
	output, err = Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}

	return false
}

func IsLWSInstalled() bool {
	lwsCRDs := []string{
		"leaderworkersets.leaderworkerset.x-k8s.io",
	}

	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	crdList := GetNonEmptyLines(output)
	allCRDsPresent := true
	for _, crd := range lwsCRDs {
		found := false
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				found = true
				break
			}
		}
		if !found {
			allCRDsPresent = false
			break
		}
	}

	if allCRDsPresent {
		return true
	}

	cmd = exec.Command("helm", "list", "-A", "-q", "-f", "^lws$")
	output, err = Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}

	return false
}

func IsOdinCRDInstalled() bool {
	odinCRDs := []string{
		"inferenceservices.odin.moreh.io",
		"inferenceservicetemplates.odin.moreh.io",
	}

	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	crdList := GetNonEmptyLines(output)
	crdFound := make(map[string]bool, len(odinCRDs))
	for _, line := range crdList {
		for _, crd := range odinCRDs {
			if strings.Contains(line, crd) {
				crdFound[crd] = true
			}
		}
	}

	allFound := true
	for _, crd := range odinCRDs {
		if !crdFound[crd] {
			allFound = false
			break
		}
	}

	if allFound {
		return true
	}

	cmd = exec.Command("helm", "list", "-A", "-q", "-f", "^odin-crd$")
	output, err = Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}

	return false
}

func IsPrometheusInstalled() bool {
	prometheusCRDs := []string{
		"prometheuses.monitoring.coreos.com",
		"prometheusrules.monitoring.coreos.com",
		"servicemonitors.monitoring.coreos.com",
	}

	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	crdList := GetNonEmptyLines(output)
	foundCRDs := make(map[string]bool, len(prometheusCRDs))
	for _, crd := range prometheusCRDs {
		foundCRDs[crd] = false
	}

	for _, line := range crdList {
		for crd := range foundCRDs {
			if strings.Contains(line, crd) {
				foundCRDs[crd] = true
			}
		}
	}

	allPresent := true
	for _, present := range foundCRDs {
		if !present {
			allPresent = false
			break
		}
	}

	if allPresent {
		return true
	}

	cmd = exec.Command("helm", "list", "-A", "-q", "-f", "^kube-prometheus-stack$|^prometheus-stack$")
	output, err = Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}

	cmd = exec.Command("kubectl", "get", "prometheus", "-A", "--no-headers")
	output, err = Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}

	return false
}

func IsMIFInstalled(namespace string) bool {
	cmd := exec.Command("helm", "list", "-n", namespace, "-q", "-f", "^moai-inference-framework$")
	output, err := Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}
	return false
}

func IsPresetInstalled(namespace string) bool {
	cmd := exec.Command("helm", "list", "-n", namespace, "-q", "-f", "^moai-inference-preset$")
	output, err := Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}
	return false
}

func IsGatewayAPIInstalled() bool {
	cmd := exec.Command("kubectl", "get", "crd", "gateways.gateway.networking.k8s.io", "--ignore-not-found")
	output, err := Run(cmd)
	if err == nil && strings.Contains(output, "gateways.gateway.networking.k8s.io") {
		return true
	}
	return false
}

func IsGatewayInferenceExtensionInstalled() bool {
	cmd := exec.Command("kubectl", "get", "crd", "inferencepools.inference.networking.k8s.io", "--ignore-not-found")
	output, err := Run(cmd)
	if err == nil && strings.Contains(output, "inferencepools.inference.networking.k8s.io") {
		return true
	}
	return false
}

func IsIstioInstalled() bool {
	cmd := exec.Command("helm", "list", "-n", "istio-system", "-q", "-f", "^istiod$|^istio-base$")
	output, err := Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}
	return false
}

func IsKgatewayInstalled() bool {
	cmd := exec.Command("helm", "list", "-n", "kgateway-system", "-q", "-f", "^kgateway$|^kgateway-crds$")
	output, err := Run(cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return true
	}
	return false
}

func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, fmt.Errorf("failed to get current working directory: %w", err)
	}

	dir := wd
	for {
		if _, err := os.Stat(dir + "/go.mod"); err == nil {
			return dir, nil
		}
		parent := dir + "/.."
		parentAbs, err := filepath.Abs(parent)
		if err != nil {
			return wd, fmt.Errorf("failed to resolve parent directory: %w", err)
		}
		if parentAbs == dir {
			return wd, fmt.Errorf("go.mod not found in any parent directory")
		}
		dir = parentAbs
	}
}

func CreateKindCluster(clusterName string) error {
	k8sVersion := os.Getenv("KIND_K8S_VERSION")
	args := []string{"create", "cluster", "--name", clusterName, "-v", "1"}
	if k8sVersion != "" {
		nodeImage := fmt.Sprintf("kindest/node:%s", k8sVersion)
		args = append(args, "--image", nodeImage)
	}

	cmd := exec.Command("kind", args...)
	dir, _ := GetProjectDir()
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter

	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %q\n", command)
	var err error
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("failed to create kind cluster: %w", err)
	}

	cmd = exec.Command("kind", "export", "kubeconfig", "--name", clusterName)
	_, err = Run(cmd)
	if err != nil {
		return fmt.Errorf("failed to export kubeconfig for kind cluster %s: %w", clusterName, err)
	}

	contextName := fmt.Sprintf("kind-%s", clusterName)
	cmd = exec.Command("kubectl", "cluster-info", "--context", contextName)
	_, err = Run(cmd)
	if err != nil {
		return fmt.Errorf("failed to verify kubectl context %s for kind cluster %s: %w", contextName, clusterName, err)
	}

	return nil
}

func DeleteKindCluster(clusterName string) error {
	cmd := exec.Command("kind", "delete", "cluster", "--name", clusterName)
	_, err := Run(cmd)
	return err
}

func IsKindClusterExists(clusterName string) bool {
	cmd := exec.Command("kind", "get", "clusters")
	output, err := Run(cmd)
	if err != nil {
		return false
	}
	clusters := GetNonEmptyLines(output)
	for _, cluster := range clusters {
		if strings.TrimSpace(cluster) == clusterName {
			return true
		}
	}
	return false
}

func InstallHeimdall(namespace string, valuesPath string) error {
	By("adding moreh Helm repository")
	cmd := exec.Command("helm", "repo", "add", "moreh", "https://moreh-dev.github.io/helm-charts")
	if _, err := Run(cmd); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to add moreh helm repo: %w", err)
		}
	}

	By("updating moreh Helm repository")
	cmd = exec.Command("helm", "repo", "update", "moreh")
	if _, err := Run(cmd); err != nil {
		return fmt.Errorf("failed to update moreh helm repo: %w", err)
	}

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
	cmd = exec.Command("helm", helmArgs...)
	_, err := Run(cmd)
	return err
}

func UninstallHeimdall(namespace string) error {
	cmd := exec.Command("helm", "uninstall", "heimdall", "-n", namespace, "--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}

func DeployMIFPreset(namespace string, chartPath string) error {
	By("deploying moai-inference-preset")
	helmArgs := []string{
		"upgrade", "--install", "moai-inference-preset",
		chartPath,
		"--namespace", namespace,
		"--wait",
		"--timeout", "10m",
	}
	cmd := exec.Command("helm", helmArgs...)
	_, err := Run(cmd)
	return err
}

func UninstallMIFPreset(namespace string) error {
	cmd := exec.Command("helm", "uninstall", "moai-inference-preset", "-n", namespace, "--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}

func InstallGatewayAPI() error {
	cmd := exec.Command("kubectl", "apply", "--server-side",
		"-f", "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml")
	_, err := Run(cmd)
	return err
}

func InstallGatewayInferenceExtension() error {
	cmd := exec.Command("kubectl", "apply",
		"-f", "https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v1.1.0/manifests.yaml")
	_, err := Run(cmd)
	return err
}

func InstallKgatewayCRDs() error {
	cmd := exec.Command("helm", "upgrade", "-i", "kgateway-crds",
		"oci://cr.kgateway.dev/kgateway-dev/charts/kgateway-crds",
		"--version", "v2.1.1",
		"-n", "kgateway-system",
		"--create-namespace")
	_, err := Run(cmd)
	return err
}

func InstallKgateway(valuesPath string) error {
	helmArgs := []string{
		"upgrade", "-i", "kgateway",
		"oci://cr.kgateway.dev/kgateway-dev/charts/kgateway",
		"--version", "v2.1.1",
		"-n", "kgateway-system",
	}
	if valuesPath != "" {
		helmArgs = append(helmArgs, "-f", valuesPath)
	}
	cmd := exec.Command("helm", helmArgs...)
	_, err := Run(cmd)
	return err
}

func InstallIstioBase() error {
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
	_, err := Run(cmd)
	return err
}

func InstallIstiod(valuesPath string) error {
	helmArgs := []string{
		"upgrade", "-i", "istiod", "istio/istiod",
		"--version", "1.28.1",
		"-n", "istio-system",
	}
	if valuesPath != "" {
		helmArgs = append(helmArgs, "-f", valuesPath)
	}
	cmd := exec.Command("helm", helmArgs...)
	_, err := Run(cmd)
	return err
}

func InstallInferenceService(namespace, valuesPath string) error {
	if valuesPath == "" {
		return fmt.Errorf("inference service manifest path is required (e.g., path/to/manifest.yaml)")
	}

	kubectlArgs := []string{
		"apply",
		"-f", valuesPath,
	}

	if namespace != "" {
		kubectlArgs = append(kubectlArgs, "-n", namespace)
	}

	cmd := exec.Command("kubectl", kubectlArgs...)
	_, err := Run(cmd)
	return err
}

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

func UninstallGatewayInferenceExtension() error {
	cmd := exec.Command("kubectl", "delete",
		"-f", "https://github.com/kubernetes-sigs/gateway-api-inference-extension/releases/download/v1.1.0/manifests.yaml",
		"--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}

func UninstallGatewayAPI() error {
	cmd := exec.Command("kubectl", "delete",
		"-f", "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml",
		"--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}
