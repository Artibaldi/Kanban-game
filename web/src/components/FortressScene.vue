<script setup lang="ts">
import { computed } from 'vue'
import { enemyName } from '../lib/fortress'
import type { Enemy, Fortress } from '../types'

// The team fortress at the bottom, the horde in four rows by priority above it (P1 nearest the walls),
// overdue enemies at the wall itself, and the backlog as fog on the horizon.
const props = defineProps<{ fortress: Fortress; rows: Enemy[][]; fog: number; selectedId?: string | null }>()
const emit = defineEmits<{ select: [enemy: Enemy] }>()

const W = 640, H = 430, PER_ROW = 9
const WALL_Y = 330
const rowY = [236, 188, 140, 100] // P1 … P4
const size: Record<Enemy['kind'], number> = { goblin: 9, knight: 12, giant: 15, dragon: 18 }
const colors: Record<Enemy['kind'], [string, string]> = {
  goblin: ['#6f9b3a', '#4d6e24'], knight: ['#8a8f99', '#5f646d'], giant: ['#a3683f', '#74462a'], dragon: ['#b3402f', '#7f2a1f'],
}

interface Placed { e: Enemy; x: number; y: number; r: number }
const placed = computed(() => {
  const out: Placed[] = []
  const atWall = props.rows.flat().filter(e => e.overdue)
  const spread = (list: Enemy[], y: number) => {
    const shown = list.slice(0, PER_ROW)
    shown.forEach((e, i) => out.push({ e, x: (W / (shown.length + 1)) * (i + 1), y, r: size[e.kind] }))
  }
  spread(atWall, WALL_Y - 50)
  props.rows.forEach((row, i) => spread(row.filter(e => !e.overdue), rowY[i]))
  return out
})
const hidden = computed(() => props.rows.map(r => Math.max(0, r.filter(e => !e.overdue).length - PER_ROW)))
const cracks = computed(() => (props.fortress.hp < 40 ? 3 : props.fortress.hp < 70 ? 2 : props.fortress.hp < 95 ? 1 : 0))
const label = (e: Enemy) =>
  `${enemyName(e.kind)}: ${e.number} ${e.title}, ${e.points} балл., ${e.priority}${e.overdue ? ', просрочена' : e.at_gates ? ', срок близко' : ''}`
</script>

<template>
  <svg class="scene" :viewBox="`0 0 ${W} ${H}`" role="group" aria-label="Крепость команды и орда задач" preserveAspectRatio="xMidYMid meet">
    <defs>
      <linearGradient id="fs-sky" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="var(--art-sky)" />
        <stop offset="1" stop-color="var(--art-grass-1)" />
      </linearGradient>
    </defs>
    <rect x="0" y="0" :width="W" :height="H" fill="url(#fs-sky)" />
    <rect x="0" y="80" :width="W" :height="H - 80" fill="var(--art-grass-2)" opacity="0.55" />
    <!-- fog: the backlog -->
    <g class="fog" aria-hidden="true">
      <ellipse v-for="i in 7" :key="i" :cx="(i - 0.5) * (W / 7)" cy="52" :rx="70" ry="26" fill="#fff" opacity="0.75" />
    </g>
    <text :x="W / 2" y="58" text-anchor="middle" class="fog-text">Туман: {{ fog }} в бэклоге</text>
    <!-- row hints -->
    <g aria-hidden="true">
      <text v-for="(y, i) in rowY" :key="y" x="10" :y="y + 4" class="row-text">P{{ i + 1 }}</text>
      <text v-for="(n, i) in hidden" v-show="n > 0" :key="'h' + i" :x="W - 10" :y="rowY[i] + 4" text-anchor="end" class="row-text">+{{ n }}</text>
    </g>
    <!-- enemies -->
    <g v-for="p in placed" :key="p.e.task_id" class="enemy" :class="{ overdue: p.e.overdue, gates: p.e.at_gates, sel: p.e.task_id === selectedId }"
      role="button" tabindex="0" :aria-label="label(p.e)" :transform="`translate(${p.x} ${p.y})`"
      @click="emit('select', p.e)" @keydown.enter.prevent="emit('select', p.e)" @keydown.space.prevent="emit('select', p.e)">
      <title>{{ label(p.e) }}</title>
      <ellipse :cy="p.r * 0.95" :rx="p.r * 0.9" :ry="p.r * 0.3" fill="#000" opacity="0.18" />
      <circle :r="p.r" :fill="colors[p.e.kind][0]" :stroke="colors[p.e.kind][1]" stroke-width="2" />
      <circle :cx="-p.r * 0.35" :cy="-p.r * 0.2" :r="p.r * 0.18" fill="#fff" />
      <circle :cx="p.r * 0.35" :cy="-p.r * 0.2" :r="p.r * 0.18" fill="#fff" />
      <circle :cx="-p.r * 0.32" :cy="-p.r * 0.17" :r="p.r * 0.08" fill="#1d2521" />
      <circle :cx="p.r * 0.38" :cy="-p.r * 0.17" :r="p.r * 0.08" fill="#1d2521" />
      <path v-if="p.e.kind === 'dragon'" :d="`M ${-p.r} ${-p.r * 0.6} L ${-p.r * 1.7} ${-p.r * 1.3} L ${-p.r * 0.7} ${-p.r * 0.9} Z M ${p.r} ${-p.r * 0.6} L ${p.r * 1.7} ${-p.r * 1.3} L ${p.r * 0.7} ${-p.r * 0.9} Z`" :fill="colors.dragon[1]" />
      <path v-if="p.e.kind === 'knight'" :d="`M ${-p.r * 0.8} ${-p.r * 0.55} Q 0 ${-p.r * 1.5} ${p.r * 0.8} ${-p.r * 0.55} Z`" fill="#c9ccd2" />
      <circle v-if="p.e.overdue || p.e.at_gates" class="ring" :r="p.r + 5" fill="none" />
      <text :y="p.r + 14" text-anchor="middle" class="num-text">{{ p.e.number }}</text>
    </g>
    <!-- fortress -->
    <g class="fortress" aria-hidden="true">
      <rect x="40" :y="WALL_Y" :width="W - 80" height="56" fill="var(--art-stone)" stroke="var(--art-stone-dark)" stroke-width="2" />
      <rect v-for="i in 14" :key="'m' + i" :x="40 + (i - 1) * ((W - 80) / 14) + 6" :y="WALL_Y - 12" :width="(W - 80) / 14 - 12" height="12" fill="var(--art-stone-light)" stroke="var(--art-stone-dark)" stroke-width="1.5" />
      <rect x="20" :y="WALL_Y - 40" width="56" height="96" fill="var(--art-stone-light)" stroke="var(--art-stone-dark)" stroke-width="2" />
      <rect :x="W - 76" :y="WALL_Y - 40" width="56" height="96" fill="var(--art-stone-light)" stroke="var(--art-stone-dark)" stroke-width="2" />
      <path :d="`M 16 ${WALL_Y - 40} L 48 ${WALL_Y - 74} L 80 ${WALL_Y - 40} Z`" fill="var(--art-roof-hall)" />
      <path :d="`M ${W - 80} ${WALL_Y - 40} L ${W - 48} ${WALL_Y - 74} L ${W - 16} ${WALL_Y - 40} Z`" fill="var(--art-roof-hall)" />
      <path :d="`M ${W / 2 - 34} ${WALL_Y + 56} L ${W / 2 - 34} ${WALL_Y + 18} Q ${W / 2} ${WALL_Y - 8} ${W / 2 + 34} ${WALL_Y + 18} L ${W / 2 + 34} ${WALL_Y + 56} Z`" fill="var(--art-wood)" stroke="var(--art-wood-dark)" stroke-width="2" />
      <line x1="48" :y1="WALL_Y - 74" x2="48" :y2="WALL_Y - 104" stroke="var(--art-wood-dark)" stroke-width="3" />
      <path :d="`M 48 ${WALL_Y - 104} L 76 ${WALL_Y - 96} L 48 ${WALL_Y - 88} Z`" fill="var(--art-banner)" />
      <path v-if="cracks >= 1" :d="`M 150 ${WALL_Y + 4} l 10 14 l -8 10 l 12 16`" stroke="var(--art-stone-dark)" stroke-width="2.5" fill="none" />
      <path v-if="cracks >= 2" :d="`M 470 ${WALL_Y + 6} l -9 12 l 10 12 l -7 18`" stroke="var(--art-stone-dark)" stroke-width="2.5" fill="none" />
      <path v-if="cracks >= 3" :d="`M 250 ${WALL_Y + 2} l 6 18 l -12 8 l 8 20`" stroke="var(--art-stone-dark)" stroke-width="2.5" fill="none" />
      <g v-if="fortress.siege">
        <circle v-for="i in 4" :key="'s' + i" :cx="160 + i * 70" :cy="WALL_Y - 30 - (i % 2) * 14" :r="16 + (i % 3) * 4" fill="#6b6b6b" opacity="0.35" />
      </g>
    </g>
    <rect x="0" :y="WALL_Y + 56" :width="W" :height="H - WALL_Y - 56" fill="var(--art-soil)" opacity="0.85" />
  </svg>
</template>

<style scoped>
.scene { display: block; width: 100%; height: auto; border-radius: var(--radius-md); }
.fog-text { font: 700 14px var(--font-sans); fill: var(--color-text-muted); }
.row-text { font: 700 11px var(--font-sans); fill: rgba(29, 37, 33, 0.55); }
.num-text { font: 650 10px var(--font-sans); fill: #1d2521; paint-order: stroke; stroke: rgba(255, 255, 255, 0.85); stroke-width: 3px; }
.enemy { cursor: pointer; outline: none; transition: transform var(--duration-fast) var(--ease-out); }
.enemy .ring { stroke: var(--color-warning); stroke-width: 2; stroke-dasharray: 4 3; }
.enemy.overdue .ring { stroke: var(--color-danger); stroke-dasharray: none; }
.enemy:hover circle:first-of-type, .enemy:focus-visible circle:first-of-type { filter: brightness(1.12); }
.enemy:focus-visible .num-text, .enemy.sel .num-text { fill: var(--color-focus); }
.enemy.sel circle:nth-of-type(1) { stroke: var(--color-focus); stroke-width: 3; }
@media (prefers-reduced-motion: no-preference) {
  body:not(.work-mode) .enemy.overdue { animation: bob 1.6s ease-in-out infinite; }
}
@keyframes bob { 50% { translate: 0 -3px; } }
</style>
