import { listUnits } from '$lib/aoh/dispatch/units.server';
import type { PageServerLoad } from './$types';

/**
 * Load the roster on the server.
 *
 * The browser never talks to the dispatch service: no CORS to configure, and no service
 * URL in the client bundle. All URL and transport wiring lives behind `listUnits()`, so
 * this function stays a seam rather than a client.
 *
 * A failure is returned as data, not thrown: an unreachable service is an expected
 * operational condition the console renders an explicit state for, and throwing here
 * would replace the whole page with the generic error surface.
 */
export const load: PageServerLoad = async () => {
	try {
		return { units: await listUnits(), unavailable: false };
	} catch {
		// The cause is deliberately not forwarded to the client — it is developer-facing.
		return { units: [], unavailable: true };
	}
};
