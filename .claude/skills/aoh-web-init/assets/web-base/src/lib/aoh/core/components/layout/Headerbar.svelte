<svelte:options runes={true} />

<script lang="ts">
	// Top header bar: breadcrumb (derived from the active route + ./nav.ts), the
	// mobile sidebar trigger, the signed-in user, and the logout action. The
	// @mssfoobar/ui Sidebar footer surfaces the user but has no logout control,
	// so logout lives here. Add app-specific header controls (e.g. a
	// notification bell) as children before the user block.
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { Navbar, type NavbarBreadcrumb } from '@mssfoobar/ui/navbar';
	import { MobileTrigger } from '@mssfoobar/ui/sidebar';
	import { Button } from '@mssfoobar/ui/button';
	import { Avatar, AvatarFallback } from '@mssfoobar/ui/avatar';
	import { navItems } from '$lib/aoh/core/components/layout/nav';

	let { name = 'User' }: { name?: string } = $props();

	const initials = $derived(
		name
			.split(' ')
			.map((part) => part[0])
			.filter(Boolean)
			.slice(0, 2)
			.join('')
			.toUpperCase() || 'U'
	);

	const breadcrumbs = $derived.by<NavbarBreadcrumb[]>(() => {
		const path = page.url.pathname.replace(/\/$/, '');
		const match = navItems.find((item) => item.url.replace(/\/$/, '') === path);
		if (!match) return [];
		return match.group
			? [{ label: match.group }, { label: match.name, current: true }]
			: [{ label: match.name, current: true }];
	});

	const logoutUrl = resolve('/aoh/api/auth/logout');
</script>

<Navbar {breadcrumbs} data-testid="topnav">
	<MobileTrigger />
	<div class="flex items-center gap-2">
		<Avatar class="size-8">
			<AvatarFallback class="text-xs">{initials}</AvatarFallback>
		</Avatar>
		<span class="hidden text-sm font-medium sm:inline">{name}</span>
	</div>
	<!-- Full-page navigation: logout is a server endpoint that clears the SDS
	     session and redirects, not a client-routable page. -->
	<Button variant="outline" size="sm" href={logoutUrl} data-sveltekit-reload aria-label="Log out">
		<!-- Inline SVG (Lucide "log-out") keeps the bundle free of a lucide dep. -->
		<svg
			viewBox="0 0 24 24"
			class="size-4"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
			aria-hidden="true"
		>
			<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
			<polyline points="16 17 21 12 16 7" />
			<line x1="21" x2="9" y1="12" y2="12" />
		</svg>
		<span class="hidden sm:inline">Logout</span>
	</Button>
</Navbar>
