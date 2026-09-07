import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test } from '@playwright/test';

/**
 * The no-authentication posture, from specs/dispatch-console
 * ("The application has no authentication").
 *
 * This is the regression that matters most: the scaffold this app came from performed
 * OIDC discovery in a top-level `await`, which made every route return HTTP 500 when no
 * identity provider was reachable. These assertions fail loudly if any of that returns.
 */

const ROUTES_DIR = fileURLToPath(new URL('../../../src/routes', import.meta.url));
const SRC_DIR = fileURLToPath(new URL('../../../src', import.meta.url));

function walk(dir: string): string[] {
	return readdirSync(dir).flatMap((entry) => {
		const full = join(dir, entry);
		return statSync(full).isDirectory() ? walk(full) : [full];
	});
}

/** Every route directory SvelteKit will serve, as URL-ish paths. */
function routePaths(): string[] {
	return walk(ROUTES_DIR)
		.filter((f) => /[\\/]\+(page\.svelte|server\.ts)$/.test(f))
		.map((f) => relative(ROUTES_DIR, f).split(sep).slice(0, -1).join('/'))
		.map((p) => '/' + p.replace(/\([^)]*\)\/?/g, '')) // strip route groups
		.map((p) => p.replace(/\/+$/, '') || '/');
}

const AUTH_ROUTE_PATTERN = /login|logout|callback|refresh|oauth|oidc|auth|gateway|iams|session/i;

const KNOWN_GONE = [
	'/aoh/api/auth/login',
	'/aoh/api/auth/logout',
	'/aoh/api/auth/callback',
	'/aoh/api/auth/refresh',
	'/aoh/api/auth/context',
	'/aoh/gateway/anything',
	'/getting-started'
];

test.describe('no authentication', () => {
	test('serves the console with no identity provider running', async ({ request }) => {
		// The whole suite runs with no containers; a 200 here is the proof.
		const response = await request.get('/units');
		expect(response.status()).toBe(200);
	});

	test('health endpoints survive the removal of the route groups', async ({ request }) => {
		expect((await request.get('/livez')).status()).toBe(200);
		expect((await request.get('/readyz')).status()).toBe(200);
	});

	test('the route tree contains no authentication route', () => {
		// Enumerated from the filesystem rather than an allowlist, so a reintroduced
		// /login or a gateway mounted at a new path is caught rather than skipped.
		const offenders = routePaths().filter((p) => AUTH_ROUTE_PATTERN.test(p));
		expect(offenders, 'no route may look like an auth or gateway route').toEqual([]);

		// Sanity-check the enumeration itself: if it silently found nothing, the
		// assertion above would be vacuous.
		expect(routePaths()).toContain('/units');
	});

	test('the known auth routes no longer resolve', async ({ request }) => {
		for (const route of KNOWN_GONE) {
			const response = await request.get(route, { maxRedirects: 0 });
			expect(response.status(), `${route} must not resolve`).toBe(404);
		}
	});

	test('no server-side module performs OIDC discovery or reaches an identity provider', () => {
		// The failure this guards is server-side, so the browser cannot observe it.
		// Assert it against the source instead.
		const sources = walk(SRC_DIR).filter((f) => /\.(ts|js|svelte)$/.test(f));
		const offenders = sources.filter((f) => {
			const body = readFileSync(f, 'utf8').replace(/\/\*[\s\S]*?\*\/|\/\/.*$/gm, '');
			return /openid-client|discovery\(|SdsClient|IAM_URL|authenticate\(/.test(body);
		});
		expect(
			offenders.map((f) => relative(SRC_DIR, f)),
			'no source file may reference the removed auth stack'
		).toEqual([]);
	});

	test('loading the console sets no session cookie', async ({ page, context }) => {
		await page.goto('/units');
		const cookies = await context.cookies();
		const sessionCookies = cookies.filter((c) => /auth|session|token/i.test(c.name));
		expect(sessionCookies).toEqual([]);
	});
});
