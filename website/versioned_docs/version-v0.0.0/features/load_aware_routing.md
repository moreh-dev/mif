---
sidebar_position: 5

title: 'Load-aware routing'
---

# Load-aware routing

Load-aware routing monitors the number of assigned requests and real-time utilization metrics of each inference instance (pod) to determine where the next request should be routed. Since individual requests have different workload characteristics and processing times, applying load-aware routing can achieve higher system-level efficiency than round-robin routing and especially help reduce latency variance across requests. Similar to other routing strategies such as prefix cache-aware routing, load-aware routing cannot serve as the sole routing criterion and should be combined with other metrics for optimal decision-making.

## Key features

- The **Heimdall** scheduler supports various scoring methods for load-aware routing.
- The framework can dynamically adjust the importance of load-aware routing based on defined service level objectives (SLOs) and the current traffic volume.

---

## Scorer

The **Heimdall** scheduler currently supports five scoring methods that can be manually enabled, disabled, or weighted to adjust their influence. All scores are normalized to values between 0 and 1, and a higher score indicates a lighter load &mdash; meaning the pod is more preferred for routing. The following configuration file shows an example of manully enabling all scorers and assigning them equal weights.

```yaml heimdall-values.yaml
...
config:
  apiVersion: inference.networking.x-k8s.io/v1alpha1
  kind: EndpointPickerConfig
  plugins:
    ...
    - type: queue-scorer
    - type: load-aware-scorer
      parameters:
        threshold: 128
    - type: active-request-scorer
      parameters:
        requestTimeout: "2m"
    - type: session-affinity-scorer
    - type: no-hit-lru-scorer
    - type: max-score-picker
      parameters:
        maxNumOfEndpoints: 2
  schedulingProfiles:
    - name: default
      plugins:
        ...
        - pluginRef: queue-scorer
        - pluginRef: load-aware-scorer
        - pluginRef: active-request-scorer
        - pluginRef: session-affinity-scorer
        - pluginRef: no-hit-lru-scorer
        - pluginRef: max-score-picker
        ...
...
```

### queue-scorer

It assigns scores based on the number of queued requests. The pod with the fewest queued requests receives a score of 1.0, and the one with the most receives 0.0. The others are assigned proportionally based on their relative queue lengths.

### load-aware-scorer

It assigns scores also based on the number of queued requests. A pod with no queued requests receives a score of 0.5. If the number of queued requests exceeds the threshold, it receives a score of 1.0. For values in between, the score is proportional to how many requests are waiting relative to the threshold (i.e., `0.5 + (waitingRequests / threshold)`).

Unlike the queue-scorer, this method prevents excessive score gaps between pods when there are not many pending requests or little variation in their numbers across pods.

**Parameters:**

- `threshold`: the threshold that serves as the criterion for overload

### active-request-scorer

It assigns scores based on the number of active request. A pod with no active requests receives a score of 1.0. while the pod with the most active requests receives a score of 0.0. All other pods are assigned scores proportional to their relative number of active requests.

**Parameters:**

- `requestTimeout`: If a response is not received within this time, the request is assumed to have been timed out by the inference engine (vLLM). Since the scorer does not have visibility into individual request timeouts, this assumption is necessary &mdash; otherwise, timed-out requests would remain counted as active indefinitely.

### session-affinity-scorer

It assigns a higher score if a pod has previously handled a request from the same session (with the same `x-session-token` value in the HTTP header). This indirectly produces a similar effect to prefix cache-aware routing.

### no-hit-lru-scorer

To ensure that cold requests (those without prefix cache hits) are evenly distributed across pods, scores from 0.0 to 1.0 are assigned in order from the pod that most recently received a cold request to the one that received it the longest time ago. This helps ensure that the size of the KV cache stored in either GPU memory or main memory increases evenly across pods.

On the other hand, for hot requests (those with (partial) prefix cache hits), a score of 0.5 is assigned to all pods. That means, unlike other scorers, the `no-hit-lru-scorer` is influenced not only by the state of the pods but also by the input prompts of incoming requests.

To determine whether each request has a prefix cache hit, it operates in integration with the prefix caching plugin.

**Parameters:**

- `prefixPluginName`: the name of the prefix caching plugin used to determine whether a cache hit occurs. The default value is `prefix-cache-scorer`. You must specify the actual plugin that is currently enabled.
- `lruSize`: the maximum number of pods to track for least recently used (LRU) status. The default value is 1024, meaning that only the most recent 1024 pods that received cold requests are tracked and assigned scores between 0.0 and 1.0. All other pods beyond this range are just assigned a score of 1.0.

### Checking active scorers

If you set the log level of the Heimdall scheduler to 4 (by adding `-v=4` to `extraArgs`), Heimdall will print logs like the following each time it receives a request. The scorers listed in the logs indicate that they are active and functioning.

```shell
kubectl logs -n mif -f -l app=heimdall | jq -r 'select(.msg | test("Running scorer")) | [.scorer, .msg]'
```

```shell Expected output
Defaulted container "main" out of: main, traffic-agent
[
  "queue-scorer",
  "Running scorer"
]
[
  "load-aware-scorer",
  "Running scorer"
]
[
  "active-request-scorer",
  "Running scorer"
]
[
  "session-affinity-scorer",
  "Running scorer"
]
...
```

---

## Example: round-robin routing vs load-aware routing

This example shows how load-aware routing can shorten request processing time compared to simple round-robin routing that does not account for imbalance among pods. To emulate real-world scenarios where diverse request patterns coexist, we prepare a workload consisting of requests with varying input and output sequence lengths. Adjusting these parameters and the request order, combined with round-robin routing, allows us to closely mirror real-world conditions where utilization imbalance between pods occurs. By introducing load-aware routing, we can prevent overload on any single pod, resulting in higher overall efficiency and reduced total wall-clock time for processing all requests.

### Benchmarking environment and configuration

| Item | Description |
| --- | --- |
| Servers | 2x servers, each equipped with 4x AMD MI250 GPUs |
| Networking | InfiniBand HDR |
| Inference Engine | vLLM |
| Model | `meta-llama/Llama-3.3-70B-Instruct` |
| Pods | 4x, each using 2x AMD MI250 GPUs |
| Scorers used | queue-scorer and no-hit-lru-scorer, both with a weight of 1 |

### Workload generator

The following is a Python script used to send requests with varying input and output sequence lengths.

<details>
<summary>Source code of the workload generator (`workload.py`)</summary>
```python workload.py
#!/usr/bin/env python3
import asyncio, aiohttp, time, os, argparse, statistics as stats, random, json
from typing import Dict, Any, List, Tuple
import string

# -------------------- CLI --------------------

def parse_args(): p = argparse.ArgumentParser(description="Generate a JSQ-favorable workload with inline heavy requests") p.add_argument("--api-base", default=os.environ.get("API_BASE", "http:127.0.0.1:8000"), help="Base URL (e.g., https://host)") p.add_argument("--api-key", default=os.environ.get("API_KEY", ""), help="Bearer token if your gateway requires it") p.add_argument("--model", default=os.environ.get("MODEL", "meta-llama/Llama-3.3-70B-Instruct"), help="Model name served by your endpoint") p.add_argument("--workers", "-N", type=int, default=int(os.environ.get("WORKERS", 4)), help="Logical workers behind the router (for sizing rationale only)") p.add_argument("--total", type=int, default=int(os.environ.get("TOTAL", 1000)), help="Total number of requests to send (including seeds + inline heavies)") p.add_argument("--heavy-seeds", type=int, default=int(os.environ.get("HEAVY_SEEDS", 2)), help="Heavy requests sent first to pin some workers") p.add_argument("--inline-every", type=int, default=int(os.environ.get("INLINE_EVERY", 50)), help="Insert 1 inline heavy after every N non-heavy burst items (ignored if --inline-heavy-count set)") p.add_argument("--inline-heavy-count", type=int, default=int(os.environ.get("INLINE_HEAVY_COUNT", 0)), help="Override: explicit number of inline heavy requests to insert within the burst (0 = auto from --inline-every)") p.add_argument("--seed-interval", type=float, default=float(os.environ.get("SEED_INTERVAL", 0.05)), help="Seconds between heavy seeds") p.add_argument("--burst-start", type=float, default=float(os.environ.get("BURST_START", 0.30)), help="When the burst begins (seconds after start)") p.add_argument("--burst-interval", type=float, default=float(os.environ.get("BURST_INTERVAL", 0.010)), help="Base spacing between burst requests (seconds)") p.add_argument("--jitter", type=float, default=float(os.environ.get("JITTER", 0.003)), help="Uniform ±jitter added to each burst spacing (seconds)") p.add_argument("--inline-extra-gap", type=float, default=float(os.environ.get("INLINE_EXTRA_GAP", 0.06)), help="Extra gap AFTER any inline heavy to help it pin a worker (seconds)") p.add_argument("--timeout", type=float, default=600.0, help="Per-request timeout in seconds") p.add_argument("--print-output", action="store_true", help="Print model output text for each request") p.add_argument("--save-json", default="", help="Optional path to save per-request results as JSON") return p.parse_args()

# -------------------- Helpers --------------------

def make_prompt(words: int, tag: str) -> str: """ Generate deterministic but unique prompt text per request. - Each run produces the same prompts (fixed random seed). - Each request gets different text content. - 'words' roughly controls input sequence length (ISL). """ # Use a fixed global seed for reproducibility across runs base_seed = 12345 # Derive a stable per-tag seed so same tag → same text random.seed(base_seed + sum(ord(c) for c in tag))

    vocab = [
        "quantum", "neural", "matrix", "optimization", "graph", "tensor",
        "latency", "throughput", "bandwidth", "kernel", "scheduler", "routing",
        "prefill", "decode", "scaling", "gradient", "cache", "expert",
        "activation", "pipeline", "distributed", "dynamic", "cluster", "epoch",
        "inference", "token", "sampling", "prefetch", "adapter", "mixture",
        "routing", "context", "parallelism", "load", "dispatch", "bandwidth",
        "coherence", "gradient", "topology", "fabric", "kernel"
    ]

    # Pick random words deterministically
    text_words = [random.choice(vocab) for _ in range(words)]
    # Add a little random noise (letters) to vary structure
    noise = ''.join(random.choices(string.ascii_lowercase + " ", k=min(words * 5, 2000)))

    text = f"[{tag}] " + ' '.join(text_words) + " " + noise
    # Keep safely under context limits
    return text[:min(len(text), 65000)]

def req(payload_id: str, isl_words: int, osl_tokens: int, model: str) -> Dict[str, Any]: return { "id": payload_id, "json": { "model": model, "prompt": make_prompt(isl_words, payload_id), "max_tokens": osl_tokens, "temperature": 0, "top_p": 1, "stream": False, }, }

def mk_id(prefix: str, i: int, width: int) -> str: return f"{prefix}{i:0{width}d}"

# -------------------- Workload builders --------------------

HEAVY_SPECS: List[Tuple[int, int]] = [ (1300, 320), (1500, 350), (2700, 780), (4000, 1050), ]

def build_workload(args) -> List[Dict[str, Any]]: """ Build list of request dicts: - 'heavy-seeds' at the very beginning - a burst mixing short/medium requests - inline heavy requests sprinkled inside the burst Total count ~= args.total """ random.seed(42) workload: List[Dict[str, Any]] = []

    # 1) Seed heavies
    for i in range(args.heavy_seeds):
        isl, osl = HEAVY_SPECS[i % len(HEAVY_SPECS)]
        workload.append(req(mk_id("S", i+1, 3) + "_HEAVY", isl, osl, args.model))

    remaining = max(0, args.total - args.heavy_seeds)

    # Determine inline heavy count
    if args.inline_heavy_count and args.inline_heavy_count > 0:
        inline_heavy_count = min(args.inline_heavy_count, remaining)
    else:
        # auto: ~1 inline heavy after every 'inline_every' non-heavy
        inline_heavy_count = max(0, remaining // (args.inline_every + 1))

    non_heavy_burst = max(0, remaining - inline_heavy_count)
    short_cnt = non_heavy_burst // 2
    medium_cnt = non_heavy_burst - short_cnt

    # Build burst pool and shuffle
    burst_pool: List[Tuple[str, int, int]] = []
    for _ in range(short_cnt):
        isl = random.randint(10, 20)
        osl = random.randint(25, 40)
        burst_pool.append(("SHORT", isl, osl))
    for _ in range(medium_cnt):
        isl = random.randint(120, 180)
        osl = random.randint(60, 90)
        burst_pool.append(("MED", isl, osl))
    random.shuffle(burst_pool)

    # Compute insertion positions to spread inline heavies across the burst
    inline_pos = []
    if inline_heavy_count > 0 and len(burst_pool) > 0:
        step = max(1, len(burst_pool) // inline_heavy_count)
        inline_pos = [min(len(burst_pool), (k+1)*step) for k in range(inline_heavy_count)]

    # Assemble burst + inline heavies
    inline_used = 0
    heavy_cursor = 0
    built_burst: List[Tuple[str, int, int]] = []
    for idx, item in enumerate(burst_pool, start=1):
        built_burst.append(item)
        if inline_used < len(inline_pos) and idx == inline_pos[inline_used]:
            isl, osl = HEAVY_SPECS[heavy_cursor % len(HEAVY_SPECS)]
            built_burst.append(("HEAVY_INLINE", isl, osl))
            heavy_cursor += 1
            inline_used += 1

    # Convert to request dicts with IDs
    b_short = b_med = b_heavy_inline = 0
    for kind, isl, osl in built_burst:
        if kind == "SHORT":
            b_short += 1
            workload.append(req(mk_id("B", b_short, 4) + "_SHORT", isl, osl, args.model))
        elif kind == "MED":
            b_med += 1
            workload.append(req(mk_id("M", b_med, 4) + "_MED", isl, osl, args.model))
        else:
            b_heavy_inline += 1
            workload.append(req(mk_id("I", b_heavy_inline, 3) + "_HEAVY", isl, osl, args.model))

    # Quick summary
    print(f"[Workload] workers={args.workers}  total={len(workload)} | "
          f"seed_heavy={args.heavy_seeds}, inline_heavy={inline_heavy_count}, "
          f"nonheavy={non_heavy_burst} (short={short_cnt}, med={medium_cnt})")

    return workload

def build_schedule(workload: List[Dict[str, Any]], args) -> Dict[str, float]: """ Assign a launch time to every request: - seeds: 0, seed_interval, 2\*seed_interval, ... - burst: starts at burst_start, spaced by burst_interval ± jitter add inline_extra_gap AFTER any inline heavy """ # Map id -> time sched: Dict[str, float] = {}

    # Seeds first
    for i in range(args.heavy_seeds):
        sched[workload[i]["id"]] = i * args.seed_interval

    # Burst
    t = args.burst_start
    for w in workload[args.heavy_seeds:]:
        rid = w["id"]
        sched[rid] = t
        # base spacing with jitter
        dt = args.burst_interval + (random.uniform(-args.jitter, args.jitter) if args.jitter > 0 else 0.0)
        # inline heavy → add an extra gap AFTER it
        if rid.endswith("_HEAVY") and rid.startswith("I"):
            dt += args.inline_extra_gap
        t += max(0.0, dt)

    print(f"[Timing] seed_interval={args.seed_interval:.3f}s  burst_start={args.burst_start:.3f}s  "
          f"burst_interval={args.burst_interval:.3f}s  jitter=±{args.jitter:.3f}s  "
          f"inline_extra_gap={args.inline_extra_gap:.3f}s")
    return sched

# -------------------- HTTP runner --------------------

async def post_one(session: aiohttp.ClientSession, url: str, headers: Dict[str, str], r: Dict[str, Any], timeout: float, print_output: bool): t0 = time.perf_counter() async with session.post(url, json=r["json"], headers=headers, timeout=timeout) as resp: if resp.status == 200: data = await resp.json() else: # capture some text for debugging on error data = {"error_text": (await resp.text())[:1000]} t1 = time.perf_counter()

    text = ""
    if resp.status == 200 and isinstance(data, dict) and "choices" in data and data["choices"]:
        text = data["choices"][0].get("text", "").strip()

    if print_output:
        print(f"\n=== {r['id']} ===")
        print(f"Status: {resp.status}  Latency: {t1 - t0:.3f}s")
        print("=" * 60)

    return {
        "id": r["id"],
        "status": resp.status,
        "latency_s": t1 - t0,
        "resp_text": text,
        "error": None if resp.status == 200 else data,
    }

# -------------------- Main --------------------

async def main(): args = parse_args() random.seed(42) # deterministic timing/jitter

    # Build workload & schedule
    workload = build_workload(args)
    schedule = build_schedule(workload, args)

    url = f"{args.api_base.rstrip('/')}/v1/completions"
    headers = {"Content-Type": "application/json"}
    if args.api_key:
        headers["Authorization"] = f"Bearer {args.api_key}"

    start = time.perf_counter()
    results: List[Dict[str, Any]] = []

    conn = aiohttp.TCPConnector(
        limit=0,  # 0 = no limit
        limit_per_host=0,   # no per-host limit
        ttl_dns_cache=300
    )

    async with aiohttp.ClientSession(connector=conn) as session:
        tasks = []

        async def launcher(r):
            launch_at = schedule[r["id"]]
            # sleep until its scheduled time relative to 'start'
            await asyncio.sleep(max(0.0, launch_at - (time.perf_counter() - start)))

            now = time.perf_counter() - start
            print(f"[SEND] {r['id']:<15}  scheduled={launch_at:.3f}s  actual={now:.3f}s")

            return await post_one(session, url, headers, r, args.timeout, args.print_output)

        for r in workload:
            tasks.append(asyncio.create_task(launcher(r)))

        # Gather as they finish
        for t in tasks:
            res = await t
            results.append(res)
            # quick per-request line:
            print(f"[{res['id']}] status={res['status']} latency={res['latency_s']:.3f}s")

    # Summaries
    lat_ok = [r["latency_s"] for r in results if r["status"] == 200]
    summary = {}
    if lat_ok:
        avg = sum(lat_ok) / len(lat_ok)
        p50 = stats.median(lat_ok)
        p95 = sorted(lat_ok)[max(0, int(len(lat_ok) * 0.95) - 1)]
        print("\n=== Latency Summary (successful requests) ===")
        print(f"count={len(lat_ok)}  avg={avg:.3f}s  p50={p50:.3f}s  p95={p95:.3f}s")
        summary.update({"count": len(lat_ok), "avg": avg, "p50": p50, "p95": p95})

    # Total E2E wall-clock
    end = time.perf_counter()
    total_e2e = end - start
    print(f"\n=== Total End-to-End (E2E) Latency ===")
    print(f"All requests completed in {total_e2e:.3f} seconds")
    summary["total_e2e"] = total_e2e

    # Optional save
    if args.save_json:
        try:
            payload = {"requests": results, "summary": summary}
            with open(args.save_json, "w", encoding="utf-8") as f:
                json.dump(payload, f, ensure_ascii=False, indent=2)
            print(f"\nSaved results to {args.save_json}")
        except Exception as e:
            print(f"\nFailed to save JSON: {e}")

if **name** == "**main**": asyncio.run(main())

````
</details>

You can execute the workload generator as follows. Please replace `{SERVER_ENDPOINT}` and `{SAVE_DIRECTORY}` with your own values.

```sh
python3 workload.py --api-base {SERVER_ENDPOINT} --save-json {SAVE_DIRECTORY} --heavy-seeds 400 --seed-interval 0 --total 800 --burst-start 5 --burst-interval 0.5 --jitter 0.5 --inline-extra-gap 0.5 --inline-every 24 --timeout 10000
````

It sends a total of 800 requests, with the first 400 designated as heavy workloads. Among these heavy requests, one super-heavy request is sent after every three normal ones. The purpose of sending 400 heavy requests first is to quickly saturate the pods, causing subsequent requests to be queued. Afterwards, shorter requests are used to evaluate the effect of the routing mechanisms. To introduce additional stress to the system, a heavy request is injected after every 24 requests.

### Experimental results

The following table shows the total wall-clock time elapsed to process all requests when applying round-robin routing versus load-aware routing. When load-aware routing was applied, the total wall-clock time was reduced by ~18%.

| Metric                | Round-robin routing | Load-aware routing     |
| --------------------- | ------------------- | ---------------------- |
| Total wall-clock time | 560.02 s            | 458.87 s **(-18.06%)** |

A deeper analysis can be conducted by examining the queuing behavior under both routing mechanisms. The following images illustrate the queue size (the number of pending requests) of each pod while processing incoming requests.

| Routing | Queue size trend of each pod over time |
| --- | --- |
| Round-robin | ![Round Robin](./images/load_aware_routing__round_robin_queue.png) |
| Load-aware | ![Load-aware](./images/load_aware_routing__load_aware_queue.png) |

Under round-robin routing, after one minute has passed, the pod represented by the blue line begins to lag behind, taking longer to complete its requests and accumulating a larger queue. This imbalance continues to widen, and even after all other pods finish processing their requests, the blue pod continues handling the requests assigned to it for roughly four minutes.

In contrast, load-aware routing does not suffer from such imbalance &mdash; it maintains balanced queues across the pods. The pod represented by the green line takes more time to complete, but the difference from other pods is only about two minutes. This indicates that load-aware routing achieves a much more balanced workload distribution and improves overall processing efficiency.
