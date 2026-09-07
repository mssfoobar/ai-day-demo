/**
 * Presentation helpers shared by the console's components. Pure functions, no DOM, so
 * they are unit-tested directly.
 */
import type { UnitStatus } from './types';

export type BadgeColor = 'success' | 'info' | 'warning' | 'destructive' | 'default';

/**
 * Status → `Badge` palette. `Badge` splits shape (`variant`) from palette (`color`).
 * Idle is `default`, not `warning`: an idle unit is neutral, not faulted.
 */
export const statusColor: Record<UnitStatus, BadgeColor> = {
	Available: 'success',
	'En route': 'info',
	Idle: 'default'
};

/**
 * Status → the thin colour rail on each row, as a Tailwind background class. These are the
 * package's semantic tokens (`--color-success`, `--color-bg-info-strong`), so they follow
 * the theme; never raw colours.
 */
export const statusRail: Record<UnitStatus, string> = {
	Available: 'bg-success',
	'En route': 'bg-info-strong',
	Idle: 'bg-muted-foreground/40'
};

/** Priority is the dispatcher's triage signal, so P1 reads destructive. */
export function priorityColor(priority: string): BadgeColor {
	if (priority === 'P1') return 'destructive';
	if (priority === 'P2') return 'warning';
	return 'default';
}

/**
 * "3 min ago" — a dispatcher reads recency, not a wall-clock timestamp.
 *
 * Takes `now` explicitly so callers can drive it from a ticking clock and tests can pin it.
 */
export function sinceLabel(iso: string, now: number = Date.now()): string {
	const then = Date.parse(iso);
	if (Number.isNaN(then)) return '—';
	const minutes = Math.max(0, Math.round((now - then) / 60000));
	if (minutes < 1) return 'just now';
	if (minutes === 1) return '1 min ago';
	if (minutes < 60) return `${minutes} min ago`;
	const hours = Math.round(minutes / 60);
	if (hours < 24) return hours === 1 ? '1 hr ago' : `${hours} hrs ago`;
	const days = Math.round(hours / 24);
	return days === 1 ? '1 day ago' : `${days} days ago`;
}

/**
 * How stale a last-contact time is, for the recency dot: fresh under 10 minutes,
 * ageing under 30, stale beyond. Thresholds are a workshop default, not doctrine.
 */
export function recency(iso: string, now: number = Date.now()): 'fresh' | 'ageing' | 'stale' {
	const then = Date.parse(iso);
	if (Number.isNaN(then)) return 'stale';
	const minutes = (now - then) / 60000;
	if (minutes < 10) return 'fresh';
	if (minutes < 30) return 'ageing';
	return 'stale';
}
