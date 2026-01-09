package utils

import (
	"os/exec"
	"strings"
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

// IsCertManagerCRDsInstalled checks if cert-manager CRDs are installed.
func IsCertManagerCRDsInstalled() bool {
	certManagerCRDs := []string{
		"certificates.cert-manager.io",
		"issuers.cert-manager.io",
		"clusterissuers.cert-manager.io",
		"certificaterequests.cert-manager.io",
		"orders.acme.cert-manager.io",
		"challenges.acme.cert-manager.io",
	}

	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	crdList := GetNonEmptyLines(output)
	foundCRDs := make(map[string]bool, len(certManagerCRDs))
	for _, crd := range certManagerCRDs {
		foundCRDs[crd] = false
	}

	for _, line := range crdList {
		for _, crd := range certManagerCRDs {
			if strings.Contains(line, crd) {
				foundCRDs[crd] = true
				break
			}
		}
	}

	for _, found := range foundCRDs {
		if !found {
			return false
		}
	}

	return true
}
