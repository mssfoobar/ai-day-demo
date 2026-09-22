/**
 * Form parsing for the unit add/edit form. Pure — no SvelteKit, no DOM.
 *
 * This is the console's own first pass; the service validates again and the database has
 * the last word. Duplicating the *required* checks here means a blank field is caught
 * without a round trip, and the wording is the console's, not the service's.
 */
import {
	KNOWN_STATUSES,
	type Position,
	type UnitField,
	type UnitInput,
	type UnitStatus
} from './types';

/** What the form sends, as strings — the shape we echo back on a failed submit. */
export type UnitFormValues = Record<UnitField, string>;

/**
 * The three hidden inputs that carry a unit's existing position through an edit.
 *
 * The console does not author positions — but a write is a **replace**, so a form that
 * sent no position would clear the unit's fix and delete its marker from the map. They
 * are not `UnitField`s: they have no label, no required check, and the operator never
 * sees them. `at` rides along too, so an edit that reported no new location leaves the
 * fix time where it was rather than looking like a fresh GPS report.
 */
export const POSITION_FIELDS = ['positionLon', 'positionLat', 'positionAt'] as const;

export const EMPTY_VALUES: UnitFormValues = {
	unitCode: '',
	callSign: '',
	status: 'Available',
	unitType: '',
	station: '',
	sector: '',
	radioChannel: '',
	shift: '',
	capabilities: ''
};

export type UnitFormErrors = Partial<Record<UnitField | 'form', string>>;

export type ParsedUnitForm =
	| { ok: true; input: UnitInput; occLock: number | null; values: UnitFormValues }
	| { ok: false; errors: UnitFormErrors; values: UnitFormValues };

/**
 * Read the round-tripped position back off the form.
 *
 * Returns `undefined` when the unit had none — which is also what an absent `position`
 * means to the service. A malformed value is a wiring bug rather than user input, so it
 * is reported under the form rather than under a field the operator can see.
 */
function parsePosition(data: FormData): { position?: Position; error?: string } {
	const raw = (key: (typeof POSITION_FIELDS)[number]) => {
		const value = data.get(key);
		return typeof value === 'string' ? value.trim() : '';
	};
	const lon = raw('positionLon');
	const lat = raw('positionLat');
	const at = raw('positionAt');

	if (!lon && !lat) return {};

	const lonNum = Number(lon);
	const latNum = Number(lat);
	if (!lon || !lat || Number.isNaN(lonNum) || Number.isNaN(latNum)) {
		return { error: 'This edit carried an incomplete position. Reload and try again.' };
	}
	if (lonNum < -180 || lonNum > 180 || latNum < -90 || latNum > 90) {
		return { error: 'This edit carried an out-of-range position. Reload and try again.' };
	}
	return { position: { lon: lonNum, lat: latNum, at } };
}

const REQUIRED: UnitField[] = [
	'callSign',
	'unitType',
	'station',
	'sector',
	'radioChannel',
	'shift'
];

const LABELS: Record<UnitField, string> = {
	unitCode: 'Unit ID',
	callSign: 'Call sign',
	status: 'Status',
	unitType: 'Type',
	station: 'Station',
	sector: 'Sector',
	radioChannel: 'Radio channel',
	shift: 'Shift',
	capabilities: 'Capabilities'
};

/** Field label for messages and for the form itself, so the two never disagree. */
export function fieldLabel(field: UnitField): string {
	return LABELS[field];
}

/** Read every known field from FormData as a trimmed string. */
export function valuesFrom(data: FormData): UnitFormValues {
	const get = (key: UnitField) => {
		const raw = data.get(key);
		return typeof raw === 'string' ? raw.trim() : '';
	};
	return {
		unitCode: get('unitCode'),
		callSign: get('callSign'),
		status: get('status'),
		unitType: get('unitType'),
		station: get('station'),
		sector: get('sector'),
		radioChannel: get('radioChannel'),
		shift: get('shift'),
		capabilities: get('capabilities')
	};
}

/** "ALS, Water rescue" → ["ALS", "Water rescue"]; blanks and duplicates dropped. */
export function parseCapabilities(raw: string): string[] {
	const seen = new Set<string>();
	for (const part of raw.split(',')) {
		const c = part.trim();
		if (c) seen.add(c);
	}
	return [...seen];
}

/**
 * Parse and validate the form.
 *
 * `requireCode` is true for add (the user supplies the unit ID) and false for edit (the
 * ID is fixed and comes from the selected unit, not the form).
 */
export function parseUnitForm(data: FormData, options: { requireCode: boolean }): ParsedUnitForm {
	const values = valuesFrom(data);
	const errors: UnitFormErrors = {};

	if (options.requireCode) {
		if (!values.unitCode) errors.unitCode = `${LABELS.unitCode} is required.`;
		else if (values.unitCode.length > 32)
			errors.unitCode = `${LABELS.unitCode} must be 32 characters or fewer.`;
	}
	for (const field of REQUIRED) {
		if (!values[field]) errors[field] = `${LABELS[field]} is required.`;
		else if (values[field].length > 120)
			errors[field] = `${LABELS[field]} must be 120 characters or fewer.`;
	}
	if (!values.status) {
		errors.status = `${LABELS.status} is required.`;
	} else if (!(KNOWN_STATUSES as readonly string[]).includes(values.status)) {
		errors.status = `${LABELS.status} must be one of ${KNOWN_STATUSES.join(', ')}.`;
	}

	// occ_lock rides along on edit; a non-integer is a programming error, not user input.
	const rawLock = data.get('occLock');
	const occLock = typeof rawLock === 'string' && /^\d+$/.test(rawLock) ? Number(rawLock) : null;

	// The unit's existing position, carried through untouched.
	const { position, error: positionError } = parsePosition(data);
	if (positionError) errors.form = positionError;

	if (Object.keys(errors).length > 0) {
		return { ok: false, errors, values };
	}

	return {
		ok: true,
		occLock,
		values,
		input: {
			unitCode: values.unitCode,
			callSign: values.callSign,
			status: values.status as UnitStatus,
			unitType: values.unitType,
			station: values.station,
			sector: values.sector,
			radioChannel: values.radioChannel,
			shift: values.shift,
			capabilities: parseCapabilities(values.capabilities),
			position
		}
	};
}
