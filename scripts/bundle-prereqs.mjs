#!/usr/bin/env node
/**
 * Builds the offline prerequisite bundle: the Node, Go, pnpm, Claude Code and
 * Python installers, for every platform in the room.
 *
 *   pnpm bundle:prereqs
 *   pnpm bundle:prereqs --out /Volumes/USB/prereq-bundle --platform win32-x64
 *
 * Needs internet. What it writes installs without any.
 */
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { createReadStream, createWriteStream, existsSync } from 'node:fs';
import { chmod, copyFile, mkdir, readFile, rm, stat, writeFile } from 'node:fs/promises';
import { Readable } from 'node:stream';
import { pipeline } from 'node:stream/promises';
import { basename, dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');

const NODE_MAJOR = 24;
const GO_MIN = '1.25';
const PYTHON_SERIES = '3.13';
const CLAUDE_RELEASES = 'https://downloads.claude.ai/claude-code-releases';

const PLATFORMS = {
	'darwin-arm64': {
		label: 'macOS, Apple Silicon',
		nodeFile: (v) => `node-v${v}-darwin-arm64.tar.gz`,
		goFile: (v) => `go${v}.darwin-arm64.tar.gz`,
		claudeBinary: 'claude',
		python: false
	},
	'linux-x64': {
		label: 'Linux, x86-64',
		nodeFile: (v) => `node-v${v}-linux-x64.tar.gz`,
		goFile: (v) => `go${v}.linux-amd64.tar.gz`,
		claudeBinary: 'claude',
		python: false
	},
	'win32-x64': {
		label: 'Windows, x86-64',
		nodeFile: (v) => `node-v${v}-win-x64.zip`,
		goFile: (v) => `go${v}.windows-amd64.zip`,
		claudeBinary: 'claude.exe',
		python: true
	}
};

// ---------------------------------------------------------------- arguments

const args = process.argv.slice(2);
let out = join(ROOT, 'prereq-bundle');
let force = false;
let zip = false;
const wanted = [];

for (let i = 0; i < args.length; i++) {
	const arg = args[i];
	if (arg === '--out') out = args[++i];
	else if (arg === '--platform') wanted.push(args[++i]);
	else if (arg === '--force') force = true;
	else if (arg === '--zip') zip = true;
	else if (arg === '--help' || arg === '-h') {
		console.log(
			[
				'pnpm bundle:prereqs [--out DIR] [--platform ID]... [--zip] [--force]',
				'',
				`  --out       where to write the bundle (default ${join(ROOT, 'prereq-bundle')})`,
				`  --platform  one of ${Object.keys(PLATFORMS).join(', ')}; repeatable, default all`,
				'  --zip       also write one zip per platform, ready to hand out',
				'  --force     re-download assets already in the bundle'
			].join('\n')
		);
		process.exit(0);
	} else {
		console.error(`unknown option: ${arg}`);
		process.exit(2);
	}
}

const targets = wanted.length ? wanted : Object.keys(PLATFORMS);
for (const id of targets) {
	if (!PLATFORMS[id]) {
		console.error(`unknown platform: ${id}. Pick from ${Object.keys(PLATFORMS).join(', ')}`);
		process.exit(2);
	}
}

// ---------------------------------------------------------------- helpers

const BOLD = '\u001b[1m';
const DIM = '\u001b[2m';
const RESET = '\u001b[0m';

const step = (text) => console.log(`\n${BOLD}==> ${text}${RESET}`);
const info = (text) => console.log(`    ${text}`);

/** Formats a byte count as MB, one decimal place. */
const mb = (bytes) => `${(bytes / 1024 / 1024).toFixed(1)} MB`;

async function getText(url) {
	const res = await fetch(url);
	if (!res.ok) throw new Error(`${res.status} ${res.statusText} for ${url}`);
	return res.text();
}

async function getJson(url) {
	return JSON.parse(await getText(url));
}

async function sha256File(path) {
	const hash = createHash('sha256');
	await pipeline(createReadStream(path), hash);
	return hash.digest('hex');
}

/** True when dotted version `a` is greater than or equal to `b`. */
function versionGte(a, b) {
	const left = a.split('.').map(Number);
	const right = b.split('.').map(Number);
	for (let i = 0; i < Math.max(left.length, right.length); i++) {
		const diff = (left[i] ?? 0) - (right[i] ?? 0);
		if (diff !== 0) return diff > 0;
	}
	return true;
}

/**
 * Downloads `url` to `dest` and returns its sha256, skipping the transfer when
 * the file is already there and matches `expected`.
 */
async function fetchAsset(url, dest, expected) {
	const name = basename(dest);
	if (!force && existsSync(dest)) {
		const have = await sha256File(dest);
		if (!expected || have === expected) {
			info(`${DIM}have  ${name}${RESET}`);
			return have;
		}
	}
	const res = await fetch(url);
	if (!res.ok) throw new Error(`${res.status} ${res.statusText} for ${url}`);
	const total = Number(res.headers.get('content-length') ?? 0);
	process.stdout.write(`    get   ${name}${total ? `  ${mb(total)}` : ''}`);
	await pipeline(Readable.fromWeb(res.body), createWriteStream(dest));
	const actual = await sha256File(dest);
	if (expected && actual !== expected) {
		throw new Error(`checksum mismatch for ${url}\n  want ${expected}\n  got  ${actual}`);
	}
	process.stdout.write(expected ? '  verified\n' : '\n');
	return actual;
}

// ---------------------------------------------------------------- versions

/** Latest Node release in the pinned major, with the release's own checksums. */
async function resolveNode() {
	const releases = await getJson('https://nodejs.org/dist/index.json');
	const release = releases.find((r) => r.version.startsWith(`v${NODE_MAJOR}.`));
	if (!release) throw new Error(`no Node ${NODE_MAJOR}.x release on nodejs.org`);
	const version = release.version.slice(1);
	const sums = await getText(`https://nodejs.org/dist/v${version}/SHASUMS256.txt`);
	const checksums = {};
	for (const line of sums.trim().split('\n')) {
		const [sha, name] = line.trim().split(/\s+/);
		checksums[name] = sha;
	}
	return { version, checksums, base: `https://nodejs.org/dist/v${version}` };
}

/** Latest stable Go release, with the checksums go.dev publishes alongside it. */
async function resolveGo() {
	const releases = await getJson('https://go.dev/dl/?mode=json');
	const release = releases[0];
	const version = release.version.replace(/^go/, '');
	if (!versionGte(version, GO_MIN)) {
		throw new Error(`go.dev offers ${version}, older than ${GO_MIN}`);
	}
	const checksums = Object.fromEntries(release.files.map((f) => [f.filename, f.sha256]));
	return { version, checksums, base: 'https://go.dev/dl' };
}

/** Latest Claude Code release, with the per-platform checksums from its manifest. */
async function resolveClaude() {
	const version = (await getText(`${CLAUDE_RELEASES}/latest`)).trim();
	if (!/^\d+\.\d+\.\d+/.test(version)) {
		throw new Error(`unexpected version from ${CLAUDE_RELEASES}/latest`);
	}
	const manifest = await getJson(`${CLAUDE_RELEASES}/${version}/manifest.json`);
	return { version, platforms: manifest.platforms, base: `${CLAUDE_RELEASES}/${version}` };
}

/** Newest patch in the pinned Python series that still ships a Windows installer. */
async function resolvePython() {
	const index = await getText('https://www.python.org/ftp/python/');
	const pattern = new RegExp(`${PYTHON_SERIES.replace('.', '\\.')}\\.(\\d+)/`, 'g');
	const patches = [...new Set([...index.matchAll(pattern)].map((m) => Number(m[1])))].sort(
		(a, b) => b - a
	);
	for (const patch of patches) {
		const version = `${PYTHON_SERIES}.${patch}`;
		const url = `https://www.python.org/ftp/python/${version}/python-${version}-amd64.exe`;
		const res = await fetch(url, { method: 'HEAD' });
		if (res.ok) return { version, url };
	}
	throw new Error(`no ${PYTHON_SERIES}.x Windows installer on python.org`);
}

/** The pnpm version this repo's `packageManager` field pins. */
async function resolvePnpm() {
	const pkg = JSON.parse(await readFile(join(ROOT, 'package.json'), 'utf8'));
	const version = (pkg.packageManager ?? '').replace(/^pnpm@/, '');
	if (!version) throw new Error('package.json has no packageManager pin for pnpm');
	return { version, url: `https://registry.npmjs.org/pnpm/-/pnpm-${version}.tgz` };
}

// ---------------------------------------------------------------- build

step('Resolving versions');
const [node, go, claude, pnpm] = await Promise.all([
	resolveNode(),
	resolveGo(),
	resolveClaude(),
	resolvePnpm()
]);
const python = targets.includes('win32-x64') ? await resolvePython() : null;

info(`node    ${node.version}`);
info(`go      ${go.version}`);
info(`pnpm    ${pnpm.version}`);
info(`claude  ${claude.version}`);
if (python) info(`python  ${python.version}  (Windows only)`);

const written = [];

/** Writes the SHA256SUMS file the offline installers verify against. */
async function writeSums(dir, entries) {
	const body = entries.map(({ sha, name }) => `${sha}  ${name}`).join('\n');
	await writeFile(join(dir, 'SHA256SUMS'), `${body}\n`);
}

async function record(dir, name, path) {
	written.push({ dir, name, bytes: (await stat(path)).size });
}

step('pnpm, shared by every platform');
const commonDir = join(out, 'common');
await mkdir(commonDir, { recursive: true });
const pnpmName = `pnpm-${pnpm.version}.tgz`;
const pnpmSha = await fetchAsset(pnpm.url, join(commonDir, pnpmName));
await writeSums(commonDir, [{ sha: pnpmSha, name: pnpmName }]);
await record('common', pnpmName, join(commonDir, pnpmName));

for (const id of targets) {
	const platform = PLATFORMS[id];
	step(`${id}  ${DIM}${platform.label}${RESET}`);
	const dir = join(out, id);
	await mkdir(dir, { recursive: true });
	const sums = [];

	const nodeName = platform.nodeFile(node.version);
	sums.push({
		name: nodeName,
		sha: await fetchAsset(`${node.base}/${nodeName}`, join(dir, nodeName), node.checksums[nodeName])
	});

	const goName = platform.goFile(go.version);
	sums.push({
		name: goName,
		sha: await fetchAsset(`${go.base}/${goName}`, join(dir, goName), go.checksums[goName])
	});

	const claudeEntry = claude.platforms[id];
	if (!claudeEntry) throw new Error(`Claude Code ${claude.version} has no ${id} build`);
	const claudePath = join(dir, platform.claudeBinary);
	sums.push({
		name: platform.claudeBinary,
		sha: await fetchAsset(`${claude.base}/${id}/${claudeEntry.binary}`, claudePath, claudeEntry.checksum)
	});
	await chmod(claudePath, 0o755);

	if (platform.python) {
		const pythonName = `python-${python.version}-amd64.exe`;
		sums.push({ name: pythonName, sha: await fetchAsset(python.url, join(dir, pythonName)) });
	}

	await writeSums(dir, sums);
	for (const { name } of sums) await record(id, name, join(dir, name));
}

// ---------------------------------------------------------------- manifest

step('Bundle files');
const installers = ['install-prereqs-offline.sh', 'install-prereqs-offline.ps1'];
for (const name of installers) {
	await copyFile(join(ROOT, 'scripts', name), join(out, name));
	info(name);
}
await chmod(join(out, 'install-prereqs-offline.sh'), 0o755);

const built = new Date();
await writeFile(
	join(out, 'versions.json'),
	`${JSON.stringify(
		{
			built: built.toISOString(),
			platforms: targets,
			node: node.version,
			go: go.version,
			pnpm: pnpm.version,
			claude: claude.version,
			python: python?.version ?? null
		},
		null,
		'\t'
	)}\n`
);
info('versions.json');

await writeFile(
	join(out, 'README.md'),
	[
		'# Workshop prerequisites, offline',
		'',
		'Node, Go, pnpm, Claude Code and, on Windows, Python. Everything here',
		'installs with no network.',
		'',
		'```sh',
		'./install-prereqs-offline.sh            # macOS and Linux',
		'```',
		'',
		'```powershell',
		'.\\install-prereqs-offline.ps1           # Windows, from PowerShell',
		'```',
		'',
		'Add `--check` (`-Check` on Windows) to see what you are missing without',
		'installing anything. Windows needs no Administrator. macOS and Linux',
		'install into `/usr/local` and ask for `sudo`, falling back to',
		'`~/.local` when there is none.',
		'',
		'Open a new terminal afterwards, so every PATH change takes effect.',
		'',
		'Podman is not in here. Install Podman Desktop from',
		'<https://podman-desktop.io> while you still have internet.',
		'',
		`Versions are in \`versions.json\`. Built ${built.toISOString().slice(0, 10)}.`,
		''
	].join('\n')
);
info('README.md');

/**
 * Zips `entries`, all relative to `cwd`, into `dest`. Tries zip, then the
 * bsdtar that ships with macOS and Windows.
 */
function makeZip(dest, cwd, entries) {
	const attempts = [
		['zip', ['-r', '-q', '-1', dest, ...entries]],
		['tar', ['-a', '-c', '-f', dest, ...entries]]
	];
	for (const [command, commandArgs] of attempts) {
		const result = spawnSync(command, commandArgs, { cwd, stdio: 'inherit' });
		if (!result.error && result.status === 0) return command;
	}
	throw new Error(`could not write ${dest}: neither zip nor tar worked`);
}

const zips = [];
if (zip) {
	step('Zipping');
	const root = resolve(out);
	const parent = dirname(root);
	const name = basename(root);
	for (const id of targets) {
		const dest = join(parent, `${name}-${id}.zip`);
		// zip appends to an archive that is already there.
		await rm(dest, { force: true });
		const entries = [
			`${name}/README.md`,
			`${name}/versions.json`,
			`${name}/install-prereqs-offline.sh`,
			`${name}/install-prereqs-offline.ps1`,
			`${name}/common`,
			`${name}/${id}`
		];
		makeZip(dest, parent, entries);
		const bytes = (await stat(dest)).size;
		zips.push({ dest, bytes });
		info(`${basename(dest)}  ${mb(bytes)}`);
	}
}

step('Summary');
console.log();
console.log(`    ${'DIR'.padEnd(14)} ${'FILE'.padEnd(34)} SIZE`);
console.log(`    ${'-----'.padEnd(14)} ${'------'.padEnd(34)} ------`);
let total = 0;
for (const { dir, name, bytes } of written) {
	total += bytes;
	console.log(`    ${dir.padEnd(14)} ${name.padEnd(34)} ${mb(bytes).padStart(9)}`);
}
console.log(`\n    ${'total'.padEnd(49)} ${mb(total).padStart(9)}`);
console.log(`\n    Written to ${out}`);
if (zips.length) {
	console.log('\n    One zip per platform, each carrying only what that machine needs:\n');
	for (const { dest, bytes } of zips) console.log(`    ${dest}  ${mb(bytes)}`);
	console.log('\n    Hand out the zip for the machine. Unzip it, then run the installer inside.\n');
} else {
	console.log('    Copy the whole folder to a USB stick, or pass --zip to package it.\n');
}
