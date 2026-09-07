#!/usr/bin/env python3
"""
Scaffold a Go microservice inside a Turborepo monorepo.

Generates all infrastructure files plus a minimal example entity so the
scaffold compiles, passes lint, and passes tests out of the box.

Example files are annotated with TODELETE(scaffold) comments. To find
all scaffold examples to clean up: grep -r "TODELETE(scaffold)"

Usage:
    python3 scaffold.py \
        --name <service-name> \
        --module <module-path> \
        --repo-root <path-to-turborepo-root> \
        [--target-dir <relative-path>] \
        [--description <service-description>] \
        [--go-version <version>] \
        [--alpine-version <version>] \
        [--golangci-lint-version <version>] \
        [--swag-version <version>] \
        [--mockery-version <version>]

By default the scaffold lands at `apps/<name>/` (consumer-app monorepo shape).
For the ops-hub platform-monorepo shape, pass `--target-dir modules/<name>/service`
(or any other relative path under the repo root). The flag changes both the on-disk
location and the `go.work` `use (...)` entry that gets written; everything else
(asset files, Go source tree, turbo.json registration) is unchanged.
"""

import argparse
import json
import os
import re
import subprocess
import sys
from pathlib import Path

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------

# Hard fallback used only when `go version` is not on PATH at scaffold time.
# Real default is detected from the host (see detect_host_go_minor below).
FALLBACK_GO_VERSION = "1.24"


def detect_host_go_minor() -> str:
    """Return the host's Go version as major.minor (e.g. "1.26").

    Used as the default for `--go-version`. We deliberately strip the patch
    so the Dockerfile tag (`golang:1.26`) picks up the latest patch on rebuild,
    and so a host running `go 1.26.1` doesn't lock the image to 1.26.1
    when 1.26.3 already exists. If `go.mod` declares a patch-level version,
    `GOTOOLCHAIN=auto` in the image will fetch it.

    Falls back to FALLBACK_GO_VERSION when `go` is not on PATH (e.g. the
    skill is being invoked from an environment without a Go toolchain).
    """
    try:
        result = subprocess.run(
            ["go", "version"], capture_output=True, text=True, check=True
        )
        # Output looks like: "go version go1.26.1 darwin/arm64"
        parts = result.stdout.split()
        if len(parts) >= 3 and parts[2].startswith("go"):
            full = parts[2][2:]  # strip leading "go"
            mm = ".".join(full.split(".")[:2])  # "1.26.1" -> "1.26"
            if mm:
                return mm
    except (FileNotFoundError, subprocess.CalledProcessError):
        pass
    return FALLBACK_GO_VERSION


DEFAULT_GO_VERSION = detect_host_go_minor()
DEFAULT_ALPINE_VERSION = "3.21"
DEFAULT_GOLANGCI_LINT_VERSION = "v2.10.1"
DEFAULT_SWAG_VERSION = "v1.16.4"
DEFAULT_MOCKERY_VERSION = "v3.3.2"
DEFAULT_MIGRATE_VERSION = "v4.19.1"

# Pinned library versions — must match tool versions to avoid incompatibilities.
# e.g., swag CLI v1.16.4 generates code requiring swag library v1.16.x
PINNED_DEPS = [
    "github.com/go-chi/chi/v5@v5.2.5",
    "github.com/golang-migrate/migrate/v4@v4.19.1",
    "github.com/go-chi/cors@v1.2.2",
    "github.com/go-chi/render@v1.0.3",
    "github.com/google/uuid@v1.6.0",
    "github.com/jmoiron/sqlx@v1.4.0",
    "github.com/lib/pq@v1.12.3",
    "github.com/mitchellh/mapstructure@v1.5.0",
    "github.com/spf13/pflag@v1.0.10",
    "github.com/spf13/viper@v1.21.0",
    "github.com/stretchr/testify@v1.11.1",
    "github.com/swaggo/files@v1.0.1",
    "github.com/swaggo/http-swagger@v1.3.4",
    "github.com/swaggo/swag@v1.16.6",
    "go.uber.org/zap@v1.27.1",
]

# All pinned deps are public — they're fetched with a normal `go get`.
PUBLIC_DEPS = PINNED_DEPS

# The shared Go foundation library (incl. the `otel` subpackage). A normal
# registry dependency since AOH-8074: scaffolded services `require` it at a
# released `packages/aoh-golib/vX.Y.Z` tag and fetch it from the private
# ops-hub repo (GitHub auth or a spoke GOPROXY mirror) — no local `replace`,
# no vendored copy. The version below is load-bearing; bump it when a new
# golib tag ships.
AOH_GOLIB_MODULE = "github.com/mssfoobar/ops-hub/packages/aoh-golib"
AOH_GOLIB_VERSION = "v0.2.0"


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def write_file(path: Path, content: str):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content)


def error_code_prefix(name: str) -> str:
    """The service's errorCode namespace, e.g. `incident-svc` -> `INCIDENT_SVC`.

    Codes must match aoherr's `^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$` or MustCode
    panics at service startup, so the prefix is normalized here rather than
    trusting the raw --name: upper-cased, every run of non-alphanumerics
    collapsed to one underscore, and a leading digit prefixed with `SVC_`
    (a code may not start with a digit).
    """
    prefix = re.sub(r"[^A-Z0-9]+", "_", name.upper()).strip("_")
    if not prefix:
        return "SVC"
    if prefix[0].isdigit():
        return f"SVC_{prefix}"
    return prefix


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
    print(f"  COPY {asset_name}")


def render_tree(src_root: Path, dst_root: Path, replacements: dict):
    """Walk src_root, substitute placeholders, write to dst_root.

    Files ending in `.tmpl` have placeholders replaced and the `.tmpl` suffix
    stripped on write. Files without `.tmpl` are copied verbatim.

    Placeholder convention: `__AOH_NAME__`, `__AOH_MODULE__`, `__AOH_TITLE__` —
    safe markers that never appear in Go, YAML, or Dockerfile syntax.
    """
    for src in sorted(src_root.rglob("*")):
        if not src.is_file():
            continue
        rel = src.relative_to(src_root)
        is_template = rel.suffix == ".tmpl"
        rel_out = rel.with_suffix("") if is_template else rel
        dst = dst_root / rel_out

        if is_template:
            content = src.read_text()
            for placeholder, value in replacements.items():
                content = content.replace(placeholder, value)
            dst.parent.mkdir(parents=True, exist_ok=True)
            dst.write_text(content)
        else:
            dst.parent.mkdir(parents=True, exist_ok=True)
            shutil_copy(src, dst)
        print(f"  COPY {rel_out}")


def shutil_copy(src: Path, dst: Path):
    import shutil
    shutil.copyfile(src, dst)


# ---------------------------------------------------------------------------
# Asset files
# ---------------------------------------------------------------------------

def copy_assets(asset_dir: Path, service_dir: Path, repo_root: Path, args):
    replacements = {
        "__AOH_MODULE__": args.module,
        "__AOH_NAME__": args.name,
        "__AOH_DESCRIPTION__": args.description,
        "{GO_VERSION}": args.go_version,
        "{ALPINE_VERSION}": args.alpine_version,
        "{GOLANGCI_LINT_VERSION}": args.golangci_lint_version,
        "{SWAG_VERSION}": args.swag_version,
        "{MOCKERY_VERSION}": args.mockery_version,
        "{MIGRATE_VERSION}": args.migrate_version,
    }

    golangci_dest = repo_root / ".golangci.yml"
    if not golangci_dest.exists():
        copy_asset(asset_dir, ".golangci.yml", golangci_dest, {})
    else:
        print("  SKIP .golangci.yml (already exists)")

    copy_asset(asset_dir, ".gitignore", service_dir / ".gitignore", {})
    copy_asset(asset_dir, ".dockerignore", service_dir / ".dockerignore", {})
    copy_asset(asset_dir, "Makefile", service_dir / "Makefile", replacements)
    copy_asset(asset_dir, "Dockerfile", service_dir / "Dockerfile", replacements)
    copy_asset(asset_dir, "AGENTS.md", service_dir / "AGENTS.md", replacements)
    copy_asset(asset_dir, "CLAUDE.md", service_dir / "CLAUDE.md", {})
    copy_asset(asset_dir, "README.md.tmpl", service_dir / "README.md", replacements)

    # .mockery.yaml — inject the example interface only when the example is kept.
    mockery_src = asset_dir / ".mockery.yaml"
    if mockery_src.exists():
        content = mockery_src.read_text()
        content = content.replace("__AOH_MODULE__", args.module)
        if not getattr(args, "no_example", False):
            content = content.replace(
                "    interfaces:",
                "    interfaces:\n"
                "      ExampleService: # TODELETE(scaffold): remove when adding real entities",
                1,
            )
        (service_dir / ".mockery.yaml").write_text(content)
        print("  COPY .mockery.yaml")


# ---------------------------------------------------------------------------
# Workspace setup
# ---------------------------------------------------------------------------

def setup_package_json(service_dir: Path, service_name: str):
    pkg = {
        "name": service_name,
        "version": "0.0.0",
        "private": True,
        "scripts": {
            "build": "go build -o bin/server ./cmd/server",
            "dev": "go run ./cmd/server --log.level=debug",
            "test": "make test",
            "lint": "make lint-code",
        },
    }
    (service_dir / "package.json").write_text(json.dumps(pkg, indent=2) + "\n")


def setup_go_mod(service_dir: Path, module_path: str):
    if (service_dir / "go.mod").exists():
        print("  SKIP go.mod (already exists)")
        return
    os.system(f'cd "{service_dir}" && go mod init {module_path}')


def setup_aoh_golib_require(service_dir: Path):
    """Pin aoh-golib as a normal registry dependency (no `replace`).

    The module is fetched from the private ops-hub repo at a released
    `packages/aoh-golib/vX.Y.Z` tag — the same way the scaffolded service's
    Dockerfile resolves it in-image.
    """
    os.system(
        f'cd "{service_dir}" && '
        f'go mod edit -require={AOH_GOLIB_MODULE}@{AOH_GOLIB_VERSION}'
    )
    print(f"  go.mod: require {AOH_GOLIB_MODULE} {AOH_GOLIB_VERSION}")


def setup_golib_fetch_env():
    """Let `go` fetch the private aoh-golib module during Step 7.

    GONOSUMDB skips the public checksum DB for first-party modules while still
    honoring whatever GOPROXY the host has configured — a spoke mirror keeps
    working, and the default proxy chain falls through to a direct git fetch
    using the developer's normal GitHub credentials. Skipped when the host
    already configures GOPRIVATE / GONOSUMDB (probed via `go env`, which sees
    both process env and `go env -w` values). Set via os.environ (child
    processes inherit) so it works on Windows too — no shell env-prefix.
    """
    try:
        result = subprocess.run(
            ["go", "env", "GOPRIVATE", "GONOSUMDB"],
            capture_output=True, text=True, check=True,
        )
        if any(line.strip() for line in result.stdout.splitlines()):
            print("  (using host GOPRIVATE/GONOSUMDB as-is)")
            return
    except (FileNotFoundError, subprocess.CalledProcessError):
        pass
    os.environ["GONOSUMDB"] = "github.com/mssfoobar/*"
    print("  GONOSUMDB=github.com/mssfoobar/* (private-module fetch)")


def _is_path_in_use_directive(content: str, path: str) -> bool:
    """True iff `path` appears in a real `use` directive in a go.work file.

    Strips // line comments before checking so a path that appears only inside
    a comment (e.g., a stub `go.work` with `// First entry expected: ./foo`)
    is NOT treated as "already listed". Without this, the substring match
    used previously collided with documentation comments.

    Matches both go.work shapes:
      use ./path/here
      use ( ... ./path/here ... )
    """
    # Drop // line comments (go.work doesn't support /* ... */).
    no_comments = re.sub(r"//[^\n]*", "", content)
    # Single-line form: `use ./path`
    if re.search(rf"(?m)^\s*use\s+\./{re.escape(path)}\s*$", no_comments):
        return True
    # Block form: scan inside each `use (...)` block.
    for block in re.finditer(r"use\s*\(([^)]*)\)", no_comments, re.DOTALL):
        for line in block.group(1).splitlines():
            if line.strip() == f"./{path}":
                return True
    return False


def setup_go_work(repo_root: Path, service_rel_path: str):
    go_work = repo_root / "go.work"
    use_line = f"\t./{service_rel_path}"

    if go_work.exists():
        content = go_work.read_text()
        if _is_path_in_use_directive(content, service_rel_path):
            print("  SKIP go.work (already listed)")
            return
        # Two cases:
        # (a) Existing `use (...)` block — insert the new entry before the close paren.
        # (b) No `use (...)` block yet (stub with only `go X.Y` + comments, as the
        #     ops-hub repo ships pre-AOH-7270) — append a fresh `use (...)` block.
        if re.search(r"use\s*\([^)]*\n\s*\)", content, re.DOTALL):
            content = re.sub(r"(use\s*\([^)]*)(\n\s*\))",
                             rf"\1\n{use_line}\2", content, count=1, flags=re.DOTALL)
        else:
            if not content.endswith("\n"):
                content += "\n"
            content += f"\nuse (\n{use_line}\n)\n"
        go_work.write_text(content)
        print(f"  UPDATE go.work")
    else:
        # `go.work` keeps the full host version (incl. patch) since it pins
        # the toolchain. The Dockerfile only needs major.minor; see
        # detect_host_go_minor().
        go_version = DEFAULT_GO_VERSION
        try:
            result = subprocess.run(
                ["go", "version"], capture_output=True, text=True, check=True
            )
            parts = result.stdout.split()
            if len(parts) >= 3 and parts[2].startswith("go"):
                go_version = parts[2][2:]
        except (FileNotFoundError, subprocess.CalledProcessError):
            pass
        content = f"go {go_version}\n\nuse (\n{use_line}\n)\n"
        go_work.write_text(content)
        print("  CREATE go.work")


def setup_turbo_json(repo_root: Path, service_name: str):
    turbo_path = repo_root / "turbo.json"
    if not turbo_path.exists():
        print("  SKIP turbo.json (not found)")
        return

    content = turbo_path.read_text()
    turbo = json.loads(content)

    override_key = f"{service_name}#build"
    if override_key in turbo.get("tasks", {}):
        print(f"  SKIP turbo.json ({override_key} exists)")
        return

    turbo.setdefault("tasks", {})[override_key] = {
        "inputs": ["**/*.go", "go.mod", "go.sum", "config.yaml*", "schema/**"],
        "outputs": ["bin/**"],
    }
    turbo_path.write_text(json.dumps(turbo, indent=2) + "\n")
    print(f"  UPDATE turbo.json")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(description="Scaffold a Go microservice")
    parser.add_argument("--name", required=True, help="Service name (e.g., incidents)")
    parser.add_argument("--module", required=True, help="Go module path")
    parser.add_argument("--repo-root", required=True, help="Turborepo root directory")
    parser.add_argument(
        "--target-dir",
        default=None,
        help="Relative path under the repo root where the service should be created "
             "(e.g., 'modules/gis/service' for the ops-hub platform-monorepo shape). "
             "Defaults to 'apps/<name>' for the consumer-app monorepo shape.",
    )
    parser.add_argument("--description", default="A Go microservice", help="Service description")
    parser.add_argument(
        "--no-example",
        action="store_true",
        help="Skip the TODELETE(scaffold) example entity. Use when you already know "
             "the first entity you want to build and don't need the reference files.",
    )
    parser.add_argument("--go-version", default=DEFAULT_GO_VERSION)
    parser.add_argument("--alpine-version", default=DEFAULT_ALPINE_VERSION)
    parser.add_argument("--golangci-lint-version", default=DEFAULT_GOLANGCI_LINT_VERSION)
    parser.add_argument("--swag-version", default=DEFAULT_SWAG_VERSION)
    parser.add_argument("--mockery-version", default=DEFAULT_MOCKERY_VERSION)
    parser.add_argument("--migrate-version", default=DEFAULT_MIGRATE_VERSION)
    args = parser.parse_args()

    repo_root = Path(args.repo_root).resolve()
    service_rel_path = args.target_dir if args.target_dir else f"apps/{args.name}"
    # Reject absolute paths and `..` traversal so the scaffold can never escape repo_root.
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

    # Shared placeholder values for rendered Go / YAML templates.
    go_tree_replacements = {
        "__AOH_MODULE__": args.module,
        "__AOH_NAME__": args.name,
        "__AOH_TITLE__": args.name.replace("-", " ").title(),
        "__AOH_CODE_PREFIX__": error_code_prefix(args.name),
    }

    print(f"\nScaffolding Go microservice: {args.name}")
    print(f"  Module: {args.module}")
    print(f"  Target: {service_dir}")
    if args.no_example:
        print(f"  Example entity: SKIPPED (--no-example)")
    print()

    # Step 1: Create service directory & package.json
    service_dir.mkdir(parents=True, exist_ok=True)
    print("[1/7] Package setup")
    setup_package_json(service_dir, args.name)

    # Step 2: Copy asset files (Makefile, Dockerfile, AGENTS.md, CLAUDE.md, .golangci.yml, etc.)
    print("\n[2/7] Asset files")
    copy_assets(asset_dir, service_dir, repo_root, args)

    # Step 3: Render Go source tree (infrastructure + optional example entity).
    print("\n[3/7] Go source tree")
    render_tree(asset_dir / "go", service_dir, go_tree_replacements)
    if not args.no_example:
        render_tree(asset_dir / "go-example", service_dir, go_tree_replacements)
    (service_dir / "schema").mkdir(parents=True, exist_ok=True)

    # When skipping the example, also strip the example wiring from init.go and router.go
    # so the scaffold compiles cleanly out of the box.
    if args.no_example:
        strip_example_wiring(service_dir)

    # Normalize formatting: template edits and the example-strip above can leave
    # gofmt-dirty output (e.g. a struct over-aligned after a field is removed),
    # which trips the gci/gofmt formatters in `make lint-code`. gofmt -w keeps
    # the generated infrastructure lint-clean out of the box.
    os.system(f'cd "{service_dir}" && gofmt -w . 2>/dev/null')

    go_files = list(service_dir.rglob("*.go"))
    print(f"  Generated {len(go_files)} Go files total")

    # Step 4: Initialize Go module + pin aoh-golib at its released tag.
    print("\n[4/7] Go module")
    setup_go_mod(service_dir, args.module)
    setup_aoh_golib_require(service_dir)

    # Step 5: Go workspace
    print("\n[5/7] Go workspace")
    setup_go_work(repo_root, service_rel_path)

    # Step 6: Turbo config
    print("\n[6/7] Turbo config")
    setup_turbo_json(repo_root, args.name)

    # Step 7: Install dependencies with pinned versions.
    # `go mod tidy` fetches aoh-golib (incl. its otel subpackage) from the
    # private ops-hub repo at the pinned tag, plus the public deps — the host
    # needs GitHub access for github.com/mssfoobar/ops-hub (normal git
    # credentials) or a spoke GOPROXY mirror.
    print("\n[7/7] Dependencies")
    setup_golib_fetch_env()
    print(f"  Pinning {len(PUBLIC_DEPS)} public dependencies...")
    if os.system(f'cd "{service_dir}" && go get {" ".join(PUBLIC_DEPS)} 2>&1') != 0:
        print("  WARNING: failed to pin public dependencies (network/version issue).")
        print(f'           Re-run: cd "{service_dir}" && go get {" ".join(PUBLIC_DEPS)}')

    print("  Running go mod tidy...")
    if os.system(f'cd "{service_dir}" && go mod tidy 2>&1') != 0:
        print("  WARNING: go mod tidy had issues.")
        print("           The aoh-golib fetch needs GitHub access to the private")
        print("           ops-hub repo (normal git credentials; GONOSUMDB is already")
        print("           set) or a spoke registry mirror (GOPROXY=<mirror> GOSUMDB=off).")

    print(f"\n{'=' * 50}")
    print(f"Scaffold complete: {service_rel_path}/")
    print(f"{'=' * 50}")
    print(f"\nNext steps:")
    print(f"  cd {service_rel_path}")
    print(f"  make swag          # generate swagger docs")
    print(f"  make mock          # generate mocks (overwrites placeholders)")
    print(f"  go mod tidy        # finalize deps after generation")
    print(f"  make lint-code     # verify linting passes")
    print(f"  make test          # regenerates mocks then runs go test")
    if not args.no_example:
        print(f"\nTo find all scaffold examples to clean up:")
        print(f'  grep -r "TODELETE(scaffold)" {service_rel_path}/')


def strip_example_wiring(service_dir: Path):
    """When --no-example is passed, init.go and router.go still reference the
    example entity. Rewrite them to the minimal no-example form so the service
    compiles out of the box."""
    import re

    init_go = service_dir / "cmd/server/init.go"
    if init_go.exists():
        content = init_go.read_text()
        # Drop the TODELETE wiring block and trim the 3rd NewHTTPServer arg.
        content = content.replace(
            "\t// TODELETE(scaffold): Replace example wiring with real entities\n"
            "\texampleRepo := repo.NewExampleRepository(db)\n"
            "\texampleSvc := service.NewExampleService(exampleRepo)\n\n"
            "\tsrv := handler.NewHTTPServer(*cfg, db, exampleSvc)\n",
            "\tsrv := handler.NewHTTPServer(*cfg, db)\n",
        )
        # The service import is now unused. Match whatever module path was
        # substituted in at render time.
        content = re.sub(r'\t"[^"]+/internal/service"\n', "", content)
        init_go.write_text(content)

    router_go = service_dir / "internal/handler/router.go"
    if router_go.exists():
        content = router_go.read_text()
        # Drop the example field from the struct.
        content = content.replace(
            "\texampleSvc service.ExampleService // TODELETE(scaffold): replace with real services\n",
            "",
        )
        # Simplify the constructor signature/body.
        content = content.replace(
            "func NewHTTPServer(cfg config.Config, db *repo.DB, exampleSvc service.ExampleService) *HTTPServer {\n"
            "\treturn &HTTPServer{\n"
            "\t\tcfg:        cfg,\n"
            "\t\tdb:         db,\n"
            "\t\texampleSvc: exampleSvc,\n"
            "\t}\n"
            "}",
            "func NewHTTPServer(cfg config.Config, db *repo.DB) *HTTPServer {\n"
            "\treturn &HTTPServer{\n"
            "\t\tcfg: cfg,\n"
            "\t\tdb:  db,\n"
            "\t}\n"
            "}",
        )
        # Drop the example route Mount.
        content = content.replace(
            "\n\t// TODELETE(scaffold): Replace example route with real entity routes\n"
            "\tr.Mount(\"/examples\", getExampleHandler(s.exampleSvc, s.cfg.IAMS))\n",
            "",
        )
        # service import is now unused.
        content = re.sub(r'\t"[^"]+/internal/service"\n', "", content)
        router_go.write_text(content)


if __name__ == "__main__":
    main()
