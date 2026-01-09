//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting MIF E2E test suite\n")
	stopInterruptHandler := setupInterruptHandler()
	defer stopInterruptHandler()
	RunSpecs(t, "MIF E2E Suite")
}

var _ = BeforeSuite(func() {
	By("checking prerequisites")
	checkPrerequisites()

	if !cfg.skipKind {
		setupKindCluster()
	} else {
		cfg.isUsingKindCluster = false
		_, _ = fmt.Fprintf(GinkgoWriter, "Using existing cluster (kubeconfig). Resource cleanup will be skipped for safety.\n")
	}

	setupPrerequisites()
})

var _ = AfterSuite(func() {
	cleanupE2ETempFiles()

	if cfg.skipCleanup {
		_, _ = fmt.Fprintf(GinkgoWriter, "%s=true: skipping test namespace, resources, and kind cluster deletion.\n", envSkipCleanup)
		return
	}

	if !cfg.isUsingKindCluster {
		_, _ = fmt.Fprintf(GinkgoWriter, "Using existing cluster (kubeconfig). Skipping resource cleanup for safety.\n")
		return
	}

	cleanupKindResources()
})