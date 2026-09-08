/**
 * The three workshop exercises — stubbed in the UI and the API, not built.
 *
 * This is the one place their titles, stories, stub routes and completion state are written
 * down for the console; WORKSHOP.md at the repo root is the full brief. The console's
 * **Exercises** button lists whatever is not complete and can focus the console on an
 * exercise's placeholder. When you finish one, flip its `complete` to `true`: it drops out
 * of the list, and the count on the button goes down. When all three are done, delete this
 * module and every `Workshop` placeholder with it.
 */
export type ExerciseNumber = 1 | 2 | 3;

export interface Exercise {
	number: ExerciseNumber;
	title: string;
	/** Flip to `true` when the exercise passes its acceptance criteria. */
	complete: boolean;
	/** Where the placeholder sits in the console. */
	where: string;
	/** The user story, in one sentence. */
	story: string;
	/** Acceptance criteria, abbreviated — WORKSHOP.md has the full list. */
	criteria: string[];
	/** The service routes that answer 501 until the exercise is built. */
	routes: string[];
}

export const EXERCISES: Record<ExerciseNumber, Exercise> = {
	1: {
		number: 1,
		title: 'Dispatch a unit',
		complete: false,
		where: 'The Dispatch / Stand down button in the unit header.',
		story:
			'As a dispatcher, I want to dispatch an available unit to an incident and stand it down when the job is done, so the console shows who is working what.',
		criteria: [
			'Dispatch opens a form: incident code, title, priority (P1–P3), location — all required.',
			'Saving sets the assignment and the status becomes En route; the pane, row and tiles update.',
			'Stand down clears the assignment and the status returns to Available.',
			'A stale occ_lock is a 409, shown in the form like edit does today.',
			'Survives a service restart.'
		],
		routes: ['POST /v1/units/{unit_code}/assignment', 'DELETE /v1/units/{unit_code}/assignment']
	},
	2: {
		number: 2,
		title: 'Unit activity timeline',
		complete: false,
		where: 'The dashed Activity section in the unit detail.',
		story:
			'As a dispatcher, I want to see a unit’s recent status and assignment changes with timestamps, so I can tell what happened to it during the shift.',
		criteria: [
			'Lists the last 20 events, newest first: when, and what changed.',
			'Every write to a unit records an event in the same transaction.',
			'A unit with no history shows “No activity yet.”',
			'Seed data includes a few events so the section is not empty on first boot.',
			'Survives a service restart.'
		],
		routes: ['GET /v1/units/{unit_code}/events']
	},
	3: {
		number: 3,
		title: 'Manage crew',
		complete: false,
		where: 'The Manage button beside the Crew heading.',
		story:
			'As a dispatcher, I want to add and remove the crew on a unit, so the roster matches who is actually on the vehicle this shift.',
		criteria: [
			'Manage opens a form listing the crew, each removable, plus one row to add (name, role).',
			'Saving replaces the crew in one request; the section and its count update.',
			'Two members with the same name on one unit is refused, beside the offending row.',
			'A stale occ_lock is a 409.',
			'Survives a service restart.'
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
 * Shared look for the exercise cards. The workshop palette (`--workshop-*`, defined in
 * app.css) is used by nothing else in the console, so the layer reads as an annotation over
 * the UI rather than part of it.
 */
export const PLACEHOLDER_CLASS =
	'border border-dashed border-(--workshop-border) bg-(--workshop-muted) text-(--workshop-text) transition-shadow duration-300';

/** Added to a placeholder while the console is focused on its exercise. */
export const FOCUS_CLASS =
	'animate-pulse ring-2 ring-(--workshop) ring-offset-2 ring-offset-background';
