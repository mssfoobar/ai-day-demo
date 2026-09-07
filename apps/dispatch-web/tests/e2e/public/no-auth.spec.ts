import { expect, test } from '@playwright/test';

/**
 * The no-authentication posture, from specs/dispatch-console
 * ("The application has no authentication").
 *
 * This is the regression that matters most: the scaffold this app came from performed
 * OIDC discovery in a top-level `await`, which made every route return HTTP 500 when no
 * identity provider was reachable. These assertions fail loudly if any of that returns.
 */

const AUTH_ROUTES = [
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

	test('no authentication route resolves', async ({ request }) => {
		for (const route of AUTH_ROUTES) {
			const response = await request.get(route, { maxRedirects: 0 });
			expect(response.status(), `${route} must not resolve`).toBe(404);
		}
	});

	test('loading the console sets no session cookie', async ({ page, context }) => {
		await page.goto('/units');
		const cookies = await context.cookies();
		const sessionCookies = cookies.filter((c) => /auth|session|token/i.test(c.name));
		expect(sessionCookies).toEqual([]);
	});
});
