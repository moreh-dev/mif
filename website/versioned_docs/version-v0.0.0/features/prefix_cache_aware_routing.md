---
sidebar_position: 4

title: 'Prefix cache-aware routing'
---

import Tabs from '@theme/Tabs'; import TabItem from '@theme/TabItem';

# Prefix cache-aware routing

Prefix caching refers to a technique that stores the KV cache from previous queries, allowing subsequent queries with an identical prefix to reuse it, thereby eliminating redundant computation and improving performance. Since multiple queries often share common prefixes &mdash; such as system prompts, conversation history, or contextual documents, &mdash; recomputing the KV cache for every request would be highly inefficient.

In a system composed of multiple inference instances (Pods), each instance maintains its own (L1) prefix cache in GPU memory. As a result, the cache hit rate (the length of the cached prefix) can vary depending on which instance a request is routed to. Prefix cache-aware routing calculates the cache hit rate of the given request for each Pod and prioritizes routing to the Pod with the highest cache coverage. This reduces redundant KV computation and improves both time to first token (TTFT) and overall throughput.

However, in real-world inference systems, the cache hit rate alone cannot serve as the sole routing criterion. It must be considered alongside other factors &mdash; such as the workload characteristics of the requests and the current state of each Pod &mdash; to make optimal routing decisions.

## Key features

- The **Heimdall** scheduler tokenizes the request prompt, calculates the cache hit rate for each Pod, and assigns a normalized score to each Pod so that it can be used as a routing decision criterion. It continuously receives updates on each Pod's cache status through ZMQ events.
- The framework can determine how much importance to assign to prefix cache-aware routing based on the given service level objectives (SLOs) and the computation characteristics of the GPUs (the penalty of KV cache recomputation).

---

## Scorer

Prefix cache-aware routing is applied by enabling and configuring **precise-prefix-cache-scorer** in the **Heimdall** scheduler. The following configuration file shows an example of setting up the scorer, including each pod's prefix cache information and the model tokenizer details.

```yaml heimdall-values.yaml {24}
...
config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    ...
    - type: precise-prefix-cache-scorer
      parameters:
        indexerConfig:
          prefixStoreConfig:
            cacheSize: 500000
            blockSize: 256
          tokenProcessorConfig:
            blockSize: 16
            hashSeed: "12345"
          kvBlockIndexConfig:
            inMemoryConfig:
              size: 100000000
              podCacheSize: 10
            enableMetrics: true
          tokenizersPoolConfig:
            workersCount: 8
            minPrefixOverlapRatio: 0.8
            huggingFaceToken: <huggingFaceToken>
            tokenizersCacheDir: "/tmp"
        kvEventsConfig:
          zmqEndpoint: "tcp://*:5557"
          topicFilter: "kv@"
          concurrency: 16
    - type: max-score-picker
      parameters:
        maxNumOfEndpoints: 2
  schedulingProfiles:
    - name: default
      plugins:
        ...
        - pluginRef: precise-prefix-cache-scorer
        - pluginRef: max-score-picker
        ...
...
```

### Tokenizer and prefix store

Each vLLM instance (pod) manages its own prefix cache, which uses tokenized sequences rather than raw prompts as cache keys. Therefore, to determine prefix cache hit rates, the scorer must first tokenize each incoming prompt using the model's tokenizer.

Although multiple worker processes are used to increase concurrency, tokenizing every request can still introduce significant overhead. To reduce this cost, the scorer maintains a cache called the **prefix store**, which holds previously computed tokenization results. It uses a combination of model (to identify the tokenizer) and prompt as the key, and the tokenized sequence as the value.

If a large portion of a new request's prompt prefix &mdash; for example, more than 80% &mdash; has already been tokenized, the scorer simply reuses cached results to estimate prefix cache hit rates. It even skips tokenizing the remaining unmatched suffix, since calculating the rate using only the first >80% of the prompt yields results nearly identical to those obtained with the full prompt. This is especially reasonable because the goal is not to compute an exact hit rate value, but to identify pods with more prefix cache hits.

**Parameters:**

- `indexerConfig.prefixStoreConfig`: configuration for the prefix store. `cacheSize * blockSize` will be the capacity of the prefix store.
  - `cacheSize`: the maximum number of blocks.
  - `blockSize`: the number of characters in each block.
- `indexerConfig.tokenizersPoolConfig`: configuration for the tokenizer worker processes.
  - `workersCount`: the number of workers.
  - `minPrefixOverlapRatio`: threshold for reusing cached results in the prefix store. A value between 0 and 1.
  - `huggingFaceToken`: Hugging Face token required to download the tokenizer.
  - `tokenizersCacheDir`: Tokenizer download path.

### Token processor

vLLM uses block hashes to efficiently look up all possible prefixes of the current input sequence in the prefix cache. For a given sequence, the first block (the first _B_ tokens) has one hash value, the first and second blocks together (the first _2B_ tokens) have another, ..., and finally the entire sequence has its own last hash value. These hash values serve as actual keys for the prefix cache. For detailed behavior, please refer to the [Automatic Prefix Caching](https://docs.vllm.ai/en/latest/design/prefix_caching/) page.

To emulate vLLM's prefix cache access, the scorer must also compute the block hashes for a given request in the same way as vLLM does.

**Parameters:**

- `indexerConfig.tokenProcessorConfig`
  - `blockSize`: hash block size, must be identical to the vLLM configuration of each pod.
  - `hashSeed`: must be identical to vLLM's `PYTHONHASHSEED` of each pod.

### KV block index

Each vLLM instance (pod) publishes events (`BlockStored`, `BlockRemoved`, and `AllCacheCleared`) via ZMQ whenever its prefix cache is updated. The scorer subscribes to the ZMQ channels of all pods to receive these events, thereby maintaining a complete view of the overall prefix cache status. This information is stored in a data structure called the **KV block index**.

The KV block index uses a prefix hash value as the key and stores a list of pods that hold the corresponding KV cache as the value. KV cache values for the same prefix may exist on multiple pods (since previous requests with that prefix could have been routed to different pods), so the value in the KV block index must be a list of pods.

For each incoming request, the scorer tokenizes the prompt into blocks and hashes each block to query the KV block index. The pod that holds the largest number of matching blocks is considered to have the highest cache hit rate.

**Parameters:**

- `indexerConfig.kvBlockIndexConfig`: configurations for the KV block index.
  - `inMemoryConfig.size`: the maximum number of entries (hash keys).
  - `inMemoryConfig.podCacheSize`: the maximum length of the pod list for each hash key. If the KV cache for a given prefix is actually stored in more than `podCacheSize` pods, only `podCacheSize` of them are selectively recorded. However, if this value is larger than the total number of vLLM worker pods, it does not affect the result of selecting the pod with the highest cache hit rate.
  - `enableMetrics`: enables Prometheus to collect metrics related to the KV block index.
- `kvEventsConfig`: ZMQ subscription configuration.
  - `zmqEndpoint`: ZMQ endpoint for communication with vLLM pods.
  - `topicFilter`: a string used to filter prefix cache events only.
  - `concurrency`: the number of workers for receiving ZMQ events and maintaining the KV block index.

---

## Example: random routing vs prefix cache-aware routing

This example shows how prefix cache-aware routing can improve time to first token (TTFT) and end-to-end latency compared with random routing which selects one of the pods at random for each request.

### Benchmarking environment and configuration

| Item             | Description                                      |
| ---------------- | ------------------------------------------------ |
| Servers          | 4x servers, each equipped with 4x AMD MI250 GPUs |
| Networking       | InfiniBand HDR                                   |
| Inference Engine | vLLM                                             |
| Model            | `Qwen/Qwen3-32B`                                 |
| Pods             | 8x, each using 2x AMD MI250 GPUs                 |

### Deployment

The following configuration file shows how to set up the **precise-prefix-cache-scorer** on the **Heimdall** scheduler. Each of the eight instances runs on two AMD MI250 GPUs and maintains its own separate prefix cache.

:::info
In the `inference-service-values.yaml` file, the number of `amd.com/gpu` is set to 4 because each MI250 GPU is recognized as two logical devices at the device driver level. Therefore, four logical devices correspond to two physical GPUs. This behavior is specific to the MI250 model.
:::

<Tabs>
<TabItem value="heimdall" label="Heimdall scheduler configuration" default>

```yaml heimdall-values.yaml {55}
global:
  imagePullSecrets:
    - name: moreh-registry

config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    - type: single-profile-handler
    - type: queue-scorer
    - type: kv-cache-utilization-scorer
    - type: max-score-picker
    - type: precise-prefix-cache-scorer
      parameters:
        indexerConfig:
          prefixStoreConfig:
            cacheSize: 500000
            blockSize: 256
          tokenProcessorConfig:
            blockSize: 32
            hashSeed: '12345'
          kvBlockIndexConfig:
            inMemoryConfig:
              size: 100000000
              podCacheSize: 10
            enableMetrics: true
          tokenizersPoolConfig:
            workersCount: 8
            minPrefixOverlapRatio: 0.8
            tokenizersCacheDir: '/tmp'
        kvEventsConfig:
          zmqEndpoint: 'tcp://*:5557'
          topicFilter: 'kv@'
          concurrency: 16
  schedulingProfiles:
    - name: default
      plugins:
        - pluginRef: queue-scorer
          weight: 2
        - pluginRef: kv-cache-utilization-scorer
          weight: 2
        - pluginRef: precise-prefix-cache-scorer
          weight: 3
        - pluginRef: max-score-picker
gateway:
  name: mif
  gatewayClassName: istio

serviceMonitor:
  labels:
    release: prometheus-stack

extraEnvVars:
  - name: HF_TOKEN
    value: <huggingFaceToken>
```

</TabItem>
<TabItem value="odin" label="Odin inference service configuration">

```yaml inference-service-values.yaml {41}
global:
  imagePullSecrets:
    - name: moreh-registry

extraArgs:
  - Qwen/Qwen3-32B
  - --no-enable-log-requests
  - --disable-uvicorn-access-log
  - --quantization
  - 'None'
  - --kv-transfer-config
  - '{"kv_connector":"NixlConnector", "kv_role":"kv_both"}'
  - --max-num-batched-tokens
  - '65536'
  - --max-num-seqs
  - '512'
  - --prefix-caching-hash-algo
  - sha256_cbor
  - --block-size
  - '32'
  - --kv-events-config
  - |
    {
      "enable_kv_cache_events": true,
      "publisher": "zmq",
      "endpoint": "tcp://heimdall.mif.svc.cluster.local:5557",
      "topic": "kv@$(POD_IP)@Qwen/Qwen3-32B",
      "buffer_steps": 1,
      "hwm": 10000,
      "max_queue_size": 10000
    }

extraEnvVars:
  - name: VLLM_NIXL_SIDE_CHANNEL_HOST
    valueFrom:
      fieldRef:
        fieldPath: status.podIP
  - name: UCX_TLS
    value: rocm_copy,rocm_ipc,self,sm,rc_x
  - name: HF_TOKEN
    value: '<huggingFaceToken>'
  - name: PYTHONHASHSEED
    value: '12345'
  - name: VLLM_PORT
    value: '8000'

_common: &common
  image:
    repository: '255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/quickstart/moreh-vllm'
    tag: '20250915.1'

  podMonitor:
    labels:
      release: prometheus-stack

decode:
  replicas: 8

  <<: *common

  extraArgs:
    - --tensor-parallel-size
    - '4'

  resources:
    limits:
      amd.com/gpu: '4'
      mellanox/hca: '1'
    requests:
      amd.com/gpu: '4'
      mellanox/hca: '1'

prefill:
  enabled: false
```

</TabItem>
</Tabs>

### Experimental result

We alternated among 230 different prompt groups, each containing 5 unique user prompts. Each request consisted of an 8,000-token system prompt and a 1,000-token user prompt (9,000 tokens total), generating 1,000 output tokens. Starting with 46 requests per second (RPS) for warmup and gradually increasing from 3 to 100 RPS across multiple stages, we measured time to first token (TTFT) across different load levels.

As a result, when prefix cache-aware routing was applied, the median TTFT was dramatically reduced from 4.46 seconds to 0.22 seconds under random routing &mdash; approximately 20 times faster. The P75 TTFT improved from 9.29 seconds to 0.72 seconds, and the P90 TTFT improved from 17.96 seconds to 1.11 seconds. These results demonstrate that prefix cache-aware routing significantly reduces TTFT by effectively reusing cached prefixes across multiple inference instances, particularly in scenarios with shared system prompts.

**TTFT (time to first token):**

| Routing                    | P50 (ms)    | P75 (ms)    | P90 (ms)     |
| -------------------------- | ----------- | ----------- | ------------ |
| Random routing             | 4464.611    | 9292.547    | 17961.767    |
| Prefix cache-aware routing | **217.383** | **718.785** | **1106.871** |
