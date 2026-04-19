#!/usr/bin/env python3

import sys
from pathlib import Path

import yaml


def main() -> int:
    if len(sys.argv) != 3:
        print("usage: sync_bundle_metadata.py <base_csv> <bundle_csv>", file=sys.stderr)
        return 1

    base_path = Path(sys.argv[1])
    bundle_path = Path(sys.argv[2])

    with base_path.open("r", encoding="utf-8") as f:
        base = yaml.safe_load(f)
    with bundle_path.open("r", encoding="utf-8") as f:
        bundle = yaml.safe_load(f)

    base_annotations = ((base or {}).get("metadata") or {}).get("annotations") or {}
    bundle_metadata = (bundle or {}).setdefault("metadata", {})
    bundle_annotations = bundle_metadata.setdefault("annotations", {})

    for key in ("alm-examples", "capabilities"):
        if key in base_annotations:
            bundle_annotations[key] = base_annotations[key]

    base_owned = (
        ((base or {}).get("spec") or {}).get("customresourcedefinitions") or {}
    ).get("owned") or []
    bundle_spec = (bundle or {}).setdefault("spec", {})
    bundle_crds = bundle_spec.setdefault("customresourcedefinitions", {})
    bundle_owned = bundle_crds.setdefault("owned", [])

    base_owned_by_name = {
        item.get("name"): item for item in base_owned if item.get("name")
    }
    for item in bundle_owned:
        name = item.get("name")
        if not name or name not in base_owned_by_name:
            continue
        base_item = base_owned_by_name[name]
        for key in (
            "displayName",
            "description",
            "resources",
            "specDescriptors",
            "statusDescriptors",
        ):
            if key in base_item:
                item[key] = base_item[key]

    with bundle_path.open("w", encoding="utf-8") as f:
        yaml.safe_dump(bundle, f, sort_keys=False)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
