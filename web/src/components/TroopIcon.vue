<script setup lang="ts">
// Original troop portraits: Страж (balanced), Налётчик (high attack), Таран (siege).
withDefaults(defineProps<{ type: string; size?: number; faded?: boolean }>(), { size: 48 })
</script>

<template>
  <svg :width="size" :height="size" viewBox="0 0 48 48" aria-hidden="true" :opacity="faded ? 0.35 : 1" class="troop">
    <circle cx="24" cy="24" r="23" :fill="type === 'guard' ? '#dcebd9' : type === 'raider' ? '#f4dcd6' : '#efe2cf'" />
    <g v-if="type === 'guard'">
      <path d="M14 44c0-9 4-14 10-14s10 5 10 14z" fill="#3f7a55" />
      <circle cx="24" cy="22" r="7" fill="#e9c7a4" />
      <path d="M16 21a8 8 0 0 1 16 0v-3a8 8 0 0 0-16 0z" fill="#8c96a0" /><rect x="23" y="11" width="2" height="5" fill="#8c96a0" />
      <path d="M30 28h10v8c0 5-5 8-5 8s-5-3-5-8z" fill="#3f6fb0" stroke="#2f548a" stroke-width="1.2" /><path d="M35 30v11" stroke="#e3b341" stroke-width="1.6" />
    </g>
    <g v-else-if="type === 'raider'">
      <path d="M13 44c0-9 5-14 11-14s11 5 11 14z" fill="#b34a37" />
      <circle cx="24" cy="22" r="7" fill="#d9ad86" />
      <path d="M16 21c1-7 15-7 16 0l-2-6h-12z" fill="#4a3526" /><rect x="17" y="20" width="14" height="3" fill="#2d2d2d" opacity="0.8" />
      <path d="M8 38 18 26" stroke="#c9ced3" stroke-width="2.4" stroke-linecap="round" /><path d="M40 38 30 26" stroke="#c9ced3" stroke-width="2.4" stroke-linecap="round" />
      <path d="M9 34l4 3M39 34l-4 3" stroke="#7c5528" stroke-width="2" />
    </g>
    <g v-else>
      <rect x="8" y="20" width="28" height="9" rx="4" fill="#a8783f" stroke="#7c5528" stroke-width="1.2" />
      <path d="M36 18l7 3.5v6L36 31z" fill="#6f7780" stroke="#4e555c" stroke-width="1" />
      <path d="M12 20v-6h20v6" fill="none" stroke="#7c5528" stroke-width="2" /><path d="M10 14h24" stroke="#b34a37" stroke-width="3" stroke-linecap="round" />
      <circle cx="14" cy="34" r="5" fill="#5a3f22" stroke="#3d2a16" stroke-width="1.2" /><circle cx="30" cy="34" r="5" fill="#5a3f22" stroke="#3d2a16" stroke-width="1.2" />
      <circle cx="14" cy="34" r="1.5" fill="#c89a5e" /><circle cx="30" cy="34" r="1.5" fill="#c89a5e" />
    </g>
  </svg>
</template>

<style scoped>.troop { flex: none; }</style>
