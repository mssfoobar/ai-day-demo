<svelte:options runes={true} />

<script lang="ts">
	// App sidebar: renders the hardcoded `navItems` (see ./nav.ts) onto the
	// @mssfoobar/ui Sidebar primitive. Entries are grouped by their `group`
	// label and role-filtered so a user only sees pages their active-tenant
	// roles permit. Must render inside `Sidebar.Provider` (see (private)/+layout.svelte).
	import { page } from '$app/state';
	import { SvelteMap } from 'svelte/reactivity';
	import { Sidebar } from '@mssfoobar/ui/sidebar';
	import type { SidebarGroup, SidebarItem } from '@mssfoobar/ui/sidebar';
	import { navItems, filterByRoles } from '$lib/aoh/core/components/layout/nav';

	let {
		title = 'AGIL Ops Hub',
		description,
		roles = [],
		userName = 'User',
		userEmail,
		defaultGroup = 'Navigation'
	}: {
		title?: string;
		description?: string;
		roles?: string[];
		userName?: string;
		userEmail?: string;
		defaultGroup?: string;
	} = $props();

	const groups = $derived.by<SidebarGroup[]>(() => {
		const path = page.url.pathname.replace(/\/$/, '');
		const order: string[] = [];
		const byGroup = new SvelteMap<string, SidebarItem[]>();
		for (const item of filterByRoles(navItems, roles)) {
			const label = item.group ?? defaultGroup;
			if (!byGroup.has(label)) {
				byGroup.set(label, []);
				order.push(label);
			}
			const url = item.url.replace(/\/$/, '');
			byGroup.get(label)!.push({
				title: item.name,
				url: item.url,
				icon: item.icon,
				// Exact match so a child route (e.g. `/foo/new`) highlights its own
				// entry, not a parent entry at `/foo`.
				isActive: path === url
			});
		}
		return order.map((label) => ({ label, items: byGroup.get(label)! }));
	});
</script>

<Sidebar {title} {description} {groups} user={{ name: userName, email: userEmail }} />
