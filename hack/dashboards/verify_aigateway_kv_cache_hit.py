#!/usr/bin/env python3
"""Run the AIGateway dashboard's request-observed KV cache hit panel queries
against a local Prometheus fed by fake gateways, and check the values each
panel reports.

The fake gateways model a rolling update from the old counters
(aigateway_prompt_tokens_observed_total, aigateway_cached_tokens_reports_total)
to the per-request histograms (aigateway_engine_input_tokens,
aigateway_engine_cached_tokens, aigateway_predicted_cached_tokens). Per
second:

- model A, new pod: three requests of 100 input tokens. Two had 64 cached
  tokens and a 48-token prediction; one had no cached_tokens.
- model A, old pod: 64 cached and 36 prefilled tokens from one reported
  request, plus one absent request.
- model B, old pod: two absent requests only, like a vLLM worker without
  --enable-prompt-tokens-details. The old gateway never creates the token
  series for it.
- model C, new pod: one request of 50 input tokens with no cached_tokens.
  The new gateway still creates the cached series, at count 0.

Usage:
    python3 hack/dashboards/verify_aigateway_kv_cache_hit.py --prometheus /path/to/prometheus

The Prometheus binary comes from https://github.com/prometheus/prometheus/releases.
Exits non-zero when a panel query returns a value other than the expected one.
"""

import argparse
import http.server
import json
import math
import pathlib
import socket
import subprocess
import sys
import tempfile
import threading
import time
import urllib.parse
import urllib.request

DASHBOARD = (
    pathlib.Path(__file__).resolve().parents[2]
    / "deploy/helm/moai-inference-framework/files/dashboards/mif-aigateway.json"
)

LABELS = 'namespace="ns",aigateway="gw"'

# The gateway's token buckets: 1 to 262144, four times apart.
BUCKETS = [4**i for i in range(10)]


def histogram(name, model, observations):
    """Series of one histogram, as (series, value per second).

    `observations` lists (count per second, tokens per observation).
    """
    labels = f'model="{model}"'
    series = []
    for le in BUCKETS:
        count = sum(c for c, v in observations if v <= le)
        series.append((f'{name}_bucket{{{labels},le="{le}"}}', count))
    total = sum(c for c, _ in observations)
    series.append((f'{name}_bucket{{{labels},le="+Inf"}}', total))
    series.append((f"{name}_sum{{{labels}}}", sum(c * v for c, v in observations)))
    series.append((f"{name}_count{{{labels}}}", total))
    return series


PODS = {
    "new-a": histogram("aigateway_engine_input_tokens", "A", [(3, 100)])
    + histogram("aigateway_engine_cached_tokens", "A", [(2, 64)])
    + histogram("aigateway_predicted_cached_tokens", "A", [(2, 48)]),
    "old-a": [
        ('aigateway_prompt_tokens_observed_total{model="A",source="cache"}', 64),
        ('aigateway_prompt_tokens_observed_total{model="A",source="prefill"}', 36),
        ('aigateway_cached_tokens_reports_total{model="A",result="reported"}', 1),
        ('aigateway_cached_tokens_reports_total{model="A",result="absent"}', 1),
    ],
    "old-b": [
        ('aigateway_cached_tokens_reports_total{model="B",result="absent"}', 2),
    ],
    "new-c": histogram("aigateway_engine_input_tokens", "C", [(1, 50)])
    + histogram("aigateway_engine_cached_tokens", "C", []),
}

NAN = float("nan")

# Expected value per (panel id, target refId), keyed by model. A missing model
# means the query must return no series for it.
EXPECTED = {
    # Cached over input tokens across both pods of model A. Model B exported no
    # tokens, so it has no ratio instead of a false 0%.
    (84, "A"): {"A": (128 + 64) / (300 + 100), "C": 0.0},
    # Only the new pod predicts, over its own input tokens.
    (84, "B"): {"A": 96 / 300},
    # Requests with cached tokens over all requests, either version. Model B
    # has none and shows 0%, not a missing series.
    (85, "A"): {"A": (2 + 1) / (3 + 2), "B": 0.0, "C": 0.0},
    # Every request of model A with cached tokens on the new pod was predicted.
    (85, "B"): {"A": 1.0},
    (86, "A"): {"A": 300 + 100, "C": 50},
    (86, "B"): {"A": 128 + 64, "C": 0.0},
    (86, "C"): {"A": 96},
    # Every cached observation of model A is 64 tokens, in the (16, 64] bucket.
    (87, "A"): {"A": 16 + (64 - 16) * 0.50, "C": NAN},
    (87, "B"): {"A": 16 + (64 - 16) * 0.95, "C": NAN},
}

RATE_WINDOW = "20s"


def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def serve_pod(port, series, started):
    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            elapsed = time.time() - started
            body = "".join(
                f"{name.replace('{', '{' + LABELS + ',', 1)} {rate * elapsed}\n"
                for name, rate in series
            )
            self.send_response(200)
            self.send_header("Content-Type", "text/plain; version=0.0.4")
            self.end_headers()
            self.wfile.write(body.encode())

        def log_message(self, *args):
            pass

    server = http.server.HTTPServer(("127.0.0.1", port), Handler)
    threading.Thread(target=server.serve_forever, daemon=True).start()


def query(base, expr):
    url = f"{base}/api/v1/query?" + urllib.parse.urlencode({"query": expr})
    with urllib.request.urlopen(url) as resp:
        body = json.load(resp)
    if body["status"] != "success":
        raise RuntimeError(f"query failed: {body}")
    return {r["metric"].get("model"): float(r["value"][1]) for r in body["data"]["result"]}


def close(got, want):
    if math.isnan(want):
        return math.isnan(got)
    return math.isclose(got, want, rel_tol=1e-3, abs_tol=1e-9)


def panel_exprs():
    dashboard = json.loads(DASHBOARD.read_text())
    panels = {p["id"]: p for p in dashboard["panels"]}
    for (panel_id, ref_id) in EXPECTED:
        target = next(t for t in panels[panel_id]["targets"] if t["refId"] == ref_id)
        expr = (
            target["expr"]
            .replace("$namespace", ".*")
            .replace("$aigateway", ".*")
            .replace("$model", ".*")
            .replace("$__rate_interval", RATE_WINDOW)
        )
        yield (panel_id, ref_id), panels[panel_id]["title"], expr


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("--prometheus", default="prometheus", help="Prometheus binary")
    args = parser.parse_args()

    started = time.time()
    targets = []
    for series in PODS.values():
        port = free_port()
        serve_pod(port, series, started)
        targets.append(f"127.0.0.1:{port}")

    prom_port = free_port()
    base = f"http://127.0.0.1:{prom_port}"
    with tempfile.TemporaryDirectory() as tmp:
        config = pathlib.Path(tmp, "prometheus.yml")
        config.write_text(
            "global: {scrape_interval: 1s}\n"
            "scrape_configs:\n"
            "  - job_name: fake-aigateway\n"
            "    honor_labels: true\n"
            f"    static_configs: [{{targets: {json.dumps(targets)}}}]\n"
        )
        prom = subprocess.Popen(
            [
                args.prometheus,
                f"--config.file={config}",
                f"--web.listen-address=127.0.0.1:{prom_port}",
                f"--storage.tsdb.path={tmp}/data",
            ],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        try:
            # Enough samples in every series for a full rate window.
            deadline = time.time() + 120
            while True:
                try:
                    counts = query(base, "min(count_over_time(up[1m]))")
                    if counts and min(counts.values()) >= 25:
                        break
                except OSError:
                    pass
                if time.time() > deadline:
                    sys.exit("Prometheus collected no samples within 120s")
                time.sleep(1)

            failed = False
            for key, title, expr in panel_exprs():
                got = query(base, expr)
                want = EXPECTED[key]
                ok = got.keys() == want.keys() and all(close(got[m], want[m]) for m in want)
                failed |= not ok
                print(f"{'ok  ' if ok else 'FAIL'} panel {key[0]} {key[1]} {title}: got {got}, want {want}")
            sys.exit(1 if failed else 0)
        finally:
            prom.terminate()
            prom.wait()


if __name__ == "__main__":
    main()
