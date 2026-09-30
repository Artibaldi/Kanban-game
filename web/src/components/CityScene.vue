<script setup lang="ts">
import { computed } from 'vue'
import BuildingSprite from './BuildingSprite.vue'
import type { BuildingType, CityBuilding } from '../types'

const props = withDefaults(defineProps<{
  buildings: CityBuilding[]; grid?: number; selectedId?: string | null; placing?: BuildingType | null
  now?: number; ruined?: BuildingType[]; showLevels?: boolean; label?: string; compact?: boolean
}>(), { grid: 12, selectedId: null, placing: null, now: 0, ruined: () => [], showLevels: true, label: 'Город' })
const emit = defineEmits<{ select: [b: CityBuilding]; cell: [x: number, y: number] }>()

const TW = 32, TH = 16
const pos = (x: number, y: number) => ({ sx: (x - y) * TW, sy: (x + y) * TH })
const tiles = computed(() => {
  const out: { x: number; y: number; d: string; alt: boolean; free: boolean }[] = []
  const used = new Set(props.buildings.map(b => `${b.x},${b.y}`))
  for (let y = 0; y < props.grid; y++) for (let x = 0; x < props.grid; x++) {
    const { sx, sy } = pos(x, y)
    out.push({ x, y, alt: (x + y) % 2 === 0, free: !used.has(`${x},${y}`), d: `${sx},${sy - TH} ${sx + TW},${sy} ${sx},${sy + TH} ${sx - TW},${sy}` })
  }
  return out
})
const sorted = computed(() => [...props.buildings].sort((a, b) => a.x + a.y - (b.x + b.y) || a.x - b.x))
const N = computed(() => props.grid)
const edge = computed(() => {
  const n = N.value
  const top = [0, -TH], right = [n * TW, (n - 1) * TH], bottom = [0, (2 * n - 1) * TH], left = [-n * TW, (n - 1) * TH]
  const depth = 14
  return {
    left: `${left[0]},${left[1]} ${bottom[0]},${bottom[1]} ${bottom[0]},${bottom[1] + depth} ${left[0]},${left[1] + depth}`,
    right: `${bottom[0]},${bottom[1]} ${right[0]},${right[1]} ${right[0]},${right[1] + depth} ${bottom[0]},${bottom[1] + depth}`,
    top: `${top[0]},${top[1]} ${right[0]},${right[1]} ${bottom[0]},${bottom[1]} ${left[0]},${left[1]}`,
  }
})
const viewBox = computed(() => {
  const n = N.value
  return `${-n * TW - 12} ${-TH - 110} ${2 * n * TW + 24} ${(2 * n - 1) * TH + TH + 110 + 30}`
})
function progress(b: CityBuilding) {
  if (!b.busy_until || !b.started_at) return 0
  const s = new Date(b.started_at).getTime(), e = new Date(b.busy_until).getTime()
  return e <= s ? 1 : (props.now - s) / (e - s)
}
function describe(b: CityBuilding) {
  return `${b.name}, уровень ${b.level}${b.busy_until ? ', идёт строительство' : ''}, клетка ${b.x + 1}:${b.y + 1}`
}
</script>

<template>
  <svg class="scene" :class="{ compact, placing: !!placing }" :viewBox="viewBox" role="group" :aria-label="label" preserveAspectRatio="xMidYMid meet">
    <polygon :points="edge.left" fill="var(--art-soil)" />
    <polygon :points="edge.right" fill="var(--art-soil)" opacity="0.8" />
    <polygon :points="edge.top" fill="var(--art-grass-edge)" />
    <g class="tiles">
      <polygon v-for="t in tiles" :key="`${t.x}-${t.y}`" :points="t.d" class="tile"
        :class="{ placeable: placing && t.free, blocked: placing && !t.free }"
        :fill="t.alt ? 'var(--art-grass-1)' : 'var(--art-grass-2)'"
        :tabindex="placing && t.free ? 0 : undefined" :role="placing && t.free ? 'button' : undefined"
        :aria-label="placing && t.free ? `Построить в клетке ${t.x + 1}:${t.y + 1}` : undefined"
        @click="placing && t.free && emit('cell', t.x, t.y)" @keydown.enter="placing && t.free && emit('cell', t.x, t.y)" />
    </g>
    <g v-for="b in sorted" :key="b.id" class="b" :tabindex="compact ? undefined : 0" :role="compact ? 'img' : 'button'" :aria-label="describe(b)"
      :aria-pressed="compact ? undefined : selectedId === b.id" @click="emit('select', b)" @keydown.enter="emit('select', b)">
      <polygon v-if="selectedId === b.id" :points="`${pos(b.x, b.y).sx},${pos(b.x, b.y).sy - TH} ${pos(b.x, b.y).sx + TW},${pos(b.x, b.y).sy} ${pos(b.x, b.y).sx},${pos(b.x, b.y).sy + TH} ${pos(b.x, b.y).sx - TW},${pos(b.x, b.y).sy}`"
        fill="var(--art-highlight)" stroke="var(--color-focus)" stroke-width="2" />
      <BuildingSprite :type="b.type" :level="b.level || 1" :cx="pos(b.x, b.y).sx" :cy="pos(b.x, b.y).sy" :building="!!b.busy_until" :progress="progress(b)"
        :ruined="ruined.includes(b.type) || (ruined.includes('storage') && b.type === 'barracks')" :selected="selectedId === b.id" :show-level="showLevels" />
    </g>
  </svg>
</template>

<style scoped>
.scene { width: 100%; height: auto; display: block; user-select: none; }
.tile { stroke: rgba(0, 0, 0, 0.08); stroke-width: 0.8; transition: fill var(--duration-fast); }
.tile.placeable { cursor: pointer; fill: var(--art-valid); }
.tile.placeable:hover, .tile.placeable:focus { fill: rgba(40, 150, 90, 0.75); outline: none; }
.tile.blocked { fill-opacity: 0.85; }
.b { cursor: pointer; outline: none; }
.compact .b { cursor: default; }
/* While placing, buildings let clicks through to the cells behind them. */
.placing .b { pointer-events: none; opacity: 0.85; }
.b:focus-visible :deep(polygon) { stroke: var(--color-focus); stroke-width: 1.5; }
</style>
