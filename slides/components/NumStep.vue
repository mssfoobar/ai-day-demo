<script setup lang="ts">
withDefaults(defineProps<{ n: number | string; body?: string; pad?: string }>(), {
  body: 'text-xl',
  pad: 'py-2.5',
})
</script>

<template>
  <div class="deck-row" :class="pad">
    <div class="font-mono text-base w-10 text-brand">{{ n }}</div>
    <div class="text-ink" :class="body"><slot /></div>
  </div>
</template>
