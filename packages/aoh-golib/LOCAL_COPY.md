# Local copy of `aoh-golib`

This directory is a verbatim snapshot of
`github.com/mssfoobar/ops-hub/packages/aoh-golib` at tag **`packages/aoh-golib/v0.3.0`**.

`apps/dispatch-svc/go.mod` points at it with a `replace` directive, so building the
service needs **no access to the private `ops-hub` repository** and no `GOPRIVATE`
setting. That is the whole reason it is here: the workshop should need one credential
(the npm token for `@mssfoobar/ui`), not two.

This is the same pattern the platform monorepo uses for its own services — the upstream
`package.json` describes it as "in-repo Go services consume it via go.work + a go.mod
replace directive". The `aoh-go-init` skill's guidance for *external* consumers is to
depend on the published tag instead; we deviate deliberately, for the reason above.

## Do not edit in place

Treat it as read-only. A fix that belongs in `aoh-golib` belongs upstream in `ops-hub`;
edit it here and the next refresh silently discards your change.

## Refreshing to a newer version

```sh
# from the repo root, with access to ops-hub
GOFLAGS=-mod=mod go mod download github.com/mssfoobar/ops-hub/packages/aoh-golib@vX.Y.Z
rm -rf packages/aoh-golib
cp -r "$(go env GOMODCACHE)/github.com/mssfoobar/ops-hub/packages/aoh-golib@vX.Y.Z" packages/aoh-golib
chmod -R u+w packages/aoh-golib
rm packages/aoh-golib/package.json      # see below
# then update the version noted at the top of this file and in apps/dispatch-svc/go.mod
```

Only two files differ from upstream:

- `package.json` is removed. Upstream ships one so *its* Turborepo can run `go test
  -race`; here it would make pnpm treat this directory as a workspace package and turbo
  would try to run those scripts (and `-race` needs a C toolchain on Windows).
- This file.

The `LICENSE` is upstream's and applies unchanged.
