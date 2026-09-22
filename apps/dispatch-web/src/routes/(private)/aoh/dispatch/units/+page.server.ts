import { fail, redirect } from '@sveltejs/kit';
import { StatusCodes } from 'http-status-codes';
import { LOGIN_API } from '$lib/aoh/core/provider/auth/auth';
import { canWrite } from '$lib/aoh/dispatch/permissions';
import { parseUnitForm, type UnitFormErrors, type UnitFormValues } from '$lib/aoh/dispatch/forms';
import {
	bearerFrom,
	createUnit,
	deleteUnit,
	DispatchServiceError,
	fieldFromWire,
	listUnits,
	updateUnit
} from '$lib/aoh/dispatch/units.server';
import type { Actions, PageServerLoad } from './$types';

/** Why the roster is not on screen. `ok` means it is. */
export type RosterState = 'ok' | 'unavailable' | 'denied';

/**
 * Load the roster on the server, as the signed-in operator.
 *
 * The browser never talks to the dispatch service: no CORS to configure, and no service
 * URL in the client bundle. The bearer is read from SDS here and sent upstream; it is
 * never returned from this load, because everything a load returns is serialised into
 * the page payload.
 *
 * Failures are returned as data rather than thrown — an unreachable service and a refused
 * role are both expected operational conditions the console renders explicit states for.
 * A 401 is the exception: a dead session is not a state to render, it is a reason to sign
 * in again.
 */
export const load: PageServerLoad = async ({ locals }) => {
	const roles = locals.authResult.success
		? (locals.authResult.claims.active_tenant?.roles ?? [])
		: [];
	const token = bearerFrom(locals);
	if (!token) redirect(StatusCodes.TEMPORARY_REDIRECT, LOGIN_API);

	try {
		return { units: await listUnits(token), state: 'ok' as RosterState, canWrite: canWrite(roles) };
	} catch (err) {
		if (err instanceof DispatchServiceError && err.isUnauthenticated) {
			redirect(StatusCodes.TEMPORARY_REDIRECT, LOGIN_API);
		}
		const state: RosterState =
			err instanceof DispatchServiceError && err.isForbidden ? 'denied' : 'unavailable';
		return { units: [], state, canWrite: canWrite(roles) };
	}
};

/** What a failed write hands back to the form. */
export interface UnitActionFailure {
	intent: 'create' | 'update' | 'delete';
	errors: UnitFormErrors;
	values?: UnitFormValues;
	/** True when the service refused the operator's role, not their input. */
	denied?: boolean;
}

/** What a successful write hands back so the page can toast and select. */
export interface UnitActionSuccess {
	intent: 'create' | 'update' | 'delete';
	unitCode: string;
	callSign: string;
}

const CONFLICT_MESSAGE = (callSign: string) =>
	`${callSign} was changed by someone else. Reload and try again.`;

/** The platform's permission-denied line. Names the action; offers no retry. */
const DENIED_MESSAGE = (action: string) =>
	`Access to ${action} is restricted. For assistance with access, please contact your administrator.`;

const ACTION_NAMES: Record<'create' | 'update' | 'delete', string> = {
	create: 'adding a unit',
	update: 'editing a unit',
	delete: 'deleting a unit'
};

/**
 * Turn a service failure into the form's error shape. Developer-facing text from the
 * service never reaches the user; the field details and error code do.
 */
function errorsFrom(
	err: unknown,
	callSign: string,
	intent: 'create' | 'update' | 'delete'
): { status: number; errors: UnitFormErrors; denied?: boolean } {
	if (err instanceof DispatchServiceError) {
		if (err.isValidation && err.details.length > 0) {
			const errors: UnitFormErrors = {};
			for (const d of err.details) {
				errors[fieldFromWire(d.field) as keyof UnitFormErrors] = humanise(d);
			}
			return { status: 400, errors };
		}
		if (err.isForbidden) {
			// The operator's roles do not permit this. A retry will never succeed, so this
			// is deliberately not the generic "try again" message.
			return { status: 403, errors: { form: DENIED_MESSAGE(ACTION_NAMES[intent]) }, denied: true };
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
	const msg = d.message.replace(/^[a-z_.]+:\s*/, '');
	return `${label} ${msg}${msg.endsWith('.') ? '' : '.'}`;
}

/**
 * The bearer for a write, or a redirect to sign in.
 *
 * A write with no session is not a form error — there is nobody to show one to.
 */
function writeToken(locals: App.Locals): string {
	const token = bearerFrom(locals);
	if (!token) redirect(StatusCodes.TEMPORARY_REDIRECT, LOGIN_API);
	return token;
}

export const actions: Actions = {
	create: async ({ request, locals }) => {
		const token = writeToken(locals);
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
			const unit = await createUnit(token, parsed.input);
			return {
				intent: 'create',
				unitCode: unit.id,
				callSign: unit.callSign
			} satisfies UnitActionSuccess;
		} catch (err) {
			if (err instanceof DispatchServiceError && err.isUnauthenticated) {
				redirect(StatusCodes.TEMPORARY_REDIRECT, LOGIN_API);
			}
			const { status, errors, denied } = errorsFrom(err, parsed.input.callSign, 'create');
			return fail(status, {
				intent: 'create',
				errors,
				values: parsed.values,
				denied
			} satisfies UnitActionFailure);
		}
	},

	update: async ({ request, locals }) => {
		const token = writeToken(locals);
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
			const unit = await updateUnit(token, unitCode, parsed.occLock, {
				...parsed.input,
				unitCode
			});
			return {
				intent: 'update',
				unitCode: unit.id,
				callSign: unit.callSign
			} satisfies UnitActionSuccess;
		} catch (err) {
			if (err instanceof DispatchServiceError && err.isUnauthenticated) {
				redirect(StatusCodes.TEMPORARY_REDIRECT, LOGIN_API);
			}
			const { status, errors, denied } = errorsFrom(err, parsed.input.callSign, 'update');
			return fail(status, {
				intent: 'update',
				errors,
				values: parsed.values,
				denied
			} satisfies UnitActionFailure);
		}
	},

	delete: async ({ request, locals }) => {
		const token = writeToken(locals);
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
			await deleteUnit(token, unitCode, Number(rawLock));
			return { intent: 'delete', unitCode, callSign } satisfies UnitActionSuccess;
		} catch (err) {
			if (err instanceof DispatchServiceError && err.isUnauthenticated) {
				redirect(StatusCodes.TEMPORARY_REDIRECT, LOGIN_API);
			}
			const { status, errors, denied } = errorsFrom(err, callSign, 'delete');
			return fail(status, { intent: 'delete', errors, denied } satisfies UnitActionFailure);
		}
	}
};
