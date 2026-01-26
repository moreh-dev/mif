//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
)

// CreateInferenceServiceTemplate creates an InferenceServiceTemplate CR in the given namespace.
func CreateInferenceServiceTemplate(namespace, manifestPath string) error {
	if manifestPath == "" {
		return fmt.Errorf("inference service template manifest path is required (e.g., path/to/template.yaml)")
	}

	kubectlArgs := []string{
		"apply",
		"-f", manifestPath,
	}

	if namespace != "" {
		kubectlArgs = append(kubectlArgs, "-n", namespace)
	}

	cmd := exec.Command("kubectl", kubectlArgs...)
	_, err := Run(cmd)
	return err
}

// DeleteInferenceServiceTemplate deletes an InferenceServiceTemplate from the given namespace.
func DeleteInferenceServiceTemplate(namespace, templateName string) error {
	By(fmt.Sprintf("deleting InferenceServiceTemplate %s", templateName))
	cmd := exec.Command("kubectl", "delete", "inferenceservicetemplate", templateName,
		"-n", namespace, "--ignore-not-found=true")
	_, err := Run(cmd)
	return err
}

func createCommonTemplate() (string, error) {
	projectDir, err := GetProjectDir()
	if err != nil {
		return "", err
	}

	rendered, err := renderTemplateFile("inference-service-template-common.yaml.tmpl", getInferenceServiceData())
	if err != nil {
		return "", fmt.Errorf("failed to render common template: %w", err)
	}

	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceTemplateCommon)
	if err := ensureParentDir(manifestPath); err != nil {
		return "", err
	}
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write common template file: %w", err)
	}

	return manifestPath, nil
}

func createPrefillMetaTemplate() (string, error) {
	projectDir, err := GetProjectDir()
	if err != nil {
		return "", err
	}

	rendered, err := renderTemplateFile("inference-service-template-prefill-meta.yaml.tmpl", getInferenceServiceData())
	if err != nil {
		return "", fmt.Errorf("failed to render prefill meta template: %w", err)
	}

	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceTemplatePrefillMeta)
	if err := ensureParentDir(manifestPath); err != nil {
		return "", err
	}
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write prefill meta template file: %w", err)
	}

	return manifestPath, nil
}

func createDecodeMetaTemplate() (string, error) {
	projectDir, err := GetProjectDir()
	if err != nil {
		return "", err
	}

	rendered, err := renderTemplateFile("inference-service-template-decode-meta.yaml.tmpl", getInferenceServiceData())
	if err != nil {
		return "", fmt.Errorf("failed to render decode meta template: %w", err)
	}

	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceTemplateDecodeMeta)
	if err := ensureParentDir(manifestPath); err != nil {
		return "", err
	}
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write decode meta template file: %w", err)
	}

	return manifestPath, nil
}

func createDecodeProxyTemplate() (string, error) {
	projectDir, err := GetProjectDir()
	if err != nil {
		return "", err
	}

	rendered, err := renderTemplateFile("inference-service-template-decode-proxy.yaml.tmpl", getInferenceServiceData())
	if err != nil {
		return "", fmt.Errorf("failed to render decode proxy template: %w", err)
	}

	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceTemplateDecodeProxy)
	if err := ensureParentDir(manifestPath); err != nil {
		return "", err
	}
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write decode proxy template file: %w", err)
	}

	return manifestPath, nil
}
