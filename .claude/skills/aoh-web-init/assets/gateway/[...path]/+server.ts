import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { StatusCodes } from 'http-status-codes';
import { resolveModule } from '$root/gateway.config';
import { logger } from '@mssfoobar/logger';

type CacheEntry = { data: unknown; expiresAt: number };

const cache = new Map<string, CacheEntry>();

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function buildTargetUrl(resolvedUrl: string, searchParams: URLSearchParams): string {
	const upstream = new URLSearchParams(searchParams);
	upstream.delete('cache_ttl');
	upstream.delete('cache_bust');
	const query = upstream.toString();

	return query ? `${resolvedUrl}?${query}` : resolvedUrl;
}

function tryServeFromCache(key: string, bust: boolean): Response | null {
	if (bust) return null;
	const cached = cache.get(key);
	if (!cached) return null;
	if (Date.now() < cached.expiresAt) {
		logger.debug(`Gateway cache hit: ${key}`);
		return jsonResponse(cached.data);
	}
	cache.delete(key);
	logger.debug(`Gateway cache expired: ${key}`);
	return null;
}

async function buildHeaders(
	resolved: {
		config: {
			getHeaders?: (opts: {
				accessToken: string;
				requestHeaders: Headers;
			}) => Headers | Promise<Headers>;
		};
	},
	accessToken: string,
	requestHeaders: Headers
): Promise<Headers> {
	const headers = resolved.config.getHeaders
		? await resolved.config.getHeaders({ accessToken, requestHeaders })
		: new Headers([['Authorization', `Bearer ${accessToken}`]]);
	headers.set('Content-Type', requestHeaders.get('Content-Type') || 'application/json');
	return headers;
}

const proxy: RequestHandler = async ({ request, params, locals, fetch, url }) => {
	if (!locals.authResult.success || !locals.authResult.access_token)
		throw error(StatusCodes.UNAUTHORIZED);

	const accessToken = locals.authResult.access_token;
	const key = params.path;
	const ttl = Number(url.searchParams.get('cache_ttl'));
	const isGet = request.method === 'GET';

	if (isGet) {
		const cached = tryServeFromCache(key, url.searchParams.get('cache_bust') === 'true');
		if (cached) return cached;
	}

	const resolved = resolveModule(params.path);
	if (!resolved) {
		logger.error(`Gateway: unknown module in path "${params.path}"`);
		throw error(StatusCodes.BAD_REQUEST);
	}

	const targetUrl = buildTargetUrl(resolved.url, url.searchParams);
	logger.debug(`Gateway proxy: ${request.method} ${params.path} -> ${targetUrl}`);

	const headers = await buildHeaders(resolved, accessToken, request.headers);

	let res: Response;
	try {
		res = await fetch(targetUrl, {
			method: request.method,
			headers,
			body: request.body,
			duplex: 'half'
		} as RequestInit);
	} catch (err) {
		logger.error(err, `Gateway: failed to reach ${request.method} ${targetUrl}`);
		throw error(StatusCodes.BAD_GATEWAY, 'Bad Gateway');
	}

	if (!res.ok) {
		logger.error(
			{ status: res.status, statusText: res.statusText, url: targetUrl },
			'Gateway upstream non-ok'
		);
	}

	const isJson = (res.headers.get('Content-Type') || '').includes('application/json');

	if (isGet && isJson && res.ok) {
		const body = await res.json();
		if (ttl > 0) {
			cache.set(key, { data: body, expiresAt: Date.now() + ttl * 1000 });
			logger.debug(`Gateway cache set: ${key} (ttl: ${ttl}s)`);
		}
		return jsonResponse(body, res.status);
	}

	// Stream the body through — binary-safe. Preserve the upstream headers (so
	// Content-Disposition, Cache-Control, ETag, … survive) but DROP the two that break
	// a decompressed stream: Node's fetch already decompressed gzip/br, so a forwarded
	// Content-Encoding would make the browser double-decode and the upstream
	// Content-Length under-declares the decompressed body and truncates it. Add
	// `nosniff` + `sandbox allow-downloads` so a user-uploaded image/svg+xml served
	// same-origin can't run script on a top-level open, while still permitting file
	// downloads (harmless to <img> embedding and fetch() consumers).
	const headers = new Headers(res.headers);
	headers.delete('content-length');
	headers.delete('content-encoding');
	headers.set('X-Content-Type-Options', 'nosniff');
	headers.set('Content-Security-Policy', 'sandbox allow-downloads');
	return new Response(res.body, {
		status: res.status,
		statusText: res.statusText,
		headers
	});
};

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const PATCH = proxy;
export const DELETE = proxy;
export const OPTIONS = proxy;
export const HEAD = proxy;
