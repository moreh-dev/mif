---
sidebar_position: 2

title: 'Prefill-decode disaggregation'
---

import Tabs from '@theme/Tabs'; import TabItem from '@theme/TabItem';

# Prefill-decode disaggregation

During LLM inference, computation occurs in two stages: prefill and decode. In the prefill phase, the model processes the entire input prompt to generate the first token &mdash; a highly parallel, compute-bound process. The decode phase then predicts one token at a time, reusing the growing KV cache, and is memory-bound.

Because these phases have fundamentally different characteristics, prefill-decode (PD) disaggregation executes them on separate GPU resources. The prefill runs first on compute-optimized machines, then the KV cache is transferred to memory-optimized ones for decoding. This separation allows each phase to use its optimal parallelization, batch size, and configurations, while preventing interference between concurrent requests.

PD disaggregation can improve key metrics such as time to first token (TTFT) and time per output token (TPOT) &mdash; since TTFT depends on prefill and TPOT on decode, dedicated optimization for each leads to better overall performance. However, because it also introduces communication overhead, which may negatively affect TTFT, PD disaggregation should be applied judiciously to ensure net efficiency gains.

## Key features

- The **Heimdall** scheduler runs prefill-only and decode-only instances separately, allowing each to scale independently and managing request routing between them.
- The framework can automatically determine whether to apply PD disaggregation and how to scale each phase according to defined service level objectives (SLOs).
- Moreh vLLM is optimized to efficiently execute both prefill and decode phases of various models on AMD MI200 and MI300 series GPUs. It applies distinct parallelization and optimization strategies tailored to prefill-only and decode-only instances.

---

## Manual configuration of PD disaggregation

To enable PD disaggregation, configure the prefill pod separately from the decode pod in the **Odin** inference service.

```yaml inference-service-values.yaml

...
decode: ...

prefill:
  enabled: true

  replicas: 4

  resources:
    requests:
      amd.com/gpu: '2'
    limits:
      amd.com/gpu: '2'

  extraArgs: ...
```

Within the `decode` and `prefill` section, you can independently configure not only the number of replicas (`replicas`) but also the type and number of GPUs (`resources`) and the argument passed to the inference engine (`extraArgs`). Make sure that the options specified in the `extraArgs` field of the `decode` or `prefill` sections do not duplicate those defined in the global `extraArgs`.

Additionally, in the **Heimdall** scheduler, you must define separate scheduling profiles for prefill and decode as shown below.

```yaml heimdall-values.yaml
...
config:
  ...
  plugins:
    - type: pd-profile-handler
    - type: prefill-filter
    - type: decode-filter
    ...
  schedulingProfiles:
    - name: prefill
      plugins:
        - pluginRef: prefill-filter
        - pluginRef: queue-scorer
        - pluginRef: max-score-picker
    - name: decode
      plugins:
        - pluginRef: decode-filter
        - pluginRef: queue-scorer
        - pluginRef: max-score-picker
...
```

---

## Example: PD disaggregation on Llama 3.3 70B

### Benchmarking environment and configuration

| Item                  | Description                                                                                 |
| --------------------- | ------------------------------------------------------------------------------------------- |
| Servers               | 4x servers, each equipped with 4x AMD MI250 GPUs                                            |
| Networking            | InfiniBand HDR                                                                              |
| Inference Engine      | vLLM (0.10.1rc2.dev59+g0167efe20)                                                           |
| Model                 | `meta-llama/Llama-3.3-70B-Instruct`                                                         |
| PD disaggregation     | 6x prefill, 2x decode instances                                                             |
| Benchmarking tool     | [genai-bench](https://github.com/sgl-project/genai-bench)                                   |
| Benchmarking scenario | Input sequence length ~ N(3000, 300), output sequence length ~ N(200, 20), concurrency = 64 |

### Deployment

The following configuration files show how to set up PD disaggregation on the **Heimdall** scheduler and the **Odin** inference service. Prefill-only and decode-only vLLM instances each use two AMD MI250 GPUs. Six prefill instances and two decode instances &mdash; a total of eight instances run across four servers in this example.

:::info
In the `inference-service-values.yaml` file, the number of `amd.com/gpu` is set to 4 because each MI250 GPU is recognized as two logical devices at the device driver level. Therefore, four logical devices correspond to two physical GPUs. This behavior is specific to the MI250 model.
:::

<Tabs>
<TabItem value="heimdall" label="Heimdall scheduler configuration" default>

```yaml heimdall-values.yaml
global:
  imagePullSecrets:
    - name: moreh-registry

config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    - type: pd-profile-handler
    - type: prefill-filter
    - type: decode-filter
    - type: queue-scorer
    - type: max-score-picker
      parameters:
        maxNumOfEndpoints: 2
  schedulingProfiles:
    - name: prefill
      plugins:
        - pluginRef: prefill-filter
        - pluginRef: queue-scorer
        - pluginRef: max-score-picker
    - name: decode
      plugins:
        - pluginRef: decode-filter
        - pluginRef: queue-scorer
        - pluginRef: max-score-picker

gateway:
  name: mif
  gatewayClassName: istio

serviceMonitor:
  labels:
    release: prometheus-stack
```

</TabItem>
<TabItem value="odin" label="Odin inference service configuration">

```yaml inference-service-values.yaml {27}
global:
  imagePullSecrets:
    - name: moreh-registry

extraArgs:
  - meta-llama/Llama-3.3-70B-Instruct
  - --no-enable-log-requests
  - --disable-uvicorn-access-log
  - --quantization
  - 'None'
  - --kv-transfer-config
  - '{"kv_connector":"NixlConnector", "kv_role":"kv_both"}'
  - --no-enable-prefix-caching
  - --tensor-parallel-size
  - '4'
  - --max-num-batched-tokens
  - '8192'

extraEnvVars:
  - name: VLLM_NIXL_SIDE_CHANNEL_HOST
    valueFrom:
      fieldRef:
        fieldPath: status.podIP
  - name: UCX_TLS
    value: rocm_copy,rocm_ipc,self,sm,rc_x
  - name: HF_TOKEN
    value: '<huggingFaceToken>'

_common: &common
  image:
    repository: 255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/quickstart/moreh-vllm
    tag: '20250915.1'

  resources:
    requests:
      amd.com/gpu: '4'
      mellanox/hca: '1'
    limits:
      amd.com/gpu: '4'
      mellanox/hca: '1'

  podMonitor:
    labels:
      release: prometheus-stack

decode:
  replicas: 2

  <<: *common

prefill:
  replicas: 6

  <<: *common
```

</TabItem>
</Tabs>

Run the following command to deploy the services.

```shell
helm upgrade -i heimdall moreh/heimdall \
    --version v0.5.0 \
    -n mif \
    -f heimdall-values.yaml
```

```shell
helm upgrade -i inference-service moreh/inference-service \
    --version v0.3.1 \
    -n mif \
    -f inference-service-values.yaml
```

### Benchmarking

Use the **genai-bench** tool as follows to measure performance for the benchmarking scenario described above. Note that the `--api-base` option must be set to your actual endpoint URL.

```shell
genai-bench benchmark \
  --api-backend vLLM \
  --api-key anything \
  --api-base http://mif-istio.mif.svc.cluster.local \
  --api-model-name meta-llama/Llama-3.3-70B-Instruct \
  --model-tokenizer meta-llama/Llama-3.3-70B-Instruct \
  --task text-to-text \
  --max-time-per-run 1000 \
  --max-requests-per-run 3200 \
  --server-engine vLLM \
  --traffic-scenario "N(3000,300)/(200,20)" \
  --num-concurrency 64 \
  --warmup-ratio 0.05 \
  --cooldown-ratio 0.05
```

### Experimental results

We compared the performance of our PD disaggregation setup with that of a baseline configuration using a Kubernetes Service, where requests were simply distributed in a round-robin manner across eight vLLM instances without disaggregation. Time per output token (TPOT) was reduced by approximately 30% (133 → 96 ms), and as a result, the total benchmark runtime decreased by about 20% (1428.6 → 1165.8 s), even though time to first token (TTFT) was sacrificed.

**End-to-end latency:**

| Router      | PD disaggregation | Total duration (s) | Mean   | P50    | P90    | P95    | P99    |
| ----------- | ----------------- | ------------------ | ------ | ------ | ------ | ------ | ------ |
| Heimdall    | Applied           | 1165.8             | 22.826 | 22.620 | 26.734 | 28.103 | 30.582 |
| K8s Service | Not applied       | 1428.6             | 28.107 | 27.858 | 31.723 | 33.231 | 35.352 |

**TTFT (time to first token):**

| Router      | PD disaggregation | Mean (s) | P50    | P90    | P95    | P99    |
| ----------- | ----------------- | -------- | ------ | ------ | ------ | ------ |
| Heimdall    | Applied           | 3.7633   | 3.1132 | 6.9578 | 8.3785 | 10.023 |
| K8s Service | Not applied       | 1.6022   | 1.5994 | 1.8094 | 1.9000 | 2.0133 |

**TPOT (time per output token):**

| Router      | PD disaggregation | Mean (ms) | P50    | P90    | P95    | P99    |
| ----------- | ----------------- | --------- | ------ | ------ | ------ | ------ |
| Heimdall    | Applied           | 96.029    | 96.166 | 101.29 | 103.34 | 105.94 |
| K8s Service | Not applied       | 133.34    | 133.14 | 141.30 | 143.36 | 147.86 |

However, the degradation in TTFT also implies that PD disaggregation should be applied carefully depending on the SLO. The method for automating scheduling in an SLO-driven manner is described in a separate document.
