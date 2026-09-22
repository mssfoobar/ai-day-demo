#!/usr/bin/env node
/**
 * End-to-end smoke test, against a running stack.
 *
 *   pnpm e2e                 # or: node scripts/e2e-smoke.mjs
 *
 * Exercises the three things unit tests cannot: that IAMS really issues tokens carrying
 * the application roles, that `dispatch-svc` really enforces them and really scopes to a
 * tenant, and that a unit write really reaches `gis-service` as a geo-entity.
 *
 * No manual step, and no fixtures to load first: the roster it asserts against is seeded
 * by the service itself on the first dispatcher request, which this script makes.
 *
 * Written in Node rather than a shell script or a Makefile because this repo is developed
 * on native Windows as well as POSIX (see AGENTS.md / aoh-scripting-conventions).
 *
 * Env comes from `compose/.env` — a bare `node scripts/e2e-smoke.mjs` loads no app's
 * `.env`, so that file is the documented source. Anything in the real environment wins.
 */
import { spawnSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { setTimeout as sleep } from 'node:timers/promises';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const COMPOSE_FILE = join(ROOT, 'compose', 'compose.yml');

/* -------------------------------------------------------------------------- */
/*                                   CONFIG                                   */
/* -------------------------------------------------------------------------- */

function composeEnv() {
	try {
		const body = readFileSync(join(ROOT, 'compose', '.env'), 'utf8');
		return Object.fromEntries(
			body
				.split('\n')
				.map((line) => line.trim())
				.filter((line) => line && !line.startsWith('#') && line.includes('='))
				.map((line) => {
					const i = line.indexOf('=');
					return [line.slice(0, i).trim(), line.slice(i + 1).trim()];
				})
		);
	} catch {
		return {};
	}
}

const FILE_ENV = composeEnv();
/** The real environment wins, so CI can point this at another stack. */
const cfg = (key, fallback) => process.env[key] ?? FILE_ENV[key] ?? fallback;

const DEV_DOMAIN = cfg('DEV_DOMAIN', '127.0.0.1.nip.io');
const IAM_URL = cfg('IAM_URL', `http://iams-keycloak.${DEV_DOMAIN}/realms/aoh`);
const IAM_CLIENT_ID = cfg('IAM_CLIENT_ID', 'web');
const DISPATCHER_USER = cfg('DEV_USER', 'admin');
const DISPATCHER_PASSWORD = cfg('DEV_PASSWORD', 'P@ssw0rd');
const VIEWER_USER = cfg('VIEWER_USER', 'viewer');
// The viewer shares DEV_PASSWORD — realm-import.json seeds it with that credential.
const VIEWER_PASSWORD = cfg('VIEWER_PASSWORD', DISPATCHER_PASSWORD);
const SVC_URL = cfg('DISPATCH_SVC_URL', 'http://localhost:8081').replace(/\/+$/, '');
const GIS_URL = cfg('GIS_URL', `http://gis.${DEV_DOMAIN}`).replace(/\/+$/, '');
const WEB_URL = cfg('WEB_URL', `http://${DEV_DOMAIN}:5173`).replace(/\/+$/, '');

/** The unit this run creates and cleans up. Suffixed so two runs never collide. */
const PROBE_CODE = `FU-E2E${String(Date.now()).slice(-4)}`;
const OTHER_TENANT = 'e2e-other-tenant';
/** Shares a code with a unit in the caller's tenant; only exists in the other tenant. */
const SHARED_CODE = 'FU-101';
const OTHER_ONLY_CODE = 'FU-E2E-OTHER';

/* -------------------------------------------------------------------------- */
/*                                 ASSERTIONS                                 */
/* -------------------------------------------------------------------------- */

let checks = 0;
const failures = [];

function pass(name) {
	checks++;
	process.stdout.write(`  \x1b[32m✓\x1b[0m ${name}\n`);
}

function fail(name, detail) {
	checks++;
	failures.push(name);
	process.stdout.write(`  \x1b[31m✗\x1b[0m ${name}\n      ${detail}\n`);
}

function check(name, condition, detail = '') {
	if (condition) pass(name);
	else fail(name, detail);
}

function eq(name, actual, expected) {
	check(name, actual === expected, `expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
}

function section(title) {
	process.stdout.write(`\n\x1b[1m${title}\x1b[0m\n`);
}

/* -------------------------------------------------------------------------- */
/*                                   HELPERS                                  */
/* -------------------------------------------------------------------------- */

async function token(username, password) {
	const body = new URLSearchParams({
		grant_type: 'password',
		client_id: IAM_CLIENT_ID,
		// NOT optional. AOH services validate through Keycloak's userinfo endpoint, which
		// 403s on a token issued without it — surfacing as an opaque 401 from the service.
		scope: 'openid',
		username,
		password
	});
	const response = await fetch(`${IAM_URL}/protocol/openid-connect/token`, {
		method: 'POST',
		headers: { 'content-type': 'application/x-www-form-urlencoded' },
		body
	});
	if (!response.ok) {
		throw new Error(`token for ${username}: HTTP ${response.status} ${await response.text()}`);
	}
	const json = await response.json();
	if (!json.access_token) throw new Error(`token for ${username}: no access_token in response`);
	return json.access_token;
}

function claims(jwt) {
	return JSON.parse(Buffer.from(jwt.split('.')[1], 'base64url').toString('utf8'));
}

function svc(path, { method = 'GET', bearer, body } = {}) {
	return fetch(`${SVC_URL}${path}`, {
		method,
		headers: {
			accept: 'application/json',
			...(bearer ? { authorization: `Bearer ${bearer}` } : {}),
			...(body !== undefined ? { 'content-type': 'application/json' } : {})
		},
		body: body !== undefined ? JSON.stringify(body) : undefined
	});
}

function gis(path, bearer) {
	return fetch(`${GIS_URL}${path}`, { headers: { authorization: `Bearer ${bearer}` } });
}

async function unitsOf(bearer) {
	const response = await svc('/v1/units', { bearer });
	if (!response.ok) throw new Error(`GET /v1/units: HTTP ${response.status}`);
	return (await response.json()).data ?? [];
}

function newUnit(code, position) {
	return {
		unit_code: code,
		call_sign: 'E2E probe',
		status: 'Available',
		unit_type: 'Ambulance',
		station: 'E2E Station',
		sector: 'E2E Sector',
		radio_channel: 'TAC-E2E',
		shift: 'Day',
		capabilities: ['ALS'],
		...(position ? { position } : {})
	};
}

/**
 * Poll until `predicate` holds, because the projection is delivered after the write
 * commits — asserting immediately would be a race, and a fixed sleep would be a slower
 * race.
 */
async function eventually(describe, predicate, { attempts = 30, delay = 500 } = {}) {
	let last;
	for (let i = 0; i < attempts; i++) {
		try {
			last = await predicate();
			if (last) return last;
		} catch (err) {
			last = err;
		}
		await sleep(delay);
	}
	throw new Error(`${describe}: still false after ${(attempts * delay) / 1000}s (${last})`);
}

/** Compose runtime, for the one step that needs raw SQL. Podman preferred, docker accepted. */
function composeRunner() {
	for (const candidate of ['podman', 'docker']) {
		// No `shell: true`: passing an args array through a shell triggers Node's DEP0190
		// and quotes differently per platform. The binary resolves fine without one.
		const probe = spawnSync(candidate, ['compose', 'version'], { stdio: 'ignore' });
		if (probe.status === 0) return candidate;
	}
	return null;
}

/**
 * Run SQL in the dispatch database.
 *
 * The statement goes in on **stdin**, not as `-tAc <sql>`, and the runner is spawned with
 * no shell. SQL is full of quotes, commas and newlines; handing it to a shell means it is
 * quoted correctly on exactly one platform, and this repo is developed on Windows as well
 * as POSIX. stdin sidesteps the question.
 */
function psql(sql) {
	const runner = composeRunner();
	if (!runner) throw new Error('neither podman nor docker is on PATH');
	const result = spawnSync(
		runner,
		['compose', '-f', COMPOSE_FILE, 'exec', '-T', 'postgres', 'psql', '-U', 'dispatch', '-d', 'dispatch', '-tA', '-v', 'ON_ERROR_STOP=1'],
		{ input: sql, encoding: 'utf8' }
	);
	if (result.error) throw new Error(`could not run ${runner}: ${result.error.message}`);
	if (result.status !== 0) {
		throw new Error(`psql failed: ${(result.stderr || result.stdout || '').trim()}`);
	}
	return (result.stdout ?? '').trim();
}

/* -------------------------------------------------------------------------- */
/*                                    TESTS                                   */
/* -------------------------------------------------------------------------- */

async function authentication() {
	section('Authentication');

	const dispatcher = await token(DISPATCHER_USER, DISPATCHER_PASSWORD);
	const viewer = await token(VIEWER_USER, VIEWER_PASSWORD);
	pass('both seeded accounts complete the password grant with scope=openid');

	const d = claims(dispatcher).active_tenant ?? {};
	const v = claims(viewer).active_tenant ?? {};
	check('dispatcher carries dispatch-dispatcher', (d.roles ?? []).includes('dispatch-dispatcher'), JSON.stringify(d.roles));
	check('viewer carries dispatch-viewer', (v.roles ?? []).includes('dispatch-viewer'), JSON.stringify(v.roles));
	check(
		'viewer does NOT carry dispatch-dispatcher',
		!(v.roles ?? []).includes('dispatch-dispatcher'),
		JSON.stringify(v.roles)
	);
	check('both are in the same tenant', Boolean(d.tenant_id) && d.tenant_id === v.tenant_id, `${d.tenant_id} vs ${v.tenant_id}`);

	// A missing credential is the easy case; these are the two that a hand-rolled
	// validator typically lets through.
	eq('no token is 401', (await svc('/v1/units')).status, 401);
	eq('a malformed token is 401', (await svc('/v1/units', { bearer: 'not-a-jwt' })).status, 401);

	// `exp` in the past. Unsigned and unverifiable, which is the point: whatever the
	// service does with it, the answer must be 401 and never 200.
	const expired = [
		Buffer.from(JSON.stringify({ alg: 'none', typ: 'JWT' })).toString('base64url'),
		Buffer.from(
			JSON.stringify({ sub: 'expired', exp: Math.floor(Date.now() / 1000) - 3600 })
		).toString('base64url'),
		''
	].join('.');
	eq('an expired token is 401', (await svc('/v1/units', { bearer: expired })).status, 401);

	// Health probes are outside the authenticated surface — Kubernetes has no token.
	eq('service /livez is open', (await fetch(`${SVC_URL}/livez`)).status, 200);
	eq('service /readyz is open', (await fetch(`${SVC_URL}/readyz`)).status, 200);

	// Every write route is behind the bearer.
	for (const [method, path] of [
		['POST', `/v1/units/${SHARED_CODE}/assignment`],
		['DELETE', `/v1/units/${SHARED_CODE}/assignment`]
	]) {
		eq(`${method} ${path} is 401 unauthenticated`, (await svc(path, { method })).status, 401);
	}

	return { dispatcher, viewer };
}

async function seedAndRoster(dispatcher) {
	section('Seeded roster');

	const units = await unitsOf(dispatcher);
	check('the dispatcher sees a seeded roster', units.length >= 5, `got ${units.length} units`);

	const positioned = units.filter((u) => u.position);
	check('some seeded units carry a position', positioned.length > 0, `${positioned.length} positioned`);
	check(
		'at least one seeded unit carries none',
		units.some((u) => !u.position),
		'every unit has a position; the "not shown" state is unreachable'
	);
	check(
		'positions are in range',
		positioned.every((u) => u.position.lon >= -180 && u.position.lon <= 180 && u.position.lat >= -90 && u.position.lat <= 90),
		'a seeded position is out of range'
	);
	check(
		'a seeded unit carries crew',
		units.some((u) => (u.crew ?? []).length > 0),
		'no seeded unit has crew'
	);
	check(
		'a seeded unit carries an assignment',
		units.some((u) => u.assignment),
		'no seeded unit has an assignment'
	);
	const statuses = new Set(units.map((u) => u.status));
	check(
		'the status vocabulary is covered',
		['Available', 'En route', 'Idle'].every((s) => statuses.has(s)),
		[...statuses].join(', ')
	);

	// Seeding is idempotent: a second request must not add or bump anything.
	const again = await unitsOf(dispatcher);
	eq('a second request seeds nothing further', again.length, units.length);

	return units;
}

async function seededProjections(dispatcher, units) {
	section('Seeded projections reached GIS');

	// This is what proves the lazy seed's projections were DELIVERED, not just enqueued.
	for (const unit of units.filter((u) => u.position)) {
		await eventually(`geo-entity for seeded ${unit.unit_code}`, async () => {
			const response = await gis(`/geoentity/entity_id/${unit.unit_code}`, dispatcher);
			return response.ok;
		});
	}
	pass(`every seeded positioned unit has a geo-entity (${units.filter((u) => u.position).length})`);

	// And an un-positioned one has none — a delete intent that is a no-op in GIS.
	const unpositioned = units.find((u) => !u.position);
	if (unpositioned) {
		const response = await gis(`/geoentity/entity_id/${unpositioned.unit_code}`, dispatcher);
		eq(`un-positioned ${unpositioned.unit_code} has no geo-entity`, response.status, 404);
	}
}

async function roleGating(dispatcher, viewer) {
	section('Role gating');

	const body = newUnit(PROBE_CODE, { lon: 103.85, lat: 1.29 });

	eq('a viewer may read', (await svc('/v1/units', { bearer: viewer })).status, 200);
	eq('a viewer may not create', (await svc('/v1/units', { method: 'POST', bearer: viewer, body })).status, 403);
	eq(
		'a viewer may not replace',
		(await svc(`/v1/units/${SHARED_CODE}`, { method: 'PUT', bearer: viewer, body: { ...body, occ_lock: 0 } })).status,
		403
	);
	eq(
		'a viewer may not delete',
		(await svc(`/v1/units/${SHARED_CODE}?occ_lock=0`, { method: 'DELETE', bearer: viewer })).status,
		403
	);

	const created = await svc('/v1/units', { method: 'POST', bearer: dispatcher, body });
	eq('a dispatcher may create', created.status, 201);
	return (await created.json()).data;
}

async function projectionLifecycle(dispatcher, unit) {
	section('Projection lifecycle');

	const entity = await eventually(`geo-entity for ${PROBE_CODE}`, async () => {
		const response = await gis(`/geoentity/entity_id/${PROBE_CODE}`, dispatcher);
		return response.ok ? (await response.json()).data : false;
	});
	eq('entity_type is track', entity.entity_type, 'track');
	eq('properties.kind is field-unit', entity.geojson.properties.kind, 'field-unit');
	check(
		'coordinates match the unit',
		entity.geojson.geometry.coordinates[0] === 103.85 && entity.geojson.geometry.coordinates[1] === 1.29,
		JSON.stringify(entity.geojson.geometry.coordinates)
	);

	// Move it: the entity must follow.
	const moved = await svc(`/v1/units/${PROBE_CODE}`, {
		method: 'PUT',
		bearer: dispatcher,
		body: { ...newUnit(PROBE_CODE, { lon: 104.05, lat: 1.42 }), occ_lock: unit.occ_lock }
	});
	eq('a dispatcher may move a unit', moved.status, 200);
	const movedUnit = (await moved.json()).data;

	await eventually('the entity follows the unit', async () => {
		const response = await gis(`/geoentity/entity_id/${PROBE_CODE}`, dispatcher);
		if (!response.ok) return false;
		const [lon, lat] = (await response.json()).data.geojson.geometry.coordinates;
		return lon === 104.05 && lat === 1.42;
	});
	pass('the entity follows the unit');

	// Clear the position: the entity must be REMOVED, not left at the last known spot.
	const cleared = await svc(`/v1/units/${PROBE_CODE}`, {
		method: 'PUT',
		bearer: dispatcher,
		body: { ...newUnit(PROBE_CODE), occ_lock: movedUnit.occ_lock }
	});
	eq('a replace with no position succeeds', cleared.status, 200);
	const clearedUnit = (await cleared.json()).data;
	check('the unit now omits `position`', !('position' in clearedUnit), JSON.stringify(clearedUnit.position));

	await eventually('clearing a position removes the entity', async () => {
		const response = await gis(`/geoentity/entity_id/${PROBE_CODE}`, dispatcher);
		return response.status === 404;
	});
	pass('clearing a position removes the entity');

	eq(
		'a dispatcher may delete',
		(await svc(`/v1/units/${PROBE_CODE}?occ_lock=${clearedUnit.occ_lock}`, { method: 'DELETE', bearer: dispatcher })).status,
		204
	);
	eq('the deleted unit is gone', (await svc(`/v1/units/${PROBE_CODE}`, { bearer: dispatcher })).status, 404);
}

async function tenantIsolation(dispatcher) {
	section('Tenant isolation');

	/*
	 * The stack runs one tenant, so the other tenant's rows are inserted directly. The
	 * predicate under test is the `WHERE tenant_id = $n` clause, not Keycloak — standing
	 * up a second identity would test something else and cost far more.
	 *
	 * Two rows, because one cannot satisfy all four assertions: one reuses a code the
	 * caller's tenant already has (for the list and read checks) and one uses a code the
	 * caller's tenant does not (so "exists only under the other tenant" and "the same code
	 * may exist in two tenants" both have a subject).
	 */
	psql(`
		INSERT INTO dispatch.unit (unit_code, call_sign, status, unit_type, station, sector, radio_channel, shift, tenant_id, created_by, updated_by) VALUES
		  ('${SHARED_CODE}','Other-1','Idle','Ambulance','Elsewhere','Sector 9','TAC-9','Day','${OTHER_TENANT}','e2e','e2e'),
		  ('${OTHER_ONLY_CODE}','Other-2','Idle','Ambulance','Elsewhere','Sector 9','TAC-9','Day','${OTHER_TENANT}','e2e','e2e')
	`);

	try {
		const units = await unitsOf(dispatcher);
		check(
			"another tenant's units are absent from the list",
			!units.some((u) => u.call_sign === 'Other-1' || u.unit_code === OTHER_ONLY_CODE),
			units.map((u) => `${u.unit_code}/${u.call_sign}`).join(' ')
		);

		const own = await svc(`/v1/units/${SHARED_CODE}`, { bearer: dispatcher });
		eq('a shared code resolves to the caller\'s own unit', own.status, 200);
		check(
			"...and never the other tenant's",
			(await own.json()).data.call_sign !== 'Other-1',
			'the other tenant\'s row was returned'
		);

		eq(
			"another tenant's unit is not found",
			(await svc(`/v1/units/${OTHER_ONLY_CODE}`, { bearer: dispatcher })).status,
			404
		);
		eq(
			"another tenant's unit cannot be replaced",
			(await svc(`/v1/units/${OTHER_ONLY_CODE}`, {
				method: 'PUT',
				bearer: dispatcher,
				body: { ...newUnit(OTHER_ONLY_CODE), call_sign: 'Hijacked', occ_lock: 0 }
			})).status,
			404
		);
		eq(
			"another tenant's unit cannot be deleted",
			(await svc(`/v1/units/${OTHER_ONLY_CODE}?occ_lock=0`, { method: 'DELETE', bearer: dispatcher })).status,
			404
		);

		const row = psql(
			`SELECT call_sign || '|' || occ_lock FROM dispatch.unit WHERE unit_code = '${OTHER_ONLY_CODE}' AND tenant_id = '${OTHER_TENANT}'`
		);
		eq("the other tenant's row is byte-identical", row, 'Other-2|0');

		// The same code may exist in two tenants: unit_code is unique per tenant, not
		// globally, and GIS is itself tenant-partitioned.
		const created = await svc('/v1/units', {
			method: 'POST',
			bearer: dispatcher,
			body: newUnit(OTHER_ONLY_CODE)
		});
		eq("a code that exists only in another tenant is free to create", created.status, 201);
		const mine = (await created.json()).data;
		eq(
			'cleaning up the created unit',
			(await svc(`/v1/units/${OTHER_ONLY_CODE}?occ_lock=${mine.occ_lock}`, { method: 'DELETE', bearer: dispatcher })).status,
			204
		);
	} finally {
		// Torn down unconditionally, so the script stays re-runnable after a failure.
		psql(`DELETE FROM dispatch.unit WHERE tenant_id = '${OTHER_TENANT}'`);
	}
}

async function sessionLifecycle() {
	section('Console session');

	/*
	 * A cookie jar just big enough to drive the OIDC flow the way a browser does — and,
	 * crucially, one that records WHO set each cookie. The flow crosses two origins, and
	 * Keycloak sets its own SSO cookies (KEYCLOAK_IDENTITY among them) on its own host.
	 * Those are the identity provider's business; the assertion below is about what the
	 * *console* puts in the browser, so a flat jar would fail on someone else's cookie.
	 */
	const jar = new Map(); // name -> { value, setBy }
	const consoleHost = new URL(WEB_URL).hostname;

	const cookieHeader = () => [...jar].map(([name, c]) => `${name}=${c.value}`).join('; ');
	const remember = (response, url) => {
		const setBy = new URL(url).hostname;
		for (const raw of response.headers.getSetCookie?.() ?? []) {
			const [pair] = raw.split(';');
			const i = pair.indexOf('=');
			const name = pair.slice(0, i).trim();
			const value = pair.slice(i + 1).trim();
			if (value === '' || /Max-Age=0/i.test(raw)) jar.delete(name);
			else jar.set(name, { value, setBy });
		}
	};

	const hop = async (url, init = {}) => {
		let current = url;
		for (let i = 0; i < 12; i++) {
			const response = await fetch(current, {
				...init,
				headers: { ...(init.headers ?? {}), cookie: cookieHeader() },
				redirect: 'manual'
			});
			remember(response, current);
			const location = response.headers.get('location');
			if (!location || response.status < 300 || response.status >= 400) return response;
			current = new URL(location, current).toString();
			init = {}; // only the first hop carries a method or a body
		}
		throw new Error(`too many redirects from ${url}`);
	};

	const loginPage = await hop(`${WEB_URL}/aoh/api/auth/login`);
	const html = await loginPage.text();
	const action = html.match(/action="([^"]+)"/)?.[1]?.replace(/&amp;/g, '&');
	if (!action) throw new Error('no Keycloak login form found — is the console running?');

	await hop(action, {
		method: 'POST',
		headers: { 'content-type': 'application/x-www-form-urlencoded' },
		body: new URLSearchParams({ username: DISPATCHER_USER, password: DISPATCHER_PASSWORD }).toString()
	});

	const sessionId = jar.get('web_auth_session_id')?.value;
	check('signing in sets a session-id cookie', Boolean(sessionId), 'no web_auth_session_id');

	// Only the console's own cookies. Keycloak's SSO cookies live on the IdP's host and
	// are not this app's to police.
	const consoleCookies = [...jar].filter(([, c]) => c.setBy === consoleHost);
	check(
		'the console sets no token-shaped cookie',
		!consoleCookies.some(([, c]) => /^eyJ[A-Za-z0-9_-]{20,}\./.test(c.value)),
		consoleCookies.map(([name]) => name).join(', ')
	);
	check(
		'the session id is all the console stores',
		consoleCookies.every(([name]) => name === 'web_auth_session_id'),
		consoleCookies.map(([name]) => name).join(', ')
	);

	const console_ = await hop(`${WEB_URL}/aoh/dispatch/units`);
	eq('the console renders for a signed-in operator', console_.status, 200);
	const body = await console_.text();
	check(
		'no JWT is serialised into the page payload',
		!/eyJ[A-Za-z0-9_-]{20,}\./.test(body),
		'a JWT-shaped string is present in the HTML'
	);

	// Capture the session id, sign out, then replay it.
	await hop(`${WEB_URL}/aoh/api/auth/logout`);

	const replayed = await fetch(`${WEB_URL}/aoh/dispatch/units`, {
		headers: { cookie: `web_auth_session_id=${sessionId}` },
		redirect: 'manual'
	});
	const location = replayed.headers.get('location') ?? '';
	check(
		'a replayed session id is dead after sign-out',
		replayed.status >= 300 && replayed.status < 400 && location.includes('/aoh/api/auth/login'),
		`HTTP ${replayed.status} -> ${location || '(no redirect)'}`
	);

	// Health probes stay open on the console too.
	eq('console /livez is open', (await fetch(`${WEB_URL}/livez`)).status, 200);
	eq('console /readyz is open', (await fetch(`${WEB_URL}/readyz`)).status, 200);
}

/* -------------------------------------------------------------------------- */
/*                                    MAIN                                    */
/* -------------------------------------------------------------------------- */

process.stdout.write(`e2e-smoke — ${SVC_URL}, ${GIS_URL}, ${WEB_URL}\n`);

try {
	const { dispatcher, viewer } = await authentication();
	const units = await seedAndRoster(dispatcher);
	await seededProjections(dispatcher, units);
	const created = await roleGating(dispatcher, viewer);
	await projectionLifecycle(dispatcher, created);
	await tenantIsolation(dispatcher);
	await sessionLifecycle();
} catch (err) {
	process.stdout.write(`\n\x1b[31mABORTED\x1b[0m ${err.message}\n`);
	process.exit(1);
}

process.stdout.write(`\n${checks - failures.length}/${checks} checks passed\n`);
if (failures.length > 0) {
	process.stdout.write(`\x1b[31mFAILED\x1b[0m\n${failures.map((f) => `  - ${f}`).join('\n')}\n`);
	process.exit(1);
}
process.stdout.write('\x1b[32mOK\x1b[0m\n');
