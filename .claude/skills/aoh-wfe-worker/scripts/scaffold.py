#!/usr/bin/env python3
"""
Scaffold a WFE Temporal activity worker.

Generates a self-contained, standalone activity worker (aoh-golib as a normal
registry dependency — no local `replace`) with a single HelloWorld starter
activity, so it builds, lints, tests, and seeds the WFE designer catalog out of
the box.

Usage:
    python3 scaffold.py \
        --name <name> \
        --module <go-module-path> \
        --repo-root <path-to-repo-root> \
        [--target-dir <relative-path>] \
        [--description <one-line description>]

By default the worker lands at `apps/<name>/`. Pass `--target-dir` for another
relative path under the repo root (e.g. `modules/<x>/worker`). The flag changes
both the on-disk location and the `go.work` `use (...)` entry written (only when a
go.work already exists — the scaffold never forces a workspace on a standalone repo).

Placeholder convention in assets/: `__AOH_MODULE__`, `__AOH_NAME__`,
`__AOH_TITLE__` — safe markers that never appear in Go, YAML, or Dockerfile syntax.
Files ending in `.tmpl` have placeholders substituted and the suffix stripped.
"""

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
from pathlib import Path


def gofmt_tree(service_dir: Path) -> None:
    """gofmt -w the rendered Go. The templates can't pre-sort imports against an
    unknown module path, so a path like github.com/acme/demo would mis-sort and
    fail `make lint-code` (gci/goimports) on a fresh scaffold. gofmt sorts them
    for the actual module path, keeping the worker gofmt/lint-clean out of the box."""
    try:
        subprocess.run(["gofmt", "-w", str(service_dir)], check=True)
        print("  gofmt -w (rendered Go)")
    except FileNotFoundError:
        print("  SKIP gofmt (not on PATH) — run `make fmt` after scaffolding")
    except subprocess.CalledProcessError as e:
        print(f"  WARNING: gofmt failed ({e}) — run `make fmt` after scaffolding")


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def copy_asset(asset_dir: Path, asset_name: str, dest: Path, replacements: dict):
    src = asset_dir / asset_name
    if not src.exists():
        print(f"  SKIP {asset_name} (not found)")
        return
    content = src.read_text()
    for placeholder, value in replacements.items():
        content = content.replace(placeholder, value)
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_text(content)
    print(f"  COPY {dest.name}")


def render_tree(src_root: Path, dst_root: Path, replacements: dict):
    """Walk src_root, substitute placeholders, write to dst_root.

    Files ending in `.tmpl` have placeholders replaced and the `.tmpl` suffix
    stripped on write. Files without `.tmpl` are copied verbatim.
    """
    for src in sorted(src_root.rglob("*")):
        if not src.is_file():
            continue
        rel = src.relative_to(src_root)
        is_template = rel.suffix == ".tmpl"
        rel_out = rel.with_suffix("") if is_template else rel
        dst = dst_root / rel_out
        dst.parent.mkdir(parents=True, exist_ok=True)
        if is_template:
            content = src.read_text()
            for placeholder, value in replacements.items():
                content = content.replace(placeholder, value)
            dst.write_text(content)
        else:
            shutil.copyfile(src, dst)
        print(f"  COPY {rel_out}")


# ---------------------------------------------------------------------------
# Asset files
# ---------------------------------------------------------------------------

def copy_assets(asset_dir: Path, service_dir: Path, replacements: dict):
    # Flat infra files (rendered to the worker root).
    copy_asset(asset_dir, ".gitignore", service_dir / ".gitignore", {})
    copy_asset(asset_dir, ".dockerignore", service_dir / ".dockerignore", {})
    copy_asset(asset_dir, ".golangci.yml", service_dir / ".golangci.yml", {})
    copy_asset(asset_dir, "Makefile", service_dir / "Makefile", {})
    copy_asset(asset_dir, "worker.config.yaml", service_dir / "worker.config.yaml", {})
    copy_asset(asset_dir, "go.mod.tmpl", service_dir / "go.mod", replacements)
    copy_asset(asset_dir, "worker.Dockerfile.tmpl", service_dir / "worker.Dockerfile", replacements)
    copy_asset(asset_dir, "AGENTS.md.tmpl", service_dir / "AGENTS.md", replacements)
    copy_asset(asset_dir, "README.md.tmpl", service_dir / "README.md", replacements)


# ---------------------------------------------------------------------------
# Workspace setup
# ---------------------------------------------------------------------------

def setup_package_json(service_dir: Path, service_name: str):
    pkg_path = service_dir / "package.json"
    if pkg_path.exists():
        print("  SKIP package.json (already exists)")
        return
    pkg = {
        "name": service_name,
        "version": "0.0.0",
        "private": True,
        "scripts": {
            "build": "go build -o bin/worker ./cmd/worker",
            "dev": "go run ./cmd/worker --log.level=debug",
            "test": "make test",
            "lint": "make lint-code",
        },
    }
    pkg_path.write_text(json.dumps(pkg, indent=2) + "\n")
    print("  COPY package.json")


def _is_path_in_use_directive(content: str, path: str) -> bool:
    """True iff `path` appears in a real `use` directive in a go.work file.

    Strips // line comments first so a path mentioned only in a comment isn't
    treated as already listed. Matches both `use ./path` and `use ( ... )` forms.
    """
    no_comments = re.sub(r"//[^\n]*", "", content)
    if re.search(rf"(?m)^\s*use\s+\./{re.escape(path)}\s*$", no_comments):
        return True
    for block in re.finditer(r"use\s*\(([^)]*)\)", no_comments, re.DOTALL):
        for line in block.group(1).splitlines():
            if line.strip() == f"./{path}":
                return True
    return False


def setup_go_work(repo_root: Path, service_rel_path: str):
    """Add the worker to an EXISTING go.work. Never create one — a standalone
    worker doesn't need a workspace, and forcing one would surprise consumers."""
    go_work = repo_root / "go.work"
    if not go_work.exists():
        print("  SKIP go.work (none at repo root — standalone module)")
        return
    content = go_work.read_text()
    if _is_path_in_use_directive(content, service_rel_path):
        print("  SKIP go.work (already listed)")
        return
    use_line = f"\t./{service_rel_path}"
    if re.search(r"use\s*\([^)]*\n\s*\)", content, re.DOTALL):
        content = re.sub(r"(use\s*\([^)]*)(\n\s*\))",
                         rf"\1\n{use_line}\2", content, count=1, flags=re.DOTALL)
    else:
        if not content.endswith("\n"):
            content += "\n"
        content += f"\nuse (\n{use_line}\n)\n"
    go_work.write_text(content)
    print("  UPDATE go.work")


def setup_turbo_json(repo_root: Path, service_name: str):
    turbo_path = repo_root / "turbo.json"
    if not turbo_path.exists():
        print("  SKIP turbo.json (not found)")
        return
    turbo = json.loads(turbo_path.read_text())
    override_key = f"{service_name}#build"
    if override_key in turbo.get("tasks", {}):
        print(f"  SKIP turbo.json ({override_key} exists)")
        return
    turbo.setdefault("tasks", {})[override_key] = {
        "inputs": ["**/*.go", "go.mod", "go.sum", "config.yaml*"],
        "outputs": ["bin/**"],
    }
    turbo_path.write_text(json.dumps(turbo, indent=2) + "\n")
    print("  UPDATE turbo.json")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(description="Scaffold a WFE Temporal activity worker")
    parser.add_argument("--name", required=True, help="Worker name (e.g. incident-worker)")
    parser.add_argument("--module", required=True, help="Go module path (e.g. github.com/acme/incident-worker)")
    parser.add_argument("--repo-root", required=True, help="Repo root directory")
    parser.add_argument(
        "--target-dir",
        default=None,
        help="Relative path under the repo root for the worker (default: apps/<name>).",
    )
    parser.add_argument("--description", default="A WFE activity worker", help="Worker description")
    args = parser.parse_args()

    repo_root = Path(args.repo_root).resolve()
    service_rel_path = args.target_dir if args.target_dir else f"apps/{args.name}"
    normalized = Path(service_rel_path)
    if normalized.is_absolute() or ".." in normalized.parts:
        print(f"ERROR: --target-dir must be a relative path under repo-root: {service_rel_path}")
        sys.exit(1)
    service_dir = repo_root / service_rel_path
    script_dir = Path(__file__).parent.parent
    asset_dir = script_dir / "assets"

    if not repo_root.exists():
        print(f"ERROR: Repo root not found: {repo_root}")
        sys.exit(1)
    if not asset_dir.exists():
        print(f"ERROR: Assets directory not found: {asset_dir}")
        sys.exit(1)

    replacements = {
        "__AOH_MODULE__": args.module,
        "__AOH_NAME__": args.name,
        "__AOH_TITLE__": args.name.replace("-", " ").title(),
        "__AOH_DESC__": args.description,
    }

    print(f"\nScaffolding WFE activity worker: {args.name}")
    print(f"  Module: {args.module}")
    print(f"  Target: {service_dir}")
    print()

    service_dir.mkdir(parents=True, exist_ok=True)

    print("[1/6] Package setup")
    setup_package_json(service_dir, args.name)

    print("\n[2/6] Asset files")
    copy_assets(asset_dir, service_dir, replacements)

    print("\n[3/6] Worker source tree")
    render_tree(asset_dir / "src", service_dir, replacements)
    go_files = list(service_dir.rglob("*.go"))
    print(f"  Generated {len(go_files)} Go files total")

    print("\n[4/6] Format Go")
    gofmt_tree(service_dir)

    print("\n[5/6] Go workspace")
    setup_go_work(repo_root, service_rel_path)

    print("\n[6/6] Turbo config")
    setup_turbo_json(repo_root, args.name)

    print(f"\n{'=' * 50}")
    print(f"Scaffold complete: {service_rel_path}/")
    print(f"{'=' * 50}")
    print("\nNext steps:")
    print(f"  cd {service_rel_path}")
    print("  go mod tidy                       # resolve deps + write go.sum")
    print("  cp worker.config.yaml config.yaml # set temporal.task_queue to the engine's")
    print("  make test                         # run the activity tests")
    print("  make run                          # build + run with debug logging")
    print("\nRegister the activities + events in the WFE designer (against the MANAGER's Postgres):")
    print('  psql "$WFE_MANAGER_DSN" -f sql/seed_catalog.sql')


if __name__ == "__main__":
    main()
