//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/moreh-dev/mif/test/utils"
)

// setupInterruptHandler sets up signal handlers for graceful shutdown.
func setupInterruptHandler() func() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})

	go func() {
		select {
		case sig := <-sigChan:
			_, _ = fmt.Fprintf(GinkgoWriter, "\nReceived signal: %v. Initiating immediate cleanup...\n", sig)
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

// cleanupKindCluster deletes the kind cluster if it exists.
func cleanupKindCluster() {
	if cfg.skipKind {
		return
	}

	if !utils.IsKindClusterExists(cfg.kindClusterName) {
		_, _ = fmt.Fprintf(GinkgoWriter, "Kind cluster %s does not exist, skipping deletion\n", cfg.kindClusterName)
		return
	}

	By("deleting kind cluster (always cleanup)")
	_, _ = fmt.Fprintf(GinkgoWriter, "Deleting kind cluster %s...\n", cfg.kindClusterName)

	if err := utils.DeleteKindCluster(cfg.kindClusterName); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to delete kind cluster %s: %v\n", cfg.kindClusterName, err)
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Successfully deleted kind cluster %s\n", cfg.kindClusterName)
	}
}

func cleanupMIFNamespace() {
	if err := utils.DeleteNamespace(cfg.mifNamespace); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to delete MIF namespace: %v\n", err)
	}
}

// cleanupE2ETempFiles removes temporary files created during E2E tests.
func cleanupE2ETempFiles() {
	projectDir, err := utils.GetProjectDir()
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
	}

	for _, rel := range tempFiles {
		path := filepath.Join(projectDir, rel)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			_, _ = fmt.Fprintf(GinkgoWriter, "warning: failed to remove temp file %s: %v\n", path, err)
		}
	}
}

// checkPrerequisites verifies that required tools are available.
func checkPrerequisites() {
	requiredTools := []string{"kubectl", "helm"}
	if !cfg.skipKind {
		requiredTools = append(requiredTools, "kind")
		cmd := exec.Command("which", "kind")
		if err := cmd.Run(); err != nil {
			Fail(fmt.Sprintf("kind is required but not available. Install kind or set %s=true to use existing cluster", envSkipKind))
		}
	}
	for _, tool := range requiredTools {
		cmd := exec.Command("which", tool)
		if err := cmd.Run(); err != nil {
			Fail(fmt.Sprintf("Required tool %s is not available", tool))
		}
	}

	if !cfg.skipKind {
		return
	}

	By("verifying Kubernetes cluster connectivity")
	cmd := exec.Command("kubectl", "cluster-info", "--request-timeout=30s")
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Cannot connect to Kubernetes cluster. Please check your kubeconfig.")
}
