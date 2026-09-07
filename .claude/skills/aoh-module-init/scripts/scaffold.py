#!/usr/bin/env python3
"""
Scaffold an AOH module inside the ops-hub platform-monorepo.

An "AOH module" is a vertically-sliced feature in ops-hub:
modules/<name>/{web,client,types,service,deploy}/. This script creates
the sibling packages (web, client, types, deploy) plus a module-level
README + AGENTS.md (with a one-line `@AGENTS.md` CLAUDE.md import), then
delegates the Go service scaffold to
aoh-go-init's existing scaffold.py via --target-dir.

Usage:
    python3 scaffold.py \\
        --name <module-name> \\
        --description "<one-liner>" \\
        --repo-root <path-to-ops-hub> \\
        [--go-module-path <path>] \\
        [--skip-service]

Preconditions:
    * <repo-root> must look like an ops-hub repo:
      - has a modules/ directory
      - has a pnpm-workspace.yaml with modules/*/* globs
      - has a go.work
    * modules/<name>/ must NOT already exist.

Refer to the aoh-module-init SKILL.md for the higher-level procedure.
"""

import argparse
import os
import re
import shutil
import subprocess
import sys
from pathlib import Path


# Valid module names: lowercase, start with a letter, kebab-case allowed.
# Matches npm package naming + Go module identifier rules + kebab→Pascal
# conversion safety. Rejects: leading hyphen, trailing hyphen, double
# hyphens, leading digit, uppercase, underscore, Unicode.
_NAME_REGEX = re.compile(r"^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$")


# ---------------------------------------------------------------------------
# Placeholders
# ---------------------------------------------------------------------------
#
# Same convention as aoh-go-init: __AOH_*__ markers that never appear in
# real Go, TS, YAML, or Dockerfile syntax.

PH_MODULE_NAME = "__AOH_MODULE_NAME__"
PH_MODULE_DESCRIPTION = "__AOH_MODULE_DESCRIPTION__"
PH_PASCAL_NAME = "__AOH_PASCAL_NAME__"


# ---------------------------------------------------------------------------
# Case helpers
# ---------------------------------------------------------------------------

def kebab_to_pascal(name: str) -> str:
    """`my-module` -> `MyModule`. Used for class / type names."""
    return "".join(part.capitalize() for part in name.split("-") if part)


# ---------------------------------------------------------------------------
# Preflight
# ---------------------------------------------------------------------------

def preflight(repo_root: Path, module_name: str) -> Path:
    """Validate ops-hub shape + that the module doesn't already exist.

    Returns the modules/<name>/ destination path on success. Exits with
    a clear message on any failure — we'd rather fail loudly than
    corrupt an unrelated repo.
    """
    if not repo_root.is_dir():
        sys.exit(f"ERROR: --repo-root {repo_root} is not a directory")

    modules_dir = repo_root / "modules"
    if not modules_dir.is_dir():
        sys.exit(
            f"ERROR: {repo_root} does not look like an ops-hub repo "
            "(no modules/ directory). aoh-module-init only supports the "
            "ops-hub platform-monorepo shape; for the consumer-app "
            "shape use aoh-go-init directly."
        )

    pnpm_ws = repo_root / "pnpm-workspace.yaml"
    if not pnpm_ws.is_file():
        sys.exit(f"ERROR: {pnpm_ws} not found")
    pnpm_ws_text = pnpm_ws.read_text()
    # Check for each sub-package glob individually. A workspace that
    # globs only `modules/*` (no sub-package suffix) would pass a loose
    # substring check but pnpm wouldn't actually discover any of the
    # four sub-packages — leaving the user with a "scaffold ran fine
    # but pnpm install ignored it" silent failure.
    missing_globs = [
        glob for glob in ("modules/*/web", "modules/*/client",
                          "modules/*/types", "modules/*/service")
        if glob not in pnpm_ws_text
    ]
    if missing_globs:
        sys.exit(
            f"ERROR: {pnpm_ws} is missing required globs: "
            f"{', '.join(missing_globs)}.\n"
            "Refusing to scaffold a module that pnpm won't pick up. "
            "Add the missing glob(s) under 'packages:' then re-run."
        )

    if not (repo_root / "go.work").is_file():
        sys.exit(
            f"ERROR: {repo_root}/go.work not found. ops-hub should have "
            "one at the root."
        )

    dest = modules_dir / module_name
    if dest.exists():
        sys.exit(
            f"ERROR: {dest} already exists. Refusing to overwrite. If "
            "this is a re-scaffold, remove the directory first; "
            "otherwise pick a different --name."
        )

    return dest


# ---------------------------------------------------------------------------
# Template copy
# ---------------------------------------------------------------------------

def render_template_tree(
    src_root: Path,
    dst_root: Path,
    replacements: dict,
):
    """Walk src_root, substitute placeholders, write to dst_root.

    Files ending in `.tmpl` have placeholders substituted and the
    `.tmpl` suffix stripped on write. Non-`.tmpl` files are copied
    verbatim — those are config files (tsconfig.json, vitest.config.ts,
    etc.) that don't need any substitution.
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
# Service delegation
# ---------------------------------------------------------------------------

def find_aoh_go_init_script() -> Path:
    """Find aoh-go-init's scaffold.py. It lives as a sibling skill.

    Layout (when invoked from .claude/skills/ or skills/):
        aoh-module-init/scripts/scaffold.py    <-- THIS FILE
        aoh-go-init/scripts/scaffold.py        <-- target
    """
    here = Path(__file__).resolve()
    candidate = here.parent.parent.parent / "aoh-go-init" / "scripts" / "scaffold.py"
    if not candidate.is_file():
        sys.exit(
            f"ERROR: could not find sibling aoh-go-init scaffold.py at "
            f"{candidate}. aoh-module-init requires aoh-go-init to be "
            "installed alongside it."
        )
    return candidate


def scaffold_service(
    repo_root: Path,
    module_name: str,
    description: str,
    go_module_path: str,
):
    """Delegate Go scaffold to aoh-go-init via --target-dir."""
    go_init_script = find_aoh_go_init_script()
    target_dir = f"modules/{module_name}/service"
    cmd = [
        sys.executable,
        str(go_init_script),
        "--name", module_name,
        "--module", go_module_path,
        "--repo-root", str(repo_root),
        "--target-dir", target_dir,
        "--description", description,
    ]
    print(f"\n[2/2] Delegating service scaffold to aoh-go-init")
    print(f"  $ {' '.join(cmd)}")
    result = subprocess.run(cmd)
    if result.returncode != 0:
        sys.exit(
            f"ERROR: aoh-go-init exited with status {result.returncode}. "
            "The sibling packages (web/client/types/deploy) were already "
            "scaffolded; you can re-run aoh-go-init manually or remove "
            f"modules/{module_name}/ and start over."
        )


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Scaffold an AOH module under ops-hub/modules/<name>/.",
    )
    parser.add_argument("--name", required=True,
                        help="Module name (kebab-case). E.g. 'inventory'.")
    parser.add_argument("--description", required=True,
                        help="One-line module description (used in READMEs).")
    parser.add_argument("--repo-root", required=True, type=Path,
                        help="Absolute path to the ops-hub repo root.")
    parser.add_argument("--go-module-path",
                        help="Go module path for the service. "
                             "Defaults to github.com/mssfoobar/ops-hub/modules/<name>/service.")
    parser.add_argument("--skip-service", action="store_true",
                        help="Skip the Go service scaffold (sibling packages only).")
    args = parser.parse_args()

    # ---- Validate inputs ----

    if not _NAME_REGEX.fullmatch(args.name):
        sys.exit(
            f"ERROR: --name {args.name!r} must be lowercase kebab-case starting "
            "with a letter (e.g. 'inventory' or 'asset-tracking'). "
            "Rejects leading/trailing/double hyphens, leading digits, uppercase, "
            "underscores, and Unicode — those break npm package names, Go module "
            "identifiers, or kebab→Pascal conversion downstream."
        )

    repo_root: Path = args.repo_root.resolve()
    dest = preflight(repo_root, args.name)

    go_module_path = (
        args.go_module_path
        or f"github.com/mssfoobar/ops-hub/modules/{args.name}/service"
    )

    # ---- Scaffold sibling packages from assets/ ----

    assets_root = Path(__file__).resolve().parent.parent / "assets"
    if not assets_root.is_dir():
        sys.exit(f"ERROR: assets/ not found at {assets_root}")

    replacements = {
        PH_MODULE_NAME: args.name,
        PH_MODULE_DESCRIPTION: args.description,
        PH_PASCAL_NAME: kebab_to_pascal(args.name),
    }

    print(f"[1/2] Scaffolding modules/{args.name}/ from templates")
    dest.mkdir(parents=True, exist_ok=False)
    render_template_tree(assets_root, dest, replacements)

    # ---- Delegate to aoh-go-init for service/ ----

    if args.skip_service:
        print("\n[2/2] --skip-service set — leaving service/ unscaffolded.")
    else:
        scaffold_service(
            repo_root=repo_root,
            module_name=args.name,
            description=args.description,
            go_module_path=go_module_path,
        )

    # ---- Summary ----

    print(f"\n==============================================================")
    print(f"Module scaffolded: modules/{args.name}/")
    print(f"==============================================================")
    print()
    print(f"Packages created:")
    print(f"  modules/{args.name}/types/         (@mssfoobar/{args.name}-types)")
    print(f"  modules/{args.name}/client/        (@mssfoobar/{args.name}-client)")
    print(f"  modules/{args.name}/web/           (@mssfoobar/{args.name}-web-sdk)")
    print(f"  modules/{args.name}/deploy/        (compose snippet)")
    if not args.skip_service:
        print(f"  modules/{args.name}/service/       (Go service via aoh-go-init)")
    print()
    print(f"Next steps:")
    steps = [
        f"cd {repo_root}",
        f"pnpm install",
        f"pnpm exec turbo run check --filter=./modules/{args.name}/*",
    ]
    if not args.skip_service:
        steps.append(
            f"cd modules/{args.name}/service && make mock && make build && go test ./..."
        )
    steps.append(
        f"Edit modules/{args.name}/README.md and AGENTS.md to add module-specific context"
    )

    fill_in_bullets = [
        f"TS mirrors in modules/{args.name}/types/src/index.ts",
        f"Client resource APIs in modules/{args.name}/client/src/index.ts",
        f"SDK components in modules/{args.name}/web/sdk/",
    ]
    if not args.skip_service:
        # Service-related fill-in only when the service was actually scaffolded.
        fill_in_bullets.insert(
            0, f"Wire DTOs in modules/{args.name}/service/internal/dto/*.go"
        )
    steps.append("Begin filling in:")

    for i, step in enumerate(steps, 1):
        print(f"  {i}. {step}")
    for bullet in fill_in_bullets:
        print(f"     - {bullet}")

    if args.skip_service:
        print()
        print(
            "Note: --skip-service was set. modules/{name}/deploy/compose.snippet.yml "
            "still references ../service for its build context — `docker compose up` "
            "will fail until the service is scaffolded (run aoh-go-init with "
            "--target-dir modules/{name}/service) or you remove the build: block.".format(
                name=args.name
            )
        )


if __name__ == "__main__":
    main()
