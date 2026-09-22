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
	/** What the incident is about, as the call was taken. */
	description: string;
	priority: string;
	location: string;
	/** Absent when the address has not been resolved to a coordinate. */
	point?: IncidentPoint;
	since: string;
}

/** An incident's location as a coordinate. */
export interface IncidentPoint {
	lon: number;
	lat: number;
}

/**
 * Where a unit was last located.
 *
 * `at` is the **fix time** — when the location was reported — which is deliberately not
 * the unit's last contact. A unit can be heard from without reporting a position, so the
 * two move independently and the detail pane labels them separately.
 */
export interface Position {
	lon: number;
	lat: number;
	at: string;
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
	/** Absent — not a zero coordinate — when the unit has never reported a location. */
	position?: Position;
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
	/**
	 * Carried through, not edited. Positions are seeded and set through the API, not
	 * authored in the console — but a write is a **replace**, so the form has to send the
	 * unit's existing position back or the write would clear it and the unit's marker
	 * would vanish from the map. Round-tripping `at` unchanged is also what keeps the fix
	 * time from advancing on an edit that reported no new location.
	 */
	position?: Position;
}

/**
 * Field names a form can carry, and the key an error is reported under.
 *
 * Spelled out rather than `keyof UnitInput`: `position` is round-tripped through hidden
 * inputs rather than being an editable field, so it has no label and no required check.
 */
export type UnitField =
	| 'unitCode'
	| 'callSign'
	| 'status'
	| 'unitType'
	| 'station'
	| 'sector'
	| 'radioChannel'
	| 'shift'
	| 'capabilities';
