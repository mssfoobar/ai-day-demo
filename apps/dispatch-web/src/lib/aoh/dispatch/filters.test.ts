import { describe, expect, it } from 'vitest';
import { countByStatus, filterUnits, fleetSummary, matchesQuery, sortUnits } from './filters';
import type { FieldUnit } from './types';

const unit = (overrides: Partial<FieldUnit>): FieldUnit => ({
	id: 'FU-000',
	callSign: 'Zulu-9',
	status: 'Idle',
	unitType: 'Van',
	station: 'Nowhere',
	sector: 'Sector 0',
	radioChannel: 'TAC-0',
	shift: 'Day',
	capabilities: [],
	crew: [],
	lastContact: '2026-09-07T12:00:00Z',
	...overrides
});

const alpha1 = unit({
	id: 'FU-101',
	callSign: 'Alpha-1',
	status: 'Available',
	unitType: 'Ambulance',
	station: 'Marina Bay Station 4',
	capabilities: ['ALS'],
	crew: [{ name: 'J. Tan', role: 'Paramedic' }],
	lastContact: '2026-09-07T12:57:00Z'
});
const alpha2 = unit({
	id: 'FU-102',
	callSign: 'Alpha-2',
	status: 'En route',
	unitType: 'Ambulance',
	assignment: {
		incidentCode: 'INC-2841',
		title: 'Cardiac arrest',
		priority: 'P1',
		location: '12 Raffles Quay',
		since: '2026-09-07T12:46:00Z'
	},
	lastContact: '2026-09-07T12:59:00Z'
});
const bravo1 = unit({
	id: 'FU-204',
	callSign: 'Bravo-1',
	status: 'Idle',
	unitType: 'Fire engine',
	lastContact: '2026-09-07T12:38:00Z'
});
const charlie1 = unit({
	id: 'FU-311',
	callSign: 'Charlie-1',
	status: 'En route',
	unitType: 'Patrol car',
	assignment: {
		incidentCode: 'INC-2839',
		title: 'Traffic obstruction',
		priority: 'P3',
		location: 'Nicoll Highway',
		since: '2026-09-07T12:25:00Z'
	},
	lastContact: '2026-09-07T12:58:00Z'
});
const ALL = [alpha1, alpha2, bravo1, charlie1];

describe('matchesQuery', () => {
	it('matches on call sign, id, type, station, incident, crew and capability', () => {
		for (const q of ['alpha-1', 'fu-101', 'ambulance', 'marina', 'j. tan', 'als']) {
			expect(matchesQuery(alpha1, q), q).toBe(true);
		}
		expect(matchesQuery(alpha2, 'inc-2841')).toBe(true);
		expect(matchesQuery(alpha2, 'cardiac')).toBe(true);
	});

	it('is case-insensitive and requires every term', () => {
		expect(matchesQuery(alpha2, 'AMBULANCE cardiac')).toBe(true);
		expect(matchesQuery(alpha2, 'ambulance traffic')).toBe(false);
	});

	it('treats an empty or whitespace query as a match', () => {
		expect(matchesQuery(bravo1, '')).toBe(true);
		expect(matchesQuery(bravo1, '   ')).toBe(true);
	});
});

describe('filterUnits', () => {
	it('shows the whole fleet when no status is selected', () => {
		expect(filterUnits(ALL, '', [])).toHaveLength(4);
	});

	it('narrows by status', () => {
		expect(filterUnits(ALL, '', ['En route']).map((u) => u.id)).toEqual(['FU-102', 'FU-311']);
	});

	it('combines status with search', () => {
		expect(filterUnits(ALL, 'traffic', ['En route']).map((u) => u.id)).toEqual(['FU-311']);
		expect(filterUnits(ALL, 'traffic', ['Available'])).toEqual([]);
	});
});

describe('sortUnits', () => {
	it('sorts by call sign', () => {
		expect(
			sortUnits([charlie1, bravo1, alpha2, alpha1], 'callSign').map((u) => u.callSign)
		).toEqual(['Alpha-1', 'Alpha-2', 'Bravo-1', 'Charlie-1']);
	});

	it('sorts En route first, then Available, then Idle, ties by call sign', () => {
		expect(sortUnits(ALL, 'status').map((u) => u.id)).toEqual([
			'FU-102',
			'FU-311',
			'FU-101',
			'FU-204'
		]);
	});

	it('sorts P1 first and unassigned last', () => {
		expect(sortUnits(ALL, 'priority').map((u) => u.id)).toEqual([
			'FU-102',
			'FU-311',
			'FU-101',
			'FU-204'
		]);
	});

	it('sorts most recent contact first', () => {
		expect(sortUnits(ALL, 'lastContact').map((u) => u.id)).toEqual([
			'FU-102',
			'FU-311',
			'FU-101',
			'FU-204'
		]);
	});

	it('does not mutate its input', () => {
		const input = [bravo1, alpha1];
		sortUnits(input, 'callSign');
		expect(input.map((u) => u.id)).toEqual(['FU-204', 'FU-101']);
	});
});

describe('fleetSummary', () => {
	it('splits assigned from free and lists P1 units', () => {
		const s = fleetSummary(ALL);
		expect(s.total).toBe(4);
		expect(s.assigned).toBe(2);
		expect(s.free).toBe(2);
		expect(s.p1.map((u) => u.id)).toEqual(['FU-102']);
	});

	it('names the unit heard from least recently', () => {
		expect(fleetSummary(ALL).quietest?.id).toBe('FU-204');
	});

	it('copes with an empty fleet', () => {
		expect(fleetSummary([])).toEqual({ total: 0, assigned: 0, free: 0, p1: [], quietest: null });
	});
});

describe('countByStatus', () => {
	it('counts every status, including zero', () => {
		expect(countByStatus([alpha1, alpha2, charlie1])).toEqual({
			Available: 1,
			'En route': 2,
			Idle: 0
		});
	});
});
