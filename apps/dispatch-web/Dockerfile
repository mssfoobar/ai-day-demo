FROM node:22-alpine AS base
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
# corepack resolves the exact pnpm version from package.json's
# `packageManager` field on first use — no version to keep in sync here.
RUN corepack enable

# ---- Build Stage ----
# NOTE: build context is the repo root so the full monorepo workspace is
# available (required by `pnpm deploy --filter=dispatch-web`).
FROM base AS build
WORKDIR /app

# Fetch all packages to store (lockfile + .npmrc needed for good layer caching;
# root package.json carries the `packageManager` pin corepack activates).
COPY package.json pnpm-lock.yaml ./
COPY apps/dispatch-web/.npmrc ./
RUN --mount=type=cache,id=pnpm,target=/pnpm/store \
    --mount=type=secret,id=access_token \
    pnpm config set //npm.pkg.github.com/:_authToken=$(cat /run/secrets/access_token) && \
    pnpm fetch

# Install from store and build
COPY . .
RUN --mount=type=cache,id=pnpm,target=/pnpm/store \
    pnpm install --offline --frozen-lockfile
RUN pnpm --filter=dispatch-web run build

# Production deploy
RUN --mount=type=cache,id=pnpm,target=/pnpm/store \
    pnpm deploy --filter=dispatch-web --prod /prod/web

# pnpm deploy does not copy build artifacts — bring over SvelteKit output.
RUN cp -r /app/apps/dispatch-web/build /prod/web/build

# ---- Runtime Stage ----
FROM base
COPY --from=build /prod/web /prod/web
WORKDIR /prod/web

# This app has no authentication, so the scaffold's IAM_URL / IAM_CLIENT_ID / SDS_URL /
# PUBLIC_COOKIE_PREFIX / LOGIN_DESTINATION defaults were removed along with the code that
# read them. LOGIN_DESTINATION also pointed at /getting-started, a route that no longer
# exists. See apps/dispatch-web/AGENTS.md.
#
# What remains is exactly what src/hooks.server.ts reads. The two security headers must be
# defaulted here: unset, the container serves no CSP and no X-Frame-Options at all.
# `'self'` is double-quoted so the CSP keyword survives — bare `self` is parsed as a host
# name and matches nothing.
ENV FRAME_ANCESTORS="'self'"
ENV X_FRAME_OPTIONS="SAMEORIGIN"

# ORIGIN has NO default on purpose. hooks.server.ts treats it as required and exits(1)
# when unset, and adapter-node needs the real public origin — an empty default would only
# turn a clear startup failure into a subtly wrong one. Inject it at `docker run`.
ENV NODE_ENV="production"
ENV PORT=3000

EXPOSE 3000

USER node

CMD ["node", "build"]
