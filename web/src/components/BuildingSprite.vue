<script setup lang="ts">
// Original isometric building art, drawn from prisms and pyramids. Size and detail grow with the level.
import { computed } from 'vue'
import type { BuildingType } from '../types'

const props = defineProps<{ type: BuildingType; level: number; cx: number; cy: number; building?: boolean; progress?: number; ruined?: boolean; selected?: boolean; showLevel?: boolean }>()

interface Poly { d: string; fill: string; stroke?: string }
const P = (pts: number[][]) => pts.map(p => p.map(v => v.toFixed(1)).join(',')).join(' ')

function prism(cx: number, by: number, w: number, h: number, top: string, left: string, right: string): Poly[] {
  const d = w / 2
  const L = [cx - w, by], R = [cx + w, by], F = [cx, by + d]
  const Lt = [cx - w, by - h], Rt = [cx + w, by - h], Ft = [cx, by + d - h], Bt = [cx, by - d - h]
  return [
    { d: P([Lt, Ft, F, L]), fill: left },
    { d: P([Ft, Rt, R, F]), fill: right },
    { d: P([Bt, Rt, Ft, Lt]), fill: top },
  ]
}
function roof(cx: number, by: number, w: number, h: number, rh: number, light: string, dark: string): Poly[] {
  const d = w / 2
  const Lt = [cx - w, by - h], Rt = [cx + w, by - h], Ft = [cx, by + d - h], Bt = [cx, by - d - h], A = [cx, by - h - rh]
  return [
    { d: P([Lt, Bt, A]), fill: dark },
    { d: P([Bt, Rt, A]), fill: dark },
    { d: P([Lt, Ft, A]), fill: light },
    { d: P([Ft, Rt, A]), fill: dark },
  ]
}
const v = (n: string) => `var(--art-${n})`

const art = computed(() => {
  const L = Math.max(1, props.level || 1)
  const { cx, cy } = props
  const shapes: Poly[] = []
  let topY = cy
  const extra: { kind: string; x: number; y: number; [k: string]: number | string }[] = []
  const stone = props.ruined ? [v('ruin'), v('ruin'), v('ruin')] : [v('stone-light'), v('stone'), v('stone-dark')]
  switch (props.type) {
    case 'townhall': {
      const w = 22 + L * 1.6, h = 16 + L * 4
      shapes.push(...prism(cx, cy, w, h, stone[0], stone[1], stone[2]))
      let base = cy - h, ww = w
      if (L >= 3) {
        const h2 = 8 + L * 2
        ww = w * 0.6
        shapes.push(...prism(cx, base, ww, h2, stone[0], stone[1], stone[2]))
        base -= h2
      }
      if (!props.ruined) shapes.push(...roof(cx, base, ww, 0, 14 + L * 2, v('roof-hall'), v('roof-hall-dark')))
      // Door on the left face and windows.
      shapes.push({ d: P([[cx - w * 0.55, cy + w * 0.22 - 1], [cx - w * 0.3, cy + w * 0.35 - 1], [cx - w * 0.3, cy + w * 0.35 - 12], [cx - w * 0.55, cy + w * 0.22 - 12]]), fill: v('wood-dark') })
      topY = base - (props.ruined ? 0 : 14 + L * 2)
      if (!props.ruined) extra.push({ kind: 'flag', x: cx, y: topY })
      if (L >= 4 && !props.ruined) extra.push({ kind: 'trim', x: cx, y: cy - h, w })
      break
    }
    case 'barracks': {
      const w = 20 + L * 1.2, h = 10 + L * 3
      const wood = props.ruined ? stone : [v('wood-light'), v('wood'), v('wood-dark')]
      shapes.push(...prism(cx, cy, w, h, wood[0], wood[1], wood[2]))
      if (!props.ruined) shapes.push(...roof(cx, cy, w, h, 11 + L, v('roof-barracks'), v('roof-barracks-dark')))
      topY = cy - h - 11 - L
      extra.push({ kind: 'spears', x: cx + w * 0.5, y: cy - h * 0.2 })
      break
    }
    case 'hut': shapes.push(...prism(cx, cy, 16, 12, v('wood-light'), v('wood'), v('wood-dark')), ...roof(cx, cy, 16, 12, 10, v('gold'), v('gold-dark'))); topY = cy - 34; break
    case 'market': shapes.push(...prism(cx, cy, 20, 9, v('wood-light'), v('wood'), v('wood-dark')), ...roof(cx, cy, 20, 9, 7, '#e05a4a', '#b23a2e')); topY = cy - 34; break
    case 'well': shapes.push(...prism(cx, cy, 18, 7, v('stone-light'), v('stone'), v('stone-dark')), ...prism(cx, cy - 7, 6, 16, '#c4b2ff', '#9c84ff', '#7a5cff')); topY = cy - 26; break
    case 'archery': shapes.push(...prism(cx, cy, 22, 3, v('stone-light'), v('stone'), v('stone-dark')), ...prism(cx - 6, cy - 3, 4, 12, '#f2e3b0', '#e2b65a', '#c9973d'), ...prism(cx + 7, cy - 3, 4, 12, '#f2e3b0', '#e2b65a', '#c9973d')); topY = cy - 22; break
    case 'workshop': shapes.push(...prism(cx, cy, 20, 13, v('stone-light'), v('stone'), v('stone-dark')), ...roof(cx, cy, 20, 13, 9, '#9aa6ba', '#7d8ba3')); topY = cy - 34; break
    case 'mortar': shapes.push(...prism(cx, cy, 18, 8, v('stone-light'), v('stone'), v('stone-dark')), ...prism(cx, cy - 8, 7, 12, '#5a5e6a', '#4a4e5a', '#3c3f48')); topY = cy - 26; break
    case 'trap': shapes.push(...prism(cx, cy, 16, 2, v('wood-light'), v('wood'), v('wood-dark'))); topY = cy - 4; break
    case 'stable': shapes.push(...prism(cx, cy, 24, 10, v('wood-light'), v('wood'), v('wood-dark')), ...roof(cx, cy, 24, 10, 8, v('gold'), v('gold-dark'))); topY = cy - 30; break
    case 'crossbow': shapes.push(...prism(cx, cy, 13, 30, v('stone-light'), v('stone'), v('stone-dark')), ...prism(cx, cy - 30, 16, 3, '#9aa6ba', '#7d8ba3', '#5d6b83')); topY = cy - 42; break
    // Decor in the flat view: small and simple.
    case 'road': shapes.push(...prism(cx, cy, 26, 2, v('stone-light'), v('stone'), v('stone-dark'))); topY = cy - 4; break
    case 'pond': shapes.push(...prism(cx, cy, 24, 3, '#4db4f0', v('stone'), v('stone-dark'))); topY = cy - 5; break
    case 'flowers': shapes.push(...prism(cx, cy, 18, 5, '#ff8ac0', v('soil'), v('wood-dark'))); topY = cy - 7; break
    case 'tree': shapes.push(...prism(cx, cy, 4, 12, v('wood-light'), v('wood'), v('wood-dark')), ...roof(cx, cy, 14, 12, 26, v('grass-1'), v('grass-edge'))); topY = cy - 38; break
    case 'lantern': shapes.push(...prism(cx, cy, 3, 26, '#6b6b78', '#4a4a58', '#3a3a46'), ...prism(cx, cy - 26, 6, 7, '#ffe08a', '#ffd34a', '#e6b02a')); topY = cy - 34; break
    case 'tavern': {
      const w = 19 + L * 1.4, h = 11 + L * 2.5
      const wood = props.ruined ? stone : [v('wood-light'), v('wood'), v('wood-dark')]
      shapes.push(...prism(cx, cy, w, h, wood[0], wood[1], wood[2]))
      if (!props.ruined) shapes.push(...roof(cx, cy, w, h, 10 + L, v('gold'), v('gold-dark')))
      topY = cy - h - 10 - L
      extra.push({ kind: 'barrel', x: cx - w - 3, y: cy + 4 })
      if (L >= 2) extra.push({ kind: 'flag', x: cx, y: topY })
      break
    }
    case 'storage': {
      const w = 18 + L * 1.2, h = 9 + L * 2.5
      shapes.push(...prism(cx, cy, w, h, stone[0], stone[1], stone[2]))
      if (!props.ruined) shapes.push(...roof(cx, cy, w, h, 7 + L, v('roof-storage'), v('roof-storage-dark')))
      topY = cy - h - 7 - L
      extra.push({ kind: 'coin', x: cx - w * 0.5, y: cy - h * 0.35 + w * 0.12 })
      if (L >= 2) extra.push({ kind: 'barrel', x: cx + w + 3, y: cy + 4 })
      break
    }
    case 'wall': {
      const w = 15, h = 7 + L * 2.5
      const mat = props.ruined ? stone : L === 1 ? [v('wood-light'), v('wood'), v('wood-dark')] : L >= 4 ? [v('stone'), v('stone-dark'), v('ruin')] : stone
      shapes.push(...prism(cx, cy, w, h, mat[0], mat[1], mat[2]))
      if (!props.ruined && L >= 2) {
        shapes.push(...prism(cx - 8, cy - h, 4, 4, mat[0], mat[1], mat[2]))
        shapes.push(...prism(cx + 8, cy - h, 4, 4, mat[0], mat[1], mat[2]))
      }
      topY = cy - h - 4
      break
    }
    case 'tower': {
      const w = 10 + L * 0.6, h = props.ruined ? 12 : 24 + L * 6
      shapes.push(...prism(cx, cy, w, h, stone[0], stone[1], stone[2]))
      if (!props.ruined) {
        shapes.push(...prism(cx, cy - h, w + 4, 5, stone[0], stone[1], stone[2]))
        for (const dx of [-w, 0, w]) shapes.push(...prism(cx + dx * 0.8, cy - h - 5 + (dx === 0 ? -(w + 4) / 2 + 2 : 0), 2.5, 4, stone[0], stone[1], stone[2]))
        if (L >= 3) shapes.push(...roof(cx, cy - h - 5, w + 2, 0, 12 + L * 2, v('roof-barracks'), v('roof-barracks-dark')))
        extra.push({ kind: 'slit', x: cx - w * 0.5, y: cy - h * 0.5 })
      }
      topY = cy - h - (L >= 3 ? 17 + L * 2 : 9)
      break
    }
  }
  return { shapes, topY, extra }
})
</script>

<template>
  <g class="bsprite" :class="{ building, ruined, selected }" :transform="`translate(${cx} ${cy}) scale(1.3) translate(${-cx} ${-cy})`">
    <ellipse :cx="cx" :cy="cy + 3" rx="30" ry="12" fill="var(--art-shadow)" />
    <g :opacity="building ? 0.6 : 1">
      <polygon v-for="(s, i) in art.shapes" :key="i" :points="s.d" :fill="s.fill" stroke="rgba(0,0,0,0.18)" stroke-width="0.6" stroke-linejoin="round" />
      <template v-for="(e, i) in art.extra" :key="'e' + i">
        <g v-if="e.kind === 'flag'">
          <line :x1="e.x" :y1="e.y" :x2="e.x" :y2="e.y - 16" stroke="var(--art-wood-dark)" stroke-width="1.6" />
          <path :d="`M${e.x} ${e.y - 16}h11l-3 3.5 3 3.5h-11z`" fill="var(--art-gold)" stroke="var(--art-gold-dark)" stroke-width="0.6" />
        </g>
        <path v-else-if="e.kind === 'trim'" :d="`M${e.x - Number(e.w)} ${e.y}L${e.x} ${e.y + Number(e.w) / 2}L${e.x + Number(e.w)} ${e.y}`" fill="none" stroke="var(--art-gold)" stroke-width="2" />
        <g v-else-if="e.kind === 'spears'" stroke="var(--art-wood-dark)" stroke-width="1.4">
          <line :x1="e.x - 5" :y1="e.y + 5" :x2="e.x + 5" :y2="e.y - 7" /><line :x1="e.x + 5" :y1="e.y + 5" :x2="e.x - 5" :y2="e.y - 7" />
          <rect :x="e.x - 3" :y="e.y - 4" width="6" height="7" fill="var(--art-banner)" stroke="none" />
        </g>
        <g v-else-if="e.kind === 'coin'"><ellipse :cx="e.x" :cy="e.y" rx="4.5" ry="5" fill="var(--art-gold)" stroke="var(--art-gold-dark)" stroke-width="1" /></g>
        <g v-else-if="e.kind === 'barrel'"><rect :x="e.x - 4" :y="e.y - 9" width="8" height="10" rx="2" fill="var(--art-wood)" stroke="var(--art-wood-dark)" stroke-width="0.8" /><ellipse :cx="e.x" :cy="e.y - 9" rx="4" ry="1.6" fill="var(--art-wood-light)" /></g>
        <rect v-else-if="e.kind === 'slit'" :x="e.x - 1.2" :y="e.y - 5" width="2.4" height="8" fill="var(--art-soil)" />
      </template>
    </g>
    <g v-if="ruined" stroke="var(--art-soil)" stroke-width="1.2" fill="none" opacity="0.8">
      <path :d="`M${cx - 8} ${cy - 6}l4 -5 3 4 5 -6`" />
      <circle :cx="cx + 6" :cy="art.topY - 4" r="5" fill="rgba(80,80,80,0.35)" stroke="none" />
      <circle :cx="cx + 12" :cy="art.topY - 12" r="4" fill="rgba(80,80,80,0.25)" stroke="none" />
    </g>
    <g v-if="building" class="scaffold" stroke="var(--art-wood-dark)" stroke-width="1.2">
      <line :x1="cx - 22" :y1="cy" :x2="cx - 22" :y2="art.topY + 6" /><line :x1="cx + 22" :y1="cy" :x2="cx + 22" :y2="art.topY + 6" />
      <line :x1="cx - 24" :y1="(cy + art.topY) / 2" :x2="cx + 24" :y2="(cy + art.topY) / 2" /><line :x1="cx - 22" :y1="cy" :x2="cx + 22" :y2="art.topY + 6" />
      <g transform="translate(0,-10)">
        <rect :x="cx - 20" :y="art.topY - 4" width="40" height="6" rx="3" fill="rgba(0,0,0,0.45)" stroke="none" />
        <rect :x="cx - 19" :y="art.topY - 3" :width="Math.max(0, Math.min(1, progress || 0)) * 38" height="4" rx="2" fill="var(--art-gold)" stroke="none" />
      </g>
    </g>
    <g v-if="showLevel && !building && type !== 'wall'" :transform="`translate(${cx + 16}, ${art.topY + 4})`">
      <circle r="7.5" fill="var(--color-surface)" stroke="var(--art-gold-dark)" stroke-width="1.2" />
      <text text-anchor="middle" y="3.5" font-size="9.5" font-weight="800" fill="var(--color-text)">{{ level }}</text>
    </g>
  </g>
</template>

<style scoped>
.bsprite.selected polygon { stroke: var(--color-focus); stroke-width: 1.4; }
.scaffold { animation: bob 1.6s ease-in-out infinite; }
@keyframes bob { 50% { opacity: 0.7; } }
</style>
