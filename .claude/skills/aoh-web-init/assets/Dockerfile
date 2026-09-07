FROM node:{NODE_VERSION}-alpine AS base
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
# corepack resolves the exact pnpm version from package.json's
# `packageManager` field on first use — no version to keep in sync here.
RUN corepack enable

# ---- Build Stage ----
# NOTE: build context is the repo root so the full monorepo workspace is
# available (required by `pnpm deploy --filter={APP_NAME}`).
FROM base AS build
WORKDIR /app

# Fetch all packages to store (lockfile + .npmrc needed for good layer caching;
# root package.json carries the `packageManager` pin corepack activates).
COPY package.json pnpm-lock.yaml ./
COPY apps/{APP_NAME}/.npmrc ./
RUN --mount=type=cache,id=pnpm,target=/pnpm/store \
    --mount=type=secret,id=access_token \
    pnpm config set //npm.pkg.github.com/:_authToken=$(cat /run/secrets/access_token) && \
    pnpm fetch

# Install from store and build
COPY . .
RUN --mount=type=cache,id=pnpm,target=/pnpm/store \
    pnpm install --offline --frozen-lockfile
RUN pnpm --filter={APP_NAME} run build

# Production deploy
RUN --mount=type=cache,id=pnpm,target=/pnpm/store \
    pnpm deploy --filter={APP_NAME} --prod /prod/web

# pnpm deploy does not copy build artifacts — bring over SvelteKit output.
RUN cp -r /app/apps/{APP_NAME}/build /prod/web/build

# ---- Runtime Stage ----
FROM base
COPY --from=build /prod/web /prod/web
WORKDIR /prod/web

ENV IAM_URL=""
ENV IAM_CLIENT_ID="{APP_NAME}"
ENV SDS_URL=""
ENV PUBLIC_DOMAIN=""
ENV PUBLIC_COOKIE_PREFIX=""
ENV LOGIN_DESTINATION="/getting-started"
ENV ORIGIN=""
ENV NODE_ENV="production"
ENV PORT=3000

EXPOSE 3000

USER node

CMD ["node", "build"]
