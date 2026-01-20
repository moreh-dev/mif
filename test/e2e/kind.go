package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
)

// CreateKindCluster creates a kind cluster with the given name.
func CreateKindCluster(clusterName string) error {
	k8sVersion := os.Getenv("KIND_K8S_VERSION")
	args := []string{"create", "cluster", "--name", clusterName, "-v", "1"}
	if k8sVersion != "" {
		nodeImage := fmt.Sprintf("kindest/node:%s", k8sVersion)
		args = append(args, "--image", nodeImage)
	}

	cmd := exec.Command("kind", args...)
	dir, _ := GetProjectDir()
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter

	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %q\n", command)
	var err error
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("failed to create kind cluster: %w", err)
	}

	cmd = exec.Command("kind", "export", "kubeconfig", "--name", clusterName)
	_, err = Run(cmd)
	if err != nil {
		return fmt.Errorf("failed to export kubeconfig for kind cluster %s: %w", clusterName, err)
	}

	contextName := fmt.Sprintf("kind-%s", clusterName)
	cmd = exec.Command("kubectl", "cluster-info", "--context", contextName)
	_, err = Run(cmd)
	if err != nil {
		return fmt.Errorf("failed to verify kubectl context %s for kind cluster %s: %w", contextName, clusterName, err)
	}

	return nil
}

// DeleteKindCluster deletes a kind cluster with the given name.
func DeleteKindCluster(clusterName string) error {
	cmd := exec.Command("kind", "delete", "cluster", "--name", clusterName)
	_, err := Run(cmd)
	return err
}

// IsKindClusterExists checks if a kind cluster with the given name exists.
func IsKindClusterExists(clusterName string) bool {
	cmd := exec.Command("kind", "get", "clusters")
	output, err := Run(cmd)
	if err != nil {
		return false
	}
	clusters := GetNonEmptyLines(output)
	for _, cluster := range clusters {
		if strings.TrimSpace(cluster) == clusterName {
			return true
		}
	}
	return false
}
