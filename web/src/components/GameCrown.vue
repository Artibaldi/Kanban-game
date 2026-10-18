<script setup lang="ts">
// A crown slot: gold when won, slate when not. `drop` plays the stamp-in when it is won.
defineProps<{ on: boolean; mid?: boolean; drop?: boolean }>()
</script>

<template>
  <span class="crown" :class="{ mid, on, drop: drop && on }" aria-hidden="true">
    <svg viewBox="0 0 60 52">
      <template v-if="on">
        <path d="M6 44 3 14l14 12L30 4l13 22 14-12-3 30z" fill="#FFD323" stroke="#1E1E1E" stroke-width="3.5" stroke-linejoin="round" />
        <path d="M6 44h48l-1 6H7z" fill="#D18B00" stroke="#1E1E1E" stroke-width="3" stroke-linejoin="round" />
        <path d="M10 38h40" stroke="#FFF2A0" stroke-width="3" stroke-linecap="round" opacity=".8" />
        <circle cx="30" cy="30" r="5" fill="#F04848" stroke="#1E1E1E" stroke-width="2.5" />
        <circle cx="3" cy="13" r="3.5" fill="#FFD323" stroke="#1E1E1E" stroke-width="2.5" />
        <circle cx="57" cy="13" r="3.5" fill="#FFD323" stroke="#1E1E1E" stroke-width="2.5" />
        <circle cx="30" cy="4" r="4" fill="#FFD323" stroke="#1E1E1E" stroke-width="2.5" />
      </template>
      <template v-else>
        <path d="M6 44 3 14l14 12L30 4l13 22 14-12-3 30z" fill="#56608A" stroke="#1E1E1E" stroke-width="3.5" stroke-linejoin="round" />
        <path d="M6 44h48l-1 6H7z" fill="#434B70" stroke="#1E1E1E" stroke-width="3" stroke-linejoin="round" />
      </template>
    </svg>
  </span>
</template>

<style scoped>
.crown { position: relative; display: inline-block; width: var(--crown, 40px); aspect-ratio: 60 / 52; }
.crown.mid { width: calc(var(--crown, 40px) * 1.18); }
svg { display: block; width: 100%; height: 100%; overflow: visible; }
.drop { animation: stamp 0.5s cubic-bezier(0.34, 1.56, 0.64, 1); }
.drop::after { content: ""; position: absolute; inset: -14px; border-radius: 50%; background: radial-gradient(circle, rgba(255, 240, 150, 0.95), rgba(255, 211, 35, 0) 65%); animation: spark 0.6s 0.2s ease-out both; pointer-events: none; }
@keyframes stamp { 0% { transform: translateY(-70px) scale(1.4) rotate(-14deg); opacity: 0; } 55% { transform: translateY(0) scale(1.4); opacity: 1; } 100% { transform: scale(1); } }
@keyframes spark { 0% { transform: scale(0.2); opacity: 1; } 100% { transform: scale(1.8); opacity: 0; } }
</style>
