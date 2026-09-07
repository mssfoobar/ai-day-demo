import { describe, expect, it } from 'vitest';
import {
	DispatchServiceError,
	toFieldUnit,
	unitsFromEnvelope,
	type FieldUnit
} from './units.server';

/** One unit exactly as the service puts it on the wire. */
function wireUnit(overrides: Record<string, unknown> = {}) {
	return {
		unit_code: 'FU-101',
		call_sign: 'Alpha-1',
		status: 'Available',
		unit_type: 'Ambulance',
		station: 'Marina Bay Station 4',
		sector: 'Sector 4 · Marina Bay',
		radio_channel: 'TAC-2',
		shift: 'Day · 07:00-19:00',
		capabilities: ['ALS', 'Water rescue'],
		crew: [
			{ name: 'J. Tan', role: 'Paramedic' },
			{ name: 'M. Lim', role: 'EMT' }
		],
		last_contact: '2026-09-07T13:41:00Z',
		...overrides
	};
}

describe('toFieldUnit', () => {
	it('maps snake_case wire fields onto the app shape', () => {
		const unit = toFieldUnit(wireUnit());

		expect(unit).toMatchObject<Partial<FieldUnit>>({
			id: 'FU-101',
			callSign: 'Alpha-1',
			status: 'Available',
			unitType: 'Ambulance',
			station: 'Marina Bay Station 4',
			sector: 'Sector 4 · Marina Bay',
			radioChannel: 'TAC-2',
			shift: 'Day · 07:00-19:00',
			lastContact: '2026-09-07T13:41:00Z'
		});
	});

	it('preserves crew order', () => {
		const unit = toFieldUnit(wireUnit());
		expect(unit.crew.map((c) => c.name)).toEqual(['J. Tan', 'M. Lim']);
	});

	it('leaves assignment absent when the unit is unassigned', () => {
		const unit = toFieldUnit(wireUnit());
		// Absent, not an empty object — the UI branches on presence.
		expect(unit.assignment).toBeUndefined();
	});

	it('maps an assignment when present', () => {
		const unit = toFieldUnit(
			wireUnit({
				assignment: {
					incident_code: 'INC-2841',
					title: 'Cardiac arrest',
					priority: 'P1',
					location: '12 Raffles Quay',
					since: '2026-09-07T13:12:00Z'
				}
			})
		);

		expect(unit.assignment).toEqual({
			incidentCode: 'INC-2841',
			title: 'Cardiac arrest',
			priority: 'P1',
			location: '12 Raffles Quay',
			since: '2026-09-07T13:12:00Z'
		});
	});

	it('tolerates a unit with no capabilities and no crew', () => {
		const unit = toFieldUnit(wireUnit({ capabilities: [], crew: [] }));
		expect(unit.capabilities).toEqual([]);
		expect(unit.crew).toEqual([]);
	});

	it('rejects a status outside the known vocabulary', () => {
		// The database CHECK is the real constraint; this is the frontend's guard against
		// rendering a unit it cannot colour or count.
		expect(() => toFieldUnit(wireUnit({ status: 'Out of service' }))).toThrow(DispatchServiceError);
	});
});

describe('unitsFromEnvelope', () => {
	it('unwraps the AOH success envelope', () => {
		const units = unitsFromEnvelope({
			data: [wireUnit(), wireUnit({ unit_code: 'FU-102', call_sign: 'Alpha-2' })],
			message: '',
			sent_at: '2026-09-07T13:41:00Z'
		});

		expect(units.map((u) => u.id)).toEqual(['FU-101', 'FU-102']);
	});

	it('preserves the order the service returned', () => {
		const units = unitsFromEnvelope({
			data: [wireUnit({ unit_code: 'FU-311' }), wireUnit({ unit_code: 'FU-101' })]
		});
		expect(units.map((u) => u.id)).toEqual(['FU-311', 'FU-101']);
	});

	it('rejects a body that is not the envelope', () => {
		// A bare array is what a service that forgot the envelope would return.
		expect(() => unitsFromEnvelope([wireUnit()])).toThrow(DispatchServiceError);
		expect(() => unitsFromEnvelope({ units: [] })).toThrow(DispatchServiceError);
	});
});
