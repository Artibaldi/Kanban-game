<script setup lang="ts">
import { computed } from 'vue'
import Icon from '../ui/Icon.vue'
import { num } from '../lib/format'
import type { Player } from '../types'

const props = defineProps<{ player: Player; compact?: boolean }>()
const xpPct = computed(() => Math.min(100, Math.round((props.player.level_xp / Math.max(1, props.player.level_need)) * 100)))
</script>

<template>
  <div class="resources" :class="{ compact }" aria-label="Ресурсы игрока">
    <span class="res gold" :class="{ neg: player.gold < 0 }" :title="`Золото: ${num(player.gold)} из ${num(player.gold_cap)} — строительство и улучшения`">
      <Icon name="coins" :size="16" /><span class="num">{{ num(player.gold) }}</span><span v-if="!compact" class="cap num">/ {{ num(player.gold_cap) }}</span>
      <span class="sr-only">золота</span>
    </span>
    <span class="res energy" :class="{ neg: player.energy < 0 }" :title="`Энергия: ${num(player.energy)} из ${num(player.energy_cap)} — армия и бои`">
      <Icon name="zap" :size="16" /><span class="num">{{ num(player.energy) }}</span><span v-if="!compact" class="cap num">/ {{ num(player.energy_cap) }}</span>
      <span class="sr-only">энергии</span>
    </span>
    <span class="res xp" :title="`Уровень ${player.level}: ${num(player.level_xp)} из ${num(player.level_need)} опыта до следующего`">
      <span class="lvl num">{{ player.level }}</span>
      <span class="xpbar" aria-hidden="true"><span :style="{ width: xpPct + '%' }" /></span>
      <span class="sr-only">уровень {{ player.level }}, опыт {{ player.level_xp }} из {{ player.level_need }}</span>
    </span>
    <span v-if="player.debt" class="badge danger" title="Награда за возвращённую задачу отменена после того, как ресурсы были потрачены. Долг гасится следующими наградами.">Долг</span>
  </div>
</template>

<style scoped>
.resources { display: flex; align-items: center; gap: var(--space-2); flex-wrap: nowrap; }
.res { display: inline-flex; align-items: center; gap: 5px; padding: 3px 10px; border-radius: var(--radius-pill); font-weight: 700; font-size: var(--text-sm); }
.res .cap { font-weight: 500; opacity: 0.7; font-size: var(--text-xs); }
.gold { background: var(--color-gold-bg); color: var(--color-gold); }
.energy { background: var(--color-energy-bg); color: var(--color-energy); }
.xp { background: var(--color-xp-bg); color: var(--color-xp); }
.neg { background: var(--color-danger-bg); color: var(--color-danger); }
.lvl { display: inline-grid; place-items: center; min-width: 20px; height: 20px; border-radius: 50%; background: var(--color-xp); color: var(--color-surface); font-size: var(--text-xs); }
.xpbar { width: 44px; height: 6px; border-radius: 99px; background: color-mix(in srgb, var(--color-xp) 22%, transparent); overflow: hidden; }
.xpbar span { display: block; height: 100%; background: var(--color-xp); }
.compact .xpbar { width: 28px; }
.compact .res { padding: 2px 8px; }
</style>
