import { defineConfig, devices } from '@playwright/test';

// Plain localhost: the scaffold used a `127.0.0.1.nip.io` origin so Keycloak cookie
// domains and OIDC redirect URIs would resolve. This app has no authentication, so the
// nip.io indirection buys nothing. Must match `.env.development`'s ORIGIN.
const BASE_URL = process.env.E2E_BASE_URL ?? 'http://localhost:5173';

export default defineConfig({
	webServer: {
		command: 'pnpm dev --port 5173 --strictPort',
		url: BASE_URL,
		reuseExistingServer: true,
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
