import { log as logger } from '$lib/aoh/core/logger/Logger';
import { LOGIN_API } from '$lib/aoh/core/provider/auth/auth';
import { redirect } from '@sveltejs/kit';
import { StatusCodes } from 'http-status-codes';
import { CONTEXT_COOKIE_NAME } from '$lib/aoh/core/constants';
import { env as envPrivate } from '$env/dynamic/private';
import type { LayoutServerLoad } from './$types';

const log = logger.child({ src: new URL(import.meta.url).pathname });

/**
 * The guard for every page under `(private)`.
 *
 * This — not the auth handle in `hooks.server.ts` — is what redirects an unauthenticated
 * visitor to sign in. Keeping the redirect here rather than in the handle is what leaves
 * `/livez`, `/readyz` and the `(public)/aoh/api/auth/*` routes reachable with no session.
 *
 * It returns the operator's **claims**, never the access token: anything returned from a
 * `load` is serialised into the page payload and would reach the browser, which is the
 * one thing the SDS posture exists to prevent.
 */
export const load: LayoutServerLoad = async ({ locals, cookies, url }) => {
	const authResult = locals.authResult;
	const context = cookies.get(CONTEXT_COOKIE_NAME);

	if (authResult.success) {
		return {
			user: authResult.claims,
			context
		};
	}

	log.debug('user is not authenticated');
	// Avoid redirect loops by checking if we're already dealing with auth-related URLs.
	if (url.pathname.startsWith('/aoh/api/auth/')) {
		log.warn('attempted to redirect from auth endpoint, potential redirect loop');
		throw new Error('Authentication required');
	}

	// Remember where they were heading, so the callback can put them back there.
	const currentPath = url.pathname + url.search;
	cookies.set('aoh_redirect_after_auth', currentPath, {
		path: '/',
		maxAge: 60 * 10, // 10 minutes
		httpOnly: true,
		sameSite: 'lax',
		secure: (envPrivate.ORIGIN ?? '').toLowerCase().startsWith('https://')
	});
	redirect(StatusCodes.TEMPORARY_REDIRECT, LOGIN_API);
};
