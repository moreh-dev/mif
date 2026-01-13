//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moreh-dev/mif/test/utils"
)

func createCommonTemplate() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	rendered, err := renderTemplateFile("inference-service-template-common.yaml.tmpl", getInferenceServiceData())
	if err != nil {
		return "", fmt.Errorf("failed to render common template: %w", err)
	}

	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceTemplateCommon)
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write common template file: %w", err)
	}

	return manifestPath, nil
}

func createPrefillMetaTemplate() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	rendered, err := renderTemplateFile("inference-service-template-prefill-meta.yaml.tmpl", getInferenceServiceData())
	if err != nil {
		return "", fmt.Errorf("failed to render prefill meta template: %w", err)
	}

	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceTemplatePrefillMeta)
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write prefill meta template file: %w", err)
	}

	return manifestPath, nil
}

func createDecodeMetaTemplate() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	rendered, err := renderTemplateFile("inference-service-template-decode-meta.yaml.tmpl", getInferenceServiceData())
	if err != nil {
		return "", fmt.Errorf("failed to render decode meta template: %w", err)
	}

	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceTemplateDecodeMeta)
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write decode meta template file: %w", err)
	}

	return manifestPath, nil
}

func createDecodeProxyTemplate() (string, error) {
	projectDir, err := utils.GetProjectDir()
	if err != nil {
		return "", err
	}

	rendered, err := renderTemplateFile("inference-service-template-decode-proxy.yaml.tmpl", getInferenceServiceData())
	if err != nil {
		return "", fmt.Errorf("failed to render decode proxy template: %w", err)
	}

	manifestPath := filepath.Join(projectDir, tempFileInferenceServiceTemplateDecodeProxy)
	if err := os.WriteFile(manifestPath, []byte(rendered), 0600); err != nil {
		return "", fmt.Errorf("failed to write decode proxy template file: %w", err)
	}

	return manifestPath, nil
}
