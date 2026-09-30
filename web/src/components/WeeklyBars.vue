<script setup lang="ts">
import { computed } from 'vue'
import { date } from '../lib/format'
const props = defineProps<{ weeks: string[]; values: number[]; label: string }>()
const W = 360, H = 150, pad = 22
const max = computed(() => Math.max(5, ...props.values))
const bw = computed(() => (W - pad * 2) / props.values.length - 8)
const desc = computed(() => props.label + ': ' + props.weeks.map((w, i) => `неделя с ${date(w)} — ${props.values[i]}`).join('; '))
</script>

<template>
  <svg :viewBox="`0 0 ${W} ${H}`" role="img" :aria-label="desc" class="wb">
    <g v-for="(v, i) in values" :key="i">
      <rect :x="pad + i * (bw + 8) + 4" :y="H - 24 - (v / max) * (H - 50)" :width="bw" :height="Math.max(1, (v / max) * (H - 50))" rx="3" :class="{ cur: i === values.length - 1 }" />
      <text :x="pad + i * (bw + 8) + 4 + bw / 2" :y="H - 28 - (v / max) * (H - 50)" text-anchor="middle" class="v">{{ v || '' }}</text>
      <text :x="pad + i * (bw + 8) + 4 + bw / 2" :y="H - 8" text-anchor="middle" class="w">{{ date(weeks[i]) }}</text>
    </g>
  </svg>
</template>

<style scoped>
.wb { width: 100%; height: auto; display: block; }
rect { fill: color-mix(in srgb, var(--color-action-primary) 45%, transparent); }
rect.cur { fill: var(--color-action-primary); }
.v { fill: var(--color-text); font-size: 11px; font-weight: 700; }
.w { fill: var(--color-text-muted); font-size: 9.5px; }
</style>
