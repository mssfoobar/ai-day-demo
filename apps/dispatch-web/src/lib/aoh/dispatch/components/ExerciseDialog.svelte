<svelte:options runes={true} />

<!--
  The workshop layer: a dialog listing what is still to build.

  Opened from the floating button at the bottom right, or by clicking any exercise pin. Each
  card can open its story inline, and "Show me where" hands the exercise back to the page,
  which selects a unit and draws a highlighted border around the area where the feature is
  meant to be built.

  Built on AlertDialog because @mssfoobar/ui ships no plain Dialog primitive; it is the
  design system's centred modal. Delete this component once all three exercises are done;
  WORKSHOP.md is the full brief.
-->

<script lang="ts">
	import { fly, slide } from 'svelte/transition';
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
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import Construction from '@lucide/svelte/icons/construction';
	import Crosshair from '@lucide/svelte/icons/crosshair';
	import { completedExercises, incompleteExercises, type ExerciseNumber } from '../workshop';

	let {
		open = $bindable(false),
		detail = $bindable(null),
		onfocus
	}: {
		open?: boolean;
		/** The exercise whose story is open, if any. */
		detail?: ExerciseNumber | null;
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

	function toggle(n: ExerciseNumber) {
		detail = detail === n ? null : n;
	}
</script>

<AlertDialog bind:open>
	<AlertDialogContent
		class="flex max-h-[85dvh] w-full flex-col gap-0 p-0 duration-200 data-[size=default]:max-w-xl data-[size=default]:sm:max-w-xl"
	>
		<AlertDialogHeader class="px-6 pt-6 pb-3 text-left">
			<AlertDialogTitle class="flex items-center gap-2">
				<Construction class="size-4 text-(--workshop-text)" aria-hidden="true" />
				What’s left to build
			</AlertDialogTitle>
			<AlertDialogDescription>{intro}</AlertDialogDescription>
		</AlertDialogHeader>

		<ScrollArea class="min-h-0 flex-1">
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
							<div class="min-w-0 flex-1">
								<h3 class="text-sm font-bold">{ex.title}</h3>
							</div>
						</div>

						<p class="mt-2.5 text-sm leading-6">{ex.story}</p>

						<div class="mt-3 flex flex-wrap items-center gap-2">
							<Button
								size="sm"
								class="h-8 gap-1.5 bg-(--workshop-strong) text-xs font-semibold text-(--workshop-fg) transition-transform hover:bg-(--workshop-strong)/90 active:scale-95"
								onclick={() => onfocus(ex.number)}
							>
								<Crosshair class="size-3.5" aria-hidden="true" />
								Show me where
							</Button>
							<Button
								variant="ghost"
								size="sm"
								class="h-8 gap-1 text-xs"
								aria-expanded={detail === ex.number}
								onclick={() => toggle(ex.number)}
							>
								{detail === ex.number ? 'Less' : 'Tell me more'}
								<ChevronDown
									class="size-3.5 transition-transform duration-200 {detail === ex.number
										? 'rotate-180'
										: ''}"
									aria-hidden="true"
								/>
							</Button>
						</div>

						{#if detail === ex.number}
							<div
								transition:slide={{ duration: 220 }}
								class="mt-3 space-y-3 border-t border-border pt-3 text-xs"
							>
								<section>
									<h4 class="mb-1 font-semibold text-muted-foreground">Where it goes</h4>
									<p>{ex.where}</p>
								</section>
								<section>
									<h4 class="mb-1 font-semibold text-muted-foreground">You’re done when…</h4>
									<ul class="space-y-1">
										{#each ex.done as item (item)}
											<li class="flex gap-2">
												<CircleCheck
													class="mt-0.5 size-3.5 shrink-0 text-muted-foreground/60"
													aria-hidden="true"
												/>
												<span>{item}</span>
											</li>
										{/each}
									</ul>
								</section>
								<section>
									<h4 class="mb-1 font-semibold text-muted-foreground">On the service side</h4>
									<p class="mb-1 text-muted-foreground">
										These routes already exist and answer 501 until you make them real:
									</p>
									<ul class="space-y-0.5">
										{#each ex.routes as route (route)}
											<li><code class="font-mono">{route}</code></li>
										{/each}
									</ul>
								</section>
								<p class="text-muted-foreground">
									The full brief, including the code worth copying, is in
									<code class="font-mono">WORKSHOP.md</code> at the repo root.
								</p>
							</div>
						{/if}
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

		<AlertDialogFooter class="flex-row justify-end gap-2 border-t px-6 py-3">
			<AlertDialogCancel>Got it</AlertDialogCancel>
		</AlertDialogFooter>
	</AlertDialogContent>
</AlertDialog>
