import type { Component } from 'svelte';
import Map from '@lucide/svelte/icons/map';
import Truck from '@lucide/svelte/icons/truck';
import { APPLICATION_ROLES } from '$lib/aoh/dispatch/permissions';

// Sidebar navigation for the console. This is the single source of nav entries — add a
// row here when you add a feature page under src/routes/(private)/.
export type NavItem = {
	name: string;
	url: string;
	/** Optional sidebar group label. Entries with the same group render together. */
	group?: string;
	/** Optional icon component. */
	icon?: Component;
	/** Active-tenant roles allowed to see this entry. Omit = visible to everyone. */
	roles?: string[];
};

/**
 * Both surfaces are gated on holding *either* application role, which is the same gate
 * their routes apply. A token carrying neither (a platform-only `tenant-user`, say)
 * reaches a console with an empty sidebar and a permission-denied page rather than a
 * nav entry that leads to a 403.
 */
export const navItems: NavItem[] = [
	{
		name: 'Field units',
		url: '/aoh/dispatch/units',
		group: 'Dispatch',
		icon: Truck,
		roles: [...APPLICATION_ROLES]
	},
	{
		name: 'Map',
		url: '/aoh/dispatch/map',
		group: 'Dispatch',
		icon: Map,
		roles: [...APPLICATION_ROLES]
	}
];

export function filterByRoles(items: NavItem[], roles: string[]): NavItem[] {
	return items.filter((item) => !item.roles || item.roles.some((r) => roles.includes(r)));
}
