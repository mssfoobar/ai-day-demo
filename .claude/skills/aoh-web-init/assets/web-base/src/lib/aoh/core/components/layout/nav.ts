import type { Component } from 'svelte';

// Sidebar / header navigation for the app. This is the single source of nav
// entries — add a row here when you add a feature page under
// src/routes/(private)/. (There is no modlet auto-discovery; the layout reads
// this list directly.)
export type NavItem = {
	name: string;
	url: string;
	/** Optional sidebar group label. Entries with the same group render together. */
	group?: string;
	/** Optional icon component (e.g. from a Svelte icon package). */
	icon?: Component;
	/** Active-tenant roles allowed to see this entry. Omit = visible to everyone. */
	roles?: string[];
};

export const navItems: NavItem[] = [{ name: 'Getting started', url: '/getting-started' }];

export function filterByRoles(items: NavItem[], roles: string[]): NavItem[] {
	return items.filter((item) => !item.roles || item.roles.some((r) => roles.includes(r)));
}
