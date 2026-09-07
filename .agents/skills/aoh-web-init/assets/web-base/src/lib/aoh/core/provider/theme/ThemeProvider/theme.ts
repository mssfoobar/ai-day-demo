import { writable, type Writable } from 'svelte/store';

export type ThemeMode = 'light' | 'dark' | 'system';
const STORAGE_KEY = 'aoh-theme';

export const darkModeStore: Writable<boolean> = writable(false);

export function getStoredTheme(): ThemeMode {
	if (typeof localStorage === 'undefined') return 'system';
	const v = localStorage.getItem(STORAGE_KEY);
	return v === 'light' || v === 'dark' || v === 'system' ? v : 'system';
}

export function setStoredTheme(mode: ThemeMode) {
	if (typeof localStorage === 'undefined') return;
	localStorage.setItem(STORAGE_KEY, mode);
}

export function resolveTheme(mode: ThemeMode): 'light' | 'dark' {
	if (mode === 'system') {
		if (typeof window === 'undefined') return 'dark';
		return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
	}
	return mode;
}

export function applyThemeClass(mode: ThemeMode) {
	if (typeof document === 'undefined') return;
	const resolved = resolveTheme(mode);
	document.documentElement.classList.toggle('dark', resolved === 'dark');
	darkModeStore.set(resolved === 'dark');
}
