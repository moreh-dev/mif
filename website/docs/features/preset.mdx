---
sidebar_position: 1

title: 'Presets'
---

# Presets

The MoAI Inference Framework provides a set of pre-configured `InferenceServiceTemplate`s, known as presets. These presets encapsulate standard configurations for various models and hardware setups, simplifying the deployment of inference services.

## Installation

The presets are installed via the `moai-inference-preset` Helm chart.

First, add the Moreh Helm chart repository:

```shell
helm repo add moreh https://moreh-dev.github.io/helm-charts
helm repo update moreh
```

You can check the available versions of the preset chart:

```shell
helm search repo moreh/moai-inference-preset -l
```

Then, install the chart. This will create `InferenceServiceTemplate` resources in your cluster.

```shell
helm upgrade -i moai-inference-preset moreh/moai-inference-preset \
    --version v0.3.0 \
    -n mif
```

You can view the available presets in your cluster using the following command:

```shell
kubectl get inferenceservicetemplates -n mif
```

---

## Using a complete preset

To use a preset, you reference it in the `spec.templateRefs` field of your `InferenceService`. You can specify multiple templates; they will be merged in the order listed, with later templates overriding earlier ones.

`templateRefs` searches for templates in the following order:

1. The namespace where the `InferenceService` is created.
2. The `mif` namespace, where the Odin operator is typically installed.

For example, to deploy a vLLM service for the Llama 3.2 1B Instruct model on AMD MI250 GPUs, you can combine the base `vllm` template with the model-specific `vllm-meta-llama-llama-3.2-1b-instruct-amd-mi250-tp2` template:

```yaml {20}
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: vllm-llama3-1b-instruct-tp2
spec:
  replicas: 2
  inferencePoolRefs:
    - name: heimdall
  templateRefs:
    - name: vllm
    - name: vllm-meta-llama-llama-3.2-1b-instruct-amd-mi250-tp2
  parallelism:
    tensor: 2
  template:
    spec:
      containers:
        - name: main
          env:
            - name: HF_TOKEN
              value: <huggingFaceToken>
```

---

## Overriding preset configuration

You can customize or override the configuration defined in the presets by providing a `spec.template` in your `InferenceService`. The fields in `spec.template` take precedence over those in the referenced templates.

:::warning
When using certain runtime bases (e.g., `vllm-decode-dp`), `workerTemplate` is used instead of `template` to define the pod configuration. Therefore, you must use `spec.workerTemplate` instead of `spec.template` when overriding values.
:::

To identify which values to override, you can inspect the contents of the `InferenceServiceTemplate` resources. For example, to check the runtime base configuration (`vllm`) and the model-specific configuration (`vllm-meta-llama-llama-3.2-1b-instruct-amd-mi250-tp2`):

```shell
kubectl get inferenceservicetemplate vllm -n mif -o yaml
```

```shell
kubectl get inferenceservicetemplate vllm-meta-llama-llama-3.2-1b-instruct-amd-mi250-tp2 -n mif -o yaml
```

```shell Expected output
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceServiceTemplate
metadata:
  name: vllm-meta-llama-llama-3.2-1b-instruct-amd-mi250-tp2
  namespace: mif
  # ... (other fields)
spec:
  # ... (other fields)
  template:
    spec:
      # ... (other fields)
      containers:
        # ... (other fields)
        - name: main
          # ... (other fields)
          env:
            # ... (other fields)
            - name: ISVC_MODEL_NAME
              value: meta-llama/Llama-3.2-1B-Instruct
            - name: ISVC_EXTRA_ARGS
              value: $(ISVC_MODEL_NAME) --disable-uvicorn-access-log --no-enable-log-requests
                --quantization None --max-model-len 8192 --max-num-batched-tokens 32768
                --no-enable-prefix-caching --kv-transfer-config '{"kv_connector":"NixlConnector","kv_role":"kv_both"}'
```

This command reveals the default configuration, including containers, environment variables, and resource limits. You can then reference this output to determine the correct structure and values to include in your `spec.template`.

A common use case is modifying the model execution arguments. For instance, the `vllm-meta-llama-llama-3.2-1b-instruct-amd-mi250-tp2` preset disables prefix caching by default (`--no-enable-prefix-caching`) in `ISVC_EXTRA_ARGS`. You can enable it by overriding the environment variable in your `InferenceService`:

```yaml {23,26}
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: vllm-llama3-1b-instruct-tp2
spec:
  # ... (other fields)
  templateRefs:
    - name: vllm
    - name: vllm-meta-llama-llama-3.2-1b-instruct-amd-mi250-tp2
  template:
    spec:
      containers:
        - name: main
          env:
            - name: ISVC_EXTRA_ARGS
              value: >-
                $(ISVC_MODEL_NAME)
                --disable-uvicorn-access-log
                --no-enable-log-requests
                --quantization None
                --max-model-len 8192
                --max-num-batched-tokens 32768
                --enable-prefix-caching
                --kv-transfer-config '{"kv_connector":"NixlConnector","kv_role":"kv_both"}'
            - name: HF_TOKEN
              value: <huggingFaceToken>
```

---

## Using a runtime base preset

If a preset for your specific model or hardware configuration is not available, you can use only the runtime base preset (e.g., `vllm-decode-dp`) and manually define the model-specific configurations in `spec.workerTemplate`.

In this case, you need to manually specify the model name, extra arguments, resources, and scheduler requirements (node selector and tolerations).

```yaml {29}
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceService
metadata:
  name: my-custom-model
spec:
  replicas: 1
  inferencePoolRefs:
    - name: heimdall
  templateRefs:
    - name: vllm-decode-dp # Runtime base only
  parallelism:
    data: 2
    tensor: 1
  workerTemplate: # Use workerTemplate for vllm-decode-dp
    spec:
      containers:
        - name: main
          env:
            - name: ISVC_MODEL_NAME
              value: meta-llama/Llama-3.2-1B-Instruct
            - name: ISVC_EXTRA_ARGS
              value: >-
                $(ISVC_MODEL_NAME)
                --disable-uvicorn-access-log
                --no-enable-log-requests
                --quantization None
                --max-model-len 4096
            - name: HF_TOKEN
              value: <huggingFaceToken>
          resources:
            limits:
              amd.com/gpu: 1
            requests:
              amd.com/gpu: 1
      nodeSelector:
        moai.moreh.io/accelerator.vendor: amd
        moai.moreh.io/accelerator.model: mi300x
      tolerations:
        - key: amd.com/gpu
          operator: Exists
          effect: NoSchedule
```

---

## Creating a reusable preset

You can turn the configuration above into a reusable preset (`InferenceServiceTemplate`) by removing the `replicas`, `inferencePoolRefs`, `templateRefs`, and `parallelism` fields and changing the `kind` to `InferenceServiceTemplate`. Also, remove the configurations that users need to provide in the `InferenceService` (e.g., `HF_TOKEN`).

```yaml
apiVersion: odin.moreh.io/v1alpha1
kind: InferenceServiceTemplate
metadata:
  name: my-custom-preset
spec:
  workerTemplate:
    spec:
      containers:
        - name: main
          env:
            - name: ISVC_MODEL_NAME
              value: meta-llama/Llama-3.2-1B-Instruct
            - name: ISVC_EXTRA_ARGS
              value: >-
                $(ISVC_MODEL_NAME)
                --disable-uvicorn-access-log
                --no-enable-log-requests
                --quantization None
                --max-model-len 4096
          resources:
            limits:
              amd.com/gpu: 1
            requests:
              amd.com/gpu: 1
      nodeSelector:
        moai.moreh.io/accelerator.vendor: amd
        moai.moreh.io/accelerator.model: mi300x
      tolerations:
        - key: amd.com/gpu
          operator: Exists
          effect: NoSchedule
```
