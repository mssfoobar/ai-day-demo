<script setup lang="ts">
withDefaults(defineProps<{ sigil?: string; pad?: string }>(), { sigil: '>', pad: '' })
</script>

<template>
  <div class="font-mono text-lg" :class="pad">
    <span class="text-term-dim">{{ `${sigil} ` }}</span><span class="text-term-fg"><slot /></span>
  </div>
</template>
