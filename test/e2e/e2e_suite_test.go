//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"testing"

	utils "github.com/moreh-dev/mif/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting MIF E2E test suite\n")
	stopInterruptHandler := utils.SetupInterruptHandler()
	defer stopInterruptHandler()
	RunSpecs(t, "MIF E2E Suite")
}

var _ = BeforeSuite(func() {
	By("checking prerequisites")
	utils.CheckPrerequisites()

	if !utils.Cfg.SkipKind {
		utils.SetupKindCluster()
	} else {
		utils.Cfg.IsUsingKindCluster = false
		_, _ = fmt.Fprintf(GinkgoWriter, "Using existing cluster (kubeconfig). Resource cleanup will be skipped for safety.\n")
	}

	utils.SetupPrerequisites()
})

var _ = AfterSuite(func() {
	utils.CleanupE2ETempFiles()

	if utils.Cfg.SkipCleanup {
		_, _ = fmt.Fprintf(GinkgoWriter, "%s=true: skipping test namespace, resources, and kind cluster deletion.\n", utils.EnvSkipCleanup)
		return
	}

	if !utils.Cfg.IsUsingKindCluster {
		_, _ = fmt.Fprintf(GinkgoWriter, "Using existing cluster (kubeconfig). Skipping resource cleanup for safety.\n")
		return
	}

	utils.CleanupKindResources()
})
