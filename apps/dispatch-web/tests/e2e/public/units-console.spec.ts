import { expect, test, type Locator, type Page } from '@playwright/test';

/**
 * Baseline dispatch console — acceptance coverage for the openspec change
 * `baseline-dispatch-console` (specs/dispatch-console, specs/field-unit-roster).
 *
 * No storageState, no login, no container of any kind. If any assertion here starts
 * needing infra, the baseline has regressed.
 */

const EMPTY_PROMPT = 'Select a unit to see its details.';
const STATUSES = ['Available', 'En route', 'Idle'];
const UNIT_ID = /^FU-\d+$/;

/** The baseline roster, pinned — order included. Mirrors `roster.test.ts`'s EXPECTED. */
const EXPECTED_ROWS = [
	{ callSign: 'Alpha-1', id: 'FU-101', status: 'Available' },
	{ callSign: 'Alpha-2', id: 'FU-102', status: 'En route' },
	{ callSign: 'Bravo-1', id: 'FU-204', status: 'Idle' },
	{ callSign: 'Bravo-2', id: 'FU-205', status: 'Available' },
	{ callSign: 'Charlie-1', id: 'FU-311', status: 'En route' }
];

/**
 * Navigate and let the dev server's module requests settle.
 *
 * `networkidle` is used only to quiet vite's module loading before a test attaches a
 * request listener — it is NOT a hydration signal (it fires 500ms after the last
 * response; hydration is JS work that can finish later). Hydration is handled where it
 * actually matters, in `selectRow`.
 */
async function gotoUnits(page: Page) {
	await page.goto('/units');
	await expect(page.getByRole('heading', { name: 'Dispatch console' })).toBeVisible();
	await page.waitForLoadState('networkidle');
}

/**
 * Click a unit row and wait for the selection to actually take.
 *
 * The page is server-rendered, so rows are in the DOM — and pass Playwright's
 * actionability checks — before Svelte attaches its handlers. A click in that window
 * lands on a real button and silently does nothing. Retrying the click until
 * `aria-current` appears is the only signal that is genuinely tied to interactivity.
 * Clicks are pure DOM events, so retrying generates no network traffic.
 */
async function selectRow(page: Page, index: number) {
	const row = unitRows(page).nth(index);
	await expect
		.poll(
			async () => {
				await row.click();
				return row.getAttribute('aria-current');
			},
			{ message: `row ${index} never became selectable — did the page hydrate?` }
		)
		.toBe('true');
	return row;
}

/** The selectable unit rows, scoped to the list so unrelated buttons never match. */
const unitRows = (page: Page) => page.locator('ul li button');

const detailPane = (page: Page) => page.locator('[aria-live="polite"]');

/** `CardTitle` renders a `<div data-slot="card-title">`, not a heading element. */
const cardTitles = (page: Page) => page.locator('[data-slot="card-title"]');

/** Row text is `callSign\nunitId\nstatus`; parse it rather than substring-matching. */
async function readRow(row: Locator) {
	const [callSign, id, status] = (await row.innerText()).trim().split('\n');
	return { callSign: callSign?.trim(), id: id?.trim(), status: status?.trim() };
}

/** Just the background colour — the signal the selected-row treatment actually changes. */
const background = (el: Locator) => el.evaluate((n) => getComputedStyle(n).backgroundColor);

async function paint(el: Locator) {
	return el.evaluate((node) => {
		const s = getComputedStyle(node);
		return {
			backgroundColor: s.backgroundColor,
			color: s.color,
			borderColor: s.borderColor,
			boxShadow: s.boxShadow,
			outline: s.outline,
			fontWeight: s.fontWeight
		};
	});
}

test.describe('dispatch console', () => {
	test('issues no request to any other origin', async ({ page, baseURL }) => {
		const origin = new URL(baseURL as string).origin;
		const offOrigin: string[] = [];
		// An allowlist of known auth paths would pass for an IdP hosted anywhere else.
		// Assert the stronger property instead: nothing leaves this origin at all.
		page.on('request', (request) => {
			if (!request.url().startsWith(origin) && !request.url().startsWith('data:')) {
				offOrigin.push(request.url());
			}
		});

		await gotoUnits(page);

		await expect(page).toHaveURL(/\/units$/);
		expect(offOrigin, 'the console must not talk to anything but its own origin').toEqual([]);
	});

	test('redirects bare / to the console', async ({ page }) => {
		await page.goto('/');
		await expect(page).toHaveURL(/\/units$/);
		await expect(page.getByRole('heading', { name: 'Dispatch console' })).toBeVisible();
	});

	test('lists the roster in its declared order, with call sign, id and status', async ({
		page
	}) => {
		await gotoUnits(page);
		const rows = unitRows(page);

		const count = await rows.count();
		expect(count).toBeGreaterThanOrEqual(4);
		expect(count).toBeLessThanOrEqual(5);
		expect(count).toBe(EXPECTED_ROWS.length);

		for (let i = 0; i < count; i++) {
			const row = await readRow(rows.nth(i));
			// Exact equality, not substring containment: "Available soon" must not pass.
			expect(row, `row ${i}`).toEqual(EXPECTED_ROWS[i]);
			expect(row.id).toMatch(UNIT_ID);
			expect(STATUSES).toContain(row.status);
		}

		const ids = EXPECTED_ROWS.map((r) => r.id);
		expect(new Set(ids).size, 'unit ids are unique').toBe(ids.length);
	});

	test('lays both panes out side by side without horizontal page scroll', async ({ page }) => {
		await page.setViewportSize({ width: 1280, height: 900 });
		await gotoUnits(page);

		const list = (await page.locator('ul').first().boundingBox())!;
		const detail = (await detailPane(page).boundingBox())!;
		// The units list sits to the left of the detail pane, not stacked above it.
		expect(list.x + list.width).toBeLessThanOrEqual(detail.x + 1);

		const overflows = await page.evaluate(
			() => document.documentElement.scrollWidth > document.documentElement.clientWidth
		);
		expect(overflows, 'the page must not scroll horizontally at 1280px').toBe(false);
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
		const { id, callSign, status } = await readRow(first);

		await selectRow(page, 0);

		const pane = detailPane(page);
		await expect(pane).not.toContainText(EMPTY_PROMPT);
		await expect(pane).toContainText(id);
		await expect(pane).toContainText(callSign);
		await expect(pane).toContainText(status);
		await expect(pane).toContainText('Call sign');
		await expect(pane).toContainText('Unit ID');
		await expect(pane).toContainText('Status');
		await expect(page.locator('[aria-current="true"]')).toHaveCount(1);
		await expect(cardTitles(page).filter({ hasText: callSign })).toBeVisible();
	});

	test('renders the selected row visually differently from the unselected ones', async ({
		page
	}) => {
		await gotoUnits(page);
		const rows = unitRows(page);

		const before = await background(rows.nth(0));
		await selectRow(page, 0);

		// Move the pointer off the row: `:hover` also paints a background, and would
		// otherwise pass this test on a console with no selected state at all.
		await page.mouse.move(5, 5);

		// Poll rather than read once — the Button carries `transition-all`, so the
		// background is still animating away from its hover value for ~150ms after the
		// pointer leaves, and a single read catches a mid-transition colour.
		await expect
			.poll(async () => (await background(rows.nth(0))) === (await background(rows.nth(1))), {
				message: 'the selected row must not look identical to an unselected row'
			})
			.toBe(false);

		// Selection must be visible, not merely announced to assistive tech.
		expect(await background(rows.nth(0)), 'selecting must change how the row is painted').not.toBe(
			before
		);
	});

	test('switching selection replaces the detail contents', async ({ page }) => {
		await gotoUnits(page);
		const rows = unitRows(page);

		const first = await readRow(rows.nth(0));
		const second = await readRow(rows.nth(1));

		await selectRow(page, 0);
		await expect(detailPane(page)).toContainText(first.id);

		await selectRow(page, 1);
		const pane = detailPane(page);
		await expect(pane).toContainText(second.id);
		await expect(pane).not.toContainText(first.id);
		await expect(page.locator('[aria-current="true"]')).toHaveCount(1);
	});

	for (const key of ['Enter', 'Space'] as const) {
		test(`a unit can be selected with the keyboard (${key})`, async ({ page }) => {
			await gotoUnits(page);
			const first = unitRows(page).first();
			const { id } = await readRow(first);

			await first.focus();
			await expect(first).toBeFocused();

			// The design system's focus ring must be present on the focused row.
			const ring = await first.evaluate((node) => {
				const s = getComputedStyle(node);
				return { outline: s.outlineWidth, shadow: s.boxShadow };
			});
			expect(
				ring.outline !== '0px' || ring.shadow !== 'none',
				'focused row shows a focus ring'
			).toBe(true);

			await page.keyboard.press(key);

			await expect(detailPane(page)).toContainText(id);
			await expect(first).toHaveAttribute('aria-current', 'true');
		});
	}

	test('selection issues no network request', async ({ page }) => {
		await gotoUnits(page);

		const requests: string[] = [];
		page.on('request', (request) => requests.push(request.url()));

		const second = unitRows(page).nth(1);
		const { id } = await readRow(second);
		await selectRow(page, 1);
		// Tie the guard to the click: the pane must show the unit that was just clicked.
		await expect(detailPane(page)).toContainText(id);

		// Settle, so a fire-and-forget request racing the assertion is still caught.
		await page.waitForTimeout(250);
		expect(requests, 'selection is client-side only').toEqual([]);
	});

	test('the detail pane exposes no mutating control', async ({ page }) => {
		await gotoUnits(page);
		await selectRow(page, 0);
		await expect(detailPane(page)).toContainText('Call sign');

		const pane = detailPane(page);
		// The baseline is read-only: no dispatch/reassign/status-change affordance.
		await expect(pane.locator('button, a, input, select, textarea, [role="button"]')).toHaveCount(
			0
		);
	});

	test('renders legibly in the light theme', async ({ page }) => {
		// The app pins ThemeProvider to `mode="dark"`, and that provider applies the mode
		// unconditionally — it consults neither storage nor prefers-color-scheme. So
		// `emulateMedia` and a seeded localStorage key both do nothing, and a test relying
		// on either silently re-tests the dark path. Drive the light palette directly by
		// dropping the `.dark` class after hydration instead, which is what actually
		// selects the light tokens.
		await gotoUnits(page);

		const dark = await paint(page.locator('body'));

		await page.evaluate(() => document.documentElement.classList.remove('dark'));
		const light = await paint(page.locator('body'));

		expect(light.backgroundColor, 'light mode must repaint the page').not.toBe(
			dark.backgroundColor
		);

		// Prove the light tokens actually resolved rather than merely differing: a light
		// surface with dark text, and real contrast between them.
		const luminance = (rgb: string) => {
			const [r, g, b] = rgb.match(/\d+/g)!.map(Number);
			return (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;
		};
		const bg = luminance(light.backgroundColor);
		const fg = luminance(light.color);
		expect(bg, 'light-mode background is light').toBeGreaterThan(0.7);
		expect(fg, 'light-mode text is dark').toBeLessThan(0.4);
		expect(bg - fg, 'light-mode text contrasts with its background').toBeGreaterThan(0.4);

		// The console still works under the light palette.
		await expect(page.getByRole('heading', { name: 'Dispatch console' })).toBeVisible();
		await selectRow(page, 0);
		await expect(detailPane(page)).toContainText('Call sign');
	});
});
