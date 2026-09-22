<script setup lang="ts">
defineProps<{ solid?: boolean }>()
</script>

<template>
  <div
    class="rounded-lg border-2 border-brand text-lg py-3 text-center w-[30rem]"
    :class="solid ? 'font-bold bg-brand text-white' : 'font-medium bg-brand-wash text-brand'"
  >
    <slot />
  </div>
</template>
