<script lang="ts">
	import { Navbar as AohNavbar } from "@mssfoobar/ui/navbar";
	import type { NavbarBreadcrumb } from "@mssfoobar/ui/navbar";
	import { Button } from "@mssfoobar/ui/button";
	import { Input } from "@mssfoobar/ui/input";
	import { Bell, HelpCircle, Moon, Search, Sun } from "@lucide/svelte";

	type Props = {
		breadcrumbs: NavbarBreadcrumb[];
		theme?: "light" | "dark";
		onToggleTheme?: () => void;
		searchPlaceholder?: string;
		onSearch?: (q: string) => void;
	};

	let {
		breadcrumbs,
		theme = "light",
		onToggleTheme,
		searchPlaceholder = "Search incidents, dashboards…",
		onSearch,
	}: Props = $props();

	let query = $state("");
</script>

<AohNavbar {breadcrumbs}>
	<div class="hidden w-72 md:block">
		<Input
			placeholder={searchPlaceholder}
			bind:value={query}
			onkeydown={(e) => e.key === "Enter" && onSearch?.(query)}
		/>
	</div>
	<Button variant="ghost" size="icon" aria-label="Help">
		<HelpCircle />
	</Button>
	<Button variant="ghost" size="icon" aria-label="Notifications" class="relative">
		<Bell />
		<span
			class="absolute right-2 top-2 size-2 rounded-full bg-(--bg-error-strong)"
			aria-hidden="true"
		></span>
	</Button>
	<Button
		variant="ghost"
		size="icon"
		aria-label={theme === "dark" ? "Switch to light mode" : "Switch to dark mode"}
		onclick={onToggleTheme}
	>
		{#if theme === "dark"}
			<Sun />
		{:else}
			<Moon />
		{/if}
	</Button>
</AohNavbar>
