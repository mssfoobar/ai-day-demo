import { redirect } from '@sveltejs/kit';
import { StatusCodes } from 'http-status-codes';
import type { LayoutServerLoad } from './$types';

/**
 * Bare `/` lands on the dispatch console.
 *
 * The console now lives under the `(private)` group, so this redirect leads to the
 * sign-in flow for anyone without a session — the private layout's guard does that
 * part. It is deliberately a *replacement* for the baseline's `/` → `/units` redirect
 * rather than a deletion: the scaffold ships no root `+page.svelte`, so dropping it
 * would leave `/` a 404 instead of a sign-in redirect.
 *
 * `/` stays a redirect rather than hosting the console directly, so the route is free
 * to become a real landing page later.
 */
export const load: LayoutServerLoad = async ({ url }) => {
	if (url.pathname === '/') {
		redirect(StatusCodes.TEMPORARY_REDIRECT, '/aoh/dispatch/units');
	}
	return {};
};
