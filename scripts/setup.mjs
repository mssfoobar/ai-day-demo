#!/usr/bin/env node
/**
 * One-shot bootstrap: verify (and where safe, fix) every prerequisite, install
 * dependencies, and optionally start the whole stack.
 *
 *   pnpm bootstrap        # check + fix + install, then tell you what (if anything) is left
 *   pnpm launch           # the above, then start database + service + console
 *   node scripts/setup.mjs --start --no-install
 *
 * What it checks, in order:
 *   1. Node >= 24 and corepack (pnpm comes from the root package.json's packageManager pin)
 *   2. Go >= 1.25
 *   3. A container runtime (podman or docker) whose daemon is actually running —
 *      it will try to start Docker Desktop on Windows/macOS and wait for it
 *   4. Git access to the private ops-hub repo (aoh-golib), and GOPRIVATE set so `go`
 *      does not try the public proxy for it
 *   5. A GitHub Packages token in ~/.npmrc for @mssfoobar/ui
 *   6. pnpm install + go mod download
 *
 * Credentials are the one thing this cannot conjure. If GITHUB_TOKEN is set in the
 * environment it is written to ~/.npmrc; otherwise you get exact instructions.
 *
 * Node, not bash or a Makefile: this repo is developed on native Windows as well as POSIX,
 * and Node is the one interpreter every contributor has. Python is deliberately not
 * assumed — it was absent on the machine this was built on.
 */
import { spawn, spawnSync } from 'node:child_process';
import { existsSync, readFileSync, appendFileSync } from 'node:fs';
import { homedir, platform } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { setTimeout as sleep } from 'node:timers/promises';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const args = new Set(process.argv.slice(2));
const START = args.has('--start');
const INSTALL = !args.has('--no-install');

const C = { ok: '\x1b[32m', warn: '\x1b[33m', bad: '\x1b[31m', dim: '\x1b[2m', reset: '\x1b[0m' };
const results = [];

function report(status, name, detail = '') {
	results.push({ status, name, detail });
	const mark = status === 'ok' ? `${C.ok}✔` : status === 'warn' ? `${C.warn}!` : `${C.bad}✘`;
	process.stdout.write(`${mark}${C.reset} ${name}${detail ? `  ${C.dim}${detail}${C.reset}` : ''}\n`);
}

function run(cmd, cmdArgs = [], opts = {}) {
	const r = spawnSync(cmd, cmdArgs, { encoding: 'utf8', shell: true, ...opts });
	return { ok: r.status === 0, out: `${r.stdout ?? ''}${r.stderr ?? ''}`.trim(), status: r.status };
}

function has(cmd) {
	return run(platform() === 'win32' ? 'where' : 'which', [cmd]).ok;
}

function semver(text) {
	const m = text.match(/(\d+)\.(\d+)(?:\.(\d+))?/);
	return m ? [Number(m[1]), Number(m[2]), Number(m[3] ?? 0)] : null;
}

function atLeast(actual, wanted) {
	for (let i = 0; i < 3; i++) {
		if (actual[i] > wanted[i]) return true;
		if (actual[i] < wanted[i]) return false;
	}
	return true;
}

// ---------------------------------------------------------------------------------------
// 1. Node + corepack + pnpm
// ---------------------------------------------------------------------------------------
{
	const v = semver(process.version);
	if (v && atLeast(v, [24, 0, 0])) report('ok', `Node ${process.version}`);
	else report('bad', `Node ${process.version}`, 'need >= 24 — https://nodejs.org');

	if (has('corepack')) {
		// Idempotent; makes `pnpm` resolve to the version pinned in package.json.
		run('corepack', ['enable']);
		const pnpm = run('corepack', ['pnpm', '--version']);
		if (pnpm.ok) report('ok', `pnpm ${pnpm.out.split('\n').pop()}`, 'via corepack');
		else report('bad', 'pnpm', 'corepack could not activate it — is the network reachable?');
	} else {
		report('bad', 'corepack', 'ships with Node >= 16; reinstall Node');
	}
}

// ---------------------------------------------------------------------------------------
// 2. Go
// ---------------------------------------------------------------------------------------
{
	const go = run('go', ['version']);
	const v = go.ok ? semver(go.out.replace(/^go version go/, '')) : null;
	if (v && atLeast(v, [1, 25, 0])) report('ok', `Go ${v.join('.')}`);
	else if (go.ok) report('bad', `Go ${v?.join('.') ?? '?'}`, 'need >= 1.25 — https://go.dev/dl');
	else report('bad', 'Go', 'not installed — https://go.dev/dl');
}

// ---------------------------------------------------------------------------------------
// 3. Container runtime, daemon actually running
// ---------------------------------------------------------------------------------------
async function containerRuntime() {
	const bin = ['podman', 'docker'].find((b) => run(b, ['compose', 'version']).ok);
	if (!bin) {
		report('bad', 'container runtime', 'install Docker Desktop or Podman (only Postgres runs in a container)');
		return null;
	}
	if (run(bin, ['info']).ok) {
		report('ok', `${bin} daemon running`);
		return bin;
	}

	// Try to start it. Docker Desktop on Windows/macOS is the common case for a workshop.
	const os = platform();
	let launched = false;
	if (bin === 'docker' && os === 'win32') {
		const exe = 'C:\\Program Files\\Docker\\Docker\\Docker Desktop.exe';
		if (existsSync(exe)) {
			spawn(exe, { detached: true, stdio: 'ignore' }).unref();
			launched = true;
		}
	} else if (bin === 'docker' && os === 'darwin') {
		launched = run('open', ['-a', 'Docker']).ok;
	} else if (bin === 'podman') {
		launched = run('podman', ['machine', 'start']).ok;
	}

	if (!launched) {
		report(
			'bad',
			`${bin} daemon`,
			os === 'linux' ? 'not running — `sudo systemctl start docker`' : 'not running — start it, then rerun'
		);
		return null;
	}

	process.stdout.write(`${C.dim}  starting ${bin}… `);
	for (let i = 0; i < 90; i++) {
		if (run(bin, ['info']).ok) {
			process.stdout.write(`${C.reset}\n`);
			report('ok', `${bin} daemon running`, 'started for you');
			return bin;
		}
		await sleep(1000);
	}
	process.stdout.write(`${C.reset}\n`);
	report('bad', `${bin} daemon`, 'did not come up in 90s — open it manually, then rerun');
	return null;
}
const runtime = await containerRuntime();

// ---------------------------------------------------------------------------------------
// 4. Private Go module: GOPRIVATE + git access to ops-hub
// ---------------------------------------------------------------------------------------
{
	const want = 'github.com/mssfoobar/*';
	const cur = run('go', ['env', 'GOPRIVATE']).out;
	if (!cur.split(',').includes(want)) {
		run('go', ['env', '-w', `GOPRIVATE=${cur ? `${cur},` : ''}${want}`]);
		report('ok', 'GOPRIVATE', `set to include ${want}`);
	} else {
		report('ok', 'GOPRIVATE', want);
	}

	const git = run('git', ['ls-remote', 'https://github.com/mssfoobar/ops-hub', 'HEAD'], {
		env: { ...process.env, GIT_TERMINAL_PROMPT: '0' }
	});
	if (git.ok) report('ok', 'git access to mssfoobar/ops-hub', 'aoh-golib will resolve');
	else {
		// Surface git's own words: "authentication failed" and "could not resolve host" need
		// different fixes, and a generic hint sends people down the wrong one.
		const why = git.out.split('\n').filter(Boolean).slice(-2).join(' · ') || `exit ${git.status}`;
		report(
			'bad',
			'git access to mssfoobar/ops-hub',
			`${why}\n      The Go service depends on the private aoh-golib. Sign in with \`gh auth login\` or set up a git credential for github.com.`
		);
	}
}

// ---------------------------------------------------------------------------------------
// 5. GitHub Packages token for @mssfoobar/ui
// ---------------------------------------------------------------------------------------
{
	const npmrc = join(homedir(), '.npmrc');
	const line = /\/\/npm\.pkg\.github\.com\/:_authToken=\S+/;
	const current = existsSync(npmrc) ? readFileSync(npmrc, 'utf8') : '';

	if (line.test(current)) {
		report('ok', '~/.npmrc GitHub Packages token', 'present');
	} else if (process.env.GITHUB_TOKEN) {
		const entry = `//npm.pkg.github.com/:_authToken=${process.env.GITHUB_TOKEN}\n`;
		if (current && !current.endsWith('\n')) appendFileSync(npmrc, '\n');
		appendFileSync(npmrc, entry);
		report('ok', '~/.npmrc GitHub Packages token', 'written from GITHUB_TOKEN');
	} else {
		report(
			'bad',
			'~/.npmrc GitHub Packages token',
			'@mssfoobar/ui is on GitHub Packages. Create a token with read:packages, then either rerun with GITHUB_TOKEN=<token> or add this line to ~/.npmrc:\n      //npm.pkg.github.com/:_authToken=<token>'
		);
	}
}

// ---------------------------------------------------------------------------------------
// 6. Dependencies
// ---------------------------------------------------------------------------------------
const blocking = results.filter((r) => r.status === 'bad');

if (INSTALL && blocking.length === 0) {
	process.stdout.write(`\n${C.dim}installing…${C.reset}\n`);
	const pnpm = spawnSync('corepack', ['pnpm', 'install'], { cwd: ROOT, stdio: 'inherit', shell: true });
	report(pnpm.status === 0 ? 'ok' : 'bad', 'pnpm install');

	const gomod = spawnSync('go', ['mod', 'download'], {
		cwd: join(ROOT, 'apps', 'dispatch-svc'),
		stdio: 'inherit',
		shell: true
	});
	report(gomod.status === 0 ? 'ok' : 'bad', 'go mod download');
} else if (INSTALL) {
	report('warn', 'install', 'skipped until the items above are fixed');
}

// ---------------------------------------------------------------------------------------
// Summary
// ---------------------------------------------------------------------------------------
const stillBlocking = results.filter((r) => r.status === 'bad');
process.stdout.write('\n');
if (stillBlocking.length > 0) {
	process.stdout.write(`${C.bad}${stillBlocking.length} thing(s) to fix${C.reset} — rerun \`pnpm bootstrap\` afterwards.\n`);
	process.exit(1);
}

process.stdout.write(`${C.ok}Ready.${C.reset}`);
if (START) {
	process.stdout.write(' Starting the stack…\n\n');
	const dev = spawn('node', [join(ROOT, 'scripts', 'dev.mjs')], { stdio: 'inherit', shell: true });
	dev.on('exit', (code) => process.exit(code ?? 0));
} else {
	process.stdout.write(` Run ${C.dim}pnpm start${C.reset} (or ${C.dim}pnpm launch${C.reset} next time to do both).\n`);
	if (!runtime) process.exit(1);
}
