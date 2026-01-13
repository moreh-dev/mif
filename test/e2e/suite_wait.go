//go:build e2e && !printenv
// +build e2e,!printenv

package e2e

import (
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/moreh-dev/mif/test/utils"
)

// waitForMIFComponents waits for MIF components to be ready.
func waitForMIFComponents() {
	By("waiting for Odin controller deployment")
	cmd := exec.Command("kubectl", "get", "deployment",
		"-n", cfg.mifNamespace,
		"-l", "app.kubernetes.io/name=odin",
		"-o", "jsonpath={.items[0].metadata.name}")
	output, err := utils.Run(cmd)
	if err != nil || strings.TrimSpace(output) == "" {
		cmd = exec.Command("kubectl", "get", "deployment",
			"-n", cfg.mifNamespace,
			"-o", "jsonpath={.items[?(@.metadata.name=~\"odin.*\")].metadata.name}")
		output, err = utils.Run(cmd)
		if err != nil || strings.TrimSpace(output) == "" {
			_, _ = fmt.Fprintf(GinkgoWriter, "Warning: Could not find Odin deployment by label, trying common name pattern\n")
			cmd = exec.Command("kubectl", "wait", "--for=condition=Available",
				fmt.Sprintf("deployment/%s-odin", helmReleaseMIF),
				"-n", cfg.mifNamespace,
				fmt.Sprintf("--timeout=%v", timeoutMedium))
			_, _ = utils.Run(cmd)
			return
		}
	}
	odinDeploymentName := strings.TrimSpace(output)
	_, _ = fmt.Fprintf(GinkgoWriter, "Found Odin deployment: %s\n", odinDeploymentName)

	cmd = exec.Command("kubectl", "wait", "--for=condition=Available",
		fmt.Sprintf("deployment/%s", odinDeploymentName),
		"-n", cfg.mifNamespace,
		fmt.Sprintf("--timeout=%v", timeoutLong))
	output, err = utils.Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Warning: Odin deployment wait completed with error (may already be ready): %v\n", err)
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Odin deployment is available\n")
	}

	By("waiting for all running pods to be ready")
	cmd = exec.Command("kubectl", "wait", "pod",
		"--for=condition=Ready",
		"--all",
		"--field-selector=status.phase!=Succeeded",
		"-n", cfg.mifNamespace,
		fmt.Sprintf("--timeout=%v", timeoutLong))
	output, err = utils.Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Warning: Some pods may not be ready yet: %v\n", err)
		cmd = exec.Command("kubectl", "get", "pods",
			"-n", cfg.mifNamespace,
			"--field-selector=status.phase!=Succeeded",
			"-o", "wide")
		statusOutput, _ := utils.Run(cmd)
		_, _ = fmt.Fprintf(GinkgoWriter, "Current pod status:\n%s\n", statusOutput)
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "All pods are ready\n")
	}
}

// ensureECRTokenRefresherSecret ensures the ECR token refresher secret is created.
func ensureECRTokenRefresherSecret(namespace string) {
	cronJobName := cronJobNameECRRefresher
	jobName := jobNameECRRefresher
	secretName := secretNameMorehRegistry
	ecrCredsSecretName := secretNameECRCreds

	By(fmt.Sprintf("waiting for secret %s to be created", ecrCredsSecretName))
	Eventually(func() bool {
		cmd := exec.Command("kubectl", "get", "secret", ecrCredsSecretName, "-n", namespace)
		_, err := utils.Run(cmd)
		return err == nil
	}, timeoutShort, intervalMedium).Should(BeTrue(), fmt.Sprintf("Secret %s should be created", ecrCredsSecretName))

	cmd := exec.Command("kubectl", "get", "secret", secretName, "-n", namespace)
	_, err := utils.Run(cmd)
	if err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Secret %s already exists. Skipping ecrTokenRefresher job execution.\n", secretName)
		return
	}

	By(fmt.Sprintf("waiting for CronJob %s to be created", cronJobName))
	Eventually(func() bool {
		cmd = exec.Command("kubectl", "get", "cronjob", cronJobName, "-n", namespace)
		_, err = utils.Run(cmd)
		return err == nil
	}, timeoutShort, intervalMedium).Should(BeTrue(), fmt.Sprintf("CronJob %s should be created", cronJobName))

	cmd = exec.Command("kubectl", "delete", "job", jobName, "-n", namespace, "--ignore-not-found=true")
	_, _ = utils.Run(cmd)

	By(fmt.Sprintf("creating manual job from CronJob %s", cronJobName))
	cmd = exec.Command("kubectl", "create", "job", "--from=cronjob/"+cronJobName, jobName, "-n", namespace)
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create job from CronJob %s", cronJobName))

	By(fmt.Sprintf("waiting for job %s to complete", jobName))
	Eventually(func() bool {
		cmd := exec.Command("kubectl", "get", "job", jobName, "-n", namespace, "-o", "jsonpath={.status.conditions[?(@.type==\"Complete\")].status}")
		output, err := utils.Run(cmd)
		return err == nil && strings.TrimSpace(output) == "True"
	}, timeoutMedium, intervalMedium).Should(BeTrue(), fmt.Sprintf("Job %s should complete successfully", jobName))

	cmd = exec.Command("kubectl", "get", "job", jobName, "-n", namespace, "-o", "jsonpath={.status.conditions[?(@.type==\"Failed\")].status}")
	output, _ := utils.Run(cmd)
	if strings.TrimSpace(output) == "True" {
		cmd = exec.Command("kubectl", "logs", "-n", namespace, "job/"+jobName)
		logs, _ := utils.Run(cmd)
		Fail(fmt.Sprintf("Job %s failed. Logs: %s", jobName, logs))
	}

	By(fmt.Sprintf("waiting for secret %s to be created", secretName))
	Eventually(func() bool {
		cmd := exec.Command("kubectl", "get", "secret", secretName, "-n", namespace)
		_, err := utils.Run(cmd)
		return err == nil
	}, timeoutShort, intervalShort).Should(BeTrue(), fmt.Sprintf("Secret %s should be created", secretName))

	_, _ = fmt.Fprintf(GinkgoWriter, "Successfully created secret %s via ecrTokenRefresher\n", secretName)
}
