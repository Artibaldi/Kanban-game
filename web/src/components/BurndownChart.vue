<script setup lang="ts">
// Remaining backlog points (line) with scope growth above the axis and completed work below it,
// so a change of scope is visible apart from the burn.
import { computed, ref } from 'vue'
import { date } from '../lib/format'

interface Day { date: string; added: number; reestimated: number; reopened: number; completed: number; remaining: number }
const props = defineProps<{ days: Day[] }>()
const W = 1000, H = 300, padL = 36, padR = 12, padT = 16, padB = 28
const hover = ref<number | null>(null)
const maxRem = computed(() => Math.max(8, ...props.days.map(d => d.remaining)))
const maxBar = computed(() => Math.max(4, ...props.days.map(d => Math.max(d.added + Math.max(0, d.reestimated) + d.reopened, d.completed))))
const xs = (i: number) => padL + (i + 0.5) * ((W - padL - padR) / props.days.length)
const lineH = H - padT - padB - 70
const y = (v: number) => padT + lineH - (v / maxRem.value) * lineH
const barBase = H - padB - 35
const barScale = (v: number) => (v / maxBar.value) * 32
const bw = computed(() => Math.max(3, (W - padL - padR) / props.days.length - 4))
const path = computed(() => props.days.map((d, i) => `${i ? 'L' : 'M'}${xs(i).toFixed(1)} ${y(d.remaining).toFixed(1)}`).join(' '))
const ticks = computed(() => [0, 0.5, 1].map(f => Math.round(maxRem.value * f)))
const summary = computed(() => {
  const s = props.days.reduce((a, d) => ({ added: a.added + d.added + d.reopened + d.reestimated, done: a.done + d.completed }), { added: 0, done: 0 })
  const last = props.days[props.days.length - 1]?.remaining ?? 0
  return `За период: добавлено ${s.added} баллов (включая переоценки и возвраты), выполнено ${s.done}, осталось ${last}.`
})
const h = computed(() => (hover.value === null ? null : props.days[hover.value]))
</script>

<template>
  <figure class="chart">
    <svg :viewBox="`0 0 ${W} ${H}`" role="img" :aria-label="summary" @mouseleave="hover = null">
      <g class="grid">
        <line v-for="t in ticks" :key="t" :x1="padL" :x2="W - padR" :y1="y(t)" :y2="y(t)" />
        <text v-for="t in ticks" :key="'t' + t" :x="padL - 6" :y="y(t) + 4" text-anchor="end">{{ t }}</text>
      </g>
      <line :x1="padL" :x2="W - padR" :y1="barBase" :y2="barBase" class="axis" />
      <g v-for="(d, i) in days" :key="d.date">
        <rect :x="xs(i) - bw / 2" :y="barBase - barScale(d.added + Math.max(0, d.reestimated) + d.reopened)" :width="bw" :height="barScale(d.added + Math.max(0, d.reestimated) + d.reopened)" class="added" rx="1.5" />
        <rect :x="xs(i) - bw / 2" :y="barBase" :width="bw" :height="barScale(d.completed)" class="done" rx="1.5" />
        <rect :x="xs(i) - bw / 2 - 2" :y="padT" :width="bw + 4" :height="H - padT - padB" fill="transparent" @mouseenter="hover = i" />
      </g>
      <path :d="path" class="line" />
      <circle v-if="h" :cx="xs(hover!)" :cy="y(h.remaining)" r="4.5" class="dot" />
      <text v-for="(d, i) in days" v-show="i % Math.ceil(days.length / 6) === 0" :key="'x' + i" :x="xs(i)" :y="H - 6" text-anchor="middle" class="xl">{{ date(d.date) }}</text>
    </svg>
    <figcaption class="legend xs">
      <span><i class="k line-k" />Остаток, баллы</span><span><i class="k added-k" />Добавлено и переоценено</span><span><i class="k done-k" />Выполнено</span>
      <span v-if="h" class="tip">{{ date(h.date) }}: осталось {{ h.remaining }}, +{{ h.added }} новых, {{ h.reestimated >= 0 ? '+' : '' }}{{ h.reestimated }} переоценка, +{{ h.reopened }} возвращено, −{{ h.completed }} выполнено</span>
    </figcaption>
    <p class="sr-only">{{ summary }}</p>
  </figure>
</template>

<style scoped>
.chart { margin: 0; }
svg { width: 100%; height: auto; display: block; }
.grid line { stroke: var(--color-border); stroke-dasharray: 3 3; }
.grid text, .xl { fill: var(--color-text-muted); font-size: 11px; }
.axis { stroke: var(--color-border-strong); }
.line { fill: none; stroke: var(--color-action-primary); stroke-width: 2.5; stroke-linejoin: round; }
.dot { fill: var(--color-surface); stroke: var(--color-action-primary); stroke-width: 2.5; }
.added { fill: var(--color-warning); opacity: 0.75; }
.done { fill: var(--color-success); opacity: 0.8; }
.legend { display: flex; flex-wrap: wrap; gap: var(--space-4); color: var(--color-text-muted); margin-top: var(--space-2); }
.legend span { display: inline-flex; align-items: center; gap: 6px; }
.k { display: inline-block; width: 12px; height: 10px; border-radius: 2px; }
.line-k { height: 3px; background: var(--color-action-primary); }
.added-k { background: var(--color-warning); }
.done-k { background: var(--color-success); }
.tip { color: var(--color-text); font-weight: 600; }
</style>
