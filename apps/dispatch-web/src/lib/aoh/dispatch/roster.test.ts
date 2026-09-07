import { describe, expect, it } from 'vitest';
import { listUnits, type FieldUnit, type UnitStatus } from './roster';

const STATUSES: UnitStatus[] = ['Available', 'En route', 'Idle'];

/**
 * The baseline roster, pinned.
 *
 * The workshop's premise is that every attendee sees an identical console, so the
 * contents AND the order are part of the contract (field-unit-roster spec:
 * "Roster is identical across attendees and runs", "Accessor returns the full roster,
 * in the roster's declared order"). Comparing `listUnits()` to itself would only catch
 * nondeterminism; it would sail through a reorder, a rename, or a wholesale swap.
 *
 * If you intend to change the roster, change it here too — deliberately.
 */
const EXPECTED: FieldUnit[] = [
	{ id: 'FU-101', callSign: 'Alpha-1', status: 'Available' },
	{ id: 'FU-102', callSign: 'Alpha-2', status: 'En route' },
	{ id: 'FU-204', callSign: 'Bravo-1', status: 'Idle' },
	{ id: 'FU-205', callSign: 'Bravo-2', status: 'Available' },
	{ id: 'FU-311', callSign: 'Charlie-1', status: 'En route' }
];

describe('listUnits', () => {
	it('returns exactly the baseline roster, in declared order', () => {
		expect(listUnits()).toStrictEqual(EXPECTED);
	});

	it('returns between 4 and 5 units', () => {
		const units = listUnits();
		expect(units.length).toBeGreaterThanOrEqual(4);
		expect(units.length).toBeLessThanOrEqual(5);
	});

	it('gives every unit a non-empty call sign and id', () => {
		for (const unit of listUnits()) {
			expect(unit.callSign.length).toBeGreaterThan(0);
			expect(unit.id.length).toBeGreaterThan(0);
		}
	});

	it('keeps identifiers unique', () => {
		const ids = listUnits().map((u) => u.id);
		expect(new Set(ids).size).toBe(ids.length);
	});

	it('uses only the closed status vocabulary', () => {
		for (const unit of listUnits()) {
			expect(STATUSES).toContain(unit.status);
		}
	});

	it('represents every status at least once', () => {
		const seen = new Set(listUnits().map((u) => u.status));
		for (const status of STATUSES) {
			expect(seen).toContain(status);
		}
	});

	it('returns the same units in the same order on every call', () => {
		expect(listUnits()).toStrictEqual(EXPECTED);
		expect(listUnits()).toStrictEqual(EXPECTED);
	});

	it('does not let a caller mutate the baseline for the next reader', () => {
		const first = listUnits();
		first.pop();
		first[0].callSign = 'MUTATED';

		// Compared against the pinned roster, not against another live call — otherwise
		// a mutation that corrupted every subsequent call would compare equal to itself.
		expect(listUnits()).toStrictEqual(EXPECTED);
	});
});
