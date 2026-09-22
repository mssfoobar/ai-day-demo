<script setup lang="ts">
withDefaults(defineProps<{ title?: string }>(), { title: 'terminal' })
</script>

<template>
  <div class="flex items-center gap-2 mb-5">
    <div v-for="i in 3" :key="i" class="w-[0.7rem] h-[0.7rem] rounded-full bg-term-dimmer" />
    <div class="font-mono text-xs ml-3 text-term-dim">{{ title }}</div>
  </div>
</template>
