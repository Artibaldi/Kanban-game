<script setup lang="ts">
import Icon from '../ui/Icon.vue'
import { num } from '../lib/format'

export interface StatItem { icon: string; value: number | string; label: string; badge?: string; warn?: boolean }
defineProps<{ items: StatItem[] }>()
</script>

<template>
  <ul class="stats">
    <li v-for="s in items" :key="s.label" class="stat glass">
      <span class="ico" aria-hidden="true"><Icon :name="s.icon" :size="18" /></span>
      <div class="txt">
        <div class="v"><span class="num">{{ typeof s.value === 'number' ? num(s.value) : s.value }}</span><span v-if="s.badge" class="b" :class="{ warn: s.warn }">{{ s.badge }}</span></div>
        <div class="l">{{ s.label }}</div>
      </div>
    </li>
  </ul>
</template>

<style scoped>
.stats { display: flex; gap: 10px; margin: 18px 0 0; padding: 0; list-style: none; flex-wrap: wrap; }
.stat { display: flex; align-items: center; gap: 12px; padding: 10px 18px 10px 10px; border-radius: var(--radius-lg); }
.ico { display: grid; place-items: center; width: 40px; height: 40px; border-radius: 50%; background: var(--glass-strong); color: var(--color-text); flex: none; }
.v { display: flex; align-items: center; gap: 8px; font-size: 18px; font-weight: 700; line-height: 1; }
.l { font-size: 12px; color: var(--color-text-muted); margin-top: 4px; }
.b { font-size: 11px; font-weight: 500; padding: 3px 8px; border-radius: var(--radius-pill); background: var(--color-success-bg); color: var(--color-success); white-space: nowrap; }
.b.warn { background: var(--color-danger-bg); color: var(--color-danger); }
@media (max-width: 767px) {
  .stats { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; margin-top: 14px; }
  .stat { padding: 10px; gap: 10px; border-radius: 18px; align-items: flex-start; }
  .txt { min-width: 0; }
  .ico { width: 34px; height: 34px; }
  .v { font-size: 17px; flex-wrap: wrap; gap: 4px 6px; }
  .b { white-space: normal; }
  .l { font-size: 11px; }
}
</style>
