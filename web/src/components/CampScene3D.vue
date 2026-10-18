<script setup lang="ts">
// The army camp in 3D, loaded on demand; emits `unavailable` when WebGL is missing or lost.
import { onMounted, onUnmounted, ref, shallowRef, watch } from 'vue'
import type { CampStage } from '../game3d/campStage'

const props = defineProps<{ troops: Record<string, number> }>()
const emit = defineEmits<{ unavailable: [] }>()
const canvas = ref<HTMLCanvasElement>()
const labels = ref<HTMLElement>()
const stage = shallowRef<CampStage | null>(null)
let unmounted = false

onMounted(async () => {
  try {
    const mod = await import('../game3d/campStage')
    const ok = (await import('../game3d/fortressStage')).webglAvailable()
    if (unmounted) return
    if (!ok || !canvas.value || !labels.value) throw new Error('no webgl')
    const s = new mod.CampStage(canvas.value, labels.value, matchMedia('(prefers-reduced-motion: reduce)').matches)
    s.onLost = () => emit('unavailable')
    s.setArmy(props.troops, false)
    stage.value = s
  } catch { if (!unmounted) emit('unavailable') }
})
onUnmounted(() => { unmounted = true; stage.value?.dispose() })
watch(() => props.troops, t => stage.value?.setArmy(t, true))
</script>

<template>
  <div class="scene3d">
    <canvas ref="canvas" aria-hidden="true" />
    <div ref="labels" class="g3-labels" aria-hidden="true" />
  </div>
</template>

<style scoped>
.scene3d { position: absolute; inset: 0; background: linear-gradient(#77c6f2, #bde6fa 60%, #d9eef5); }
canvas { display: block; width: 100%; height: 100%; }
</style>
