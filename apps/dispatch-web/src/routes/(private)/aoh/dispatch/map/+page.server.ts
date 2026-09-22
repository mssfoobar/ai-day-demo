import { redirect } from '@sveltejs/kit';
import { StatusCodes } from 'http-status-codes';
import { env as envPublic } from '$env/dynamic/public';
import { LOGIN_API } from '$lib/aoh/core/provider/auth/auth';
import { canWrite } from '$lib/aoh/dispatch/permissions';
import { bearerFrom, DispatchServiceError, listUnits } from '$lib/aoh/dispatch/units.server';
import type { PageServerLoad } from './$types';

/** Why the roster counts are not available. `ok` means they are. */
export type RosterState = 'ok' | 'unavailable' | 'denied';

/**
 * Load the roster — **unit** data, not entity data.
 *
 * The markers come from the SDK's RTUS subscription and nothing else (design.md D9), so
 * this load deliberately issues no geo-entity request: a second source of entity state
 * would need a dedup step at the prepend site to survive the fetch/SSE race, and the SDK
 * already replays current state on connect.
 *
 * What the roster *is* needed for is the counts. Deriving "3 units not shown" from the
 * entities on the map would make a pending projection look like a smaller roster; derived
 * from unit data it can never under-report (spec: "Un-positioned units are accounted for,
 * not hidden").
 */
export const load: PageServerLoad = async ({ locals }) => {
	const claims = locals.authResult.success ? locals.authResult.claims : undefined;
	const roles = claims?.active_tenant?.roles ?? [];
	const token = bearerFrom(locals);
	if (!token || !claims) redirect(StatusCodes.TEMPORARY_REDIRECT, LOGIN_API);

	// The RTUS subscription needs all four of these or it silently never connects, so they
	// come from the claims rather than from `data.user.id` / `data.user.tenantId`, which do
	// not exist on this base and would be `undefined`.
	const rtus = {
		sehUrl: envPublic.PUBLIC_RTUS_SEH_URL ?? '',
		mapName: 'gis',
		userId: typeof claims.sub === 'string' ? claims.sub : '',
		tenantId: claims.active_tenant?.tenant_id ?? ''
	};

	try {
		return {
			units: await listUnits(token),
			state: 'ok' as RosterState,
			canWrite: canWrite(roles),
			rtus
		};
	} catch (err) {
		if (err instanceof DispatchServiceError && err.isUnauthenticated) {
			redirect(StatusCodes.TEMPORARY_REDIRECT, LOGIN_API);
		}
		const state: RosterState =
			err instanceof DispatchServiceError && err.isForbidden ? 'denied' : 'unavailable';
		return { units: [], state, canWrite: canWrite(roles), rtus };
	}
};
