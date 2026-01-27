//go:build e2e && !printenv
// +build e2e,!printenv

package performance

import (
	"fmt"
	"testing"

	envs "github.com/moreh-dev/mif/test/e2e/envs"
	"github.com/moreh-dev/mif/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	isCertManagerAlreadyInstalled bool
	isGatewayAPIAlreadyInstalled  bool
)

func TestPerformance(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting MIF performance test suite\n")
	RunSpecs(t, "MIF Performance Suite")
}

var _ = BeforeSuite(func() {
	if !envs.SkipKind {
		By("creating kind cluster")
		Expect(utils.CreateKindCluster()).To(Succeed())
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Using existing cluster (kubeconfig).\n")
	}

	By("installing prerequisites")
	if envs.SkipPrerequisite {
		By("skipping prerequisite installation")
		return
	}

	By("checking if cert manager is installed already")
	isCertManagerAlreadyInstalled = utils.IsCertManagerCRDsInstalled()
	if !isCertManagerAlreadyInstalled {
		By("installing CertManager")
		if err := utils.InstallCertManager(); err != nil {
			Fail(fmt.Sprintf("failed to install CertManager: %v", err))
		}
	} else {
		By("CertManager is already installed. Skipping installation")
	}

	By("installing MIF")
	if err := utils.InstallMIF(envs.MIFNamespace, envs.AWSAccessKeyID, envs.AWSSecretAccessKey); err != nil {
		Fail(fmt.Sprintf("failed to install MIF: %v", err))
	}

	By("installing MIF Preset")
	if err := utils.InstallMIFPreset(envs.MIFNamespace); err != nil {
		Fail(fmt.Sprintf("failed to install MIF Preset: %v", err))
	}

	By("checking if Gateway API is installed already")
	isGatewayAPIAlreadyInstalled = utils.IsGatewayAPICRDsInstalled()
	if !isGatewayAPIAlreadyInstalled {
		By("installing Gateway API")
		if err := utils.InstallGatewayAPI(); err != nil {
			Fail(fmt.Sprintf("failed to install Gateway API: %v", err))
		}

		By("installing Gateway API Inference Extension")
		if err := utils.InstallGatewayInferenceExtension(); err != nil {
			Fail(fmt.Sprintf("failed to install Gateway API Inference Extension: %v", err))
		}
	}

	By("installing Gateway Controller")
	if err := utils.InstallGatewayController(envs.GatewayClassName); err != nil {
		Fail(fmt.Sprintf("failed to install Gateway Controller: %v", err))
	}
})

var _ = AfterSuite(func() {
	if envs.SkipCleanup {
		By("skipping cleanup")
		return
	}

	if envs.SkipPrerequisite {
		By("skipping prerequisite uninstallation")

		if !envs.SkipKind {
			By("deleting kind cluster")
			utils.DeleteKindCluster()
		}

		return
	}

	By("uninstalling Gateway Controller")
	utils.UninstallGatewayController(envs.GatewayClassName)

	if !isGatewayAPIAlreadyInstalled {
		By("uninstalling Gateway API Inference Extension")
		utils.UninstallGatewayInferenceExtension()

		By("uninstalling Gateway API")
		utils.UninstallGatewayAPI()
	}

	By("uninstalling MIF Preset")
	utils.UninstallMIFPreset(envs.MIFNamespace)

	By("uninstalling MIF")
	utils.UninstallMIF(envs.MIFNamespace)

	if !isCertManagerAlreadyInstalled {
		By("uninstalling CertManager")
		utils.UninstallCertManager()
	}

	if envs.SkipKind {
		By("skipping kind cluster deletion")
		return
	}

	By("deleting kind cluster")
	utils.DeleteKindCluster()
})
