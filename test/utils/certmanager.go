//go:build e2e
// +build e2e

package utils

import (
	"os/exec"
)

const (
	certmanagerVersion   = "v1.18.4"
	certmanagerChart     = "oci://quay.io/jetstack/charts/cert-manager"
	certmanagerNamespace = "cert-manager"
)

// InstallCertManager installs cert-manager using Helm
func InstallCertManager() error {
	helmArgs := []string{
		"upgrade", "--install", "cert-manager",
		certmanagerChart,
		"--version", certmanagerVersion,
		"--namespace", certmanagerNamespace,
		"--create-namespace",
		"--set", "crds.enabled=true",
		"--wait",
		"--timeout", "5m",
	}
	cmd := exec.Command("helm", helmArgs...)
	if _, err := Run(cmd); err != nil {
		return err
	}

	cmd = exec.Command("kubectl", "wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", certmanagerNamespace,
		"--timeout", "5m",
	)

	_, err := Run(cmd)
	return err
}

// UninstallCertManager uninstalls the cert manager using Helm
func UninstallCertManager() {
	cmd := exec.Command("helm", "uninstall", "cert-manager",
		"--namespace", certmanagerNamespace,
		"--ignore-not-found")
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}

	kubeSystemLeases := []string{
		"cert-manager-cainjector-leader-election",
		"cert-manager-controller",
	}
	for _, lease := range kubeSystemLeases {
		cmd = exec.Command("kubectl", "delete", "lease", lease,
			"-n", "kube-system", "--ignore-not-found", "--force", "--grace-period=0")
		if _, err := Run(cmd); err != nil {
			warnError(err)
		}
	}
}
