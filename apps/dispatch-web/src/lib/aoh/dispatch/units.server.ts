/**
 * Client for the dispatch service's field-unit API.
 *
 * `.server.ts` on purpose: importing this from client code is a build error, so the
 * service URL can never reach the browser bundle. The console reads units through a
 * server `load`, which keeps the browser same-origin — no CORS, no exposed upstream.
 *
 * `listUnits()` is the seam the console has always read through. It used to return a
 * hardcoded array; now it calls the service. The page did not change to accommodate it.
 */
import { env } from '$env/dynamic/private';

/** A field unit's operational state. */
export type UnitStatus = 'Available' | 'En route' | 'Idle';

/**
 * The authoritative constraint is the CHECK on `dispatch.unit.status`. This list is the
 * frontend's runtime guard: values arrive over the wire, so a TypeScript union alone
 * cannot constrain them.
 */
const KNOWN_STATUSES: readonly UnitStatus[] = ['Available', 'En route', 'Idle'];

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
}

/** Raised when the service cannot be reached or answers unusably. */
export class DispatchServiceError extends Error {}

/** The AOH success envelope every service response is wrapped in. */
interface Envelope<T> {
	data?: T;
	message?: string;
	sent_at?: string;
}

interface WireCrew {
	name?: unknown;
	role?: unknown;
}

interface WireAssignment {
	incident_code?: unknown;
	title?: unknown;
	priority?: unknown;
	location?: unknown;
	since?: unknown;
}

interface WireUnit {
	unit_code?: unknown;
	call_sign?: unknown;
	status?: unknown;
	unit_type?: unknown;
	station?: unknown;
	sector?: unknown;
	radio_channel?: unknown;
	shift?: unknown;
	capabilities?: unknown;
	crew?: unknown;
	assignment?: unknown;
	last_contact?: unknown;
}

const str = (value: unknown): string => (typeof value === 'string' ? value : '');

function isUnitStatus(value: unknown): value is UnitStatus {
	return typeof value === 'string' && (KNOWN_STATUSES as readonly string[]).includes(value);
}

function toCrew(value: unknown): Crew[] {
	if (!Array.isArray(value)) return [];
	return value.map((member) => {
		const wire = (member ?? {}) as WireCrew;
		return { name: str(wire.name), role: str(wire.role) };
	});
}

function toAssignment(value: unknown): Assignment | undefined {
	// The service omits the key entirely for an unassigned unit; keep it absent rather
	// than manufacturing an empty object the UI would have to special-case.
	if (value === null || typeof value !== 'object') return undefined;
	const wire = value as WireAssignment;
	return {
		incidentCode: str(wire.incident_code),
		title: str(wire.title),
		priority: str(wire.priority),
		location: str(wire.location),
		since: str(wire.since)
	};
}

/**
 * Map one wire unit to the app's shape.
 *
 * Exported for unit tests: the mapping is where snake_case, absent assignments and
 * unknown statuses are actually handled, and it is worth testing without a live service.
 */
export function toFieldUnit(value: unknown): FieldUnit {
	const wire = (value ?? {}) as WireUnit;

	const status = wire.status;
	if (!isUnitStatus(status)) {
		// Fail loudly rather than rendering a unit whose status the UI cannot colour or
		// count. A new status in the database is a deliberate change, not a surprise.
		throw new DispatchServiceError(
			`unit ${str(wire.unit_code) || '(unknown)'} has unrecognised status ${JSON.stringify(status)}`
		);
	}

	return {
		id: str(wire.unit_code),
		callSign: str(wire.call_sign),
		status,
		unitType: str(wire.unit_type),
		station: str(wire.station),
		sector: str(wire.sector),
		radioChannel: str(wire.radio_channel),
		shift: str(wire.shift),
		capabilities: Array.isArray(wire.capabilities) ? wire.capabilities.map(str) : [],
		crew: toCrew(wire.crew),
		assignment: toAssignment(wire.assignment),
		lastContact: str(wire.last_contact)
	};
}

/** Unwrap the AOH envelope and map every unit. Exported for tests. */
export function unitsFromEnvelope(body: unknown): FieldUnit[] {
	const envelope = (body ?? {}) as Envelope<unknown>;
	if (!Array.isArray(envelope.data)) {
		throw new DispatchServiceError('dispatch service response had no `data` array');
	}
	return envelope.data.map(toFieldUnit);
}

function serviceUrl(): string {
	return (env.DISPATCH_SVC_URL || 'http://localhost:8081').replace(/\/+$/, '');
}

/**
 * Every field unit, in the order the service returns them.
 *
 * Throws `DispatchServiceError` when the service is unreachable or answers unusably, so
 * the caller can render an explicit error state rather than an empty fleet.
 */
export async function listUnits(fetchImpl: typeof fetch = fetch): Promise<FieldUnit[]> {
	let response: Response;
	try {
		response = await fetchImpl(`${serviceUrl()}/v1/units`, {
			headers: { accept: 'application/json' }
		});
	} catch (cause) {
		throw new DispatchServiceError('could not reach the dispatch service', { cause });
	}

	if (!response.ok) {
		throw new DispatchServiceError(`dispatch service returned HTTP ${response.status}`);
	}

	let body: unknown;
	try {
		body = await response.json();
	} catch (cause) {
		throw new DispatchServiceError('dispatch service returned a non-JSON body', { cause });
	}

	return unitsFromEnvelope(body);
}
