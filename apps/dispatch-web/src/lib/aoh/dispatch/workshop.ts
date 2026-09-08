/**
 * The three workshop exercises — stubbed in the UI and the API, not built.
 *
 * This is the one place their titles, stories, stub routes and completion state are written
 * down for the console; WORKSHOP.md at the repo root is the full brief. The floating button
 * in the console lists whatever is still to build and can point at where each one goes.
 * When you finish one, flip its `complete` to `true`: it moves to "already built" and the
 * count on the button goes down. When all three are done, delete this module and every
 * `Workshop` sketch with it.
 *
 * The copy here is written for the person doing the exercise, not for a spec reader —
 * keep it that way if you edit it.
 */
export type ExerciseNumber = 1 | 2 | 3;

export interface Exercise {
	number: ExerciseNumber;
	title: string;
	/** Flip to `true` when the exercise passes its acceptance criteria. */
	complete: boolean;
	/** Where to look in the console. */
	where: string;
	/** The user story, as the dispatcher would say it. */
	story: string;
	/** "You're done when…" — abbreviated; WORKSHOP.md has the full list. */
	done: string[];
	/** The service routes that answer 501 until the exercise is built. */
	routes: string[];
}

export const EXERCISES: Record<ExerciseNumber, Exercise> = {
	1: {
		number: 1,
		title: 'Send a unit to an incident',
		complete: false,
		where: 'Look for the dashed Dispatch button next to Edit and Delete in the unit header.',
		story:
			'As a dispatcher, I want to send an available unit to an incident and bring it back when the job is done, so that the console shows who is working what instead of only what the seed data says.',
		done: [
			'Dispatch opens a small form: incident code, title, priority (P1–P3) and location.',
			'Saving it marks the unit En route, and the pane, the row and the tiles all catch up.',
			'On an assigned unit the same button reads Stand down, and clears it.',
			'If someone else changed the unit first, you see a conflict instead of overwriting them.',
			'Restart the service and it is all still there.'
		],
		routes: ['POST /v1/units/{unit_code}/assignment', 'DELETE /v1/units/{unit_code}/assignment']
	},
	2: {
		number: 2,
		title: 'Show what happened to a unit',
		complete: false,
		where: 'Look for the sketched Activity section between Assignment and Crew.',
		story:
			'As a dispatcher, I want to see what a unit has been doing — its status and assignment changes, newest first — so that I know what happened on the shift without asking over the radio.',
		done: [
			'The unit detail shows its last 20 events, newest first: when, and what changed.',
			'Every change to a unit leaves an event behind, written in the same transaction.',
			'A quiet unit says “No activity yet” rather than showing nothing.',
			'The seeded units come with a little history, so the section is not empty on day one.',
			'Restart the service and the history is still there.'
		],
		routes: ['GET /v1/units/{unit_code}/events']
	},
	3: {
		number: 3,
		title: 'Fix up who is on the crew',
		complete: false,
		where: 'Look for the dashed Manage button beside the Crew heading.',
		story:
			'As a dispatcher, I want to add and remove the people on a unit, so that the console matches who is actually on the vehicle this shift.',
		done: [
			'Manage opens a list of the crew, each with a remove control, plus a row to add someone (name, role).',
			'Saving replaces the crew in one go, and the heading count updates.',
			'Two people with the same name on one unit is refused, with the error next to the row.',
			'If someone else changed the unit first, you see a conflict instead of overwriting them.',
			'Restart the service and the crew is still right.'
		],
		routes: ['PUT /v1/units/{unit_code}/crew']
	}
};

export function incompleteExercises(): Exercise[] {
	return Object.values(EXERCISES).filter((exercise) => !exercise.complete);
}

export function completedExercises(): Exercise[] {
	return Object.values(EXERCISES).filter((exercise) => exercise.complete);
}

/**
 * Focus mode is nothing more than a highlighted border around the area where the exercise
 * gets built. Both classes are always present on a target so the colour can transition; the
 * page swaps `transparent` for the workshop colour while pointing at it.
 */
export const FOCUS_BASE_CLASS =
	'rounded-md outline-2 outline-offset-4 outline-transparent transition-[outline-color,transform] duration-300';
export const FOCUS_ON_CLASS = 'outline-(--workshop) scale-[1.02]';
