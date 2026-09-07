import { describe, expect, it } from 'vitest';
import { parseCapabilities, parseUnitForm, valuesFrom } from './forms';

function form(fields: Record<string, string>): FormData {
	const data = new FormData();
	for (const [k, v] of Object.entries(fields)) data.set(k, v);
	return data;
}

const valid = {
	unitCode: 'FU-401',
	callSign: 'Delta-1',
	status: 'Available',
	unitType: 'Ambulance',
	station: 'Bedok Station 3',
	sector: 'Sector 9 · Bedok',
	radioChannel: 'TAC-3',
	shift: 'Day · 07:00-19:00',
	capabilities: 'ALS, Water rescue'
};

describe('parseCapabilities', () => {
	it('splits on commas, trims, and drops blanks and duplicates', () => {
		expect(parseCapabilities(' ALS ,, Water rescue , ALS ')).toEqual(['ALS', 'Water rescue']);
		expect(parseCapabilities('')).toEqual([]);
		expect(parseCapabilities('  ,  ')).toEqual([]);
	});
});

describe('valuesFrom', () => {
	it('reads every field trimmed and tolerates missing ones', () => {
		const v = valuesFrom(form({ callSign: '  Delta-1 ' }));
		expect(v.callSign).toBe('Delta-1');
		expect(v.station).toBe('');
	});
});

describe('parseUnitForm — add', () => {
	it('accepts a valid form and builds the input', () => {
		const r = parseUnitForm(form(valid), { requireCode: true });
		expect(r.ok).toBe(true);
		if (!r.ok) return;
		expect(r.input).toEqual({
			unitCode: 'FU-401',
			callSign: 'Delta-1',
			status: 'Available',
			unitType: 'Ambulance',
			station: 'Bedok Station 3',
			sector: 'Sector 9 · Bedok',
			radioChannel: 'TAC-3',
			shift: 'Day · 07:00-19:00',
			capabilities: ['ALS', 'Water rescue']
		});
		expect(r.occLock).toBeNull();
	});

	it('reports every blank required field at once, with the console wording', () => {
		const r = parseUnitForm(form({ ...valid, callSign: '', station: '   ', unitCode: '' }), {
			requireCode: true
		});
		expect(r.ok).toBe(false);
		if (r.ok) return;
		expect(Object.keys(r.errors).sort()).toEqual(['callSign', 'station', 'unitCode']);
		expect(r.errors.callSign).toBe('Call sign is required.');
		// Values are echoed so the form can re-render what the user typed.
		expect(r.values.unitType).toBe('Ambulance');
	});

	it('rejects a status outside the vocabulary', () => {
		const r = parseUnitForm(form({ ...valid, status: 'Out of service' }), { requireCode: true });
		expect(r.ok).toBe(false);
		if (r.ok) return;
		expect(Object.keys(r.errors)).toEqual(['status']);
		expect(r.errors.status).toMatch(/Available, En route, Idle/);
	});

	it('caps field lengths', () => {
		const r = parseUnitForm(
			form({ ...valid, unitCode: 'X'.repeat(33), callSign: 'Y'.repeat(121) }),
			{
				requireCode: true
			}
		);
		expect(r.ok).toBe(false);
		if (r.ok) return;
		expect(r.errors.unitCode).toMatch(/32 characters/);
		expect(r.errors.callSign).toMatch(/120 characters/);
	});
});

describe('parseUnitForm — edit', () => {
	it('does not require the unit code and carries occLock', () => {
		const r = parseUnitForm(form({ ...valid, unitCode: '', occLock: '7' }), { requireCode: false });
		expect(r.ok).toBe(true);
		if (!r.ok) return;
		expect(r.occLock).toBe(7);
	});

	it('treats a non-integer occLock as absent rather than guessing', () => {
		const r = parseUnitForm(form({ ...valid, occLock: 'seven' }), { requireCode: false });
		expect(r.ok).toBe(true);
		if (!r.ok) return;
		expect(r.occLock).toBeNull();
	});

	it('accepts occLock of zero — version 0 is a real version', () => {
		const r = parseUnitForm(form({ ...valid, occLock: '0' }), { requireCode: false });
		expect(r.ok && r.occLock).toBe(0);
	});
});
