import { test as setup } from '@playwright/test';

// One-time OIDC PKCE login against the bundled `web` Keycloak client, run before
// the feature specs (they depend on the `setup` project — see playwright.config.ts)
// and saving the SDS session cookie so `(private)` pages render authenticated.
//
// - Point E2E_LANDING at any private route your app serves (defaults to the
//   scaffold's /getting-started landing).
// - Override credentials with E2E_USER / E2E_PASSWORD (default admin / P@ssw0rd,
//   the dev IAMS admin). Grant that user the app roles it needs via the
//   project-aas roles.yaml seed.
const authFile = 'playwright/.auth/user.json';

setup('authenticate', async ({ page }) => {
	const landing = process.env.E2E_LANDING ?? '/getting-started';
	await page.goto(landing); // private route → root layout redirects to Keycloak

	// Standard Keycloak login form ids.
	await page.fill('#username', process.env.E2E_USER ?? 'admin');
	await page.fill('#password', process.env.E2E_PASSWORD ?? 'P@ssw0rd');
	await page.click('#kc-login');

	// Wait until we're redirected back to the app (no longer on the Keycloak host),
	// i.e. the callback completed and the session cookie is set.
	await page.waitForURL((url) => !url.hostname.startsWith('iams-keycloak'), {
		timeout: 30_000
	});

	await page.context().storageState({ path: authFile });
});
