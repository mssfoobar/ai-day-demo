import { redirect } from '@sveltejs/kit';
import { StatusCodes } from 'http-status-codes';
import type { LayoutServerLoad } from './$types';

/**
 * Bare `/` lands on the dispatch console.
 *
 * The scaffold shipped this redirect pointing at the OIDC login flow. This app has no
 * authentication (see the openspec change `baseline-dispatch-console`, design.md D1/D2),
 * so that destination no longer exists and `/` goes straight to the console instead.
 *
 * `/` is kept as a redirect rather than hosting the console directly, so the route stays
 * free to become a real landing page later.
 */
export const load: LayoutServerLoad = async ({ url }) => {
	if (url.pathname === '/') {
		redirect(StatusCodes.TEMPORARY_REDIRECT, '/units');
	}
	return {};
};
