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

func applyGatewayResource() {
	By("applying Gateway resource and infrastructure parameters")

	var baseYAML string

	switch cfg.gatewayClass {
	case "istio":
		baseYAML = `apiVersion: v1
kind: ConfigMap
metadata:
  name: mif-gateway-infrastructure
data:
  service: |
    spec:
      type: ClusterIP
  deployment: |
    spec:
      template:
        metadata:
          annotations:
            proxy.istio.io/config: |
              accessLogFile: /dev/stdout
              accessLogEncoding: JSON
        spec:
          containers:
            - name: istio-proxy
              resources:
                limits: null

---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: mif
spec:
  gatewayClassName: istio
  infrastructure:
    parametersRef:
      group: ""
      kind: ConfigMap
      name: mif-gateway-infrastructure
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: All
`
	case "kgateway":
		baseYAML = `apiVersion: gateway.kgateway.dev/v1alpha1
kind: GatewayParameters
metadata:
  name: mif-gateway-infrastructure
spec:
  kube:
    service:
      type: ClusterIP

---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: mif
spec:
  gatewayClassName: kgateway
  infrastructure:
    parametersRef:
      group: gateway.kgateway.dev
      kind: GatewayParameters
      name: mif-gateway-infrastructure
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: All
`
	default:
		Fail(fmt.Sprintf("Unsupported gatewayClassName: %s", cfg.gatewayClass))
	}

	cmd := exec.Command("kubectl", "apply", "-f", "-", "-n", cfg.workloadNamespace, "--request-timeout=60s")
	cmd.Stdin = strings.NewReader(baseYAML)
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to apply Gateway resources")

	By("waiting for Gateway to be accepted")
	cmd = exec.Command("kubectl", "wait", "gateway", "mif",
		"--for=condition=Accepted",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", timeoutLong))
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Gateway not accepted")

	By("waiting for Gateway pods to be created")
	Eventually(func() (string, error) {
		checkCmd := exec.Command("kubectl", "get", "pods",
			"-l", "gateway.networking.k8s.io/gateway-name=mif",
			"-n", cfg.workloadNamespace,
			"-o", "name")
		return utils.Run(checkCmd)
	}, timeoutLong, intervalShort).ShouldNot(BeEmpty())

	By("waiting for Gateway pods to be ready")
	cmd = exec.Command("kubectl", "wait", "pod",
		"-l", "gateway.networking.k8s.io/gateway-name=mif",
		"--for=condition=Ready",
		"-n", cfg.workloadNamespace,
		fmt.Sprintf("--timeout=%v", timeoutLong))
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Gateway pods not ready")
}
