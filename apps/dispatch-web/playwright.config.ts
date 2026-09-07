import { defineConfig, devices } from '@playwright/test';

// Plain localhost: the scaffold used a `127.0.0.1.nip.io` origin so Keycloak cookie
// domains and OIDC redirect URIs would resolve. This app has no authentication, so the
// nip.io indirection buys nothing. Must match `.env.development`'s ORIGIN.
const BASE_URL = process.env.E2E_BASE_URL ?? 'http://localhost:5173';

export default defineConfig({
	webServer: {
		// NOT `pnpm dev --port … --strictPort`: the `dev` script ends in a pipe to
		// pino-pretty, so pnpm appends extra args to the LAST command in the pipeline —
		// pino-pretty swallows them and vite never sees them. That silently drops
		// `--strictPort`, so a busy 5173 sends vite to 5174 while Playwright polls 5173
		// until the 120s timeout. Invoke vite directly instead.
		command: 'pnpm exec env-cmd -f .env.development vite dev --port 5173 --strictPort',
		url: BASE_URL,
		// Local only. On CI a stale dev server on 5173 would be silently reused and the
		// suite would report green against code that is not under test.
		reuseExistingServer: !process.env.CI,
		timeout: 120_000
	},
	use: { baseURL: BASE_URL },
	projects: [
		// One project, no auth. The scaffold's `setup` project (a Keycloak PKCE login
		// writing storageState) and its dependent `chromium` project were removed with
		// the auth layer — there is no identity provider to log in to.
		{
			name: 'public-chromium',
			use: { ...devices['Desktop Chrome'] },
			testMatch: ['**/*.e2e.{ts,js}', '**/tests/e2e/**/*.spec.{ts,js}']
		}
	]
});
