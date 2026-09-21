#!/usr/bin/env python3
"""Generate the Tauri updater manifest (latest.json) from release .sig assets.

Usage: gen_latest_json.py <tag> <sig_dir> <output.json>

Scans <sig_dir> for `*.sig` files downloaded from the GitHub release, maps each
to its updater platform, and emits the manifest the tauri-plugin-updater
expects. Asset URLs point at the release's renamed `cloudreve-desktop-*`
assets. The version is read from desktop/src-tauri/tauri.conf.json — desktop
has its own semver, independent of the repo's server release tags.
"""
import json
import os
import sys

PLATFORM_MAP = [
    ("_x64-setup.exe", "windows-x86_64"),
    ("_arm64-setup.exe", "windows-aarch64"),
    ("_aarch64.app.tar.gz", "darwin-aarch64"),
    ("_x64.app.tar.gz", "darwin-x86_64"),
    ("amd64.AppImage.tar.gz", "linux-x86_64"),
    ("aarch64.AppImage.tar.gz", "linux-aarch64"),
]

REPO = "Dvorinka/cloudreve"


def main() -> int:
    tag, sig_dir, out_path = sys.argv[1], sys.argv[2], sys.argv[3]

    conf = json.load(open("desktop/src-tauri/tauri.conf.json"))
    version = conf["version"]

    platforms = {}
    for fname in sorted(os.listdir(sig_dir)):
        if not fname.endswith(".sig"):
            continue
        # Release assets are renamed to `cloudreve-desktop-<basename>` at
        # upload time, so the downloaded .sig name already is the asset name.
        asset = fname[: -len(".sig")]
        for marker, platform in PLATFORM_MAP:
            if marker in asset:
                signature = open(os.path.join(sig_dir, fname)).read().strip()
                if not signature:
                    print(f"warning: empty signature in {fname}, skipping")
                    break
                platforms[platform] = {
                    "signature": signature,
                    "url": (
                        f"https://github.com/{REPO}/releases/download/{tag}/"
                        f"{asset}"
                    ),
                }
                break

    if not platforms:
        print("error: no updater artifacts found", file=sys.stderr)
        return 1

    manifest = {
        "version": version,
        "notes": f"Cloudreve Desktop {version}",
        "pub_date": "",
        "platforms": platforms,
    }
    with open(out_path, "w") as f:
        json.dump(manifest, f, indent=2)
    print(f"latest.json: version={version}, platforms={sorted(platforms)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
