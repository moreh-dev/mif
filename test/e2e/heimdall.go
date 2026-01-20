//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func installHeimdallForTest() {
	By("creating Heimdall values file for test")
	heimdallValuesPath, err := createHeimdallValuesFile()
	Expect(err).NotTo(HaveOccurred(), "Failed to create Heimdall values file for test")

	By("installing Heimdall for test")
	Expect(InstallHeimdall(cfg.workloadNamespace, heimdallValuesPath)).To(Succeed(), "Failed to install Heimdall for test")

	By("waiting for Heimdall deployment to be ready")
	cmd := exec.Command("kubectl", "wait", "deployment",
		"-l", "app.kubernetes.io/instance=heimdall",
		"--for=condition=Available",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", timeoutLong))
	_, err = Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Heimdall deployment not available")
}
