<svelte:options runes={true} />

<!--
  The workshop layer: a dialog listing what is still to build.

  Opened from the floating button at the bottom right, or by clicking the sketch that is
  showing. Each card is the user story and one action: "Show me where" hands the exercise
  back to the page, which selects a unit and draws a sketch of the missing control where the
  feature is meant to be built. The full brief lives in WORKSHOP.md, not here.

  Built on AlertDialog because @mssfoobar/ui ships no plain Dialog primitive; it is the
  design system's centred modal. Delete this component once all three exercises are done.
-->

<script lang="ts">
	import { fly } from 'svelte/transition';
	import {
		AlertDialog,
		AlertDialogCancel,
		AlertDialogContent,
		AlertDialogDescription,
		AlertDialogFooter,
		AlertDialogHeader,
		AlertDialogTitle
	} from '@mssfoobar/ui/alert-dialog';
	import { Button } from '@mssfoobar/ui/button';
	import { ScrollArea } from '@mssfoobar/ui/scroll-area';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import Construction from '@lucide/svelte/icons/construction';
	import Crosshair from '@lucide/svelte/icons/crosshair';
	import { completedExercises, incompleteExercises, type ExerciseNumber } from '../workshop';

	let {
		open = $bindable(false),
		onfocus
	}: {
		open?: boolean;
		/** Called when the user asks the console to point at where an exercise goes. */
		onfocus: (exercise: ExerciseNumber) => void;
	} = $props();

	const todo = incompleteExercises();
	const done = completedExercises();

	const intro = $derived(
		todo.length === 0
			? 'Nothing left. Nice work.'
			: todo.length === 1
				? 'One feature is still waiting for someone. Open it up and we’ll point you to where it goes.'
				: `${todo.length} features are still waiting for someone. Pick one and we’ll point you to exactly where it goes.`
	);
</script>

<AlertDialog bind:open>
	<AlertDialogContent
		class="flex w-full flex-col gap-0 p-0 duration-200 data-[size=default]:max-w-xl data-[size=default]:sm:max-w-xl"
	>
		<AlertDialogHeader class="px-6 pt-6 pb-3 text-left">
			<AlertDialogTitle class="flex items-center gap-2">
				<Construction class="size-4 text-(--workshop-text)" aria-hidden="true" />
				What’s left to build
			</AlertDialogTitle>
			<AlertDialogDescription>{intro}</AlertDialogDescription>
		</AlertDialogHeader>

		<!-- ScrollArea needs a definite height to scroll; the flex chain alone was not giving it
		     one, so cap it directly. -->
		<ScrollArea class="max-h-[60dvh]">
			<div class="space-y-3 px-6 py-2">
				{#each todo as ex, i (ex.number)}
					<!-- Staggered entrance: the cards arrive one after another when the dialog opens. -->
					<article
						in:fly={{ y: 16, duration: 280, delay: 60 + i * 70 }}
						class="rounded-md border border-border bg-card p-3 text-card-foreground shadow-[var(--shadow-button)] transition-transform duration-200 hover:-translate-y-0.5"
					>
						<div class="flex items-start gap-3">
							<span
								class="grid size-6 shrink-0 place-items-center rounded-full bg-(--workshop-strong) text-xs font-bold text-(--workshop-fg) tabular-nums"
							>
								{ex.number}
							</span>
							<h3 class="min-w-0 flex-1 text-sm font-bold">{ex.title}</h3>
						</div>

						<p class="mt-2.5 text-sm leading-6">{ex.story}</p>

						<div class="mt-3">
							<Button
								size="sm"
								class="h-8 gap-1.5 bg-(--workshop-strong) text-xs font-semibold text-(--workshop-fg) transition-transform hover:bg-(--workshop-strong)/90 active:scale-95"
								onclick={() => onfocus(ex.number)}
							>
								<Crosshair class="size-3.5" aria-hidden="true" />
								Show me where
							</Button>
						</div>
					</article>
				{/each}

				{#if todo.length === 0}
					<p class="py-8 text-center text-sm text-muted-foreground">
						Everything is built. You can delete the workshop layer now.
					</p>
				{/if}

				{#if done.length > 0}
					<section class="pt-2" in:fly={{ y: 16, duration: 280, delay: 60 + todo.length * 70 }}>
						<h3 class="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
							Already built
						</h3>
						<ul class="divide-y divide-border">
							{#each done as ex (ex.number)}
								<li class="flex items-center gap-2 py-1.5 text-sm text-muted-foreground">
									<CircleCheck class="size-4 text-success" aria-hidden="true" />
									<span class="line-through">{ex.title}</span>
								</li>
							{/each}
						</ul>
					</section>
				{/if}
			</div>
		</ScrollArea>

		<AlertDialogFooter class="flex-row items-center justify-between gap-2 border-t px-6 py-3">
			<p class="text-xs text-muted-foreground">
				Full brief in <code class="font-mono">WORKSHOP.md</code>.
			</p>
			<AlertDialogCancel>Got it</AlertDialogCancel>
		</AlertDialogFooter>
	</AlertDialogContent>
</AlertDialog>
