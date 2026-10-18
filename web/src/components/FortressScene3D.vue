<script setup lang="ts">
// The 3D fortress. The engine is loaded on demand, so the work screens never download three.js.
// It remembers the horde between visits: enemies defeated while you were away fall when you come back.
import { onMounted, onUnmounted, ref, shallowRef, watch } from 'vue'
import type { Enemy } from '../types'
import type { FortressStage, WaveEvents, WaveInput } from '../game3d/fortressStage'

const props = defineProps<{ hp: number; fog: number; enemies: Enemy[]; selectedId?: string | null; memoryKey: string }>()
const emit = defineEmits<{ select: [id: string | null]; gone: [enemy: Enemy, x: number, y: number]; ready: []; unavailable: [] }>()
const canvas = ref<HTMLCanvasElement>()
const labels = ref<HTMLElement>()
const stage = shallowRef<FortressStage | null>(null)
const failed = ref(false)

type Memo = Pick<Enemy, 'task_id' | 'kind' | 'points' | 'row' | 'overdue' | 'at_gates' | 'state' | 'number' | 'title' | 'priority' | 'due_date' | 'mine'>
function remember(list: Enemy[]) {
  try {
    const memo: Memo[] = list.map(({ task_id, kind, points, row, overdue, at_gates, state, number, title, priority, due_date, mine }) => ({ task_id, kind, points, row, overdue, at_gates, state, number, title, priority, due_date, mine }))
    localStorage.setItem(props.memoryKey, JSON.stringify(memo))
  } catch { /* storage may be unavailable: the scene works without memory */ }
}
function recall(): Enemy[] | null {
  try { const v = localStorage.getItem(props.memoryKey); return v ? (JSON.parse(v) as Enemy[]) : null } catch { return null }
}

const reduced = () => matchMedia('(prefers-reduced-motion: reduce)').matches
const onGone = (e: Enemy, [x, y]: [number, number]) => {
  const r = canvas.value?.getBoundingClientRect()
  if (r) emit('gone', e, r.left + x, r.top + y)
}

onMounted(async () => {
  try {
    const mod = await import('../game3d/fortressStage')
    if (!mod.webglAvailable() || !canvas.value || !labels.value) throw new Error('no webgl')
    const s = new mod.FortressStage(canvas.value, labels.value, reduced())
    s.onPick = id => emit('select', id)
    s.setFortress(props.hp, props.fog)
    // Last visit's horde first, then today's: the difference walks in or falls.
    const before = recall()
    if (before) {
      s.sync(before, null)
      const prevIds = new Set(before.map(e => e.task_id))
      setTimeout(() => { if (stage.value === s) s.sync(props.enemies, prevIds, onGone) }, 700)
    } else s.sync(props.enemies, null)
    remember(props.enemies)
    stage.value = s
    emit('ready')
  } catch {
    failed.value = true
    emit('unavailable')
  }
})
onUnmounted(() => { stage.value?.dispose(); stage.value = null })

watch(() => props.enemies, (list, old) => {
  const s = stage.value
  if (!s) return
  s.sync(list, new Set((old || []).map(e => e.task_id)), onGone)
  remember(list)
})
watch(() => [props.hp, props.fog] as const, ([hp, fog]) => stage.value?.setFortress(hp, fog))
watch(() => props.selectedId, id => stage.value?.select(id ?? null))

defineExpose({
  playWave: (w: WaveInput, ev: WaveEvents) => stage.value?.playWave(w, ev),
  setSpeed: (x: number) => stage.value?.setSpeed(x),
  finishWave: () => stage.value?.finishWave(),
  endWave: () => stage.value?.endWave(),
  screenOf: (id: string) => {
    const p = stage.value?.screenOf(id), r = canvas.value?.getBoundingClientRect()
    return p && r ? [r.left + p[0], r.top + p[1]] as [number, number] : null
  },
})
</script>

<template>
  <div class="scene3d" :class="{ failed }">
    <canvas ref="canvas" aria-hidden="true" />
    <div ref="labels" class="g3-labels" aria-hidden="true" />
  </div>
</template>

<style scoped>
.scene3d { position: absolute; inset: 0; background: linear-gradient(#77c6f2, #bde6fa 60%, #d9eef5); }
canvas { display: block; width: 100%; height: 100%; touch-action: none; cursor: pointer; }
.failed canvas { display: none; }
</style>
