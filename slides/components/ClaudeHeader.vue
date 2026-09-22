<script setup lang="ts">
import ClaudeMark from './ClaudeMark.vue'

withDefaults(defineProps<{ pad?: string }>(), { pad: 'mb-6' })
</script>

<template>
  <div class="flex items-center gap-8" :class="pad">
    <ClaudeMark />
    <div class="font-mono text-sm leading-[1.7]">
      <slot />
    </div>
  </div>
</template>
