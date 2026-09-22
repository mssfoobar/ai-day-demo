#!/usr/bin/env node
/**
 * One-shot dev runner: dependencies, the platform stack, backend, frontend.
 *
 *   pnpm start             # everything
 *   pnpm start --no-infra  # assume the compose stack is already up
 *
 * Written in Node rather than a shell script or a Makefile because this repo is
 * developed on native Windows as well as POSIX, and Node is the one interpreter every
 * contributor already has (see AGENTS.md / aoh-scripting-conventions).
 *
 * The two apps run NATIVELY (`go run`, `vite dev`) against a composed stack — that is the
 * AOH convention, and it keeps rebuilds instant. Everything they depend on is a container:
 * the dispatch PostgreSQL plus IAMS (Keycloak + AAS), SDS, RTUS and GIS. This used to be
 * one container; since the console gained authentication and a map it is sixteen, and
 * bringing up only the database now yields a console where every route 500s (OIDC
 * discovery runs in a top-level await) and a service that cannot validate a token.
 */
import { spawn, spawnSync } from 'node:child_process';
import { createServer } from 'node:net';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { setTimeout as sleep } from 'node:timers/promises';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const COMPOSE_DIR = join(ROOT, 'compose');

/**
 * The dev domain the whole stack is addressed on. `nip.io` wildcard-resolves to the
 * loopback on every OS, including Windows, where `*.localhost` does not.
 */
const DEV_DOMAIN = readComposeEnv('DEV_DOMAIN') ?? '127.0.0.1.nip.io';

/*
 * The console is served on the ${DEV_DOMAIN} origin, not plain localhost. Its session
 * cookie has to be issued on a parent of rtus-seh.${DEV_DOMAIN} for the map's SSE
 * subscription to carry it, and rtus-seh's CORS allow-list already contains exactly this
 * origin. On localhost the map silently never connects.
 */
const WEB_URL = `http://${DEV_DOMAIN}:5173`;
const SVC_URL = 'http://localhost:8081';

// Same defaults as the service's own config, and the same env vars override them.
const SQL_PORT = Number(process.env.SQL_PORT ?? process.env.POSTGRES_PORT ?? 5432);
// POSTGRES_PORT alone moves only the published port; the service reads SQL_PORT.
process.env.SQL_PORT = String(SQL_PORT);

const skipInfra = process.argv.includes('--no-infra') || process.argv.includes('--no-db');
const children = [];
let shuttingDown = false;

const COLOURS = { infra: '\x1b[35m', svc: '\x1b[36m', web: '\x1b[32m', run: '\x1b[33m' };
const RESET = '\x1b[0m';

function log(tag, message) {
	process.stdout.write(`${COLOURS[tag] ?? ''}[${tag}]${RESET} ${message}\n`);
}

/** Read one value out of compose/.env, so this script and the stack agree. */
function readComposeEnv(key) {
	try {
		const body = readFileSync(join(COMPOSE_DIR, '.env'), 'utf8');
		const match = body.match(new RegExp(`^${key}=(.*)$`, 'm'));
		return match ? match[1].trim() : undefined;
	} catch {
		return undefined;
	}
}

/**
 * Compose runtime: podman preferred, docker accepted.
 *
 * Both are supported everywhere else in this change, so probing only podman would make
 * `pnpm start` the one command a Docker user cannot run.
 */
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
 * Warns when the podman VM will copy its proxy settings into every container.
 *
 * `podman machine init` inherits the host's HTTP_PROXY. Containers then address each
 * other by compose name (iams-keycloak:8080), which no proxy can route, and the init
 * containers fail on a connect timeout naming the proxy rather than the cause.
 */
function warnOnContainerProxy(runner) {
	if (runner !== 'podman') return;
	const machines = spawnSync('podman', ['machine', 'list', '--format', '{{.Running}}'], {
		encoding: 'utf8'
	});
	if (!machines.stdout?.includes('true')) return;

	const probe = spawnSync(
		'podman',
		[
			'machine',
			'ssh',
			"systemctl --user show-environment 2>/dev/null | grep -i '^HTTP_PROXY=.'; " +
				"grep -rhE '^[[:space:]]*http_proxy[[:space:]]*=' /etc/containers/containers.conf " +
				'/etc/containers/containers.conf.d/ 2>/dev/null'
		],
		{ encoding: 'utf8' }
	);
	const out = probe.stdout ?? '';
	// Absent any http_proxy setting, podman defaults to passing the proxy through.
	if (!/HTTP_PROXY=\S/i.test(out) || /http_proxy\s*=\s*false/i.test(out)) return;

	log('infra', 'this podman machine passes its HTTP_PROXY into every container');
	log('infra', 'container-to-container calls cannot route through it, so the init containers will fail');
	log('infra', 'disable it with:');
	log(
		'infra',
		'  podman machine ssh \'sudo mkdir -p /etc/containers/containers.conf.d && printf "[containers]\\nhttp_proxy = false\\n" | sudo tee /etc/containers/containers.conf.d/99-no-container-proxy.conf\''
	);
}

/** pnpm invocation: the binary on PATH, falling back to corepack. */
function pnpmCommand() {
	for (const candidate of ['pnpm', 'corepack pnpm']) {
		const probe = spawnSync(`${candidate} --version`, { stdio: 'ignore', shell: true });
		if (probe.status === 0) return candidate;
	}
	log('web', 'no pnpm found — install it with `npm i -g pnpm`, then retry');
	process.exit(1);
}

/** Installs workspace dependencies. A sub-second no-op once they are present. */
function installDependencies(pnpm) {
	log('run', 'pnpm install');
	// Not --frozen-lockfile: a pnpm newer than the one that wrote the lockfile rejects
	// its settings block outright, and an attendee's pnpm version is not ours to pick.
	const result = spawnSync(`${pnpm} install`, { cwd: ROOT, stdio: 'inherit', shell: true });
	if (result.status !== 0) {
		log('run', 'install failed. A 401 means the token in .npmrc has expired');
		process.exit(result.status ?? 1);
	}
}

function startInfra(runner) {
	log('infra', `${runner} compose up -d`);
	const up = spawnSync(runner, ['compose', 'up', '-d'], {
		cwd: COMPOSE_DIR,
		stdio: 'inherit'
	});
	if (up.status !== 0) {
		log('infra', 'compose failed. Is the podman/docker machine running?');
		log('infra', 'a pull failure means no access to ghcr.io/mssfoobar — see SETUP.md');
		process.exit(up.status ?? 1);
	}
}

/** Compose project name, which labels every container in the stack. */
function composeProject() {
	try {
		const body = readFileSync(join(COMPOSE_DIR, 'compose.yml'), 'utf8');
		return body.match(/^name:\s*(\S+)/m)?.[1] ?? 'aoh';
	} catch {
		return 'aoh';
	}
}

const PROJECT = composeProject();

/**
 * One row per container in the project, read from the engine rather than from compose.
 *
 * `podman-compose ps` rejects `-a` and prints nothing for `--format json`, so asking
 * compose yields an empty list and every readiness check below silently passes.
 */
function composePs(runner) {
	const probe = spawnSync(
		runner,
		[
			'ps',
			'-a',
			'--filter',
			`label=com.docker.compose.project=${PROJECT}`,
			'--format',
			'{{.Label "com.docker.compose.service"}}\t{{.State}}\t{{.Status}}'
		],
		{ encoding: 'utf8' }
	);
	return (probe.stdout ?? '')
		.split('\n')
		.map((line) => line.trim())
		.filter(Boolean)
		.map((line) => {
			const [Service, State, Status = ''] = line.split('\t');
			// Both engines spell health and exit code into Status: "Up 3 minutes (healthy)",
			// "Exited (1) 2 minutes ago". Neither exposes them as template fields on both.
			const exited = Status.match(/^Exited \((\d+)\)/);
			return {
				Service,
				State,
				Status,
				Health: Status.match(/\((healthy|unhealthy|starting)\)/)?.[1],
				ExitCode: exited ? Number(exited[1]) : 0
			};
		});
}

/**
 * Wait for the stack to converge.
 *
 * Two distinct conditions, because `compose up -d` returns long before either holds:
 * every service that declares a healthcheck must be healthy, and the two one-shot init
 * containers must have exited 0. `project-aas-init` is the one that creates the
 * application roles, so starting the apps before it finishes yields tokens with no roles
 * and a console that 403s for no visible reason.
 */
async function waitForInfra(runner) {
	const INIT_SERVICES = ['iams-init', 'project-aas-init'];
	for (let i = 0; i < 180; i++) {
		const rows = composePs(runner);
		if (rows.length === 0) {
			// Silence here reads exactly like a slow start, so name it.
			if (i % 10 === 0) log('infra', `no containers labelled ${PROJECT} yet`);
		} else {
			const unhealthy = rows.filter((r) => r.Health && r.Health !== 'healthy');
			const initsPending = INIT_SERVICES.filter((name) => {
				const row = rows.find((r) => r.Service === name);
				return !row || row.State !== 'exited' || row.ExitCode !== 0;
			});
			const failed = rows.find((r) => r.State === 'exited' && r.ExitCode !== 0);

			if (unhealthy.length === 0 && initsPending.length === 0) {
				log('infra', `${rows.length} services ready`);
				return;
			}
			if (failed && !INIT_SERVICES.includes(failed.Service)) {
				log('infra', `${failed.Service} exited with code ${failed.ExitCode}`);
				log('infra', `inspect it: ${runner} compose -f compose/compose.yml logs ${failed.Service}`);
				process.exit(1);
			}
			if (i % 10 === 0) {
				const waiting = [...unhealthy.map((r) => r.Service), ...initsPending];
				log('infra', `waiting for ${waiting.join(', ')}`);
			}
		}
		await sleep(1000);
	}
	log('infra', 'did not converge in 180s — continuing anyway, the apps may fail to start');
}

/**
 * Fail fast when a port is already taken.
 *
 * Without this the stack starts, the new process loses the bind, and the readiness probe
 * cheerfully succeeds against whatever was *already* listening — so the run looks healthy
 * right up until the child exits. Naming the port is far more useful than that.
 */
function portInUse(port) {
	return new Promise((resolve) => {
		const probe = createServer();
		probe.once('error', () => resolve(true));
		probe.once('listening', () => probe.close(() => resolve(false)));
		// No host: bind the way the services do (all interfaces). Probing 127.0.0.1 alone
		// passed on Windows while a previous `go run` still held `[::]:8081`, and the new
		// service then died on bind — exactly the case this check exists to name.
		probe.listen(port);
	});
}

async function requireFreePort(tag, port) {
	if (await portInUse(port)) {
		log(tag, `port ${port} is already in use — stop whatever is on it, then retry`);
		process.exit(1);
	}
}

function run(tag, command, args, cwd, env) {
	// One command string, no args array: passing both with `shell: true` triggers Node's
	// DEP0190 warning. Nothing here is user input, so the concatenation is safe.
	const child = spawn([command, ...args].join(' '), {
		cwd,
		shell: true,
		stdio: ['ignore', 'pipe', 'pipe'],
		env: { ...process.env, ...env },
		// POSIX: own process group, so shutdown() can signal the whole tree with a negative
		// pid. Windows has no process groups in this sense; shutdown() uses taskkill /T there.
		detached: process.platform !== 'win32'
	});
	const relay = (stream) => {
		stream.setEncoding('utf8');
		let buffer = '';
		stream.on('data', (chunk) => {
			buffer += chunk;
			const lines = buffer.split('\n');
			buffer = lines.pop() ?? '';
			for (const line of lines) if (line.trim()) log(tag, line);
		});
	};
	relay(child.stdout);
	relay(child.stderr);

	child.on('exit', (code) => {
		if (shuttingDown) return;
		log(tag, `exited with code ${code}`);
		// One process dying leaves a half-running stack that looks alive but is not.
		shutdown(code ?? 1);
	});

	children.push(child);
	return child;
}

async function waitForHttp(tag, url, attempts = 90) {
	for (let i = 0; i < attempts; i++) {
		try {
			const response = await fetch(url, { signal: AbortSignal.timeout(2000) });
			if (response.ok) return true;
		} catch {
			// not up yet
		}
		await sleep(1000);
	}
	log(tag, `never became reachable at ${url}`);
	return false;
}

function shutdown(code = 0) {
	if (shuttingDown) return;
	shuttingDown = true;
	log('run', 'stopping…');
	for (const child of children) {
		try {
			// `child.kill()` alone is not enough: with `shell: true` the child IS the shell,
			// and killing it orphans the real process underneath (`go run`'s compiled
			// binary, vite's node). Those keep 8081/5173 bound and the next start fails the
			// preflight. Kill the whole tree.
			if (process.platform === 'win32') {
				spawnSync('taskkill', ['/pid', String(child.pid), '/T', '/F'], { stdio: 'ignore' });
			} else {
				// Negative pid targets the process group (children are spawned detached below).
				try {
					process.kill(-child.pid, 'SIGTERM');
				} catch {
					child.kill('SIGTERM');
				}
			}
		} catch {
			// already gone
		}
	}
	// The stack is deliberately left running: it holds the realm, the AAS roles and the
	// seeded data, and starting it again is the slow part. `pnpm stop` tears it down.
	setTimeout(() => process.exit(code), 300);
}

process.on('SIGINT', () => shutdown(0));
process.on('SIGTERM', () => shutdown(0));

await requireFreePort('svc', 8081);
await requireFreePort('web', 5173);

const pnpm = pnpmCommand();
installDependencies(pnpm);

if (skipInfra) {
	log('infra', 'skipped (--no-infra): assuming the compose stack is already up');
} else {
	const runner = composeRunner();
	if (!runner) {
		log('infra', 'no podman or docker found. The stack is required — see SETUP.md');
		process.exit(1);
	}
	warnOnContainerProxy(runner);
	startInfra(runner);
	await waitForInfra(runner);
}

/*
 * The service's own defaults target the compose network (`http://iams-keycloak:8080`),
 * which does not resolve from a native process. Without these it starts, and then fails
 * to validate every token with an error that names DNS rather than configuration.
 */
log('svc', 'go run ./cmd/server');
run('svc', 'go', ['run', './cmd/server'], join(ROOT, 'apps', 'dispatch-svc'), {
	SQL_HOST: process.env.SQL_HOST ?? 'localhost',
	SQL_PORT: String(SQL_PORT),
	SQL_USER: process.env.SQL_USER ?? 'dispatch',
	SQL_PASSWORD: process.env.SQL_PASSWORD ?? 'dispatch',
	SQL_DATABASE_NAME: process.env.SQL_DATABASE_NAME ?? 'dispatch',
	SQL_SCHEMA_NAME: process.env.SQL_SCHEMA_NAME ?? 'dispatch',
	SQL_SSL_MODE: process.env.SQL_SSL_MODE ?? 'disable',
	IAMS_KEYCLOAK_HOST: `http://iams-keycloak.${DEV_DOMAIN}`,
	IAMS_KEYCLOAK_PORT: '80',
	IAMS_KEYCLOAK_REALM: 'aoh',
	GIS_URL: `http://gis.${DEV_DOMAIN}`,
	HTTP_PORT: '8081',
	HTTP_ALLOWED_ORIGINS: ''
});

if (await waitForHttp('svc', `${SVC_URL}/readyz`)) {
	log('svc', `ready on ${SVC_URL}`);
}

log('web', 'vite dev');
run(
	'web',
	pnpm,
	// --host binds all interfaces rather than loopback, which is what serving on
	// ${DEV_DOMAIN} needs. vite.config.ts's allowedHosts is the other half.
	[
		'exec',
		'env-cmd',
		'-f',
		'.env.development',
		'vite',
		'dev',
		'--host',
		'--port',
		'5173',
		'--strictPort'
	],
	join(ROOT, 'apps', 'dispatch-web')
);

/*
 * Probe an UNAUTHENTICATED route. Every console page now redirects to Keycloak, and
 * `fetch` follows redirects — so probing a page would report ready as soon as the sign-in
 * form rendered, whether or not the console itself was up. `/livez` is outside the
 * (private) group precisely so it can answer this.
 */
if (await waitForHttp('web', `${WEB_URL}/livez`)) {
	log('run', `console ready → ${WEB_URL}/aoh/dispatch/units`);
	log('run', 'sign in as the seeded dispatcher or viewer — see SETUP.md');
	log('run', 'press Ctrl+C to stop (the stack keeps running; `pnpm stop` removes it)');
}
