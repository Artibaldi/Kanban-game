<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import Icon from '../ui/Icon.vue'
import TroopIcon from '../components/TroopIcon.vue'
import GameOff from '../components/GameOff.vue'
import { api } from '../api'
import { duration, num } from '../lib/format'
import { useGame } from '../stores/game'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import type { Player } from '../types'

interface QueueItem { id: string; type: string; count: number; started_at: string; finish_at: string }
interface Army { troops: Record<string, number>; queue: QueueItem[]; player: Player; server_time: string }
const game = useGame()
const session = useSession()
const toasts = useToasts()
const army = ref<Army | null>(null)
const counts = ref<Record<string, number>>({ guard: 1, raider: 1, ram: 1 })
const busy = ref('')
const now = ref(Date.now())
let offset = 0, tick = 0, reloading = false

async function load() {
  try {
    const a = await api.get<Army>('/army')
    offset = new Date(a.server_time).getTime() - Date.now()
    army.value = a
    game.setPlayer(a.player)
  } catch (e) { toasts.error(e) }
}
onMounted(async () => {
  await game.loadConfig()
  await load()
  tick = window.setInterval(async () => {
    now.value = Date.now() + offset
    if (!reloading && army.value?.queue.some(q => new Date(q.finish_at).getTime() <= now.value)) {
      reloading = true
      await load()
      reloading = false
    }
  }, 1000)
})
onUnmounted(() => clearInterval(tick))

const p = computed(() => army.value?.player)
const free = computed(() => (p.value ? p.value.army_capacity - p.value.army_housing - p.value.army_queued : 0))
function block(type: string) {
  const t = game.troop(type)
  if (!t || !p.value) return 'Загрузка…'
  const n = counts.value[type] || 0
  if (n < 1) return 'Укажите количество.'
  if (p.value.debt) return 'Сначала погасите долг.'
  if (p.value.army_capacity === 0) return 'Постройте казарму.'
  if (t.housing * n > free.value) return `Не хватает мест: свободно ${free.value}.`
  if (t.cost * n > p.value.energy) return `Нужно ${t.cost * n} энергии.`
  return ''
}
async function train(type: string) {
  busy.value = type
  try {
    army.value = await api.post<Army>('/army/train', { type, count: counts.value[type] })
    game.setPlayer(army.value.player)
    toasts.push({ kind: 'success', title: `Обучение: ${game.troop(type)?.name} × ${counts.value[type]}` }, 2500)
  } catch (e) { toasts.error(e) } finally { busy.value = '' }
}
const qProgress = (q: QueueItem) => {
  const s = new Date(q.started_at).getTime(), e = new Date(q.finish_at).getTime()
  return Math.max(0, Math.min(100, ((now.value - s) / Math.max(1, e - s)) * 100))
}
</script>

<template>
  <GameOff v-if="!session.gameOn" />
  <div v-else class="page">
    <div class="page-head">
      <div>
        <p class="eyebrow">Казарма</p>
        <h1>Армия</h1>
        <p class="muted">Войска готовятся за энергию. Энергию приносят подтверждённые задачи.</p>
      </div>
      <RouterLink to="/opponents" class="btn primary"><Icon name="swords" :size="16" />К выбору соперника</RouterLink>
    </div>

    <div v-if="p" class="capacity card panel">
      <div class="row wrap"><strong class="grow">Места в казармах: {{ p.army_housing + p.army_queued }} из {{ p.army_capacity }}</strong>
        <span class="small muted">готово {{ p.army_housing }}, в обучении {{ p.army_queued }}</span></div>
      <div class="bar"><span :style="{ width: Math.min(100, ((p.army_housing + p.army_queued) / Math.max(1, p.army_capacity)) * 100) + '%' }" /></div>
    </div>

    <div class="troops">
      <article v-for="t in game.config?.troops || []" :key="t.type" class="card troop-card">
        <div class="row">
          <TroopIcon :type="t.type" :size="64" />
          <div class="grow">
            <h2>{{ t.name }}</h2>
            <p class="small muted">{{ t.role }}</p>
          </div>
          <div class="ready"><span class="num">{{ army?.troops[t.type] ?? 0 }}</span><span class="xs muted">готово</span></div>
        </div>
        <dl class="stats small">
          <div><dt>Здоровье</dt><dd>{{ t.hp }}</dd><span class="meter"><span :style="{ width: t.hp / 1.5 + '%' }" /></span></div>
          <div><dt>Атака</dt><dd>{{ t.attack }}</dd><span class="meter atk"><span :style="{ width: t.attack / 0.42 + '%' }" /></span></div>
          <div><dt>По укреплениям</dt><dd>×{{ t.siege_mult }}</dd><span class="meter siege"><span :style="{ width: Math.min(100, t.siege_mult / 0.06) + '%' }" /></span></div>
          <div><dt>Мест</dt><dd>{{ t.housing }}</dd></div>
        </dl>
        <div class="train row wrap">
          <label class="sr-only" :for="`n-${t.type}`">Количество</label>
          <input :id="`n-${t.type}`" v-model.number="counts[t.type]" type="number" min="1" max="50" class="input count" />
          <span class="small cost"><Icon name="zap" :size="14" />{{ num(t.cost * (counts[t.type] || 0)) }}</span>
          <span class="small muted"><Icon name="clock" :size="14" />{{ duration(t.seconds * (counts[t.type] || 0) * (game.config?.settings.time_scale || 1) * 1000) }}</span>
          <button class="btn primary sm" :disabled="!!block(t.type) || !!busy" :aria-busy="busy === t.type" @click="train(t.type)">Обучить</button>
        </div>
        <p v-if="block(t.type)" class="xs reason">{{ block(t.type) }}</p>
      </article>
    </div>

    <section class="card panel stack queue">
      <h2>Очередь обучения</h2>
      <p v-if="!army?.queue.length" class="muted small">Очередь пуста.</p>
      <div v-for="q in army?.queue || []" :key="q.id" class="row">
        <TroopIcon :type="q.type" :size="36" />
        <div class="grow stack" style="gap: 4px">
          <div class="row small"><strong class="grow">{{ game.troop(q.type)?.name }} × {{ q.count }}</strong>
            <span class="num muted">{{ now < new Date(q.started_at).getTime() ? 'ждёт очереди' : duration(new Date(q.finish_at).getTime() - now) }}</span></div>
          <div class="bar"><span :style="{ width: qProgress(q) + '%' }" /></div>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.capacity { display: flex; flex-direction: column; gap: var(--space-2); padding: var(--space-4); margin-bottom: var(--space-4); }
.troops { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: var(--space-4); margin-bottom: var(--space-4); }
.troop-card { padding: var(--space-4); display: flex; flex-direction: column; gap: var(--space-3); }
.troop-card h2 { font-size: var(--text-lg); }
.ready { display: flex; flex-direction: column; align-items: center; }
.ready .num { font-size: var(--text-xl); font-weight: 800; }
.stats { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-2) var(--space-4); margin: 0; }
.stats div { display: grid; grid-template-columns: 1fr auto; gap: 2px 8px; }
.stats dt { color: var(--color-text-muted); }
.stats dd { margin: 0; font-weight: 700; }
.meter { grid-column: 1 / -1; height: 5px; border-radius: 3px; background: var(--color-surface-sunken); overflow: hidden; }
.meter span { display: block; height: 100%; background: var(--color-success); max-width: 100%; }
.meter.atk span { background: var(--color-danger); }
.meter.siege span { background: var(--color-warning); }
.count { width: 72px; }
.cost { display: inline-flex; align-items: center; gap: 3px; color: var(--color-energy); font-weight: 750; }
.train .btn { margin-left: auto; }
.reason { color: var(--color-warning); }
.queue .row { align-items: center; }
@media (max-width: 960px) { .troops { grid-template-columns: 1fr; } }
</style>
