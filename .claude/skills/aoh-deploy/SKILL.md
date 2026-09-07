---
name: aoh-deploy
description: >
  AOH deploy-plane (GitOps) conventions and judgment — how a built image becomes a
  running workload in an environment, as distinct from the publish/release plane
  (that is `aoh-monorepo-release`). Use when deploying or promoting an AOH module,
  debugging why a `develop` push did not deploy, working under `gitops/` (kustomize
  bases + per-env overlays = the version vector), wiring a secret for a deployed
  component (External Secrets Operator + AWS Secrets Manager), altering Kargo
  Warehouses/Stages or the ArgoCD ApplicationSet, accessing the ArgoCD/Kargo UIs, or
  running the dev cutover off dev-infra (AOH-7711). Thin index: the authoritative
  detail lives in `gitops/README.md` and `gitops/SECRETS.md`. Trigger keywords:
  deploy, gitops, kustomize overlay, version vector, image pin, ArgoCD, Kargo,
  ApplicationSet, Warehouse, Stage, external-secret, ESO, "not deploying", cutover,
  dev-infra.
license: Proprietary
allowed-tools: Read Grep Glob
metadata:
  owner: AOH Platform Team
  status: v1
---

# AOH Deploy-plane (GitOps)

How a built image becomes a **running workload** in an environment. This is the
*deploy* plane. The *publish* plane (Changesets, marker-driven npm/tag releases) is
a separate concern — for that, use the **`aoh-monorepo-release`** skill. Don't
conflate them: queuing a package release does not deploy anything, and deploying
does not publish anything.

> **Reference, don't duplicate.** The authoritative, kept-current detail lives in
> the repo. Read these before acting; this skill only orients you and records the
> judgment calls.
>
> | Read | For |
> |---|---|
> | `gitops/README.md` | the deploy-plane model: bases vs overlays, the version vector, who writes image pins, cluster prerequisites |
> | `gitops/SECRETS.md` | the secret convention: ESO + AWS-SM, `<module>.<purpose>` naming, `creationPolicy: Owner` for modules vs **`Merge`** for chart-owned infra Secrets, the `check-secret-wiring.py` guard |
> | `gitops/argocd/README.md` | the AppProject + ApplicationSet + addons layout |
> | `gitops/kargo/` | Warehouses (per-module image subscriptions) + Stages (promotion) |

## The model in one paragraph

`gitops/` is the single source of truth for what runs in each environment (it
replaced the external `dev-infra`/`ar2-infra` manifests; tracked under AOH-7578). A
**base** (`gitops/base/<component>/`) holds env-agnostic K8s resources with the
image tag reduced to a `:latest` placeholder. An **env overlay**
(`gitops/envs/<env>/`) pins, via the kustomize `images:` transformer, exactly which
image tag/digest of each component runs there — **that list of pins is the version
vector**. Releasing a subset of modules to an env is a diff of those pins, nothing
else. ArgoCD syncs the overlay to the cluster; Kargo drives *which* pins get written.

## Who writes the image pins

| Env | Who writes pins | Trigger |
|---|---|---|
| `dev` | the `deploy-*` GitHub workflows **and/or Kargo dev Stages** | every `develop` push (integration sandbox — always newest) |
| `qa` | **Kargo** promotion (opens a PR; a human merges) | controlled promotion of a dev-tested, immutable image set |

Pins are machine-written onto dedicated **state branches** (`stage/dev`, `stage/qa`)
so they never re-trigger app CI. The ApplicationSet tracks `stage/<env>`, not
`develop`. `base/` and the overlay *structure* change only via reviewed PRs.

## "It's not deploying" — triage order

1. Did the **image build+push** succeed for the component? (the `deploy-*` workflow /
   ghcr tag exists)
2. Did a **Kargo Warehouse** discover the new image (Freight created), and did its
   **Stage** run and write the pin to `stage/<env>`?
3. Is the **ArgoCD Application** tracking `stage/<env>` (not `develop`), Synced, and
   does the target app carry the `kargo.akuity.io/authorized-stage` annotation Kargo
   needs to trigger a sync?
4. Is a **legacy dev-infra app** still owning the workload (auto-sync + selfHeal)? The
   dev cutover (**AOH-7711**) is reconcile-first and per-module: `gitops/` is still a
   subset of dev-infra, so a naive ApplicationSet re-point can drop running workloads.
   Reconcile the base/overlay gaps for a module *before* retiring its legacy app.

## Secrets (deploy-plane)

Read `gitops/SECRETS.md`. The short version: **no secret value lives in git** — only
wiring. Values live in the per-env AWS-SM blob (`aoh-dev-secret`, …) under nested
`<module>.<purpose>` properties; an `ExternalSecret` (External Secrets Operator,
`ClusterSecretStore` `cluster-secretstore-sm`) syncs them into a K8s Secret the
Deployment references.

- **App modules**: `gitops/base/<module>/external-secret.yaml`, `creationPolicy:
  Owner`, K8s key = purpose-only snake_case. CI's `check-secret-wiring.py` enforces
  that every Deployment `secretKeyRef` is provisioned.
- **Platform/infra components** (ArgoCD, Kargo) whose Secret is **chart-owned**: use
  `creationPolicy: Merge` to inject only the needed keys without clobbering the
  chart's (`server.secretkey`, `tls.*`). Example:
  `gitops/argocd/admin-external-secret.yaml` (admin password from `argocd.*` blob
  keys). To rotate, update the SM blob and bump `*.admin_password_mtime`.

## Admin UIs

ArgoCD and Kargo are exposed on the shared ALB behind the `*.dev.agilopshub.com`
wildcard cert:

- **ArgoCD** — `https://cd-admin.dev.agilopshub.com`
- **Kargo** — `https://kargo.dev.agilopshub.com`

Admin credentials are in AWS-SM under `argocd.*` / `kargo.*` in `aoh-dev-secret`.
ArgoCD's password is ESO-managed (above); Kargo's is helm-managed (its bcrypt is a
chart value), with the plaintext mirrored into the blob for reference.
