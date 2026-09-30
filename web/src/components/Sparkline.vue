<script setup lang="ts">
import { computed } from 'vue'
const props = defineProps<{ values: number[] }>()
const max = computed(() => Math.max(1, ...props.values))
</script>

<template>
  <svg viewBox="0 0 80 24" width="80" height="24" aria-hidden="true" class="spark">
    <rect v-for="(v, i) in values" :key="i" :x="i * 10 + 1" :y="22 - (v / max) * 20" width="7" :height="Math.max(1, (v / max) * 20)" rx="1.5" :class="{ cur: i === values.length - 1 }" />
  </svg>
</template>

<style scoped>
rect { fill: color-mix(in srgb, var(--color-action-primary) 40%, transparent); }
rect.cur { fill: var(--color-action-primary); }
</style>
