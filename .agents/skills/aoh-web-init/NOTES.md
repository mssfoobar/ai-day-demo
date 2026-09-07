# NOTES — asset pins and import paths

Rationale for the non-obvious pins and import paths in `assets/`. SKILL.md's
"Verify the build" section points here: the build passing does **not** prove
these failure modes stay fixed — only scaffolded-app users hit them. Read the
relevant entry before changing a pin or import path.

## `@mssfoobar/logger` — pin `^1.0.5`, import the bare specifier

Every registry release ≤ 1.0.4 is broken for the bare
specifier: all declare `main: dist/index.js` (a file none of them ship)
and have no `exports` map. The actual file location differs by version —
`1.0.0` ships `dist/Logger.js`; `1.0.1`–`1.0.4` were published from the
retired `aoh-web-lib` repo and ship `Logger.js` at the package root.

Consequences for these assets:

- **Pin `^1.0.5`** (the first release published from ops-hub, with a
  correct `exports` map). Don't loosen the range: prereleases like
  `1.0.5-alpha.N` don't satisfy `^1.0.5`, and anything `≤ 1.0.4` breaks
  `vite build` — `@mssfoobar/gis-web-sdk` imports the bare specifier
  from ~9 compiled components, so the GIS skill is unusable on the
  broken versions.
- **Import the bare specifier `@mssfoobar/logger`** in
  `assets/gateway/[...path]/+server.ts` and
  `assets/web-base/src/lib/aoh/core/logger/Logger.ts`. The old deep
  import `@mssfoobar/logger/dist/Logger` only resolved under 1.0.0's
  layout (no exports map → arbitrary subpaths allowed); 1.0.5's exports
  map still allows it as a legacy back-compat subpath, but new code
  should not depend on it.

## `cookie` — pinned `0.7.2` (+ `@types/cookie`)

A newer `cookie` major changed the `CookieSerializeOptions` type shape
("legacy cookie type" failure mode); the auth routes' option objects
type-check against the 0.7.x API. Bump only together with a pass over
every `cookies.set/serialize` call in the auth-route assets.

## `@fontsource-variable/geist` — pinned `5.2.8`

Some Geist patch versions were unpublished from npm; carets resolved to
versions that no longer exist (`ERR_PNPM_NO_MATCHING_VERSION`). Keep an
exact, known-published pin; verify with
`pnpm view @fontsource-variable/geist versions` before bumping.

## Error contract — `@mssfoobar/ui` `^1.1.0`, `@mssfoobar/errors`, `@opentelemetry/api`

The scaffold's error page (`src/routes/+error.svelte`) and the `handleError`
hook in `assets/hooks.server.ts` implement the AOH error contract, which
constrains three pins:

- **`@mssfoobar/ui` is `^1.1.0`, not `^1.0.0`.** The `./error-surface`
  subpath (the shared `ErrorSurface`) was added in the 1.1.0 minor. A
  `^1.0.0` range can resolve 1.0.x, where the subpath does not exist and the
  import fails at build time.
- **`@mssfoobar/errors`** supplies the `errorCode` vocabulary, the
  class→status/retryability algebra and the `userMessage` catalogue seam. The
  error page and the hook must agree on those, so both read them from this one
  package rather than restating the mapping.
- **`@opentelemetry/api` is a direct dependency**, pinned to the version
  `@mssfoobar/observability` declares as a peer. `handleError` reads the active
  span's trace id from it. It is *not* re-exported by
  `@mssfoobar/observability`, so relying on hoisting would break under pnpm's
  strict layout.

Two rules the assets encode deliberately — don't "simplify" them away:
`handleError` never forwards `err.message` to the client (its returned
`message` is the resolved `userMessage`), and `trace_id` is the active span's
or absent — never a minted UUID and never the literal `unknown`.

## `app.css` — design-system token discipline

`assets/app.css` imports `@mssfoobar/ui/styles/app.css` as the single
source of theme tokens. Never redeclare `--background`, `--primary`,
`--bg-*`, `--text-*` etc. in app-level CSS ("token shadowing" failure
mode) — overrides silently diverge from the design system. Customise via
the layered override pattern documented in the package's `styles/app.css`.

## Svelte-source SDKs under the base's Vite 8 / rolldown

The scaffold pins `vite@8.0.7` (rolldown bundler) + `@sveltejs/vite-plugin-svelte@7`.
A workspace SDK that ships **raw `.svelte`** (with `<script lang="ts">`) in its
`dist/` — `svelte-package` output, e.g. `@mssfoobar/gis-web-sdk` — bundles cleanly on
this stack: `vitePreprocess()` (in the base `svelte.config.js`) + vite-plugin-svelte@7
strip the TS in each `.svelte` `<script>` before rolldown parses it, so
`ssr.noExternal: [/^@mssfoobar\//]` + `vite build` works. Verified against
`gis-web-sdk@2.0.0` on `vite@8.0.7` / `rolldown@1.0.0-rc.13`. An earlier
rolldown `[PARSE_ERROR]` on such `.svelte` was specific to an older SDK/toolchain
combination and no longer reproduces — no `vite@7` downgrade or SDK build-output
change is needed.

The one real gotcha: a **workspace-peer SDK must have its own `dist/` built before**
the consuming app's `vite build`. Its `exports` point at `dist/`, so an unbuilt peer
(e.g. `@mssfoobar/ui`, `@mssfoobar/logger`) surfaces a `[RESOLVE_ERROR]` — a
build-order issue, not a TS/Vite one. Turbo's build-order handles this in CI; if you
build by hand, run the SDKs' `package`/`build` first.
