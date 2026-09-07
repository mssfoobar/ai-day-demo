import { defineConfig, devices } from '@playwright/test';

// The dev server is what `.env.development`'s ORIGIN is pinned to — preview
// on port 4173 would 404 every OIDC callback because the cookie domain and
// redirect URI were minted against the dev origin. Reusing an existing
// `pnpm dev` keeps the test against the same surface developers see.
const BASE_URL = process.env.E2E_BASE_URL ?? 'http://127.0.0.1.nip.io:5173';

// Saved by the `setup` project (tests/e2e/auth.setup.ts) after a one-time
// Keycloak login, then reused by the feature project so `(private)` pages
// render authenticated.
const authFile = 'playwright/.auth/user.json';

export default defineConfig({
	webServer: {
		command: 'pnpm dev --port 5173 --strictPort',
		url: BASE_URL,
		reuseExistingServer: true,
		timeout: 120_000
	},
	use: { baseURL: BASE_URL },
	projects: [
		// Runs first; performs the OIDC login and writes storageState.
		{ name: 'setup', testMatch: /auth\.setup\.ts/ },
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'], storageState: authFile },
			dependencies: ['setup'],
			// Accept both `*.e2e.ts` (vitest convention shadow) and
			// `tests/e2e/**/*.spec.ts` (the more common Playwright layout) so feature
			// changes can pick either. The setup file is matched only by `setup`.
			testMatch: ['**/*.e2e.{ts,js}', '**/tests/e2e/**/*.spec.{ts,js}']
		}
	]
});
