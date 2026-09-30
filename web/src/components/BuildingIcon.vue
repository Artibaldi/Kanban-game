<script setup lang="ts">
import BuildingSprite from './BuildingSprite.vue'
import type { BuildingType } from '../types'
withDefaults(defineProps<{ type: BuildingType; level?: number; size?: number }>(), { level: 1, size: 52 })
</script>

<template>
  <svg viewBox="-46 -86 92 102" :width="size" :height="size * 1.1" aria-hidden="true" class="bicon">
    <BuildingSprite :type="type" :level="level" :cx="0" :cy="0" />
  </svg>
</template>

<style scoped>.bicon { flex: none; }</style>
