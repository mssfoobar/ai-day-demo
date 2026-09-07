/**
 * The field-unit model as the console sees it.
 *
 * Kept in a plain module (no `$env`, no `.server` suffix) so client components can import
 * the types without dragging the service client — and its URL — into the browser bundle.
 */

/** A field unit's operational state. */
export type UnitStatus = 'Available' | 'En route' | 'Idle';

/**
 * The authoritative constraint is the CHECK on `dispatch.unit.status`. This list is the
 * frontend's runtime guard: values arrive over the wire, so a TypeScript union alone
 * cannot constrain them.
 */
export const KNOWN_STATUSES: readonly UnitStatus[] = ['Available', 'En route', 'Idle'];

export interface Crew {
	name: string;
	role: string;
}

export interface Assignment {
	incidentCode: string;
	title: string;
	priority: string;
	location: string;
	since: string;
}

export interface FieldUnit {
	id: string;
	callSign: string;
	status: UnitStatus;
	unitType: string;
	station: string;
	sector: string;
	radioChannel: string;
	shift: string;
	capabilities: string[];
	crew: Crew[];
	/** Absent — not an empty object — when the unit is not committed to an incident. */
	assignment?: Assignment;
	lastContact: string;
	/**
	 * Optimistic-concurrency version. Echoed on every edit/delete; a mismatch means someone
	 * else wrote first and the service answers 409 rather than overwriting.
	 */
	occLock: number;
}

/** The writable subset of a unit — what add and edit send. Crew and assignment are not editable yet. */
export interface UnitInput {
	unitCode: string;
	callSign: string;
	status: UnitStatus;
	unitType: string;
	station: string;
	sector: string;
	radioChannel: string;
	shift: string;
	capabilities: string[];
}

/** Field names a form can carry, and the key an error is reported under. */
export type UnitField = keyof UnitInput;
