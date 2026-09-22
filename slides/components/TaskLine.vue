<script setup lang="ts">
defineProps<{ done: boolean }>()
</script>

<template>
  <div class="deck-mono transition-all duration-500">
    <span class="pr-2" :class="done ? 'text-term-ok' : 'text-term-dimmer'">{{ done ? '- [x]' : '- [ ]' }} </span>
    <span :class="done ? 'text-term-fg' : 'text-term-dim'"><slot /></span>
  </div>
</template>
