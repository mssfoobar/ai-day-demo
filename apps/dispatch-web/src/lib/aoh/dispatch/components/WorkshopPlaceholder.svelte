<svelte:options runes={true} />

<!--
  Workshop placeholder — stands in for the form an exercise will add.

  Opens where the real Sheet will open (Dispatch, Manage crew) and says, in the UI itself,
  what is meant to be built here. Delete this component once all three exercises are done;
  WORKSHOP.md at the repo root is the full brief.
-->

<script lang="ts">
	import { Badge } from '@mssfoobar/ui/badge';
	import { Button } from '@mssfoobar/ui/button';
	import { ScrollArea } from '@mssfoobar/ui/scroll-area';
	import {
		Sheet,
		SheetContent,
		SheetDescription,
		SheetFooter,
		SheetHeader,
		SheetTitle
	} from '@mssfoobar/ui/sheet';
	import Construction from '@lucide/svelte/icons/construction';
	import { EXERCISES, PLACEHOLDER_CLASS, type ExerciseNumber } from '../workshop';

	let {
		open = $bindable(false),
		exercise
	}: {
		open?: boolean;
		exercise: ExerciseNumber | null;
	} = $props();

	const ex = $derived(exercise === null ? null : EXERCISES[exercise]);
</script>

<Sheet bind:open>
	<SheetContent side="right" class="flex w-full flex-col gap-0 p-0 sm:max-w-md">
		{#if ex}
			<SheetHeader class="px-5 pt-5 pb-3">
				<Badge variant="solid" color="warning" class="w-fit">Exercise {ex.number}</Badge>
				<SheetTitle class="flex items-center gap-2">
					<Construction class="size-4 text-muted-foreground" aria-hidden="true" />
					{ex.title}
				</SheetTitle>
				<SheetDescription>Not implemented yet — this is a workshop exercise.</SheetDescription>
			</SheetHeader>

			<ScrollArea class="min-h-0 flex-1">
				<div class="space-y-5 px-5 py-2 text-sm">
					<div class="{PLACEHOLDER_CLASS} rounded-md p-5 text-center">
						<Construction class="mx-auto mb-2 size-8" aria-hidden="true" />
						<p class="text-sm font-bold tracking-wide uppercase">Build it here</p>
						<p class="mt-1 text-xs">
							Replace this Sheet with the real form when you build the exercise.
						</p>
					</div>

					<section>
						<h3 class="mb-1.5 text-xs font-medium tracking-wide text-muted-foreground uppercase">
							User story
						</h3>
						<p>{ex.story}</p>
					</section>

					<section>
						<h3 class="mb-1.5 text-xs font-medium tracking-wide text-muted-foreground uppercase">
							Acceptance criteria
						</h3>
						<ol class="list-decimal space-y-1 pl-5">
							{#each ex.criteria as criterion (criterion)}
								<li>{criterion}</li>
							{/each}
						</ol>
					</section>

					<section>
						<h3 class="mb-1.5 text-xs font-medium tracking-wide text-muted-foreground uppercase">
							Service stub — answers 501 today
						</h3>
						<ul class="space-y-1">
							{#each ex.routes as route (route)}
								<li><code class="font-mono text-xs">{route}</code></li>
							{/each}
						</ul>
					</section>

					<p class="text-xs text-muted-foreground">
						Full brief, pointers to the code to copy, and what is out of scope:
						<code class="font-mono">WORKSHOP.md</code> at the repo root.
					</p>
				</div>
			</ScrollArea>

			<SheetFooter class="flex-row justify-end gap-2 border-t px-5 py-3">
				<Button type="button" variant="outline" onclick={() => (open = false)}>Close</Button>
			</SheetFooter>
		{/if}
	</SheetContent>
</Sheet>
