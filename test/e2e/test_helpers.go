//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/moreh-dev/mif/test/utils"
)

func renderTextTemplate(templateText string, data any) (string, error) {
	t, err := template.New("manifest").Parse(templateText)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func loadTemplateFile(filename string) (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", fmt.Errorf("failed to get project directory: %w", err)
	}

	templatePath := filepath.Join(projectDir, "test", "e2e", "templates", filename)
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	return string(content), nil
}

func renderTemplateFile(filename string, data any) (string, error) {
	templateText, err := loadTemplateFile(filename)
	if err != nil {
		return "", err
	}

	return renderTextTemplate(templateText, data)
}

func verifyOdinController(g Gomega) {
	cmd := exec.Command("kubectl", "wait", "deployment",
		"-l", "app.kubernetes.io/name=odin",
		"--for=condition=Available",
		"-n", cfg.mifNamespace,
		fmt.Sprintf("--timeout=%v", timeoutLong))
	_, err := utils.Run(cmd)
	g.Expect(err).NotTo(HaveOccurred(), "Odin controller not available")
}

func verifyAllPodsReady(g Gomega) {
	cmd := exec.Command("kubectl", "wait", "pod",
		"--all",
		"--field-selector=status.phase!=Succeeded",
		"--for=condition=Ready",
		"-n", cfg.mifNamespace,
		fmt.Sprintf("--timeout=%v", timeoutVeryLong))
	_, err := utils.Run(cmd)
	g.Expect(err).NotTo(HaveOccurred(), "Some pods are not ready")
}

func getInferenceImageInfo() (repo, tag string) {
	repoDefault := imageRepoDefault
	tagDefault := imageTagDefault
	if cfg.isUsingKindCluster {
		repoDefault = imageRepoKindDefault
		tagDefault = imageTagKindDefault
	}

	repo = cfg.inferenceImageRepo
	if repo == "" {
		repo = repoDefault
	}

	tag = cfg.inferenceImageTag
	if tag == "" {
		tag = tagDefault
	}

	return repo, tag
}

func collectDebugInfo() {
	By("fetching pod logs")
	cmd := exec.Command("kubectl", "get", "pods",
		"-n", cfg.mifNamespace,
		"-o", "jsonpath={.items[*].metadata.name}")
	output, err := utils.Run(cmd)
	if err == nil {
		podNames := strings.Fields(output)
		for _, podName := range podNames {
			cmd = exec.Command("kubectl", "logs", podName, "-n", cfg.mifNamespace, "--all-containers=true", "--tail=100")
			logs, logErr := utils.Run(cmd)
			if logErr == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Pod %s logs:\n%s\n", podName, logs)
			}
		}
	}

	By("fetching Kubernetes events")
	cmd = exec.Command("kubectl", "get", "events", "-n", cfg.mifNamespace, "--sort-by=.lastTimestamp")
	eventsOutput, err := utils.Run(cmd)
	if err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s\n", eventsOutput)
	}

	By("fetching resource status")
	cmd = exec.Command("kubectl", "get", "all", "-n", cfg.mifNamespace)
	allOutput, err := utils.Run(cmd)
	if err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "All resources:\n%s\n", allOutput)
	}
}

func createWorkloadNamespace() {
	By("creating workload namespace")
	cmd := exec.Command("kubectl", "create", "ns", cfg.workloadNamespace, "--request-timeout=30s")
	_, err := utils.Run(cmd)
	if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
		Expect(err).NotTo(HaveOccurred(), "Failed to create workload namespace")
	}

	if cfg.mifNamespace != cfg.workloadNamespace {
		By("adding mif=enabled label to workload namespace for automatic secret copying")
		cmd = exec.Command("kubectl", "label", "namespace", cfg.workloadNamespace,
			"mif=enabled", "--overwrite", "--request-timeout=30s")
		_, err = utils.Run(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to add mif=enabled label to namespace: %v\n", err)
		}
	}

	if cfg.istioRev != "" {
		By(fmt.Sprintf("adding istio.io/rev=%s label to workload namespace", cfg.istioRev))
		cmd = exec.Command("kubectl", "label", "namespace", cfg.workloadNamespace,
			fmt.Sprintf("istio.io/rev=%s", cfg.istioRev), "--overwrite", "--request-timeout=30s")
		_, err = utils.Run(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to add istio.io/rev label to namespace: %v\n", err)
		}
	}
}
