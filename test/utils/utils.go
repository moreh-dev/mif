//go:build e2e
// +build e2e

package utils

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
)

// SetupInterruptHandler sets up signal handlers for graceful shutdown.
func SetupInterruptHandler() func() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})

	go func() {
		select {
		case sig := <-sigChan:
			_, _ = fmt.Fprintf(GinkgoWriter, "\nReceived signal: %v. Initiating immediate cleanup...\n", sig)
			CleanupE2ETempFiles()
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

// DeleteNamespace deletes a namespace, with fallback to force delete by removing finalizers if needed.
func DeleteNamespace(namespace string) error {
	By(fmt.Sprintf("deleting namespace %s", namespace))
	cmd := exec.Command("kubectl", "delete", "ns", namespace, "--timeout=60s", "--ignore-not-found=true")
	output, err := Run(cmd)
	if err != nil {
		By("attempting to force delete namespace by removing finalizers")
		cmd = exec.Command("kubectl", "patch", "namespace", namespace,
			"--type=json", "-p", `[{"op": "replace", "path": "/spec/finalizers", "value": []}]`, "--ignore-not-found=true")
		_, _ = Run(cmd)

		cmd = exec.Command("kubectl", "delete", "ns", namespace, "--timeout=30s", "--ignore-not-found=true")
		_, err = Run(cmd)
		if err != nil {
			return fmt.Errorf("failed to delete namespace: %w", err)
		}
	} else if output != "" {
		_, _ = fmt.Fprintf(GinkgoWriter, "Namespace deletion output: %s\n", output)
	}

	return nil
}

// CleanupE2ETempFiles removes temporary files created during E2E tests.
func CleanupE2ETempFiles() {
	projectDir, err := GetProjectDir()
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
		tempFileInferenceServicePrefill,
		tempFileInferenceServiceDecode,
		tempFileInferenceServiceTemplateCommon,
		tempFileInferenceServiceTemplatePrefillMeta,
		tempFileInferenceServiceTemplateDecodeMeta,
		tempFileInferenceServiceTemplateDecodeProxy,
	}

	for _, rel := range tempFiles {
		path := filepath.Join(projectDir, rel)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			_, _ = fmt.Fprintf(GinkgoWriter, "warning: failed to remove temp file %s: %v\n", path, err)
		}
	}
}

// writeValuesFile is a helper function to write Helm values files.
func writeValuesFile(relativePath, content string, mode os.FileMode) (string, error) {
	projectDir, err := GetProjectDir()
	if err != nil {
		return "", err
	}

	valuesPath := filepath.Join(projectDir, relativePath)
	if err := ensureParentDir(valuesPath); err != nil {
		return "", err
	}
	if err := os.WriteFile(valuesPath, []byte(content), mode); err != nil {
		return "", fmt.Errorf("failed to write values file %s: %w", relativePath, err)
	}

	return valuesPath, nil
}

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
	projectDir, err := GetProjectDir()
	if err != nil {
		return "", fmt.Errorf("failed to get project directory: %w", err)
	}

	templatePath := filepath.Join(projectDir, "test", "templates", filename)
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

func getInferenceImageInfo() (repo, tag string) {
	repoDefault := imageRepoDefault
	tagDefault := imageTagDefault
	if cfg.IsUsingKindCluster {
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
