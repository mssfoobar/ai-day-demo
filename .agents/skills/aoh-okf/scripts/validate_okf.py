#!/usr/bin/env python3
"""Validate an Open Knowledge Format (OKF) v0.1 bundle.

Dependency-free (stdlib only). Checks the spec's hard conformance rules
(https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)
plus house rules, and reports soft-guidance gaps as warnings.

Errors (exit 1):
  E1  non-reserved .md file has no parseable YAML frontmatter block
  E2  frontmatter lacks a non-empty `type` field
  E3  reserved file misuse: index.md/log.md with frontmatter (root index.md
      may carry `okf_version` only)
  E4  broken link: a link whose target does not exist on disk (out-of-bundle
      targets are allowed but are existence-checked against the working tree)
  E5  bundle-absolute link (`](/path)`) — spec-legal but breaks GitHub
      rendering; house rule is relative links only (W3 warning with --lax,
      in which case the target is still existence-checked)
  E6  unreadable content: a file that cannot be read/decoded, or a directory
      that cannot be listed (either would otherwise silently shrink coverage)
  E7  symlinked directory: not followed by the walk, so its contents would
      be silently unvalidated — restructure or validate the target directly
  E8  no markdown files found — almost certainly the wrong directory

Warnings (exit 0):
  W1  recommended frontmatter field missing (title/description/tags/timestamp)
  W2  log.md date heading not `## YYYY-MM-DD`
  W3  bundle-absolute link, under --lax only

Usage:
  validate_okf.py <bundle-dir> [--lax] [--quiet]
"""

from __future__ import annotations

import argparse
import os
import re
import sys
import urllib.parse

RESERVED = {"index.md", "log.md"}
RECOMMENDED_FIELDS = ("title", "description", "tags", "timestamp")
# Destination is either <...> (may contain spaces) or a run without
# whitespace/`)`; an optional "title" may follow. Bare destinations with
# unescaped spaces are invalid CommonMark and are not scanned.
LINK_RE = re.compile(r"\]\(\s*(<[^<>\n]*>|[^)\s]+)(?:\s+\"[^\"]*\")?\s*\)")
SKIP_SCHEMES = ("http://", "https://", "mailto:", "ssh://", "tel:", "ftp://")
KEY_RE = re.compile(r"^([A-Za-z_][A-Za-z0-9_-]*):", re.M)
FENCE_RE = re.compile(r"^\s{0,3}(`{3,}|~{3,})")
LIST_RE = re.compile(r"^\s*(?:[-*+]|\d{1,9}[.)])\s")
INLINE_CODE_RE = re.compile(r"`[^`\n]*`")


def parse_frontmatter(text: str) -> tuple[str | None, bool]:
    """Return (frontmatter body, parseable) — parseable means fenced by ---."""
    if not text.startswith("---\n"):
        return None, False
    end = text.find("\n---", 3)  # from 3, so an empty block (---\n---) parses
    if end == -1:
        return None, False
    return text[4:end + 1], True


def field_value(fm: str, key: str) -> str | None:
    m = re.search(rf"^{key}:\s*(.*)$", fm, re.M)
    return m.group(1).strip() if m else None


def strip_code(text: str) -> str:
    """Blank code content (links there are examples): fenced blocks (tracking
    fence char AND length, per CommonMark closing rules), indented code
    blocks (4-space/tab indent after a blank line outside list context), and
    inline code spans. Output preserves the line count."""
    out: list[str] = []
    fence_char, fence_len = None, 0
    in_indented = False
    prev_blank = True  # document start behaves like a preceding blank line
    last_text_is_list = False
    for line in text.split("\n"):
        if fence_char is not None:
            m = FENCE_RE.match(line)
            run = m.group(1) if m else ""
            # A closing fence is same-char, at least opener-length, nothing else.
            if run and run[0] == fence_char and len(run) >= fence_len and line.strip() == run:
                fence_char = None
                prev_blank = True  # fence end is a block boundary: indented code may start
            out.append("")
            continue
        m = FENCE_RE.match(line)
        if m:
            fence_char, fence_len = m.group(1)[0], len(m.group(1))
            in_indented = False
            prev_blank = False
            out.append("")
            continue
        if not line.strip():
            prev_blank = True
            out.append(line)
            continue
        if (line.startswith("    ") or line.startswith("\t")) and (
                in_indented or (prev_blank and not last_text_is_list)):
            in_indented = True
            prev_blank = False
            out.append("")
            continue
        in_indented = False
        last_text_is_list = bool(LIST_RE.match(line))
        prev_blank = False
        out.append(INLINE_CODE_RE.sub("", line))
    return "\n".join(out)


def check_links(path: str, dirpath: str, bundle: str, text: str, lax: bool,
                errors: list[str], warnings: list[str]) -> None:
    for m in LINK_RE.finditer(strip_code(text)):
        raw = m.group(1)
        if raw.startswith("<") and raw.endswith(">"):
            raw = raw[1:-1]
        raw = raw.split("#", 1)[0]
        if not raw or raw.startswith(SKIP_SCHEMES):
            continue
        link = urllib.parse.unquote(raw)  # decode %20 etc. for resolution only
        if raw.startswith("/"):  # classify on the authored text, not decoded
            msg = f"{path}: bundle-absolute link `{raw}` — use a relative link (breaks GitHub rendering)"
            if not lax:
                errors.append(f"E5 {msg}")
                continue  # E5 already demands a rewrite; don't also E4 it
            warnings.append(f"W3 {msg}")
            target = os.path.join(bundle, link.lstrip("/"))
        else:
            target = os.path.normpath(os.path.join(dirpath, link))
        exists = os.path.isdir(target) if link.endswith("/") else os.path.exists(target)
        if not exists:
            errors.append(f"E4 {path}: broken link `{raw}` (resolved: {target})")


def validate(bundle: str, lax: bool) -> tuple[list[str], list[str], int]:
    errors: list[str] = []
    warnings: list[str] = []
    count = 0
    root_index = os.path.normpath(os.path.join(bundle, "index.md"))

    def on_walk_error(err: OSError) -> None:
        errors.append(f"E6 {getattr(err, 'filename', bundle)}: cannot list directory ({err.strerror or err})")

    for dirpath, dirnames, files in os.walk(bundle, onerror=on_walk_error):
        for d in list(dirnames):
            if d.startswith("."):
                dirnames.remove(d)
            elif os.path.islink(os.path.join(dirpath, d)):
                errors.append(
                    f"E7 {os.path.join(dirpath, d)}: symlinked directory is not followed — "
                    "its contents would go unvalidated; restructure or validate the target directly"
                )
        for name in sorted(files):
            if not name.endswith(".md"):
                continue
            count += 1
            path = os.path.join(dirpath, name)
            try:
                with open(path, encoding="utf-8-sig") as fh:  # -sig: tolerate BOM
                    text = fh.read()
            except (OSError, UnicodeDecodeError) as exc:
                errors.append(f"E6 {path}: unreadable ({exc})")
                continue
            fm, ok = parse_frontmatter(text)
            if name in RESERVED:
                keys = set(KEY_RE.findall(fm or ""))
                # keys check: a leading `---` block with no YAML-ish keys is
                # prose under a thematic break, not frontmatter — allow it.
                if ok and keys:
                    is_root_index = os.path.normpath(path) == root_index
                    if not (is_root_index and keys <= {"okf_version"}):
                        errors.append(
                            f"E3 {path}: reserved file must not carry frontmatter"
                            + (" (root index.md may carry okf_version only)" if is_root_index else "")
                            + " — if the leading `---` is a thematic break, put a heading above it"
                        )
                if name == "log.md":
                    for h in re.findall(r"^## (.+)$", text, re.M):
                        if not re.fullmatch(r"\d{4}-\d{2}-\d{2}", h.strip()):
                            warnings.append(f"W2 {path}: log heading `## {h}` is not `## YYYY-MM-DD`")
            else:
                if not ok:
                    errors.append(f"E1 {path}: missing or unclosed YAML frontmatter block")
                else:
                    if not field_value(fm, "type"):
                        errors.append(f"E2 {path}: frontmatter has no non-empty `type` field")
                    missing = [k for k in RECOMMENDED_FIELDS if field_value(fm, k) in (None, "")]
                    if missing:
                        warnings.append(f"W1 {path}: missing recommended field(s): {', '.join(missing)}")
            check_links(path, dirpath, bundle, text, lax, errors, warnings)
    if count == 0:
        errors.append(f"E8 {bundle}: no markdown files found — is this the right directory?")
    return errors, warnings, count


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("bundle", help="path to the OKF bundle directory")
    ap.add_argument("--lax", action="store_true",
                    help="report bundle-absolute links as warnings (W3) instead of "
                         "errors (E5); broken links still fail")
    ap.add_argument("--quiet", action="store_true", help="print errors only")
    args = ap.parse_args()

    if not os.path.isdir(args.bundle):
        print(f"error: not a directory: {args.bundle}", file=sys.stderr)
        return 2

    errors, warnings, count = validate(args.bundle, args.lax)
    for e in errors:
        print(f"ERROR   {e}")
    if not args.quiet:
        for w in warnings:
            print(f"warning {w}")
    status = "FAIL" if errors else "OK"
    print(f"{status}: {count} markdown file(s), {len(errors)} error(s), {len(warnings)} warning(s)")
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main())
