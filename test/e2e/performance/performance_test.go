//go:build e2e && !printenv
// +build e2e,!printenv

package performance

import (
	"fmt"
	"os/exec"
	"strings"

	envs "github.com/moreh-dev/mif/test/e2e/envs"
	"github.com/moreh-dev/mif/test/utils"
	"github.com/moreh-dev/mif/test/utils/settings"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	InferencePerfS3PrefixBase = "vllm"
	InferencePerfPreset       = "workertemplate-vllm-common"
	InferencePerfExpType      = "performance"
	InferencePerfExpName      = "synthetic_random_i1024_o1024_c64"
)

var (
	commonTemplateName      string
	prefillMetaTemplateName string
	decodeMetaTemplateName  string
	decodeProxyTemplateName string
	prefillServiceName      string
	decodeServiceName       string
)

var _ = Describe("Inference Performance", Label("performance"), Ordered, func() {
	SetDefaultEventuallyTimeout(settings.TimeoutShort)
	SetDefaultEventuallyPollingInterval(settings.IntervalShort)

	BeforeAll(func() {
		By("creating workload namespace")
		Expect(utils.CreateWorkloadNamespace(envs.WorkloadNamespace, envs.MIFNamespace)).To(Succeed())

		By("creating Gateway resources")
		Expect(utils.CreateGatewayResource(envs.WorkloadNamespace, envs.GatewayClassName, envs.IstioRev)).To(Succeed())

		By("installing Heimdall")
		data := struct {
			MorehRegistrySecretName string
			GatewayName             string
			GatewayClass            string
		}{
			MorehRegistrySecretName: settings.MorehRegistrySecretName,
			GatewayName:             settings.GatewayName,
			GatewayClass:            envs.GatewayClassName,
		}

		values, err := utils.RenderTemplate("test/e2e/performance/config/heimdall-values.yaml.tmpl", data)
		Expect(err).NotTo(HaveOccurred(), "failed to render Heimdall values template")
		Expect(utils.InstallHeimdall(envs.WorkloadNamespace, values)).To(Succeed())

		By("creating InferenceServiceTemplates")
		isKind := !envs.SkipKind
		inferenceServiceData := utils.GetInferenceServiceData(envs.WorkloadNamespace, envs.TestModel, envs.HFToken, envs.HFEndpoint, isKind, false)
		commonTemplateName, err = utils.CreateInferenceServiceTemplate(envs.WorkloadNamespace, settings.InferenceServiceTemplateCommon, inferenceServiceData)
		Expect(err).NotTo(HaveOccurred(), "failed to create common InferenceServiceTemplate")
		prefillMetaTemplateName, err = utils.CreateInferenceServiceTemplate(envs.WorkloadNamespace, settings.InferenceServiceTemplatePrefillMeta, inferenceServiceData)
		Expect(err).NotTo(HaveOccurred(), "failed to create prefill meta InferenceServiceTemplate")
		decodeMetaTemplateName, err = utils.CreateInferenceServiceTemplate(envs.WorkloadNamespace, settings.InferenceServiceTemplateDecodeMeta, inferenceServiceData)
		Expect(err).NotTo(HaveOccurred(), "failed to create decode meta InferenceServiceTemplate")
		decodeProxyTemplateName, err = utils.CreateInferenceServiceTemplate(envs.WorkloadNamespace, settings.InferenceServiceTemplateDecodeProxy, inferenceServiceData)
		Expect(err).NotTo(HaveOccurred(), "failed to create decode proxy InferenceServiceTemplate")

		By("creating InferenceServices")
		prefillServiceName, err = utils.CreateInferenceService(envs.WorkloadNamespace, settings.InferenceServicePrefill, inferenceServiceData)
		Expect(err).NotTo(HaveOccurred(), "failed to create prefill InferenceService")
		decodeServiceName, err = utils.CreateInferenceService(envs.WorkloadNamespace, settings.InferenceServiceDecode, inferenceServiceData)
		Expect(err).NotTo(HaveOccurred(), "failed to create decode InferenceService")

		By("waiting for prefill InferenceService to be ready")
		Expect(waitForInferenceService(envs.WorkloadNamespace, prefillServiceName)).To(Succeed())

		By("waiting for decode InferenceService to be ready")
		Expect(waitForInferenceService(envs.WorkloadNamespace, decodeServiceName)).To(Succeed())
	})

	AfterAll(func() {
		if envs.SkipCleanup {
			return
		}

		By("deleting InferenceServices")
		utils.DeleteInferenceService(envs.WorkloadNamespace, prefillServiceName)
		utils.DeleteInferenceService(envs.WorkloadNamespace, decodeServiceName)

		By("deleting InferenceServiceTemplates")
		utils.DeleteInferenceServiceTemplate(envs.WorkloadNamespace, commonTemplateName)
		utils.DeleteInferenceServiceTemplate(envs.WorkloadNamespace, prefillMetaTemplateName)
		utils.DeleteInferenceServiceTemplate(envs.WorkloadNamespace, decodeMetaTemplateName)
		utils.DeleteInferenceServiceTemplate(envs.WorkloadNamespace, decodeProxyTemplateName)

		By("deleting Heimdall")
		utils.UninstallHeimdall(envs.WorkloadNamespace)

		By("deleting Gateway resources")
		utils.DeleteGatewayResource(envs.WorkloadNamespace, envs.GatewayClassName)

		By("deleting workload namespace")
		utils.DeleteNamespace(envs.WorkloadNamespace)
	})

	It("should run inference-perf performance benchmark", func() {
		By("getting Gateway service name")
		serviceName, err := utils.GetGatewayServiceName(envs.WorkloadNamespace)
		Expect(err).NotTo(HaveOccurred(), "failed to get Gateway service name")

		By("creating inference-perf job")
		jobName, err := createInferencePerfJob(envs.WorkloadNamespace, fmt.Sprintf("http://%s", serviceName))
		Expect(err).NotTo(HaveOccurred(), "failed to create inference-perf job")
		defer deleteInferencePerfJob(envs.WorkloadNamespace, jobName)

		By("waiting for inference-perf job to complete")
		Expect(waitForInferencePerfJob(envs.WorkloadNamespace, jobName)).To(Succeed())
	})
})

func waitForInferenceService(namespace string, name string) error {
	cmd := exec.Command("kubectl", "wait", "inferenceService", name,
		"--for=condition=Ready",
		"-n", namespace,
		fmt.Sprintf("--timeout=%v", settings.TimeoutVeryLong))
	_, err := utils.Run(cmd)
	return err
}

func createInferencePerfJob(namespace string, baseURL string) (string, error) {
	isKind := !envs.SkipKind
	_, imageTag := utils.GetInferenceImageInfo(isKind)

	type jobTemplateData struct {
		Namespace         string
		ModelName         string
		BaseURL           string
		HFToken           string
		HFEndpoint        string
		IsKind            bool
		S3AccessKeyID     string
		S3SecretAccessKey string
		S3Region          string
		S3Bucket          string
		S3PrefixBase      string
		VLLMTag           string
		Preset            string
		ExpType           string
		ExpName           string
	}

	data := jobTemplateData{
		Namespace:         namespace,
		ModelName:         envs.TestModel,
		BaseURL:           baseURL,
		HFToken:           envs.HFToken,
		HFEndpoint:        envs.HFEndpoint,
		IsKind:            isKind,
		S3AccessKeyID:     envs.S3AccessKeyID,
		S3SecretAccessKey: envs.S3SecretAccessKey,
		S3Region:          envs.S3Region,
		S3Bucket:          envs.S3Bucket,
		S3PrefixBase:      InferencePerfS3PrefixBase,
		VLLMTag:           imageTag,
		Preset:            InferencePerfPreset,
		ExpType:           InferencePerfExpType,
		ExpName:           InferencePerfExpName,
	}

	jobYAML, err := utils.RenderTemplate("test/e2e/performance/config/inference-perf-job.yaml.tmpl", data)
	if err != nil {
		return "", fmt.Errorf("failed to render job template: %w", err)
	}

	cmd := exec.Command("kubectl", "create", "-f", "-", "-n", namespace, "-o", "name")
	cmd.Stdin = strings.NewReader(jobYAML)
	output, err := utils.Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}
	return utils.ParseResourceName(output), nil
}

func deleteInferencePerfJob(namespace string, jobName string) {
	cmd := exec.Command("kubectl", "delete", "job", jobName,
		"-n", namespace, "--ignore-not-found=true")
	_, _ = utils.Run(cmd)
}

func waitForInferencePerfJob(namespace string, jobName string) error {
	cmd := exec.Command("kubectl", "wait", "job", jobName,
		"--for=condition=complete",
		"-n", namespace,
		fmt.Sprintf("--timeout=%v", settings.TimeoutVeryLong))
	_, err := utils.Run(cmd)
	if err != nil {
		return fmt.Errorf("inference-perf job did not complete within timeout: %w", err)
	}

	return nil
}
