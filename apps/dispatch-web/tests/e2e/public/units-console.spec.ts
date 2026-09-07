import { expect, test, type Page } from '@playwright/test';

/**
 * Baseline dispatch console — acceptance coverage for the openspec change
 * `baseline-dispatch-console` (specs/dispatch-console, specs/field-unit-roster).
 *
 * No storageState, no login, no container of any kind. If any assertion here starts
 * needing infra, the baseline has regressed.
 */

const EMPTY_PROMPT = 'Select a unit to see its details.';
const STATUSES = ['Available', 'En route', 'Idle'];
const UNIT_ID = /FU-\d+/;

/**
 * Navigate and wait for hydration before interacting.
 *
 * The page is server-rendered, so the unit rows exist in the DOM — and are clickable
 * as far as Playwright's actionability checks are concerned — before Svelte has
 * attached its handlers. Clicking in that window silently does nothing: the click
 * lands, no handler runs, and the detail pane never updates. The console issues no
 * requests of its own, so `networkidle` resolves as soon as hydration's assets settle.
 */
async function gotoUnits(page: Page) {
	await page.goto('/units', { waitUntil: 'networkidle' });
}

/** The selectable unit rows, scoped to the list so unrelated buttons never match. */
const unitRows = (page: Page) => page.locator('ul li button');

const detailPane = (page: Page) => page.locator('[aria-live="polite"]');

/** `CardTitle` renders a `<div data-slot="card-title">`, not a heading element. */
const cardTitles = (page: Page) => page.locator('[data-slot="card-title"]');

test.describe('dispatch console', () => {
	test('renders without an auth redirect or any backend call', async ({ page }) => {
		const offSiteRequests: string[] = [];
		page.on('request', (request) => {
			const url = request.url();
			if (
				url.includes('/aoh/gateway') ||
				url.includes('/aoh/api/auth') ||
				url.includes('keycloak')
			) {
				offSiteRequests.push(url);
			}
		});

		await gotoUnits(page);

		await expect(page).toHaveURL(/\/units$/);
		await expect(page.getByRole('heading', { name: 'Dispatch console' })).toBeVisible();
		expect(offSiteRequests).toEqual([]);
	});

	test('redirects bare / to the console', async ({ page }) => {
		await page.goto('/');
		await expect(page).toHaveURL(/\/units$/);
		await expect(page.getByRole('heading', { name: 'Dispatch console' })).toBeVisible();
	});

	test('lists 4-5 units, each with a call sign, id, and status label', async ({ page }) => {
		await gotoUnits(page);
		const rows = unitRows(page);

		const count = await rows.count();
		expect(count).toBeGreaterThanOrEqual(4);
		expect(count).toBeLessThanOrEqual(5);

		const seenIds = new Set<string>();
		for (let i = 0; i < count; i++) {
			const text = (await rows.nth(i).innerText()).trim();

			const id = text.match(UNIT_ID)?.[0];
			expect(id, `row ${i} shows a unit id`).toBeTruthy();
			seenIds.add(id as string);

			// Status is conveyed as a text label, never by colour alone.
			expect(STATUSES.some((status) => text.includes(status))).toBe(true);
			// A call sign remains once the id is removed.
			expect(text.replace(id as string, '').trim().length).toBeGreaterThan(0);
		}
		expect(seenIds.size, 'unit ids are unique').toBe(count);
	});

	test('shows the empty-state prompt before any selection', async ({ page }) => {
		await gotoUnits(page);

		await expect(detailPane(page)).toContainText(EMPTY_PROMPT);
		await expect(cardTitles(page).filter({ hasText: 'Unit detail' })).toBeVisible();
		await expect(page.locator('[aria-current="true"]')).toHaveCount(0);
	});

	test('selecting a unit populates the detail pane', async ({ page }) => {
		await gotoUnits(page);
		const first = unitRows(page).first();
		const id = (await first.innerText()).match(UNIT_ID)?.[0] as string;

		await first.click();

		const pane = detailPane(page);
		await expect(pane).not.toContainText(EMPTY_PROMPT);
		await expect(pane).toContainText(id);
		await expect(pane).toContainText('Call sign');
		await expect(pane).toContainText('Unit ID');
		await expect(pane).toContainText('Status');
		await expect(page.locator('[aria-current="true"]')).toHaveCount(1);
	});

	test('names the selected unit in the detail pane title', async ({ page }) => {
		await gotoUnits(page);
		const first = unitRows(page).first();
		const text = (await first.innerText()).trim();
		const id = text.match(UNIT_ID)?.[0] as string;
		const callSign = text.split('\n')[0].trim();

		await first.click();

		await expect(cardTitles(page).filter({ hasText: callSign })).toBeVisible();
		await expect(detailPane(page)).toContainText(id);
	});

	test('switching selection replaces the detail contents', async ({ page }) => {
		await gotoUnits(page);
		const rows = unitRows(page);

		const firstId = (await rows.nth(0).innerText()).match(UNIT_ID)?.[0] as string;
		const secondId = (await rows.nth(1).innerText()).match(UNIT_ID)?.[0] as string;

		await rows.nth(0).click();
		await expect(detailPane(page)).toContainText(firstId);

		await rows.nth(1).click();
		const pane = detailPane(page);
		await expect(pane).toContainText(secondId);
		await expect(pane).not.toContainText(firstId);
		await expect(page.locator('[aria-current="true"]')).toHaveCount(1);
	});

	test('a unit can be selected with the keyboard', async ({ page }) => {
		await gotoUnits(page);
		const first = unitRows(page).first();
		const id = (await first.innerText()).match(UNIT_ID)?.[0] as string;

		await first.focus();
		await expect(first).toBeFocused();
		await page.keyboard.press('Enter');

		await expect(detailPane(page)).toContainText(id);
		await expect(first).toHaveAttribute('aria-current', 'true');
	});

	test('selection issues no network request', async ({ page }) => {
		await gotoUnits(page);

		const requests: string[] = [];
		page.on('request', (request) => requests.push(request.url()));

		await unitRows(page).nth(1).click();
		await expect(detailPane(page)).not.toContainText(EMPTY_PROMPT);

		expect(requests, 'selection is client-side only').toEqual([]);
	});

	test('renders legibly in the light theme', async ({ page }) => {
		// The scaffold defaults ThemeProvider to dark, so the light path is never
		// exercised by default and can rot unnoticed. Assert it explicitly.
		await page.emulateMedia({ colorScheme: 'light' });
		await page.addInitScript(() => {
			try {
				localStorage.setItem('theme', 'light');
			} catch {
				/* storage unavailable — the media emulation above still applies */
			}
		});
		await gotoUnits(page);

		await expect(page.getByRole('heading', { name: 'Dispatch console' })).toBeVisible();
		await unitRows(page).first().click();
		await expect(detailPane(page)).toContainText('Call sign');

		const { color, background } = await page.locator('body').evaluate((el) => {
			const style = getComputedStyle(el);
			return { color: style.color, background: style.backgroundColor };
		});
		// Foreground and background must actually differ — a token that failed to
		// resolve in light mode collapses these to the same value.
		expect(color).not.toBe(background);
	});
});
