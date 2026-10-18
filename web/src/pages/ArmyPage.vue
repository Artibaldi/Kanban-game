<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import Icon from '../ui/Icon.vue'
import Portrait3D from '../components/Portrait3D.vue'
import CampScene3D from '../components/CampScene3D.vue'
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
const flat = ref(false)
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
  if (!Number.isInteger(n) || n < 1 || n > 50) return 'Количество — целое число от 1 до 50.'
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
/** clampCount: whole troops, 1 to 50, whatever was typed. */
function clampCount(type: string) { const n = Math.floor(Number(counts.value[type]) || 1); counts.value[type] = Math.min(50, Math.max(1, n)) }
function step(type: string, d: number) { counts.value[type] = Math.min(50, Math.max(1, (counts.value[type] || 0) + d)) }
const qProgress = (q: QueueItem) => {
  const s = new Date(q.started_at).getTime(), e = new Date(q.finish_at).getTime()
  return Math.max(0, Math.min(100, ((now.value - s) / Math.max(1, e - s)) * 100))
}
</script>

<template>
  <GameOff v-if="!session.gameOn" />
  <div v-else class="page army-page">
    <section class="arena" aria-label="Лагерь армии">
      <CampScene3D v-if="army && !flat" :troops="army.troops" @unavailable="flat = true" />
      <div v-else-if="army" class="flat">
        <div v-for="t in game.config?.troops || []" :key="t.type" class="flat-unit"><Portrait3D :troop="t.type" :size="96" /><b class="g-stroke">×{{ army.troops[t.type] || 0 }}</b></div>
      </div>
      <div class="hud">
        <div class="g-box title-box">
          <div class="g-stroke s2 name">Казарма</div>
          <div class="sub">Войска готовятся за энергию от подтверждённых задач</div>
        </div>
        <RouterLink to="/fortress/duels" class="g-btn blue small"><span>Дружеские бои</span></RouterLink>
      </div>
      <div v-if="p" class="g-box cap" :title="`готово ${p.army_housing}, в обучении ${p.army_queued}`">
        <div class="g-stroke s2 t">Места в лагере: {{ p.army_housing + p.army_queued }} из {{ p.army_capacity }}</div>
        <div class="g-bar hp"><i :style="{ width: Math.min(100, ((p.army_housing + p.army_queued) / Math.max(1, p.army_capacity)) * 100) + '%' }" /></div>
      </div>
    </section>
    <p v-if="p?.debt" class="notice danger debt">После отмены награды баланс отрицательный. Обучение откроется, когда следующие награды погасят долг.</p>

    <div class="troops">
      <article v-for="t in game.config?.troops || []" :key="t.type" class="tcard" :class="t.type">
        <header class="hd"><span class="g-stroke s2">{{ t.name }}</span><span class="ready g-stroke s2"><span class="sr-only">готово </span>×{{ army?.troops[t.type] ?? 0 }}</span></header>
        <div class="art"><Portrait3D :troop="t.type" :size="120" /></div>
        <p class="role">{{ t.role }}</p>
        <dl class="stats">
          <div><dt>Здоровье</dt><dd>{{ t.hp }}</dd><span class="meter hp"><span :style="{ width: Math.min(100, t.hp / 1.5) + '%' }" /></span></div>
          <div><dt>Атака</dt><dd>{{ t.attack }}</dd><span class="meter atk"><span :style="{ width: Math.min(100, t.attack / 0.42) + '%' }" /></span></div>
          <div><dt>По укреплениям</dt><dd>×{{ t.siege_mult }}</dd><span class="meter siege"><span :style="{ width: Math.min(100, t.siege_mult / 0.06) + '%' }" /></span></div>
          <div><dt>Мест</dt><dd>{{ t.housing }}</dd></div>
        </dl>
        <div class="train">
          <div class="stepper" role="group" :aria-label="`Количество: ${t.name}`">
            <button class="g-btn blue small" aria-label="Меньше" :disabled="(counts[t.type] || 1) <= 1" @click="step(t.type, -1)"><span>−</span></button>
            <label class="sr-only" :for="`n-${t.type}`">Количество</label>
            <input :id="`n-${t.type}`" v-model.number="counts[t.type]" type="number" min="1" max="50" step="1" class="count" @change="clampCount(t.type)" />
            <button class="g-btn blue small" aria-label="Больше" :disabled="(counts[t.type] || 0) >= 50" @click="step(t.type, 1)"><span>+</span></button>
          </div>
          <span class="time"><Icon name="clock" :size="14" />{{ duration(t.seconds * (counts[t.type] || 0) * (game.config?.settings.time_scale || 1) * 1000) }}</span>
        </div>
        <button class="g-btn wide" :disabled="!!block(t.type) || !!busy" :aria-busy="busy === t.type" @click="train(t.type)">
          <span>Обучить</span><span class="cost"><Icon name="zap" :size="16" />{{ num(t.cost * (counts[t.type] || 0)) }}</span>
        </button>
        <p v-if="block(t.type)" class="reason">{{ block(t.type) }}</p>
      </article>
    </div>

    <section class="g-panel queue">
      <span class="g-ribbon"><b>Очередь обучения</b></span>
      <p v-if="!army?.queue.length" class="empty">Очередь пуста — выберите отряд и нажмите «Обучить».</p>
      <div v-for="q in army?.queue || []" :key="q.id" class="slot">
        <Portrait3D :troop="q.type" :size="48" />
        <div class="grow">
          <div class="row"><strong class="grow">{{ game.troop(q.type)?.name }} × {{ q.count }}</strong>
            <span class="num">{{ now < new Date(q.started_at).getTime() ? 'ждёт очереди' : duration(new Date(q.finish_at).getTime() - now) }}</span></div>
          <div class="g-bar"><i :style="{ width: qProgress(q) + '%' }" /></div>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.army-page { max-width: none; padding-top: var(--space-4); }
.arena { position: relative; height: min(56vh, 520px); min-height: 380px; border-radius: var(--radius-xl); overflow: hidden; border: 3px solid var(--g-ink); box-shadow: 0 8px 0 rgba(30, 30, 30, 0.2), var(--shadow-md); isolation: isolate; }
.flat { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; gap: 32px; background: linear-gradient(#77c6f2, #bde6fa 60%, #d9eef5); }
.flat-unit { display: flex; flex-direction: column; align-items: center; }
.flat-unit b { font-size: 22px; }
.hud { position: absolute; top: 12px; left: 12px; right: 12px; z-index: 5; display: flex; justify-content: space-between; align-items: flex-start; gap: 10px; }
.title-box { padding: 8px 14px; }
.name { font-size: 20px; }
.sub { font: 700 13px var(--font-sans); color: #dde6ff; }
.cap { position: absolute; left: 50%; bottom: 14px; transform: translateX(-50%); z-index: 5; padding: 8px 14px 10px; width: min(380px, calc(100% - 24px)); }
.cap .t { font-size: 15px; margin-bottom: 5px; }
.cap .g-bar { height: 16px; }
.debt { margin-top: var(--space-4); }
.troops { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 22px; margin: 26px 0 40px; }
.tcard { display: flex; flex-direction: column; gap: 10px; padding: 0 16px 14px; border: 3px solid var(--g-ink); border-radius: 18px; background: var(--g-panel); color: #3a2410; box-shadow: inset 0 0 0 3px #fff6dd, 0 6px 0 #3e2914, 0 6px 0 3px var(--g-ink); overflow: hidden; }
.hd { display: flex; align-items: center; justify-content: space-between; margin: 0 -16px; padding: 8px 14px 10px; border-bottom: 3px solid var(--g-ink); background: linear-gradient(#7db8ff, #3e8ef0 55%, #2f78da); font-size: 19px; }
.raider .hd { background: linear-gradient(#ff8a7a, #f04848 55%, #cf2f33); }
.ram .hd { background: linear-gradient(#e2b47a, #b07a45 55%, #8c5e33); }
.ready { font-size: 18px; }
.art { display: grid; place-items: center; height: 128px; background: radial-gradient(closest-side, rgba(255, 255, 255, 0.7), transparent); }
.role { font: 700 14px var(--font-sans); color: #7a5530; text-align: center; }
.stats { display: grid; grid-template-columns: 1fr 1fr; gap: 6px 14px; margin: 0; font: 700 13px var(--font-sans); }
.stats div { display: grid; grid-template-columns: 1fr auto; gap: 2px 8px; }
.stats dt { color: #7a5530; }
.stats dd { margin: 0; font: 900 14px var(--font-game); }
.meter { grid-column: 1 / -1; height: 8px; border: 2px solid var(--g-ink); border-radius: 6px; background: #3a2f2a; overflow: hidden; }
.meter span { display: block; height: 100%; max-width: 100%; }
.meter.hp span { background: #5cc02e; }
.meter.atk span { background: #f04848; }
.meter.siege span { background: #ffb21e; }
.train { display: flex; align-items: center; justify-content: space-between; gap: 8px; flex-wrap: wrap; }
.stepper { display: flex; align-items: center; gap: 6px; }
.stepper .g-btn { width: 38px; padding: 5px 0; margin-bottom: 6px; }
.count { width: 52px; height: 34px; text-align: center; border: 2px solid var(--g-panel-b); border-radius: 10px; background: #fff8e6; font: 900 17px var(--font-game); color: #3a2410; }
.time { display: inline-flex; align-items: center; gap: 4px; font: 700 13px var(--font-sans); color: #5a3a1e; }
.wide { width: 100%; justify-content: space-between; }
.cost { display: inline-flex; align-items: center; gap: 4px; padding: 2px 10px; border: 2px solid var(--g-ink); border-radius: 999px; background: linear-gradient(#9c7bff, #6b4ce6); font-size: 15px; }
.reason { font: 700 12px var(--font-sans); color: #9b1f1f; }
.queue { padding-top: 40px; margin-bottom: var(--space-6); }
.empty { text-align: center; font: 600 14px var(--font-sans); color: #7a5530; }
.slot { display: flex; align-items: center; gap: 12px; margin-bottom: 10px; font: 700 14px var(--font-sans); }
.slot .g-bar { height: 14px; margin-top: 4px; }
@media (max-width: 960px) { .troops { grid-template-columns: 1fr; } }
@media (max-width: 767px) { .arena { height: 44vh; min-height: 300px; border-radius: 22px; } .sub { display: none; } .name { font-size: 16px; } }
</style>
