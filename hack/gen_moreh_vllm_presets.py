#!/usr/bin/env python3
"""Generate moreh-vllm InferenceServiceTemplate Helm preset files.

The preset list comes from the moreh-vllm container image:

    docker run --rm <image> ls /app/moreh-vllm/presets/

Usage:
    python hack/gen_moreh_vllm_presets.py \\
        --version 0.15.0-260210-rc1 \\
        --output-dir deploy/helm/moai-inference-preset/templates/presets/moreh-vllm/0.15.0-260210-rc1 \\
        $(docker run --rm <image> ls /app/moreh-vllm/presets/)
"""

import argparse
import os
import re
import shutil
import sys
from typing import Optional

ECR_IMAGE_BASE = "255250787067.dkr.ecr.ap-northeast-2.amazonaws.com/moreh-vllm"
PRESET_PATH_BASE = "/app/moreh-vllm/presets"

KNOWN_ACCEL_VENDORS = {"amd"}

# Maps the kebab-case {org}-{model} portion of a preset stem to model metadata.
#
# gpu_overrides: parallelism_suffix → gpu_count, for presets where the default
# derivation in parse_parallelism_suffix() gives the wrong value.
MODEL_REGISTRY: dict[str, dict] = {
    "deepseek-ai-deepseek-r1": {
        "org_label": "deepseek-ai",
        "name_label": "deepseek-r1",
        "hf_path": "deepseek-ai/DeepSeek-R1",
    },
    "deepseek-ai-deepseek-v3.2": {
        "org_label": "deepseek-ai",
        "name_label": "deepseek-v3.2",
        "hf_path": "deepseek-ai/DeepSeek-V3.2",
    },
    "lmsys-gpt-oss-20b-bf16": {
        "org_label": "lmsys",
        "name_label": "gpt-oss-20b-bf16",
        "hf_path": "lmsys/gpt-oss-20b-bf16",
    },
    "lmsys-gpt-oss-120b-bf16": {
        "org_label": "lmsys",
        "name_label": "gpt-oss-120b-bf16",
        "hf_path": "lmsys/gpt-oss-120b-bf16",
    },
    "openai-gpt-oss-20b": {
        "org_label": "openai",
        "name_label": "gpt-oss-20b",
        "hf_path": "openai/gpt-oss-20b",
    },
    "openai-gpt-oss-120b": {
        "org_label": "openai",
        "name_label": "gpt-oss-120b",
        "hf_path": "openai/gpt-oss-120b",
    },
    "lgai-exaone-exaone-3.5-7.8b-instruct": {
        "org_label": "lgai-exaone",
        "name_label": "exaone-3.5-7.8b-instruct",
        "hf_path": "LGAI-EXAONE/EXAONE-3.5-7.8B-Instruct",
    },
    "lgai-exaone-exaone-3.5-32b-instruct": {
        "org_label": "lgai-exaone",
        "name_label": "exaone-3.5-32b-instruct",
        "hf_path": "LGAI-EXAONE/EXAONE-3.5-32B-Instruct",
    },
    "meta-llama-llama-3.3-70b-instruct": {
        "org_label": "meta-llama",
        "name_label": "llama-3.3-70b-instruct",
        "hf_path": "meta-llama/Llama-3.3-70B-Instruct",
    },
    "qwen-qwen3-omni-30b-a3b-thinking": {
        "org_label": "qwen",
        "name_label": "qwen3-omni-30b-a3b-thinking",
        "hf_path": "Qwen/Qwen3-Omni-30B-A3B-Thinking",
    },
    "deepseek-ai-deepseek-r1-0528": {
        "org_label": "deepseek-ai",
        "name_label": "deepseek-r1-0528",
        "hf_path": "deepseek-ai/DeepSeek-R1-0528",
    },
    "meta-llama-llama-3.2-1b": {
        "org_label": "meta-llama",
        "name_label": "llama-3.2-1b",
        "hf_path": "meta-llama/Llama-3.2-1B",
    },
    "meta-llama-llama-3.2-3b": {
        "org_label": "meta-llama",
        "name_label": "llama-3.2-3b",
        "hf_path": "meta-llama/Llama-3.2-3B",
    },
    "qwen-qwen3-235b-a22b-2507-fp8": {
        "org_label": "qwen",
        "name_label": "qwen3-235b-a22b-2507-fp8",
        "hf_path": "Qwen/Qwen3-235B-A22B-2507-FP8",
    },
    "qwen-qwen3-30b-a3b-2507-fp8": {
        "org_label": "qwen",
        "name_label": "qwen3-30b-a3b-2507-fp8",
        "hf_path": "Qwen/Qwen3-30B-A3B-2507-FP8",
    },
    "deepseek-ai-deepseek-v3.2-exp": {
        "org_label": "deepseek-ai",
        "name_label": "deepseek-v3.2-exp",
        "hf_path": "deepseek-ai/DeepSeek-V3.2-Exp",
    },
    "deepseek-ai-deepseek-v3.2-speciale": {
        "org_label": "deepseek-ai",
        "name_label": "deepseek-v3.2-speciale",
        "hf_path": "deepseek-ai/DeepSeek-V3.2-Speciale",
    },
    "meta-llama-llama-3.2-1b-instruct": {
        "org_label": "meta-llama",
        "name_label": "llama-3.2-1b-instruct",
        "hf_path": "meta-llama/Llama-3.2-1B-Instruct",
    },
    "meta-llama-llama-3.2-3b-instruct": {
        "org_label": "meta-llama",
        "name_label": "llama-3.2-3b-instruct",
        "hf_path": "meta-llama/Llama-3.2-3B-Instruct",
    },
    "qwen-qwen3-235b-a22b-instruct-2507-fp8": {
        "org_label": "qwen",
        "name_label": "qwen3-235b-a22b-instruct-2507-fp8",
        "hf_path": "Qwen/Qwen3-235B-A22B-Instruct-2507-FP8",
    },
    "qwen-qwen3-235b-a22b-thinking-2507-fp8": {
        "org_label": "qwen",
        "name_label": "qwen3-235b-a22b-thinking-2507-fp8",
        "hf_path": "Qwen/Qwen3-235B-A22B-Thinking-2507-FP8",
    },
    "qwen-qwen3-30b-a3b-instruct-2507-fp8": {
        "org_label": "qwen",
        "name_label": "qwen3-30b-a3b-instruct-2507-fp8",
        "hf_path": "Qwen/Qwen3-30B-A3B-Instruct-2507-FP8",
    },
    "qwen-qwen3-30b-a3b-thinking-2507-fp8": {
        "org_label": "qwen",
        "name_label": "qwen3-30b-a3b-thinking-2507-fp8",
        "hf_path": "Qwen/Qwen3-30B-A3B-Thinking-2507-FP8",
    },
}


def parse_parallelism_suffix(
    suffix: str, gpus_per_node: int
) -> tuple[Optional[dict], int]:
    """Parse the parallelism suffix into (spec_dict, gpu_count_per_worker).

    Naming format (moreh-vllm specification §3.2):
      {parallelism}[-moe-{moe_parallelism}]

    {parallelism} examples: 1, tp4, dp8, dp4tp4, dp2pp2tp4
    {moe_parallelism}: tpN or epN (for MoE models only)

    Default gpu_count derivation rules:
      1                  – single device; no spec.parallelism, 1 GPU
      tpN                – N-way tensor parallel; N GPUs per worker
      dpN(-moe-...)      – N-way data parallel; moe_count GPUs per worker
      dpNtpM[-moe-...]   – N data-parallel + M-way tensor parallel;
                           gpus_per_node GPUs per worker, dataLocal = gpus_per_node // M

    When -moe-epN is present, expert parallelism is enabled (spec.expert = true).
    """
    # Split off MoE parallelism suffix.
    moe_strategy = None  # "ep" or "tp"
    moe_count = None
    if "-moe-" in suffix:
        base_part, moe_part = suffix.split("-moe-", 1)
        m = re.fullmatch(r"(ep|tp)(\d+)", moe_part)
        if not m:
            raise ValueError(f"Unrecognized MoE parallelism: {moe_part!r}")
        moe_strategy = m.group(1)
        moe_count = int(m.group(2))
    else:
        base_part = suffix

    if base_part == "1":
        return None, 1

    m = re.fullmatch(r"tp(\d+)", base_part)
    if m:
        n = int(m.group(1))
        spec: dict = {"tensor": n}
        if moe_strategy == "ep":
            spec["expert"] = True
        return spec, n

    m = re.fullmatch(r"dp(\d+)", base_part)
    if m:
        dp = int(m.group(1))
        spec = {"data": dp}
        if moe_strategy == "ep":
            spec["expert"] = True
        gpu_count = moe_count if moe_count else dp
        return spec, gpu_count

    m = re.fullmatch(r"dp(\d+)tp(\d+)", base_part)
    if m:
        dp, tp = int(m.group(1)), int(m.group(2))
        data_local = gpus_per_node // tp
        spec = {"tensor": tp, "data": dp, "dataLocal": data_local}
        if moe_strategy == "ep":
            spec["expert"] = True
        return spec, gpus_per_node

    raise ValueError(f"Unrecognized parallelism suffix: {suffix!r}")


def parse_preset_stem(stem: str) -> dict:
    """Parse a preset filename stem into its structural components.

    Naming convention (Moreh vLLM specification §3.2):
      {org}-{model}[-mtp][-prefill|-decode]-{accel_vendor}-{accel_model}-{parallelism}[-moe-{moe_parallelism}]
    """
    parts = stem.split("-")

    # The accelerator vendor token separates the model prefix from hardware info.
    vendor_idx = next(
        (i for i, p in enumerate(parts) if p in KNOWN_ACCEL_VENDORS), None
    )
    if vendor_idx is None:
        raise ValueError(
            f"No known accelerator vendor {KNOWN_ACCEL_VENDORS} found in: {stem!r}"
        )
    if vendor_idx + 2 >= len(parts):
        raise ValueError(f"Stem too short after accelerator vendor: {stem!r}")

    accel_vendor = parts[vendor_idx]
    accel_model = parts[vendor_idx + 1]
    parallelism_suffix = "-".join(parts[vendor_idx + 2 :])

    pre = parts[:vendor_idx]

    role = "e2e"
    if pre and pre[-1] in ("prefill", "decode"):
        role = pre[-1]
        pre = pre[:-1]

    mtp = False
    if pre and pre[-1] == "mtp":
        mtp = True
        pre = pre[:-1]

    if not pre:
        raise ValueError(f"Could not extract model key from stem: {stem!r}")

    return {
        "model_key": "-".join(pre),
        "mtp": mtp,
        "role": role,
        "accel_vendor": accel_vendor,
        "accel_model": accel_model,
        "parallelism_suffix": parallelism_suffix,
    }


def _filename_stem(parsed: dict) -> str:
    middle = []
    if parsed["mtp"]:
        middle.append("mtp")
    if parsed["role"] != "e2e":
        middle.append(parsed["role"])
    return "-".join(
        [parsed["model_key"]]
        + middle
        + [parsed["accel_vendor"], parsed["accel_model"], parsed["parallelism_suffix"]]
    )


def _generate_content(
    version: str,
    parsed: dict,
    model_info: dict,
    parallelism_spec: Optional[dict],
    gpu_count: int,
) -> tuple[str, str]:
    stem = _filename_stem(parsed)
    resource_name = f"moreh-vllm-{version}-{stem}"
    image = f"{ECR_IMAGE_BASE}:{version}"
    preset_path = f"{PRESET_PATH_BASE}/{stem}.yaml"

    org_label = model_info["org_label"]
    name_label = model_info["name_label"]
    hf_path = model_info["hf_path"]
    accel_vendor = parsed["accel_vendor"]
    accel_model = parsed["accel_model"]
    role = parsed["role"]
    mtp = parsed["mtp"]
    suffix = parsed["parallelism_suffix"]
    parallelism_label = suffix

    lines = []
    lines.append("apiVersion: odin.moreh.io/v1alpha1")
    lines.append("kind: InferenceServiceTemplate")
    lines.append("metadata:")
    lines.append(f"  name: {resource_name}")
    lines.append('  namespace: {{ include "common.names.namespace" . }}')
    lines.append("  labels:")
    lines.append('    {{- include "mif.preset.labels" . | nindent 4 }}')
    lines.append(f"    mif.moreh.io/model.org: {org_label}")
    lines.append(f"    mif.moreh.io/model.name: {name_label}")
    if mtp:
        lines.append('    mif.moreh.io/model.mtp: "true"')
    lines.append(f"    mif.moreh.io/role: {role}")
    lines.append(f"    mif.moreh.io/accelerator.vendor: {accel_vendor}")
    lines.append(f"    mif.moreh.io/accelerator.model: {accel_model}")
    lines.append(f'    mif.moreh.io/parallelism: "{parallelism_label}"')
    lines.append("spec:")
    lines.append("  framework: vllm")
    lines.append("  model:")
    lines.append(f"    name: {hf_path}")
    if parallelism_spec:
        lines.append("  parallelism:")
        for key, val in parallelism_spec.items():
            if isinstance(val, bool):
                lines.append(f"    {key}: {'true' if val else 'false'}")
            else:
                lines.append(f"    {key}: {val}")
    # dpN uses workerTemplate (LeaderWorkerSet); tpN and single-device use template.
    use_worker = parallelism_spec is not None and "data" in parallelism_spec
    template_key = "workerTemplate" if use_worker else "template"
    lines.append(f"  {template_key}:")
    lines.append("    spec:")
    lines.append("      containers:")
    lines.append("        - name: main")
    lines.append(f"          image: {image}")
    lines.append("          env:")
    lines.append("            - name: ISVC_PRESET_PATH")
    lines.append(f"              value: {preset_path}")
    lines.append("          resources:")
    lines.append("            requests:")
    lines.append(f'              amd.com/gpu: "{gpu_count}"')
    lines.append("            limits:")
    lines.append(f'              amd.com/gpu: "{gpu_count}"')
    lines.append("      nodeSelector:")
    lines.append(f"        moai.moreh.io/accelerator.vendor: {accel_vendor}")
    lines.append(f"        moai.moreh.io/accelerator.model: {accel_model}")
    lines.append("      tolerations:")
    lines.append("        - key: amd.com/gpu")
    lines.append("          operator: Exists")
    lines.append("          effect: NoSchedule")

    return stem, "\n".join(lines) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser(
        description=(
            "Generate moreh-vllm InferenceServiceTemplate Helm preset files "
            "from the preset list in the moreh-vllm container image."
        ),
        epilog=(
            "Example:\n"
            "  python hack/gen_moreh_vllm_presets.py \\\n"
            "    --version 0.15.0-260210-rc1 \\\n"
            "    --output-dir deploy/helm/.../0.15.0-260210-rc1 \\\n"
            "    $(docker run --rm <image> ls /app/moreh-vllm/presets/)"
        ),
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument(
        "--version",
        required=True,
        help="moreh-vllm image version tag (e.g. 0.15.0-260210-rc1)",
    )
    parser.add_argument(
        "--output-dir",
        required=True,
        help="Directory to write the generated .helm.yaml files into",
    )
    parser.add_argument(
        "--gpus-per-node",
        type=int,
        default=8,
        metavar="N",
        help="GPUs per node, used to compute dataLocal for dpNtpM presets (default: 8)",
    )
    parser.add_argument(
        "presets",
        nargs="+",
        metavar="PRESET",
        help=(
            "Preset filename stems (with or without the .yaml extension). "
            "Obtain via: docker run --rm <image> ls /app/moreh-vllm/presets/"
        ),
    )
    args = parser.parse_args()

    entries = []
    stems_seen: set[str] = set()

    for raw in args.presets:
        # Skip ls symlink arrow tokens (e.g. '⇒' and the target filename that
        # some ls implementations emit when listing symlinks inline).
        if not re.search(r"\.yaml@?$", raw):
            continue

        # Strip symlink indicator (@) appended by some ls implementations.
        stem = re.sub(r"(\.helm)?\.yaml@?$", "", os.path.basename(raw))

        try:
            parsed = parse_preset_stem(stem)
        except ValueError as exc:
            print(f"Error parsing {stem!r}: {exc}", file=sys.stderr)
            sys.exit(1)

        model_key = parsed["model_key"]
        model_info = MODEL_REGISTRY.get(model_key)
        if model_info is None:
            print(
                f"Error: model key {model_key!r} (from {stem!r}) is not in "
                f"MODEL_REGISTRY.\n"
                f"  Add an entry with org_label, name_label, and hf_path.",
                file=sys.stderr,
            )
            sys.exit(1)

        canonical = _filename_stem(parsed)
        if canonical in stems_seen:
            # The same canonical stem can appear more than once when ls emits
            # symlink targets inline alongside the symlink names.  Skip silently.
            continue
        stems_seen.add(canonical)

        try:
            parallelism_spec, gpu_count = parse_parallelism_suffix(
                parsed["parallelism_suffix"], args.gpus_per_node
            )
        except ValueError as exc:
            print(f"Error in {stem!r}: {exc}", file=sys.stderr)
            sys.exit(1)

        gpu_count = model_info.get("gpu_overrides", {}).get(
            parsed["parallelism_suffix"], gpu_count
        )

        entries.append((parsed, model_info, parallelism_spec, gpu_count))

    if os.path.exists(args.output_dir):
        shutil.rmtree(args.output_dir)
    os.makedirs(args.output_dir)

    for parsed, model_info, parallelism_spec, gpu_count in entries:
        stem, content = _generate_content(
            args.version, parsed, model_info, parallelism_spec, gpu_count
        )
        output_path = os.path.join(args.output_dir, f"{stem}.helm.yaml")
        with open(output_path, "w") as f:
            f.write(content)
        print(f"  {output_path}")

    print(f"\nGenerated {len(entries)} files in {args.output_dir}")


if __name__ == "__main__":
    main()
