import { json } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async () => {
	return json({
		status: 'ok',
		uptime_seconds: process.uptime(),
		version: env.PUBLIC_STATIC_BUILD_VERSION ?? 'dev',
		memory: process.memoryUsage()
	});
};
