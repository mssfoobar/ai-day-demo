<script setup lang="ts">
withDefaults(defineProps<{ width?: string }>(), { width: '46rem' })
</script>

<template>
  <div class="deck-panel" :style="{ width }">
    <slot />
  </div>
</template>
