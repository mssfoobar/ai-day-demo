import { describe, expect, it } from 'vitest';
import { priorityColor, recency, sinceLabel, statusColor, statusRail } from './format';

const NOW = Date.parse('2026-09-07T13:00:00Z');
const minutesAgo = (m: number) => new Date(NOW - m * 60000).toISOString();

describe('sinceLabel', () => {
	it('renders human recency', () => {
		expect(sinceLabel(minutesAgo(0), NOW)).toBe('just now');
		expect(sinceLabel(minutesAgo(1), NOW)).toBe('1 min ago');
		expect(sinceLabel(minutesAgo(14), NOW)).toBe('14 min ago');
		expect(sinceLabel(minutesAgo(60), NOW)).toBe('1 hr ago');
		expect(sinceLabel(minutesAgo(180), NOW)).toBe('3 hrs ago');
		expect(sinceLabel(minutesAgo(60 * 48), NOW)).toBe('2 days ago');
	});

	it('never goes negative for a timestamp slightly in the future', () => {
		expect(sinceLabel(minutesAgo(-2), NOW)).toBe('just now');
	});

	it('degrades to a dash for an unparseable value', () => {
		expect(sinceLabel('not a date', NOW)).toBe('—');
	});
});

describe('recency', () => {
	it('buckets by age', () => {
		expect(recency(minutesAgo(2), NOW)).toBe('fresh');
		expect(recency(minutesAgo(15), NOW)).toBe('ageing');
		expect(recency(minutesAgo(45), NOW)).toBe('stale');
		expect(recency('garbage', NOW)).toBe('stale');
	});
});

describe('colour maps', () => {
	it('keeps Idle neutral rather than a warning', () => {
		expect(statusColor.Idle).toBe('default');
		expect(statusColor.Available).toBe('success');
		expect(statusColor['En route']).toBe('info');
	});

	it('uses only semantic token classes for the rail', () => {
		for (const cls of Object.values(statusRail)) {
			expect(cls).not.toMatch(/#|blue|green|red|amber|gray|zinc/);
		}
	});

	it('escalates P1 to destructive', () => {
		expect(priorityColor('P1')).toBe('destructive');
		expect(priorityColor('P2')).toBe('warning');
		expect(priorityColor('P3')).toBe('default');
		expect(priorityColor('')).toBe('default');
	});
});
