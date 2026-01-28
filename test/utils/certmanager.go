//go:build e2e
// +build e2e

package utils

import (
	"fmt"
	"os/exec"

	"github.com/moreh-dev/mif/test/utils/settings"
)

// InstallCertManager installs cert-manager using Helm
func InstallCertManager() error {
	helmArgs := []string{
		"upgrade", "--install", "cert-manager",
		settings.CertManagerHelmRepoURL,
		"--version", settings.CertManagerVersion,
		"--namespace", settings.CertManagerNamespace,
		"--create-namespace",
		"--set", "crds.enabled=true",
		"--wait",
		fmt.Sprintf("--timeout=%v", settings.TimeoutMedium),
	}

	cmd := exec.Command("helm", helmArgs...)
	if _, err := Run(cmd); err != nil {
		return err
	}
	return nil
}

// UninstallCertManager uninstalls the cert manager using Helm
func UninstallCertManager() {
	cmd := exec.Command("helm", "uninstall", "cert-manager",
		"--namespace", settings.CertManagerNamespace,
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

// IsCertManagerCRDsInstalled checks if any Cert Manager CRDs are installed
// by verifying the existence of key CRDs related to Cert Manager.
func IsCertManagerCRDsInstalled() bool {
	// List of common Cert Manager CRDs
	certManagerCRDs := []string{
		"certificates.cert-manager.io",
		"issuers.cert-manager.io",
		"clusterissuers.cert-manager.io",
		"certificaterequests.cert-manager.io",
		"orders.acme.cert-manager.io",
		"challenges.acme.cert-manager.io",
	}

	// Execute the kubectl command to get all CRDs
	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	return hasAllCRDs(output, certManagerCRDs)
}
