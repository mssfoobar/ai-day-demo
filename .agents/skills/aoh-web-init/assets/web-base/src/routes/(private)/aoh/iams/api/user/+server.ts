import { json } from '@sveltejs/kit';
import { StatusCodes } from 'http-status-codes';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ locals }) => {
	if (!locals.authResult?.success) {
		return json({ error: 'unauthorized' }, { status: StatusCodes.UNAUTHORIZED });
	}
	return json({ data: locals.authResult.claims });
};
