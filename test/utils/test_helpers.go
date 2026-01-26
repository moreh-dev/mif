//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// VerifyOdinController verifies Odin controller availability.
func VerifyOdinController(g Gomega) {
	cmd := exec.Command("kubectl", "wait", "deployment",
		"-l", "app.kubernetes.io/name=odin",
		"--for=condition=Available",
		"-n", cfg.mifNamespace,
		fmt.Sprintf("--timeout=%v", TimeoutLong))
	_, err := Run(cmd)
	g.Expect(err).NotTo(HaveOccurred(), "Odin controller not available")
}

// VerifyAllPodsReady verifies pods are ready in the MIF namespace.
func VerifyAllPodsReady(g Gomega) {
	cmd := exec.Command("kubectl", "wait", "pod",
		"--all",
		"--field-selector=status.phase!=Succeeded",
		"--for=condition=Ready",
		"-n", cfg.mifNamespace,
		fmt.Sprintf("--timeout=%v", TimeoutVeryLong))
	_, err := Run(cmd)
	g.Expect(err).NotTo(HaveOccurred(), "Some pods are not ready")
}

// CollectDebugInfo gathers logs and events for debugging.
func CollectDebugInfo() {
	By("fetching pod logs")
	cmd := exec.Command("kubectl", "get", "pods",
		"-n", cfg.mifNamespace,
		"-o", "jsonpath={.items[*].metadata.name}")
	output, err := Run(cmd)
	if err == nil {
		podNames := strings.Fields(output)
		for _, podName := range podNames {
			cmd = exec.Command("kubectl", "logs", podName, "-n", cfg.mifNamespace, "--all-containers=true", "--tail=100")
			logs, logErr := Run(cmd)
			if logErr == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Pod %s logs:\n%s\n", podName, logs)
			}
		}
	}

	By("fetching Kubernetes events")
	cmd = exec.Command("kubectl", "get", "events", "-n", cfg.mifNamespace, "--sort-by=.lastTimestamp")
	eventsOutput, err := Run(cmd)
	if err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s\n", eventsOutput)
	}

	By("fetching resource status")
	cmd = exec.Command("kubectl", "get", "all", "-n", cfg.mifNamespace)
	allOutput, err := Run(cmd)
	if err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "All resources:\n%s\n", allOutput)
	}
}
