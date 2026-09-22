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
}

export const EXERCISES: Record<ExerciseNumber, Exercise> = {
	1: {
		number: 1,
		title: 'Show the incidents on the units page',
		complete: false,
		where: 'Look for the dashed Incidents panel below the roster and the detail pane.',
		story:
			'As a dispatcher, I want to see every incident the fleet is working on one panel, so that I can tell what is happening across the shift without clicking each unit in turn, and can send a unit to an incident from there.',
		done: [
			'A panel on the units page lists every incident the fleet is working, P1 first.',
			'Each row shows the incident and the unit on it; clicking one selects that unit.',
			'Dispatch sends an available unit to a new incident, and it turns En route.',
			'Stand down clears it and the unit goes back to Available.',
			'If someone else changed the unit first, you see a conflict instead of overwriting them.'
		]
	},
	2: {
		number: 2,
		title: "See a unit's location here, not on the map page",
		complete: false,
		where: 'Look for the sketched Location box under Position in the detail pane.',
		story:
			'As a dispatcher, I want to see where the selected unit is on a small map in the detail pane, so that I can place it at a glance without losing the roster, my filters and my selection to a trip to the map page.',
		done: [
			'Picking a positioned unit shows a small map of where it is, right in the pane.',
			'The coordinates and the fix time stay: you still read those out over the radio.',
			'Pick another unit and the map follows it.',
			'A unit with no position says so, and shows no empty map frame.',
			'Show on map still takes you to the full map page.'
		]
	},
	3: {
		number: 3,
		title: 'Put the incidents on the map',
		complete: false,
		where: 'Open the map. Look for the dashed Incidents card over the canvas.',
		story:
			'As a dispatcher, I want to see where the incidents actually are on the map, so that I can judge which unit is closest to one without reading addresses off a list.',
		done: [
			'Every incident with coordinates gets a marker on the map, next to the unit markers.',
			'Clicking a marker highlights it and opens a panel with the incident and a placeholder picture.',
			'The panel names the unit working the incident, and clicking through selects it.',
			'An incident with no coordinates is counted somewhere rather than silently dropped.',
			'No new request and no service change: the map page already loads the roster.'
		]
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
