import { json } from '@sveltejs/kit';
import { StatusCodes } from 'http-status-codes';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ locals }) => {
	if (!locals.authResult?.success) {
		return json({ error: 'unauthorized' }, { status: StatusCodes.UNAUTHORIZED });
	}
	const claims = locals.authResult.claims;
	return json({
		data: {
			realm_roles: claims.realm_access?.roles ?? [],
			tenant_roles: claims.active_tenant?.roles ?? []
		}
	});
};
