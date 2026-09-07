#!/usr/bin/env python3
"""
Bootstrap and extend a Docker Compose stack for AOH services.

The conversational SKILL gathers intent, then shells out here with structured
args.

What this script does:
  1. Bootstraps compose/ with Traefik + IAMS + SDS + shared .env + gitignore
     + realm-import.json + iams-init postman collection + override sample.
  2. Auto-detects runtime: if Docker is present and Podman is not, writes
     compose.override.yml to remount the Docker socket on Traefik. Otherwise
     leaves the Podman default in place.
  3. For each requested AOH service, copies its asset directory to
     compose/<svc>/ and resolves transitive dependencies from a built-in
     catalogue.
  4. For each custom service (name + type=go|web), copies the corresponding
     template to compose/<name>/compose.yml with <name> placeholders replaced.
  5. Regenerates the top-level compose/compose.yml with the union of includes.

What it does NOT do (by design):
  - Add project-specific Keycloak roles, clients, or seed users. See
    aoh-knowledge/references/keycloak-realm-guide.md for the manual
    extension procedure against the bundled aoh realm.
  - Fill in per-custom-service env vars. The template ships with commented
    samples; the developer uncomments what they need.
  - Run `docker compose up`. Developer inspects before starting.

Idempotent: re-running with the same args is a no-op. Re-running with a
larger --services list adds the missing pieces without clobbering existing
files.

Usage:
    python3 bootstrap.py \\
        --repo-root <path-to-repo> \\
        [--services iams,rtus,gis] \\
        [--custom-service name=my-svc,type=go] \\
        [--custom-service name=my-web,type=web]
"""

import argparse
import json
import os
import shutil
import subprocess
import sys
from pathlib import Path

# ---------------------------------------------------------------------------
# AOH service catalogue - mirrors references/service-catalogue.md
# ---------------------------------------------------------------------------

# Each service has:
#   asset_dir:    subdirectory under assets/ that is copied into compose/
#   deps:         list of other service ids that must also be included
#
# Core (always included via bootstrap): traefik, iams, sds, otel
# (otel = the gateway OpenTelemetry collector, the OTLP endpoint every service
#  targets; default-on. The `signoz` backend stays opt-in.)
# Supporting and auxiliary services are opt-in via --services.
SERVICES = {
    "rtus":  {"asset_dir": "rtus",  "deps": ["iams"]},
    "gis":   {"asset_dir": "gis",   "deps": ["iams", "rtus"]},
    "unh":   {"asset_dir": "unh",   "deps": ["iams"]},
    "ian":   {"asset_dir": "ian",   "deps": ["iams", "unh", "rtus"]},
    "dash":  {"asset_dir": "dash",  "deps": ["iams"]},
    "ptmgr": {"asset_dir": "ptmgr", "deps": ["iams"]},
    "amm":   {"asset_dir": "amm",   "deps": ["iams"]},
    "form":  {"asset_dir": "form",  "deps": ["iams", "amm"]},
    # wfe's human-in-the-loop Form-task steps delegate to the `form` module — add
    # `form` to your service list when your workflows use them. Not a hard dep:
    # the WFE backend has no runtime Form coupling, so plain WFE doesn't need it.
    "wfe":   {"asset_dir": "wfe",   "deps": ["iams"]},
    # The default bundled observability backend. The gateway OTEL
    # collector (`otel`) is now CORE — always included — so this only adds the
    # heavy backend (ClickHouse + ZooKeeper + UI + collector + migrator). UI at
    # signoz.${DEV_DOMAIN} via Traefik. Its `otel` dep is satisfied by core.
    "signoz": {"asset_dir": "signoz", "deps": ["otel"]},
}

CUSTOM_TEMPLATES = {
    "go":  "custom-service-template.yml",
    "web": "web-service-template.yml",
}

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def write_file(path: Path, content: str):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content)


def copy_file(src: Path, dest: Path, label: str = None):
    if dest.exists():
        print(f"  SKIP {label or dest.name} (already exists)")
        return False
    dest.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(src, dest)
    print(f"  COPY {label or dest.name}")
    return True


def copy_tree(src: Path, dest: Path, label: str):
    if dest.exists():
        print(f"  SKIP {label} (already exists)")
        return False
    shutil.copytree(src, dest)
    print(f"  COPY {label}")
    return True


def copy_template(src: Path, dest: Path, replacements: dict, label: str):
    if dest.exists():
        print(f"  SKIP {label} (already exists)")
        return False
    content = src.read_text()
    for placeholder, value in replacements.items():
        content = content.replace(placeholder, value)
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_text(content)
    print(f"  COPY {label}")
    return True


# ---------------------------------------------------------------------------
# Dependency resolution
# ---------------------------------------------------------------------------

def resolve_deps(requested: list) -> list:
    """Expand requested services with transitive deps; preserve deterministic order."""
    seen = []

    def visit(svc):
        if svc in seen:
            return
        if svc not in SERVICES:
            print(f"ERROR: Unknown service '{svc}'. Known: {sorted(SERVICES.keys())}")
            sys.exit(1)
        for dep in SERVICES[svc]["deps"]:
            # Core services (iams, sds, traefik, otel) are handled by bootstrap; skip.
            if dep in ("iams", "sds", "traefik", "otel"):
                continue
            visit(dep)
        seen.append(svc)

    for svc in requested:
        visit(svc)
    return seen


# ---------------------------------------------------------------------------
# Runtime detection
# ---------------------------------------------------------------------------

def has_runtime(cmd: str) -> bool:
    try:
        result = subprocess.run(
            [cmd, "info"], capture_output=True, timeout=5
        )
        return result.returncode == 0
    except (FileNotFoundError, subprocess.TimeoutExpired):
        return False


def write_docker_override_if_needed(compose_dir: Path) -> str:
    """If the host has Docker but not Podman, generate compose.override.yml so
    Traefik can reach the Docker socket. All other cases (Podman only, both,
    or neither) leave the Podman default in place.

    Returns a one-line verdict for the final summary — the choice is made at
    scaffold time from what's installed, which can differ from the runtime
    actually used at `up` time."""
    has_docker = has_runtime("docker")
    has_podman = has_runtime("podman")

    if has_docker and not has_podman:
        override = compose_dir / "compose.override.yml"
        if override.exists():
            print("  SKIP compose.override.yml (already exists)")
            return "Docker (compose.override.yml already present)"
        override.write_text(
            "name: aoh\n"
            "\n"
            "services:\n"
            "    traefik:\n"
            "        volumes:\n"
            "            - /var/run/docker.sock:/var/run/docker.sock\n"
        )
        print("  CREATE compose.override.yml (Docker socket)")
        return "Docker (compose.override.yml created)"

    if has_podman:
        print("  SKIP compose.override.yml (Podman default)")
        verdict = "Podman"
    elif has_docker:
        print("  SKIP compose.override.yml (both runtimes - Podman default)")
        verdict = "Podman (Docker also present)"
    else:
        print("  SKIP compose.override.yml (no runtime detected)")
        verdict = "none detected (Podman default assumed)"
    return (
        verdict
        + " - if you bring the stack up with DOCKER instead, copy"
        " compose.override.sample.yml to compose.override.yml first"
        " (without it Traefik has no socket and every route 404s)"
    )


# ---------------------------------------------------------------------------
# Bootstrap
# ---------------------------------------------------------------------------

def bootstrap(compose_dir: Path, asset_dir: Path, repo_root: Path):
    """Copy the always-required files: Traefik, IAMS (+ realm + init), SDS,
    shared .env, .gitignore, override sample, and a repo-root .dockerignore."""
    compose_dir.mkdir(parents=True, exist_ok=True)

    # Repo-root .dockerignore - keeps `COPY . .` in the web Dockerfile (build
    # context = repo root) from shipping host node_modules / .env / .git / etc.
    # into the linux build, which otherwise triggers pnpm's
    # ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY. Go services build with
    # per-app context so they don't strictly need it, but the file is also
    # useful hygiene there (smaller context, no .git, no .env leaks).
    copy_file(asset_dir / "dockerignore", repo_root / ".dockerignore", ".dockerignore")

    # Traefik
    copy_file(
        asset_dir / "infra-compose.yml",
        compose_dir / "traefik" / "compose.yml",
        "traefik/compose.yml",
    )

    # IAMS (Keycloak + AAS + Web + init)
    copy_file(
        asset_dir / "iams" / "compose.yml",
        compose_dir / "iams" / "compose.yml",
        "iams/compose.yml",
    )
    copy_file(
        asset_dir / "realm-import.json",
        compose_dir / "iams" / "keycloak" / "realm-import.json",
        "iams/keycloak/realm-import.json",
    )
    copy_file(
        asset_dir / "iams" / "init" / "iams-aas-init.postman_collection.json",
        compose_dir / "iams" / "init" / "iams-aas-init.postman_collection.json",
        "iams/init/iams-aas-init.postman_collection.json",
    )
    copy_tree(
        asset_dir / "iams" / "init" / "project-aas",
        compose_dir / "iams" / "init" / "project-aas",
        "iams/init/project-aas/ (script-based bootstrap template)",
    )

    # SDS - companion service to IAMS, always bundled
    copy_file(
        asset_dir / "sds" / "compose.yml",
        compose_dir / "sds" / "compose.yml",
        "sds/compose.yml",
    )

    # OTEL gateway collector - default-on observability. The
    # OTLP endpoint every service targets; the `signoz` backend stays opt-in.
    # Full dir copy: compose.yml + otel-collector-config.yaml.
    copy_tree(
        asset_dir / "otel",
        compose_dir / "otel",
        "otel/ (gateway collector)",
    )

    # Shared configuration
    copy_file(asset_dir / "gitignore", compose_dir / ".gitignore", ".gitignore")
    copy_file(asset_dir / "env.template", compose_dir / ".env.template", ".env.template")
    copy_file(asset_dir / "env.template", compose_dir / ".env", ".env")
    copy_file(
        asset_dir / "compose-override-sample.yml",
        compose_dir / "compose.override.sample.yml",
        "compose.override.sample.yml",
    )


# ---------------------------------------------------------------------------
# AOH catalogue services
# ---------------------------------------------------------------------------

def add_service(compose_dir: Path, asset_dir: Path, svc: str):
    """Copy the full asset directory for a catalogue service."""
    spec = SERVICES[svc]
    src = asset_dir / spec["asset_dir"]
    dest = compose_dir / svc
    copy_tree(src, dest, f"{svc}/ (full tree)")


# ---------------------------------------------------------------------------
# Custom services
# ---------------------------------------------------------------------------

def parse_custom_service_arg(value: str) -> dict:
    """Parse --custom-service 'name=foo,type=go' into {'name': 'foo', 'type': 'go'}."""
    parts = dict(kv.split("=", 1) for kv in value.split(",") if "=" in kv)
    if "name" not in parts or "type" not in parts:
        raise argparse.ArgumentTypeError(
            f"--custom-service expects 'name=<name>,type=<go|web>', got: {value!r}"
        )
    if parts["type"] not in CUSTOM_TEMPLATES:
        raise argparse.ArgumentTypeError(
            f"--custom-service type must be one of {list(CUSTOM_TEMPLATES)}, got: {parts['type']!r}"
        )
    return parts


def add_custom_service(compose_dir: Path, asset_dir: Path, name: str, svc_type: str):
    """Copy the appropriate template with <name> and <NAME> replaced.

    `<name>` is the literal service name (e.g. `incident-svc`).
    `<NAME>` is the env-var-safe form used in compose interpolations like
    `${<NAME>_TAG:-local}` - uppercased, hyphens replaced with underscores
    (compose env-var names cannot contain hyphens).
    """
    template = asset_dir / CUSTOM_TEMPLATES[svc_type]
    dest = compose_dir / name / "compose.yml"
    replacements = {
        "<name>": name,
        "<NAME>": name.upper().replace("-", "_"),
    }
    copy_template(template, dest, replacements, f"{name}/compose.yml ({svc_type} template)")


# ---------------------------------------------------------------------------
# Top-level compose.yml
# ---------------------------------------------------------------------------

def gen_top_compose_yml(
    compose_dir: Path, aoh_services: list, custom_services: list
):
    """Regenerate compose/compose.yml from the union of includes.

    Always includes traefik, iams, sds, otel (core). Adds each requested AOH
    catalogue service and each custom service. **Also preserves any
    pre-existing app includes** that were in the previous compose.yml
    but were not passed on this invocation - typically `apps/<name>/`
    services added by an earlier run with `--custom-service` and now
    living in `compose/<name>/compose.yml`. Without this, re-running
    the bootstrap to add a new AOH service silently drops the
    custom-app includes from the top-level file.
    """
    lines = ["name: aoh", "", "include:"]

    # Core - in dependency order so Compose resolves cleanly.
    core = [
        "./traefik/compose.yml",
        "./iams/compose.yml",
        "./sds/compose.yml",
        "./otel/compose.yml",
    ]
    catalogue = [f"./{svc}/compose.yml" for svc in aoh_services]
    custom = [f"./{cs['name']}/compose.yml" for cs in custom_services]

    declared = set(core + catalogue + custom)

    # Preserve any include from the previous compose.yml that we did NOT
    # just re-declare - these are the custom app entries the caller did
    # not pass on this invocation. Only preserve includes whose target
    # directory currently exists, so a stale include from a deleted
    # service doesn't get carried forward.
    preserved: list[str] = []
    prior = compose_dir / "compose.yml"
    if prior.is_file():
        for raw in prior.read_text().splitlines():
            line = raw.strip()
            if not line.startswith("- "):
                continue
            target = line[2:].strip()
            if target in declared:
                continue
            # Resolve `./foo/compose.yml` -> compose/foo/compose.yml
            if target.startswith("./"):
                local = compose_dir / target[2:]
            else:
                local = compose_dir / target
            if local.is_file():
                preserved.append(target)

    includes = core + catalogue + custom + preserved

    for inc in includes:
        lines.append(f"    - {inc}")
    lines.append("")  # trailing newline

    path = compose_dir / "compose.yml"
    path.write_text("\n".join(lines))
    suffix = f" (+{len(preserved)} preserved)" if preserved else ""
    print(f"  WRITE compose.yml ({len(includes)} includes{suffix})")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Bootstrap the AOH Docker Compose stack",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--repo-root", required=True, help="Repository root directory")
    parser.add_argument(
        "--services",
        default="",
        help="Comma-separated AOH catalogue services (e.g. rtus,gis,ian). "
             "Transitive deps are resolved automatically.",
    )
    parser.add_argument(
        "--custom-service",
        action="append",
        default=[],
        type=parse_custom_service_arg,
        help="Add a project-local service. Repeatable. "
             "Format: name=<name>,type=<go|web>",
    )
    args = parser.parse_args()

    repo_root = Path(args.repo_root).resolve()
    compose_dir = repo_root / "compose"
    script_dir = Path(__file__).parent.parent
    asset_dir = script_dir / "assets"

    if not repo_root.exists():
        print(f"ERROR: Repo root not found: {repo_root}")
        sys.exit(1)
    if not asset_dir.exists():
        print(f"ERROR: Asset dir not found: {asset_dir}")
        sys.exit(1)

    requested = [s.strip() for s in args.services.split(",") if s.strip()]
    aoh_services = resolve_deps(requested)
    custom_services = args.custom_service

    print(f"\nBootstrapping compose/ at {compose_dir}")
    if aoh_services:
        print(f"  AOH services (with deps): {aoh_services}")
    if custom_services:
        print(f"  Custom services: {[cs['name'] for cs in custom_services]}")
    print()

    # [1/5] Bootstrap core (traefik + iams + sds + shared files)
    print("[1/5] Core bootstrap")
    bootstrap(compose_dir, asset_dir, repo_root)

    # [2/5] Runtime detection
    print("\n[2/5] Container runtime")
    runtime_verdict = write_docker_override_if_needed(compose_dir)

    # [3/5] AOH catalogue services
    print("\n[3/5] AOH services")
    if aoh_services:
        for svc in aoh_services:
            add_service(compose_dir, asset_dir, svc)
    else:
        print("  (none requested)")

    # [4/5] Custom services
    print("\n[4/5] Custom services")
    if custom_services:
        for cs in custom_services:
            add_custom_service(compose_dir, asset_dir, cs["name"], cs["type"])
    else:
        print("  (none requested)")

    # [5/5] Top-level compose.yml
    print("\n[5/5] Top-level compose.yml")
    gen_top_compose_yml(compose_dir, aoh_services, custom_services)

    print(f"\n{'=' * 50}")
    print(f"Compose stack ready: {compose_dir}")
    print(f"{'=' * 50}")
    print()
    print(f"Runtime: {runtime_verdict}")
    print("Host ports claimed (defaults): TRAEFIK_HTTP_PORT=80, SDS_TCP_PORT=5333 -")
    print("  override in .env if taken (pre-existing traefik/compose.yml files from")
    print("  older scaffolds are kept as-is and may still hardcode 80/5333)")
    print()
    print("Next steps:")
    print(f"  cd {compose_dir}")
    print("  # inspect .env and fill in any required values (DEV_DOMAIN, DEV_USER, DEV_PASSWORD, image tags)")
    if custom_services:
        print("  # for each custom service, open compose/<name>/compose.yml and uncomment")
        print("    the blocks you need (env vars, depends_on, etc.)")
        print("  # pre-build each custom service image (see SKILL.md -> Verify):")
        print("  #   export your GitHub PAT and `podman build --secret id=access_token,env=... -t <name>:local ...`")
    print("  # then:")
    print("  docker compose up -d")
    print()
    print("Keycloak realm customisation (roles, seed users, custom clients) - see")
    print("  .claude/skills/aoh-knowledge/references/keycloak-realm-guide.md")


if __name__ == "__main__":
    main()
