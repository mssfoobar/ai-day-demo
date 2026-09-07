/**
 * Client for the dispatch service's field-unit API.
 *
 * `.server.ts` on purpose: importing this from client code is a build error, so the
 * service URL can never reach the browser bundle. Reads happen in a server `load`, writes
 * in form actions — the browser only ever talks to its own origin.
 */
import { env } from '$env/dynamic/private';
import {
	KNOWN_STATUSES,
	type Assignment,
	type Crew,
	type FieldUnit,
	type UnitField,
	type UnitInput,
	type UnitStatus
} from './types';

// Re-exported so existing imports keep working; the definitions live in ./types so
// client components can use them without touching this server-only module.
export type { Assignment, Crew, FieldUnit, UnitInput, UnitStatus } from './types';

/** One field-level problem, as the AOH error contract's `details` carries it. */
export interface FieldError {
	field: string;
	message: string;
}

/**
 * Raised when the service cannot be reached or answers with a failure.
 *
 * Carries what a caller needs to decide what to show: the HTTP status, the service's
 * `errorCode`, and any field details. `message` is developer-facing and must not be shown
 * to a user verbatim.
 */
export class DispatchServiceError extends Error {
	constructor(
		message: string,
		readonly status: number | null = null,
		readonly errorCode: string | null = null,
		readonly details: FieldError[] = [],
		options?: ErrorOptions
	) {
		super(message, options);
		this.name = 'DispatchServiceError';
	}

	get isConflict() {
		return this.status === 409;
	}
	get isNotFound() {
		return this.status === 404;
	}
	get isValidation() {
		return this.status === 400;
	}
}

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
	occ_lock?: unknown;
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
		lastContact: str(wire.last_contact),
		occLock: typeof wire.occ_lock === 'number' ? wire.occ_lock : 0
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

function unitFromEnvelope(body: unknown): FieldUnit {
	const envelope = (body ?? {}) as Envelope<unknown>;
	if (envelope.data === null || typeof envelope.data !== 'object') {
		throw new DispatchServiceError('dispatch service response had no `data` object');
	}
	return toFieldUnit(envelope.data);
}

/** camelCase → the snake_case body the service expects. */
export function toWireInput(input: UnitInput): Record<string, unknown> {
	return {
		unit_code: input.unitCode,
		call_sign: input.callSign,
		status: input.status,
		unit_type: input.unitType,
		station: input.station,
		sector: input.sector,
		radio_channel: input.radioChannel,
		shift: input.shift,
		capabilities: input.capabilities
	};
}

const WIRE_TO_FIELD: Record<string, UnitField> = {
	unit_code: 'unitCode',
	call_sign: 'callSign',
	status: 'status',
	unit_type: 'unitType',
	station: 'station',
	sector: 'sector',
	radio_channel: 'radioChannel',
	shift: 'shift',
	capabilities: 'capabilities'
};

/** Translate a wire field name from an error detail to the form's field name. */
export function fieldFromWire(field: string): UnitField | string {
	return WIRE_TO_FIELD[field] ?? field;
}

/**
 * Decode an AOH error payload — `{ errorCode, errorMessage, details }` — into a
 * DispatchServiceError. Exported for tests. Tolerates a non-JSON body.
 */
export async function errorFromResponse(response: Response): Promise<DispatchServiceError> {
	let body: Record<string, unknown> = {};
	try {
		body = (await response.json()) as Record<string, unknown>;
	} catch {
		// A non-JSON failure body (a proxy page, say) still yields a usable error.
	}
	const details: FieldError[] = Array.isArray(body.details)
		? body.details
				.map((d) => {
					const detail = (d ?? {}) as { field?: unknown; message?: unknown };
					// aoherr renders details as {field, message}; fall back to "field: message".
					if (typeof detail.field === 'string') {
						return { field: detail.field, message: str(detail.message) };
					}
					const [field, ...rest] = str(detail.message).split(':');
					return rest.length ? { field: field.trim(), message: rest.join(':').trim() } : null;
				})
				.filter((d): d is FieldError => d !== null)
		: [];

	return new DispatchServiceError(
		str(body.errorMessage) || `dispatch service returned HTTP ${response.status}`,
		response.status,
		str(body.errorCode) || null,
		details
	);
}

function serviceUrl(): string {
	return (env.DISPATCH_SVC_URL || 'http://localhost:8081').replace(/\/+$/, '');
}

async function call(
	fetchImpl: typeof fetch,
	method: string,
	path: string,
	body?: unknown
): Promise<Response> {
	try {
		return await fetchImpl(`${serviceUrl()}${path}`, {
			method,
			headers: {
				accept: 'application/json',
				...(body !== undefined ? { 'content-type': 'application/json' } : {})
			},
			body: body !== undefined ? JSON.stringify(body) : undefined
		});
	} catch (cause) {
		throw new DispatchServiceError('could not reach the dispatch service', null, null, [], {
			cause
		});
	}
}

async function json(response: Response): Promise<unknown> {
	try {
		return await response.json();
	} catch (cause) {
		throw new DispatchServiceError(
			'dispatch service returned a non-JSON body',
			response.status,
			null,
			[],
			{ cause }
		);
	}
}

/**
 * Every field unit, in the order the service returns them.
 *
 * Throws `DispatchServiceError` when the service is unreachable or answers unusably, so
 * the caller can render an explicit error state rather than an empty fleet.
 */
export async function listUnits(fetchImpl: typeof fetch = fetch): Promise<FieldUnit[]> {
	const response = await call(fetchImpl, 'GET', '/v1/units');
	if (!response.ok) throw await errorFromResponse(response);
	return unitsFromEnvelope(await json(response));
}

/** Create a unit. 400 → validation details, 409 → `DISPATCH_UNIT_CODE_TAKEN`. */
export async function createUnit(
	input: UnitInput,
	fetchImpl: typeof fetch = fetch
): Promise<FieldUnit> {
	const response = await call(fetchImpl, 'POST', '/v1/units', toWireInput(input));
	if (!response.ok) throw await errorFromResponse(response);
	return unitFromEnvelope(await json(response));
}

/** Replace a unit's editable fields, echoing `occLock`. 409 `DISPATCH_UNIT_STALE` on a stale lock. */
export async function updateUnit(
	unitCode: string,
	occLock: number,
	input: UnitInput,
	fetchImpl: typeof fetch = fetch
): Promise<FieldUnit> {
	const { unit_code: _omit, ...body } = toWireInput(input);
	void _omit; // the code travels in the URL, not the body
	const response = await call(fetchImpl, 'PUT', `/v1/units/${encodeURIComponent(unitCode)}`, {
		...body,
		occ_lock: occLock
	});
	if (!response.ok) throw await errorFromResponse(response);
	return unitFromEnvelope(await json(response));
}

/** Delete a unit, echoing `occLock`. 204 on success. */
export async function deleteUnit(
	unitCode: string,
	occLock: number,
	fetchImpl: typeof fetch = fetch
): Promise<void> {
	const response = await call(
		fetchImpl,
		'DELETE',
		`/v1/units/${encodeURIComponent(unitCode)}?occ_lock=${encodeURIComponent(String(occLock))}`
	);
	if (!response.ok) throw await errorFromResponse(response);
}
