//go:build e2e
// +build e2e

package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	. "github.com/onsi/ginkgo/v2"
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

// RunWithGinkgoWriter runs a command with the Ginkgo writer.
func RunWithGinkgoWriter(cmd *exec.Cmd) error {
	dir, _ := GetProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %q\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %q\n", command)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%q failed with error: %w", command, err)
	}

	return nil
}

// DeleteNamespace deletes a Kubernetes namespace, ignoring errors if it doesn't exist.
func DeleteNamespace(namespace string) {
	cmd := exec.Command("kubectl", "delete", "ns", namespace, "--timeout=60s", "--ignore-not-found=true")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// RenderTemplate renders a template file with the given data.
func RenderTemplate(filename string, data any) (string, error) {
	templateText, err := loadTemplateFile(filename)
	if err != nil {
		return "", err
	}

	return renderTextTemplate(templateText, data)
}

// GetProjectDir returns the directory where the project is.
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			return "", fmt.Errorf("project root (go.mod) not found")
		}
		wd = parent
	}
}

// ParseResourceName extracts only the resource name from kubectl's "-o name" output.
// It converts "kind.group/name" into just "name".
func ParseResourceName(output string) string {
	trimmed := strings.TrimSpace(output)
	if parts := strings.Split(trimmed, "/"); len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return trimmed
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

	templatePath := filepath.Join(projectDir, filename)
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	return string(content), nil
}

func hasAllCRDs(output string, required []string) bool {
	for _, crd := range required {
		if !strings.Contains(output, crd) {
			return false
		}
	}
	return true
}

// ParseImage parses a container image reference
func ParseImage(image string) (repo, tag string, err error) {
	lastColon := strings.LastIndex(image, ":")
	if lastColon == -1 {
		return image, "", fmt.Errorf("invalid image: %s", image)
	}
	if lastColon == len(image)-1 {
		return image[:lastColon], "", fmt.Errorf("invalid image: %s", image)
	}

	repo = image[:lastColon]
	tag = image[lastColon+1:]

	return repo, tag, nil
}
