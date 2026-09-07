import { fail } from '@sveltejs/kit';
import { parseUnitForm, type UnitFormErrors, type UnitFormValues } from '$lib/aoh/dispatch/forms';
import {
	createUnit,
	deleteUnit,
	DispatchServiceError,
	fieldFromWire,
	listUnits,
	updateUnit
} from '$lib/aoh/dispatch/units.server';
import type { Actions, PageServerLoad } from './$types';

/**
 * Load the roster on the server.
 *
 * The browser never talks to the dispatch service: no CORS to configure, and no service
 * URL in the client bundle. All URL and transport wiring lives behind the client module.
 *
 * A failure is returned as data, not thrown: an unreachable service is an expected
 * operational condition the console renders an explicit state for.
 */
export const load: PageServerLoad = async () => {
	try {
		return { units: await listUnits(), unavailable: false };
	} catch {
		return { units: [], unavailable: true };
	}
};

/** What a failed write hands back to the form. */
export interface UnitActionFailure {
	intent: 'create' | 'update' | 'delete';
	errors: UnitFormErrors;
	values?: UnitFormValues;
}

/** What a successful write hands back so the page can toast and select. */
export interface UnitActionSuccess {
	intent: 'create' | 'update' | 'delete';
	unitCode: string;
	callSign: string;
}

const CONFLICT_MESSAGE = (callSign: string) =>
	`${callSign} was changed by someone else. Reload and try again.`;

/**
 * Turn a service failure into the form's error shape. Developer-facing text from the
 * service never reaches the user; the field details and error code do.
 */
function errorsFrom(err: unknown, callSign: string): { status: number; errors: UnitFormErrors } {
	if (err instanceof DispatchServiceError) {
		if (err.isValidation && err.details.length > 0) {
			const errors: UnitFormErrors = {};
			for (const d of err.details) {
				errors[fieldFromWire(d.field) as keyof UnitFormErrors] = humanise(d);
			}
			return { status: 400, errors };
		}
		if (err.errorCode === 'DISPATCH_UNIT_CODE_TAKEN') {
			return { status: 409, errors: { unitCode: 'A unit with this ID already exists.' } };
		}
		if (err.errorCode === 'DISPATCH_UNIT_STALE') {
			return { status: 409, errors: { form: CONFLICT_MESSAGE(callSign) } };
		}
		if (err.isNotFound) {
			return {
				status: 404,
				errors: { form: `${callSign} no longer exists. Reload to refresh the list.` }
			};
		}
		if (err.status === null) {
			return { status: 503, errors: { form: 'Could not reach the dispatch service.' } };
		}
	}
	return { status: 500, errors: { form: 'The change could not be saved. Try again.' } };
}

/** "must not be empty" → "Call sign must not be empty." */
function humanise(d: { field: string; message: string }): string {
	const label = fieldFromWire(d.field)
		.replace(/([A-Z])/g, ' $1')
		.replace(/^./, (c) => c.toUpperCase())
		.replace(/^Unit Code$/, 'Unit ID');
	const msg = d.message.replace(/^[a-z_]+:\s*/, '');
	return `${label} ${msg}${msg.endsWith('.') ? '' : '.'}`;
}

export const actions: Actions = {
	create: async ({ request }) => {
		const data = await request.formData();
		const parsed = parseUnitForm(data, { requireCode: true });
		if (!parsed.ok) {
			return fail(400, {
				intent: 'create',
				errors: parsed.errors,
				values: parsed.values
			} satisfies UnitActionFailure);
		}
		try {
			const unit = await createUnit(parsed.input);
			return {
				intent: 'create',
				unitCode: unit.id,
				callSign: unit.callSign
			} satisfies UnitActionSuccess;
		} catch (err) {
			const { status, errors } = errorsFrom(err, parsed.input.callSign);
			return fail(status, {
				intent: 'create',
				errors,
				values: parsed.values
			} satisfies UnitActionFailure);
		}
	},

	update: async ({ request }) => {
		const data = await request.formData();
		const unitCode = String(data.get('unitCode') ?? '').trim();
		const parsed = parseUnitForm(data, { requireCode: false });
		if (!parsed.ok) {
			return fail(400, {
				intent: 'update',
				errors: parsed.errors,
				values: parsed.values
			} satisfies UnitActionFailure);
		}
		if (!unitCode || parsed.occLock === null) {
			// The form always carries both; their absence is a wiring bug, not user error.
			return fail(400, {
				intent: 'update',
				errors: { form: 'This edit is missing its version. Reload and try again.' },
				values: parsed.values
			} satisfies UnitActionFailure);
		}
		try {
			const unit = await updateUnit(unitCode, parsed.occLock, { ...parsed.input, unitCode });
			return {
				intent: 'update',
				unitCode: unit.id,
				callSign: unit.callSign
			} satisfies UnitActionSuccess;
		} catch (err) {
			const { status, errors } = errorsFrom(err, parsed.input.callSign);
			return fail(status, {
				intent: 'update',
				errors,
				values: parsed.values
			} satisfies UnitActionFailure);
		}
	},

	delete: async ({ request }) => {
		const data = await request.formData();
		const unitCode = String(data.get('unitCode') ?? '').trim();
		const callSign = String(data.get('callSign') ?? unitCode).trim();
		const rawLock = String(data.get('occLock') ?? '');
		if (!unitCode || !/^\d+$/.test(rawLock)) {
			return fail(400, {
				intent: 'delete',
				errors: { form: 'This delete is missing its version. Reload and try again.' }
			} satisfies UnitActionFailure);
		}
		try {
			await deleteUnit(unitCode, Number(rawLock));
			return { intent: 'delete', unitCode, callSign } satisfies UnitActionSuccess;
		} catch (err) {
			const { status, errors } = errorsFrom(err, callSign);
			return fail(status, { intent: 'delete', errors } satisfies UnitActionFailure);
		}
	}
};
