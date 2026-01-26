//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type InferenceServiceData struct {
	Name            string
	Namespace       string
	ImagePullSecret string
	Image           string
	Model           string
	HFToken         string
	HFEndpoint      string
	IsKind          bool
}

func getInferenceServiceData() InferenceServiceData {
	imageRepo, imageTag := getInferenceImageInfo()
	image := fmt.Sprintf("%s:%s", imageRepo, imageTag)

	return InferenceServiceData{
		Name:            inferenceServiceName,
		Namespace:       cfg.workloadNamespace,
		ImagePullSecret: secretNameMorehRegistry,
		Image:           image,
		Model:           cfg.testModel,
		HFToken:         cfg.hfToken,
		HFEndpoint:      cfg.hfEndpoint,
		IsKind:          cfg.IsUsingKindCluster,
	}
}

// CreateInferenceService creates an InferenceService CR in the given namespace.
func CreateInferenceService(namespace, valuesPath string) error {
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

// DeleteInferenceService deletes an InferenceService from the given namespace.
func DeleteInferenceService(namespace, inferenceServiceName string) error {
	By("deleting InferenceService")
	cmd := exec.Command("kubectl", "delete", "inferenceservice", inferenceServiceName,
		"-n", namespace, "--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}

func createPrefillInferenceServiceManifest() (string, error) {
	projectDir, err := GetProjectDir()
	if err != nil {
		return "", err
	}

	rendered, err := renderTemplateFile("inference-service-prefill.yaml.tmpl", getInferenceServiceData())
	if err != nil {
		return "", fmt.Errorf("failed to render prefill InferenceService manifest: %w", err)
	}

	manifestPath := filepath.Join(projectDir, tempFileInferenceServicePrefill)
	if err := ensureParentDir(manifestPath); err != nil {
		return "", err
	}
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write prefill InferenceService manifest file: %w", err)
	}

	return manifestPath, nil
}

func createDecodeInferenceServiceManifest() (string, error) {
	projectDir, err := GetProjectDir()
	if err != nil {
		return "", err
	}

	rendered, err := renderTemplateFile("inference-service-decode.yaml.tmpl", getInferenceServiceData())
	if err != nil {
		return "", fmt.Errorf("failed to render decode InferenceService manifest: %w", err)
	}

	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceDecode)
	if err := ensureParentDir(manifestPath); err != nil {
		return "", err
	}
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write decode InferenceService manifest file: %w", err)
	}

	return manifestPath, nil
}

// InstallPrefillDecodeInferenceServicesForTest installs test InferenceServices.
func InstallPrefillDecodeInferenceServicesForTest() {
	By("creating common InferenceServiceTemplate")
	commonTemplatePath, err := createCommonTemplate()
	Expect(err).NotTo(HaveOccurred(), "Failed to create common template")

	By("creating prefill meta InferenceServiceTemplate")
	prefillMetaTemplatePath, err := createPrefillMetaTemplate()
	Expect(err).NotTo(HaveOccurred(), "Failed to create prefill meta template")

	By("creating decode meta InferenceServiceTemplate")
	decodeMetaTemplatePath, err := createDecodeMetaTemplate()
	Expect(err).NotTo(HaveOccurred(), "Failed to create decode meta template")

	By("creating decode proxy InferenceServiceTemplate")
	decodeProxyTemplatePath, err := createDecodeProxyTemplate()
	Expect(err).NotTo(HaveOccurred(), "Failed to create decode proxy template")

	By("applying InferenceServiceTemplates")
	Expect(CreateInferenceServiceTemplate(cfg.workloadNamespace, commonTemplatePath)).To(Succeed(), "Failed to apply common template")
	Expect(CreateInferenceServiceTemplate(cfg.workloadNamespace, prefillMetaTemplatePath)).To(Succeed(), "Failed to apply prefill meta template")
	Expect(CreateInferenceServiceTemplate(cfg.workloadNamespace, decodeMetaTemplatePath)).To(Succeed(), "Failed to apply decode meta template")
	Expect(CreateInferenceServiceTemplate(cfg.workloadNamespace, decodeProxyTemplatePath)).To(Succeed(), "Failed to apply decode proxy template")

	By("creating prefill InferenceService manifest file")
	prefillManifestPath, err := createPrefillInferenceServiceManifest()
	Expect(err).NotTo(HaveOccurred(), "Failed to create prefill InferenceService manifest file")

	By("creating decode InferenceService manifest file")
	decodeManifestPath, err := createDecodeInferenceServiceManifest()
	Expect(err).NotTo(HaveOccurred(), "Failed to create decode InferenceService manifest file")

	By("creating prefill InferenceService")
	Expect(CreateInferenceService(cfg.workloadNamespace, prefillManifestPath)).To(Succeed(), "Failed to create prefill InferenceService")

	By("creating decode InferenceService")
	Expect(CreateInferenceService(cfg.workloadNamespace, decodeManifestPath)).To(Succeed(), "Failed to create decode InferenceService")

	By("waiting for prefill InferenceService pods to be created")
	Eventually(func() (string, error) {
		checkCmd := exec.Command("kubectl", "get", "pods",
			"-l", fmt.Sprintf("app.kubernetes.io/name=%s-prefill", inferenceServiceName),
			"-n", cfg.workloadNamespace,
			"-o", "name")
		return Run(checkCmd)
	}, TimeoutLong, IntervalShort).ShouldNot(BeEmpty())

	By("waiting for prefill InferenceService pods to be ready")
	cmd := exec.Command("kubectl", "wait", "pod",
		"-l", fmt.Sprintf("app.kubernetes.io/name=%s-prefill", inferenceServiceName),
		"--for=condition=Ready",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", TimeoutVeryLong))
	_, err = Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Prefill InferenceService pods not ready")

	By("waiting for decode InferenceService pods to be created")
	Eventually(func() (string, error) {
		checkCmd := exec.Command("kubectl", "get", "pods",
			"-l", fmt.Sprintf("app.kubernetes.io/name=%s-decode", inferenceServiceName),
			"-n", cfg.workloadNamespace,
			"-o", "name")
		return Run(checkCmd)
	}, TimeoutLong, IntervalShort).ShouldNot(BeEmpty())

	By("waiting for decode InferenceService pods to be ready")
	cmd = exec.Command("kubectl", "wait", "pod",
		"-l", fmt.Sprintf("app.kubernetes.io/name=%s-decode", inferenceServiceName),
		"--for=condition=Ready",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", TimeoutVeryLong))
	_, err = Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Decode InferenceService pods not ready")
}

// getGatewayServiceName gets the name of the Gateway service in the workload namespace.
func getGatewayServiceName(timeout time.Duration, interval time.Duration) string {
	var serviceName string
	Eventually(func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "service",
			"-n", cfg.workloadNamespace,
			"-l", "gateway.networking.k8s.io/gateway-name=mif",
			"-o", "jsonpath={.items[0].metadata.name}")
		output, err := Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		serviceName = strings.TrimSpace(output)
		g.Expect(serviceName).NotTo(BeEmpty(), "Gateway service not found")
	}, timeout, interval).Should(Succeed())
	return serviceName
}

// VerifyInferenceEndpoint verifies inference endpoint reachability.
func VerifyInferenceEndpoint() {
	By("verifying Gateway service exists")
	getGatewayServiceName(TimeoutLong, IntervalLong)

	By("waiting for inference-service decode pods to be ready")
	cmd := exec.Command("kubectl", "wait", "pod",
		"-l", fmt.Sprintf("app.kubernetes.io/name=%s-decode", inferenceServiceName),
		"--for=condition=Ready",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", TimeoutVeryLong))
	_, err := Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "InferenceService decode pods not ready")
}
