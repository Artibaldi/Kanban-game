<script setup lang="ts">
// The 3D city. Loaded on demand; when WebGL is unavailable the page falls back to the flat city.
// While placing, the arrow keys move a cursor over the grid and Enter builds there.
import { nextTick, onMounted, onUnmounted, ref, shallowRef, watch } from 'vue'
import { duration } from '../lib/format'
import type { BuildingType, CityBuilding } from '../types'
import type { CityStage } from '../game3d/cityStage'

const props = defineProps<{ buildings: CityBuilding[]; grid: number; selectedId: string | null; placing: BuildingType | null; now: number; readonly?: boolean }>()
const emit = defineEmits<{ select: [b: CityBuilding]; deselect: []; cell: [x: number, y: number]; unavailable: [] }>()
const canvas = ref<HTMLCanvasElement>()
const labels = ref<HTMLElement>()
const stage = shallowRef<CityStage | null>(null)
const cursor = ref({ x: Math.floor(props.grid / 2), y: Math.floor(props.grid / 2) })
const announce = ref('')
let unmounted = false

const items = () => props.buildings.map(b => ({ id: b.id, type: b.type, level: b.level || 1, x: b.x, y: b.y, busy: !!b.busy_until, name: b.name }))
function timers() {
  const t: Record<string, string> = {}
  for (const b of props.buildings) if (b.busy_until) t[b.id] = duration(new Date(b.busy_until).getTime() - props.now)
  return t
}

onMounted(async () => {
  try {
    const mod = await import('../game3d/cityStage')
    const ok = (await import('../game3d/fortressStage')).webglAvailable()
    if (unmounted) return
    if (!ok || !canvas.value || !labels.value) throw new Error('no webgl')
    const s = new mod.CityStage(canvas.value, labels.value, matchMedia('(prefers-reduced-motion: reduce)').matches, props.grid, !props.readonly)
    s.onPick = id => { const b = props.buildings.find(x => x.id === id); if (b) emit('select', b); else emit('deselect') }
    s.onCell = (x, y) => emit('cell', x, y)
    s.onLost = () => emit('unavailable')
    // Mouse and keyboard share one cursor.
    s.onHoverCell = c => { cursor.value = c }
    s.sync(items(), false)
    s.setTimers(timers())
    s.select(props.selectedId)
    s.setPlacing(!!props.placing)
    stage.value = s
  } catch { if (!unmounted) emit('unavailable') }
})
onUnmounted(() => { unmounted = true; stage.value?.dispose() })

watch(() => props.buildings, () => { stage.value?.sync(items(), true); stage.value?.setTimers(timers()) })
watch(() => props.now, () => stage.value?.setTimers(timers()))
watch(() => props.selectedId, id => { stage.value?.select(id); stage.value?.setTimers(timers()) })
watch(() => props.placing, p => {
  stage.value?.setPlacing(!!p)
  // Placing starts on the field: the arrow keys work at once.
  if (p) nextTick(() => canvas.value?.focus())
  if (p) { stage.value?.setCursor(cursor.value); announce.value = `Выберите клетку стрелками, Enter — построить. Клетка ${cursor.value.x + 1}:${cursor.value.y + 1}.` }
})

function onKey(e: KeyboardEvent) {
  if (!props.placing) return
  const d: Record<string, [number, number]> = { ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, -1], ArrowDown: [0, 1] }
  if (d[e.key]) {
    e.preventDefault()
    const c = cursor.value
    cursor.value = { x: Math.min(props.grid - 1, Math.max(0, c.x + d[e.key][0])), y: Math.min(props.grid - 1, Math.max(0, c.y + d[e.key][1])) }
    stage.value?.setCursor(cursor.value)
    const taken = props.buildings.find(b => b.x === cursor.value.x && b.y === cursor.value.y)
    announce.value = `Клетка ${cursor.value.x + 1}:${cursor.value.y + 1}${taken ? `, занята: ${taken.name}` : ', свободна'}`
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const taken = props.buildings.find(b => b.x === cursor.value.x && b.y === cursor.value.y)
    if (taken) announce.value = `Клетка ${cursor.value.x + 1}:${cursor.value.y + 1} занята: ${taken.name}. Выберите свободную.`
    else emit('cell', cursor.value.x, cursor.value.y)
  }
}
</script>

<template>
  <div class="scene3d">
    <canvas ref="canvas" :tabindex="placing ? 0 : -1" :aria-label="placing ? 'Поле города: выбор клетки' : undefined" :aria-hidden="placing ? undefined : 'true'" @keydown="onKey" />
    <div ref="labels" class="g3-labels" aria-hidden="true" />
    <p class="sr-only" aria-live="polite">{{ announce }}</p>
  </div>
</template>

<style scoped>
.scene3d { position: absolute; inset: 0; background: linear-gradient(#69bdf0, #b9e4fa 70%, #e3f4fb); }
canvas { display: block; width: 100%; height: 100%; touch-action: none; cursor: pointer; }
canvas:focus-visible { outline: 3px solid #fff; outline-offset: -6px; }
</style>
