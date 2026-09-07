/**
 * The baseline field-unit roster.
 *
 * The data here is hardcoded on purpose: the workshop baseline must render an
 * identical console for every attendee with no backend, no container, and no
 * network call. See `openspec/changes/baseline-dispatch-console`.
 *
 * Read the roster through `listUnits()` — never by importing `ROSTER` directly.
 * That accessor is the seam a later change swaps for a service call without
 * touching the console page.
 */

/** A field unit's operational state. Closed union: adding a state is a typed change. */
export type UnitStatus = 'Available' | 'En route' | 'Idle';

export interface FieldUnit {
	/** Stable, unique machine key. */
	id: string;
	/** Human-readable name an operator uses to address the unit. */
	callSign: string;
	status: UnitStatus;
}

const ROSTER: readonly FieldUnit[] = [
	{ id: 'FU-101', callSign: 'Alpha-1', status: 'Available' },
	{ id: 'FU-102', callSign: 'Alpha-2', status: 'En route' },
	{ id: 'FU-204', callSign: 'Bravo-1', status: 'Idle' },
	{ id: 'FU-205', callSign: 'Bravo-2', status: 'Available' },
	{ id: 'FU-311', callSign: 'Charlie-1', status: 'En route' }
];

/**
 * Every unit in the roster, in declared order.
 *
 * Returns fresh objects, not references into `ROSTER`: a spread of the array
 * alone would still hand out the shared unit objects, so one caller mutating a
 * call sign would change the baseline for the next reader. The roster must read
 * identically on every call.
 */
export function listUnits(): FieldUnit[] {
	return ROSTER.map((unit) => ({ ...unit }));
}
