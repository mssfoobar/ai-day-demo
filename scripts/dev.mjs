#!/usr/bin/env node
/**
 * One-shot dev runner: database, backend, frontend.
 *
 *   pnpm start          # everything
 *   pnpm start --no-db  # assume Postgres is already up
 *
 * Written in Node rather than a shell script or a Makefile because this repo is
 * developed on native Windows as well as POSIX, and Node is the one interpreter every
 * contributor already has (see AGENTS.md / aoh-scripting-conventions).
 *
 * The apps run NATIVELY (`go run`, `vite dev`) against a composed database — that is the
 * AOH convention, and it keeps rebuilds instant. Only Postgres is a container.
 */
import { spawn, spawnSync } from 'node:child_process';
import { createServer } from 'node:net';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { setTimeout as sleep } from 'node:timers/promises';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const WEB_URL = 'http://localhost:5173';
const SVC_URL = 'http://localhost:8081';

const skipDb = process.argv.includes('--no-db');
const children = [];
let shuttingDown = false;

const COLOURS = { db: '\x1b[35m', svc: '\x1b[36m', web: '\x1b[32m', run: '\x1b[33m' };
const RESET = '\x1b[0m';

function log(tag, message) {
	process.stdout.write(`${COLOURS[tag] ?? ''}[${tag}]${RESET} ${message}\n`);
}

/** Compose runtime: podman first, docker as fallback — the AOH dual form. */
function composeRunner() {
	for (const bin of ['podman', 'docker']) {
		const probe = spawnSync(bin, ['compose', 'version'], { stdio: 'ignore', shell: true });
		if (probe.status === 0) return bin;
	}
	return null;
}

function startDatabase() {
	const runner = composeRunner();
	if (!runner) {
		log('db', 'no podman or docker found — start Postgres yourself, or pass --no-db');
		process.exit(1);
	}

	log('db', `${runner} compose up -d postgres`);
	const up = spawnSync(runner, ['compose', 'up', '-d', 'postgres'], {
		cwd: join(ROOT, 'compose'),
		stdio: 'inherit',
		shell: true
	});
	if (up.status !== 0) {
		log('db', 'compose failed — is the container runtime running?');
		process.exit(up.status ?? 1);
	}
	return runner;
}

/**
 * Wait for the container to report healthy.
 *
 * `compose up -d` returns as soon as the container is *started*, which is well before
 * Postgres accepts connections. Without this the service loses the race on a cold volume
 * and exits on connect.
 */
async function waitForDatabase(runner) {
	for (let i = 0; i < 60; i++) {
		const probe = spawnSync(
			runner,
			['inspect', '-f', '{{.State.Health.Status}}', 'dispatch-postgres'],
			{ encoding: 'utf8', shell: true }
		);
		if (probe.stdout?.trim() === 'healthy') {
			log('db', 'healthy');
			return;
		}
		await sleep(1000);
	}
	log('db', 'did not report healthy in 60s — continuing anyway, the service may fail to connect');
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

function run(tag, command, args, cwd) {
	// One command string, no args array: passing both with `shell: true` triggers Node's
	// DEP0190 warning. Nothing here is user input, so the concatenation is safe.
	const child = spawn([command, ...args].join(' '), {
		cwd,
		shell: true,
		stdio: ['ignore', 'pipe', 'pipe']
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

async function waitForHttp(tag, url, attempts = 60) {
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
			child.kill();
		} catch {
			// already gone
		}
	}
	// The database is deliberately left running: it holds the seeded data, and starting it
	// again is the slow part. `pnpm stop` tears it down.
	setTimeout(() => process.exit(code), 300);
}

process.on('SIGINT', () => shutdown(0));
process.on('SIGTERM', () => shutdown(0));

await requireFreePort('svc', 8081);
await requireFreePort('web', 5173);

const runner = skipDb ? null : startDatabase();
if (runner) await waitForDatabase(runner);

log('svc', 'go run ./cmd/server');
run('svc', 'go', ['run', './cmd/server'], join(ROOT, 'apps', 'dispatch-svc'));

if (await waitForHttp('svc', `${SVC_URL}/readyz`)) {
	log('svc', `ready on ${SVC_URL}`);
}

log('web', 'vite dev');
run(
	'web',
	'pnpm',
	['exec', 'env-cmd', '-f', '.env.development', 'vite', 'dev', '--port', '5173', '--strictPort'],
	join(ROOT, 'apps', 'dispatch-web')
);

if (await waitForHttp('web', `${WEB_URL}/units`)) {
	log('run', `console ready → ${WEB_URL}`);
	log('run', 'press Ctrl+C to stop (the database keeps running; `pnpm stop` removes it)');
}
