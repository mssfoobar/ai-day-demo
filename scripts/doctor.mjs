#!/usr/bin/env node
/**
 * Checks this machine is ready to work offline: dependencies installed, the local
 * stack up, and the model reachable with replies that stream.
 *
 *   pnpm doctor
 */
import { request } from 'node:http';
import { existsSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const BASE = process.env.ANTHROPIC_BASE_URL ?? '';
const KEY = process.env.ANTHROPIC_AUTH_TOKEN ?? '';
const MODEL = process.env.ANTHROPIC_MODEL ?? '';
const TIMEOUT_MS = 30_000;

let failed = false;

function report(label, ok, detail) {
	console.log(`${ok ? 'ok  ' : 'FAIL'}  ${label.padEnd(8)} ${detail}`);
	if (!ok) failed = true;
}

/** Issues a request and records when each chunk of the response arrived. */
function send(target, { method = 'GET', body, headers = {} } = {}) {
	const url = new URL(target);
	return new Promise((resolve, reject) => {
		const started = Date.now();
		const chunkTimes = [];
		const req = request(
			{
				host: url.hostname,
				port: url.port || 80,
				method,
				path: url.pathname + url.search,
				headers
			},
			(res) => {
				res.on('data', () => chunkTimes.push(Date.now() - started));
				res.on('end', () => resolve({ status: res.statusCode, chunkTimes }));
			}
		);
		req.setTimeout(TIMEOUT_MS, () => req.destroy(new Error(`timed out after ${TIMEOUT_MS}ms`)));
		req.on('error', reject);
		if (body) req.write(body);
		req.end();
	});
}

async function check(label, fn) {
	try {
		const r = await fn();
		report(label, r.ok, r.ok ? r.detail : r.why);
	} catch (err) {
		report(label, false, err.message);
	}
}

if (!BASE) {
	console.error('ANTHROPIC_BASE_URL is not set. Is .claude/settings.json in place?');
	process.exit(1);
}

console.log(`model  ${MODEL || '(unset)'} at ${BASE}\n`);

await check('deps', async () => {
	const ok = existsSync(join(ROOT, 'node_modules'));
	return ok
		? { ok: true, detail: 'node_modules present' }
		: { ok: false, why: 'node_modules missing, run pnpm start while still online' };
});

await check('service', async () => {
	const r = await send('http://localhost:8081/healthz');
	return r.status === 200
		? { ok: true, detail: 'localhost:8081 responding' }
		: { ok: false, why: `HTTP ${r.status}` };
});

await check('model', async () => {
	const r = await send(`${BASE}/v1/models`, { headers: { authorization: `Bearer ${KEY}` } });
	return r.status === 200 ? { ok: true, detail: 'reachable' } : { ok: false, why: `HTTP ${r.status}` };
});

// A buffered reply arrives as one chunk, which looks exactly like a hung agent.
await check('stream', async () => {
	const payload = JSON.stringify({
		model: MODEL,
		max_tokens: 256,
		stream: true,
		messages: [{ role: 'user', content: 'Count slowly from 1 to 50.' }]
	});
	const r = await send(`${BASE}/v1/messages`, {
		method: 'POST',
		body: payload,
		headers: {
			'content-type': 'application/json',
			'anthropic-version': '2023-06-01',
			authorization: `Bearer ${KEY}`,
			'content-length': Buffer.byteLength(payload)
		}
	});
	if (r.status !== 200) return { ok: false, why: `HTTP ${r.status}` };
	const chunks = r.chunkTimes.length;
	if (chunks < 2) return { ok: false, why: 'arrived in one block, not streamed' };
	const spanMs = r.chunkTimes.at(-1) - r.chunkTimes[0];
	return { ok: true, detail: `${chunks} chunks over ${spanMs}ms` };
});

process.exit(failed ? 1 : 0);
