<script setup lang="ts">
// A rendered 3D portrait of a troop or a building, with the flat drawing as the fallback.
import { onMounted, ref } from 'vue'
import TroopIcon from './TroopIcon.vue'
import BuildingIcon from './BuildingIcon.vue'
import type { BuildingType } from '../types'

const props = withDefaults(defineProps<{ troop?: string; building?: BuildingType; level?: number; size?: number; faded?: boolean }>(), { size: 64, level: 1 })
const src = ref<string | null>(null)
const failed = ref(false)

onMounted(async () => {
  try {
    const m = await import('../game3d/portraits')
    src.value = props.troop ? m.troopPortrait(props.troop) : props.building ? m.buildingPortrait(props.building, props.level) : null
  } catch { src.value = null }
  failed.value = !src.value
})
</script>

<template>
  <img v-if="src" :src="src" :width="size" :height="size" alt="" class="portrait" :class="{ faded }" />
  <TroopIcon v-else-if="troop && failed" :type="troop" :size="size" :faded="faded" />
  <BuildingIcon v-else-if="building && failed" :type="building" />
  <span v-else class="ph" :style="{ width: size + 'px', height: size + 'px' }" />
</template>

<style scoped>
.portrait { flex: none; object-fit: contain; filter: drop-shadow(0 3px 0 rgba(0, 0, 0, 0.18)); }
.faded { filter: grayscale(1) opacity(0.45); }
.ph { display: inline-block; flex: none; }
</style>
