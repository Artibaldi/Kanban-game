<script setup lang="ts">
// The defender's city in 3D with the duel replay on top. Read-only: no picking, the camera still pans.
import { onMounted, onUnmounted, ref, shallowRef } from 'vue'
import type { CityBuilding } from '../types'
import type { Part } from '../lib/battleView'
import type { CityStage } from '../game3d/cityStage'
import type { BattleLayer } from '../game3d/battleLayer'

const props = defineProps<{ buildings: CityBuilding[]; grid: number }>()
const emit = defineEmits<{ ready: []; unavailable: [] }>()
const canvas = ref<HTMLCanvasElement>()
const labels = ref<HTMLElement>()
const stage = shallowRef<CityStage | null>(null)
const layer = shallowRef<BattleLayer | null>(null)
let unmounted = false

onMounted(async () => {
  try {
    const [cs, bl, fs] = await Promise.all([import('../game3d/cityStage'), import('../game3d/battleLayer'), import('../game3d/fortressStage')])
    if (unmounted) return
    if (!fs.webglAvailable() || !canvas.value || !labels.value) throw new Error('no webgl')
    const s = new cs.CityStage(canvas.value, labels.value, matchMedia('(prefers-reduced-motion: reduce)').matches, props.grid, false)
    s.onLost = () => emit('unavailable')
    s.sync(props.buildings.map(b => ({ id: b.id, type: b.type, level: b.level || 1, x: b.x, y: b.y, busy: false, name: b.name })), false)
    stage.value = s
    layer.value = new bl.BattleLayer(s)
    emit('ready')
  } catch { if (!unmounted) emit('unavailable') }
})
onUnmounted(() => { unmounted = true; stage.value?.dispose() })

defineExpose({
  start: (squad: Record<string, number>, shown: Record<string, number>) => layer.value?.start(squad, shown),
  round: (target: Part | null, ruined: Set<Part>, alive: Record<string, number>, dur: number) => layer.value?.round(target, ruined, alive, dur),
  cheer: () => layer.value?.cheer(),
  clear: () => layer.value?.clear(),
})
</script>

<template>
  <div class="scene3d">
    <canvas ref="canvas" aria-hidden="true" />
    <div ref="labels" class="g3-labels" aria-hidden="true" />
  </div>
</template>

<style scoped>
.scene3d { position: absolute; inset: 0; background: linear-gradient(#69bdf0, #b9e4fa 70%, #e3f4fb); }
canvas { display: block; width: 100%; height: 100%; touch-action: none; }
</style>
