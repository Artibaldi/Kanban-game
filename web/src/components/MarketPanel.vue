<script setup lang="ts">
// Market: trade gold for energy and back at the work-reward rate, minus the market's fee. Nothing is earned here.
import { computed, ref } from 'vue'
import { api } from '../api'
import { marketQuote } from '../lib/market'
import { num } from '../lib/format'
import { useGame } from '../stores/game'
import { useToasts } from '../stores/toast'
import type { Player } from '../types'
import Icon from '../ui/Icon.vue'

const props = defineProps<{ feePct: number; player: Player }>()
const game = useGame()
const toasts = useToasts()
const give = ref<'gold' | 'energy'>('gold')
const amount = ref<number>(100)
const busy = ref(false)

const settings = computed(() => game.config?.settings || { gold_per_point: 40, energy_per_point: 10 })
const got = computed(() => marketQuote(give.value, amount.value, settings.value, props.feePct))
const have = computed(() => (give.value === 'gold' ? props.player.gold : props.player.energy))
// Room left in the store of what you get: a trade that would overflow it is refused.
const room = computed(() => (give.value === 'gold' ? props.player.energy_cap - props.player.energy : props.player.gold_cap - props.player.gold))
const block = computed(() => {
  if (props.player.debt) return 'Баланс отрицательный после отмены награды: обмен откроется, когда долг погасят следующие награды.'
  if (!(amount.value > 0) || !Number.isInteger(amount.value)) return 'Укажите целое число больше нуля.'
  if (amount.value > have.value) return give.value === 'gold' ? 'Недостаточно золота.' : 'Недостаточно энергии.'
  if (got.value < 1) return 'Слишком мало: получится меньше одной единицы.'
  if (got.value > room.value) return 'Столько не поместится в запас.'
  return ''
})
const presets = computed(() => [0.25, 0.5, 1].map(f => Math.floor(have.value * f)).filter((v, i, a) => v > 0 && a.indexOf(v) === i))

function flip(to: 'gold' | 'energy') { give.value = to; amount.value = to === 'gold' ? 100 : 25 }

async function trade() {
  if (block.value || busy.value) return
  busy.value = true
  try {
    const before = props.player
    const p = await api.post<Player>('/city/market', { give: give.value, amount: amount.value })
    game.setPlayer(p)
    // The amount the server actually paid, not the local estimate.
    toasts.push({ kind: 'success', title: give.value === 'gold' ? `Получено ${num(p.energy - before.energy)} энергии` : `Получено ${num(p.gold - before.gold)} золота` }, 2500)
  } catch (e) { toasts.error(e) } finally { busy.value = false }
}
</script>

<template>
  <div class="market">
    <h3>Обмен</h3>
    <div class="seg" role="group" aria-label="Что отдать">
      <button :aria-pressed="give === 'gold'" :class="{ on: give === 'gold' }" @click="flip('gold')"><span class="g-coin" aria-hidden="true" />Золото → энергия</button>
      <button :aria-pressed="give === 'energy'" :class="{ on: give === 'energy' }" @click="flip('energy')"><Icon name="zap" :size="14" />Энергия → золото</button>
    </div>
    <label class="amount">
      <span>Отдать</span>
      <input v-model.number="amount" type="number" min="1" step="1" :max="Math.max(1, have)" inputmode="numeric" />
    </label>
    <div class="presets"><button v-for="p in presets" :key="p" class="chip" @click="amount = p">{{ num(p) }}</button></div>
    <p class="get">Получите <b>{{ num(got) }}</b> {{ give === 'gold' ? 'энергии' : 'золота' }} · комиссия {{ feePct }}%</p>
    <button class="g-btn wide" :disabled="!!block || busy" :aria-busy="busy" @click="trade"><span>Обменять</span></button>
    <p v-if="block" class="reason">{{ block }}</p>
  </div>
</template>

<style scoped>
.market { display: flex; flex-direction: column; gap: 8px; margin-top: 8px; padding: 10px; border-radius: 14px; background: #fff8e6; border: 2px solid var(--g-panel-b); }
h3 { font: 800 15px var(--font-game); color: #3a2410; margin: 0; }
.seg { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; }
.seg button { display: inline-flex; align-items: center; justify-content: center; gap: 4px; padding: 6px 4px; border: 2px solid var(--g-panel-b); border-radius: 10px; background: #fff; font: 700 12px var(--font-sans); color: #3a2410; cursor: pointer; }
.seg button.on { border-color: var(--g-ink); background: linear-gradient(#ffec8a, #ffd323); }
.amount { display: flex; align-items: center; gap: 8px; font: 700 13px var(--font-sans); color: #7a5530; }
.amount input { flex: 1; min-width: 0; padding: 6px 8px; border: 2px solid var(--g-panel-b); border-radius: 10px; font: 700 15px var(--font-sans); }
.presets { display: flex; gap: 6px; flex-wrap: wrap; }
.chip { padding: 3px 10px; border: 2px solid var(--g-panel-b); border-radius: 999px; background: #fff; font: 700 12px var(--font-sans); cursor: pointer; }
.get { margin: 0; font: 600 13px var(--font-sans); color: #3a2410; }
.reason { margin: 0; font: 600 12px var(--font-sans); color: #b3261e; }
</style>
