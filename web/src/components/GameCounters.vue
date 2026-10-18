<script setup lang="ts">
// Gold and energy in the header, drawn like game counters. The gold number rolls up when it grows.
import { ref, watch } from 'vue'
import { countUp } from '../ui/motion'
import { num } from '../lib/format'
import type { Player } from '../types'

const props = defineProps<{ player: Player }>()
const goldEl = ref<HTMLElement>()
const pill = ref<HTMLElement>()
watch(() => props.player.gold, (to, from) => {
  if (from === undefined || to <= from) return
  countUp(goldEl.value?.querySelector('.v') as HTMLElement, from, to, 600)
  pill.value?.classList.remove('flash'); void pill.value?.offsetWidth; pill.value?.classList.add('flash')
})
</script>

<template>
  <div class="counters" aria-label="Ресурсы игрока">
    <span id="res-gold" ref="pill" class="res gold" :class="{ neg: player.gold < 0 }" :title="`Золото: ${num(player.gold)} из ${num(player.gold_cap)} — строительство и улучшения`">
      <svg class="ico" viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="10" fill="#FFC42E" stroke="#7A4A00" stroke-width="2"/><circle cx="12" cy="12" r="6" fill="none" stroke="#B87A00" stroke-width="1.6"/><path d="M12 8.5v7" stroke="#7A4A00" stroke-width="2" stroke-linecap="round"/></svg>
      <span ref="goldEl"><span class="v num">{{ num(player.gold) }}</span></span><span class="sr-only"> золота</span>
    </span>
    <span class="res energy" :class="{ neg: player.energy < 0 }" :title="`Энергия: ${num(player.energy)} из ${num(player.energy_cap)} — армия и бои`">
      <svg class="ico" viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="10" fill="#8F6BFF" stroke="#3B2690" stroke-width="2"/><path d="M13 5 8 13h4l-1 6 5-8h-4z" fill="#fff" stroke="#3B2690" stroke-width="1.2" stroke-linejoin="round"/></svg>
      <span class="num">{{ num(player.energy) }}</span><small class="num">/{{ num(player.energy_cap) }}</small><span class="sr-only"> энергии</span>
    </span>
    <span v-if="player.debt" class="badge danger" title="Награда за возвращённую задачу отменена после того, как ресурсы были потрачены. Долг гасится следующими наградами.">Долг</span>
  </div>
</template>

<style scoped>
.counters { display: flex; align-items: center; gap: 14px; padding-left: 6px; }
.res { position: relative; display: inline-flex; align-items: center; height: 32px; padding: 0 12px 0 30px; border-radius: var(--radius-pill); font: 900 16px/1 var(--font-game); letter-spacing: 0.01em; white-space: nowrap; }
.ico { position: absolute; left: -8px; top: 50%; width: 34px; height: 34px; transform: translateY(-50%); filter: drop-shadow(0 2px 0 rgba(0, 0, 0, 0.25)); }
.gold { background: linear-gradient(180deg, var(--gold-top), var(--gold-bottom)); color: var(--gold-ink); border: 2px solid var(--gold-edge); box-shadow: inset 0 -3px 0 rgba(122, 74, 0, 0.3), 0 3px 0 var(--gold-edge); }
.energy { background: linear-gradient(180deg, var(--energy-top), var(--energy-bottom)); color: #fff; border: 2px solid var(--energy-edge); box-shadow: inset 0 -3px 0 rgba(30, 15, 90, 0.3), 0 3px 0 var(--energy-edge); text-shadow: 0 1px 0 var(--energy-edge); }
.energy small { font-size: 11px; opacity: 0.85; margin-left: 1px; }
.neg { filter: saturate(0.4); }
.flash { animation: flash 0.4s var(--ease-emphasized); }
@keyframes flash { from { box-shadow: inset 0 -3px 0 rgba(122, 74, 0, 0.3), 0 3px 0 var(--gold-edge), 0 0 0 0 rgba(255, 196, 46, 0.9); } to { box-shadow: inset 0 -3px 0 rgba(122, 74, 0, 0.3), 0 3px 0 var(--gold-edge), 0 0 0 12px rgba(255, 196, 46, 0); } }
@media (max-width: 767px) { .res { font-size: 14px; height: 30px; padding: 0 10px 0 26px; } .ico { width: 30px; height: 30px; } }
</style>
