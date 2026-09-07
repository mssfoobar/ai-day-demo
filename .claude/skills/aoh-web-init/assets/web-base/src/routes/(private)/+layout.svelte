<svelte:options runes={true} />

<script lang="ts">
	import AuthProvider from '$lib/aoh/core/provider/auth/AuthProvider/index.svelte';
	import { Toaster } from '@mssfoobar/ui/toast';
	import { Provider as SidebarProvider } from '@mssfoobar/ui/sidebar';
	import Sidebar from '$lib/aoh/core/components/layout/Sidebar.svelte';
	import Headerbar from '$lib/aoh/core/components/layout/Headerbar.svelte';

	let { data, children } = $props();

	const tenantName = $derived(data.user?.active_tenant?.tenant_name ?? '');
	// Sidebar entries gate on the active tenant's roles (see ./nav.ts). AAS
	// surfaces roles into the JWT today, not resolved permissions, so role
	// membership is the gate the sidebar can act on.
	const roles = $derived(data.user?.active_tenant?.roles ?? []);
	// IDToken claims are loosely typed (JsonValue | undefined); coerce to string
	// for the display name and only keep `email` when it's actually a string.
	const displayName = $derived(
		String(data.user?.name ?? data.user?.preferred_username ?? data.user?.sub ?? 'User')
	);
	const userEmail = $derived(typeof data.user?.email === 'string' ? data.user.email : undefined);
</script>

<AuthProvider claims={data.user}>
	<SidebarProvider>
		<Sidebar description={tenantName} {roles} userName={displayName} {userEmail} />
		<div class="flex min-h-svh min-w-0 flex-1 flex-col">
			<Headerbar name={displayName} />
			<main class="flex-1 overflow-auto bg-background text-foreground">
				{@render children()}
			</main>
		</div>
	</SidebarProvider>
	<!--
		Toaster MUST be mounted somewhere reachable from any private page so
		`toast.success(...)` calls from feature pages produce visible output.
		Without this mount, every `toast.*()` call is a silent no-op and the
		UI just looks "stuck after submit." Place it inside AuthProvider so
		auth-related toasts (e.g. session-expired notifications) work too.
	-->
	<Toaster richColors closeButton />
</AuthProvider>
