import { describe, expect, it } from 'vitest';
import { listUnits, type UnitStatus } from './roster';

const STATUSES: UnitStatus[] = ['Available', 'En route', 'Idle'];

describe('listUnits', () => {
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
		expect(listUnits()).toStrictEqual(listUnits());
	});

	it('does not let a caller mutate the baseline for the next reader', () => {
		const first = listUnits();
		first.pop();
		first[0].callSign = 'MUTATED';
		expect(listUnits()).toStrictEqual(listUnits());
		expect(listUnits()[0].callSign).not.toBe('MUTATED');
	});
});
