<script setup lang="ts">
import { provide, ref } from 'vue'
import { SpecFocusKey } from './specFocus'

/**
 * Groups SpecCards so clicking one opens it and closes the rest.
 *
 * Until a card is clicked `focused` stays null and every card falls back to its
 * own `active` prop, which is what drives the click-through on these slides.
 */
const focused = ref<string | null>(null)

provide(SpecFocusKey, {
  focused,
  toggle: (name: string) => {
    focused.value = focused.value === name ? null : name
  },
})
</script>

<template>
  <div>
    <slot />
  </div>
</template>
