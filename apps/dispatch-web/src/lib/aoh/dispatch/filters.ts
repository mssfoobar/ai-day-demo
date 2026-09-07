/**
 * Search, filter and sort for the units list. Pure, so the console's interactivity is
 * testable without rendering anything.
 */
import type { FieldUnit, UnitStatus } from './types';

export type SortKey = 'callSign' | 'status' | 'lastContact' | 'priority';

export const SORT_OPTIONS: ReadonlyArray<{ value: SortKey; label: string }> = [
	{ value: 'callSign', label: 'Call sign' },
	{ value: 'status', label: 'Status' },
	{ value: 'priority', label: 'Priority' },
	{ value: 'lastContact', label: 'Last contact' }
];

/** Everything a dispatcher might type to find a unit. */
function haystack(unit: FieldUnit): string {
	return [
		unit.callSign,
		unit.id,
		unit.unitType,
		unit.station,
		unit.sector,
		unit.radioChannel,
		unit.assignment?.incidentCode,
		unit.assignment?.title,
		unit.assignment?.location,
		...unit.crew.map((c) => c.name),
		...unit.capabilities
	]
		.filter(Boolean)
		.join(' ')
		.toLowerCase();
}

/** Every whitespace-separated term must match somewhere; order does not matter. */
export function matchesQuery(unit: FieldUnit, query: string): boolean {
	const terms = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
	if (terms.length === 0) return true;
	const text = haystack(unit);
	return terms.every((term) => text.includes(term));
}

/**
 * Apply the search box and the status toggles.
 *
 * An empty `statuses` selection means "no filter", not "nothing" — clearing every toggle
 * must show the whole fleet, or the list would go blank for no visible reason.
 */
export function filterUnits(
	units: FieldUnit[],
	query: string,
	statuses: readonly UnitStatus[]
): FieldUnit[] {
	return units.filter(
		(unit) => (statuses.length === 0 || statuses.includes(unit.status)) && matchesQuery(unit, query)
	);
}

const STATUS_ORDER: Record<UnitStatus, number> = { 'En route': 0, Available: 1, Idle: 2 };

/** P1 first; unassigned units sort after every assigned one. */
function priorityRank(unit: FieldUnit): number {
	const p = unit.assignment?.priority;
	if (!p) return 99;
	const n = Number.parseInt(p.replace(/^P/i, ''), 10);
	return Number.isNaN(n) ? 98 : n;
}

/** Stable sort; ties fall back to call sign so the order never flickers. */
export function sortUnits(units: FieldUnit[], key: SortKey): FieldUnit[] {
	const byCallSign = (a: FieldUnit, b: FieldUnit) => a.callSign.localeCompare(b.callSign);
	const compare: Record<SortKey, (a: FieldUnit, b: FieldUnit) => number> = {
		callSign: byCallSign,
		status: (a, b) => STATUS_ORDER[a.status] - STATUS_ORDER[b.status] || byCallSign(a, b),
		priority: (a, b) => priorityRank(a) - priorityRank(b) || byCallSign(a, b),
		// Most recent contact first.
		lastContact: (a, b) => Date.parse(b.lastContact) - Date.parse(a.lastContact) || byCallSign(a, b)
	};
	return [...units].sort(compare[key]);
}

export interface FleetSummary {
	total: number;
	assigned: number;
	free: number;
	/** Units committed to a P1 incident, most recently committed first. */
	p1: FieldUnit[];
	/** The unit heard from least recently, or null for an empty fleet. */
	quietest: FieldUnit | null;
}

/**
 * What the detail pane shows when nothing is selected: the fleet at a glance. No new
 * data — the same roster read a different way, so the pane is never blank.
 */
export function fleetSummary(units: FieldUnit[]): FleetSummary {
	const assigned = units.filter((u) => u.assignment).length;
	const p1 = units
		.filter((u) => u.assignment?.priority === 'P1')
		.sort((a, b) => Date.parse(b.assignment!.since) - Date.parse(a.assignment!.since));
	const quietest = units.reduce<FieldUnit | null>(
		(oldest, u) =>
			oldest === null || Date.parse(u.lastContact) < Date.parse(oldest.lastContact) ? u : oldest,
		null
	);
	return { total: units.length, assigned, free: units.length - assigned, p1, quietest };
}

/** Count per status over the FULL roster — the summary must not shrink when filtering. */
export function countByStatus(units: FieldUnit[]): Record<UnitStatus, number> {
	const counts: Record<UnitStatus, number> = { Available: 0, 'En route': 0, Idle: 0 };
	for (const unit of units) counts[unit.status] += 1;
	return counts;
}
