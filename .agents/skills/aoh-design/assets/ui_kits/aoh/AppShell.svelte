<script lang="ts">
	import { Provider as SidebarProvider } from "@mssfoobar/ui/sidebar";
	import type { SidebarGroup } from "@mssfoobar/ui/sidebar";
	import type { Snippet } from "svelte";

	import Sidebar from "./Sidebar.svelte";

	type Props = {
		/** Route key, forwarded to `Sidebar.svelte` to mark the active item. */
		active?: string;
		/** Optional override for the default AOH sidebar groups. */
		groups?: SidebarGroup[];
		children: Snippet;
	};

	let { active = "dashboard", groups, children }: Props = $props();
</script>

<SidebarProvider>
	<Sidebar {active} {groups} />
	<main class="flex min-h-svh w-full flex-1 flex-col bg-(--bg-secondary)">
		{@render children()}
	</main>
</SidebarProvider>
