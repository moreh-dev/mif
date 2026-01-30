python3 -c '
import os, yaml
with open(os.environ["ISVC_PRESET_PATH"]) as f:
    p = yaml.safe_load(f) or {}
if p.get("engine_args"):
    with open("preset_engine_args.yaml", "w") as f: yaml.dump(p["engine_args"], f)
    print("export _config_arg=\"--config preset_engine_args.yaml\"")
for k, v in p.get("env_vars", {}).items():
    if k not in os.environ: print(f"export {k}=\"{v}\"")'
