import { env as envPrivate } from '$env/dynamic/private';
import { redirect } from '@sveltejs/kit';
import { StatusCodes } from 'http-status-codes';
import { LOGIN_API } from '$lib/aoh/core/provider/auth/auth';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ url }) => {
	if (url.pathname === '/') {
		const destination = envPrivate.LOGIN_PAGE || LOGIN_API;
		redirect(StatusCodes.TEMPORARY_REDIRECT, destination);
	}
	return {};
};
