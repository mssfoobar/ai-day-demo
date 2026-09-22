<script setup lang="ts">
defineProps<{ name?: string }>()
</script>

<template>
  <div class="font-mono text-sm flex gap-3 mt-1.5">
    <span class="text-term-ok">&check;</span>
    <span v-if="name" class="text-term-fg w-36">{{ name }}</span>
    <span :class="name ? 'text-term-dim' : 'text-faint'"><slot /></span>
  </div>
</template>
