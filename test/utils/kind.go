//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// CreateKindCluster creates a kind cluster with the given name.
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

// DeleteKindCluster deletes a kind cluster with the given name.
func DeleteKindCluster(clusterName string) error {
	cmd := exec.Command("kind", "delete", "cluster", "--name", clusterName)
	_, err := Run(cmd)
	return err
}

// IsKindClusterExists checks if a kind cluster with the given name exists.
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

// SetupKindCluster creates or reuses a kind cluster for testing.
func SetupKindCluster() {
	By("creating kind cluster")
	if IsKindClusterExists(cfg.kindClusterName) {
		_, _ = fmt.Fprintf(GinkgoWriter, "Kind cluster %s already exists. Skipping creation...\n", cfg.kindClusterName)
		By("exporting kubeconfig for existing kind cluster")
		cmd := exec.Command("kind", "export", "kubeconfig", "--name", cfg.kindClusterName)
		_, err := Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to export kubeconfig for existing kind cluster")
		cfg.IsUsingKindCluster = true
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Creating kind cluster %s...\n", cfg.kindClusterName)
		if err := CreateKindCluster(cfg.kindClusterName); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "Kind cluster creation failed. Attempting to clean up partially created cluster...\n")
			if IsKindClusterExists(cfg.kindClusterName) {
				_ = DeleteKindCluster(cfg.kindClusterName)
			}
			Expect(err).NotTo(HaveOccurred(), "Failed to create kind cluster")
		}
		cfg.IsUsingKindCluster = true

		By("verifying kubectl access to kind cluster")
		contextName := fmt.Sprintf("kind-%s", cfg.kindClusterName)
		cmd := exec.Command("kubectl", "cluster-info", "--context", contextName)
		_, err := Run(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "Failed to verify kind cluster. Attempting to clean up...\n")
			if IsKindClusterExists(cfg.kindClusterName) {
				_ = DeleteKindCluster(cfg.kindClusterName)
			}
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to access kind cluster via kubectl context %s", contextName))
		}
	}

	By("adding moreh Helm repository")
	cmd := exec.Command("helm", "repo", "add", helmRepoName, helmRepoURL)
	if _, err := Run(cmd); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to add moreh helm repo: %v\n", err)
		}
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Successfully added moreh Helm repository\n")
	}

	By("updating moreh Helm repository")
	cmd = exec.Command("helm", "repo", "update", helmRepoName)
	if _, err := Run(cmd); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to update moreh helm repo: %v\n", err)
	}
}

// cleanupKindCluster deletes the kind cluster if it exists.
func cleanupKindCluster() {
	if cfg.SkipKind {
		return
	}

	if !IsKindClusterExists(cfg.kindClusterName) {
		_, _ = fmt.Fprintf(GinkgoWriter, "Kind cluster %s does not exist, skipping deletion\n", cfg.kindClusterName)
		return
	}

	By("deleting kind cluster (always cleanup)")
	_, _ = fmt.Fprintf(GinkgoWriter, "Deleting kind cluster %s...\n", cfg.kindClusterName)

	if err := DeleteKindCluster(cfg.kindClusterName); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to delete kind cluster %s: %v\n", cfg.kindClusterName, err)
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Successfully deleted kind cluster %s\n", cfg.kindClusterName)
	}
}

// CleanupKindResources cleans up resources specific to kind cluster.
func CleanupKindResources() {
	By("uninstalling moai-inference-preset")
	UninstallMIFPreset(cfg.mifNamespace)

	By("uninstalling MIF")
	cmd := exec.Command("helm", "uninstall", helmReleaseMIF, "-n", cfg.mifNamespace, "--ignore-not-found=true")
	_, _ = Run(cmd)

	By("deleting MIF namespace")
	cleanupMIFNamespace()

	if !cfg.SkipPrerequisite {
		cleanupPrerequisites()
	}

	cleanupKindCluster()
}
