<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import CityScene from '../components/CityScene.vue'
import BattleScene3D from '../components/BattleScene3D.vue'
import Portrait3D from '../components/Portrait3D.vue'
import { api } from '../api'
import { dateTime, num } from '../lib/format'
import { aliveShown, destructionAt, ruinedAt, shownCount, targetOf } from '../lib/battleView'
import { reduced } from '../ui/motion'
import { useGame } from '../stores/game'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import type { Battle, BuildingType, City } from '../types'

const route = useRoute()
const game = useGame()
const session = useSession()
const toasts = useToasts()
const battle = ref<Battle | null>(null)
const city = ref<City | null>(null)
const step = ref(0) // 0 = before the battle; n = after round n
const playing = ref(false)
const speed = ref(1)
const flat = ref(false)
const ready = ref(false)
const shownStars = ref(0)
const scene = ref<InstanceType<typeof BattleScene3D>>()
let timer = 0

onMounted(async () => {
  await game.loadConfig()
  try {
    battle.value = await api.get<Battle>(`/battles/${route.params.id}`)
    // The defender's city as it stands now — for both sides of the battle.
    city.value = await api.get<City>(`/battles/${battle.value.id}/city`).catch(() => null)
    // Without a city or in work mode there is nothing to replay on: show the result.
    if (!city.value || !session.gameOn) { flat.value = !city.value; finish() }
  } catch (e) { toasts.error(e) }
})
onUnmounted(() => clearTimeout(timer))

const b = computed(() => battle.value)
const start = computed(() => b.value ? { wall_hp: b.value.defense.wall_hp, tower_hp: b.value.defense.tower_hp, townhall_hp: b.value.defense.townhall_hp, other_hp: b.value.defense.other_hp } : null)
/** Every event up to round i: walls, towers and the town hall fall by them. */
const eventsUpTo = (i: number) => (b.value?.rounds.slice(0, i).flatMap(r => r.events) ?? [])
const stateAt = (i: number) => (!b.value || i <= 0 ? start.value! : b.value.rounds[i - 1])
const cur = computed(() => (b.value && step.value > 0 ? b.value.rounds[step.value - 1] : null))
const shown = computed(() => Object.fromEntries(Object.entries(b.value?.army || {}).map(([k, n]) => [k, shownCount(n)])))
const done = computed(() => !!b.value && step.value >= b.value.rounds.length && !playing.value)
const destruction = computed(() => (!b.value ? 0 : done.value ? b.value.destruction : destructionAt(start.value!, stateAt(step.value))))
const alive = computed(() => (cur.value ? cur.value.alive : b.value?.army || {}))
const won = computed(() => (b.value?.stars || 0) > 0)
const iAttacked = computed(() => b.value?.attacker_id === session.me?.id)
const friendly = computed(() => b.value?.kind === 'friendly')
const hp = computed(() => {
  const d = b.value?.defense, r = cur.value
  if (!d) return []
  return [
    { key: 'wall', name: 'Стены', max: d.wall_hp, now: r ? r.wall_hp : d.wall_hp },
    { key: 'tower', name: 'Башни', max: d.tower_hp, now: r ? r.tower_hp : d.tower_hp },
    { key: 'townhall', name: 'Ратуша', max: d.townhall_hp, now: r ? r.townhall_hp : d.townhall_hp },
    { key: 'other', name: 'Казармы и хранилища', max: d.other_hp, now: r ? r.other_hp : d.other_hp },
  ].filter(h => h.max > 0)
})
// The flat fallback greys out ruined buildings the old way.
const ruinedTypes = computed<BuildingType[]>(() => {
  if (!start.value) return []
  const out: BuildingType[] = []
  for (const p of ruinedAt(start.value, stateAt(step.value), eventsUpTo(step.value))) out.push(...(p === 'other' ? (['barracks', 'storage'] as BuildingType[]) : [p as BuildingType]))
  return out
})

function beat(i: number, dur: number) {
  const r = stateAt(i), round = b.value!.rounds[i - 1]
  const figs = Object.fromEntries(Object.entries(b.value!.army).map(([k, n]) => [k, aliveShown(round.alive[k] ?? 0, n, shown.value[k])]))
  scene.value?.round(targetOf(stateAt(i - 1), r), ruinedAt(start.value!, r, eventsUpTo(i)), figs, dur / 1000)
}
function play() {
  if (!b.value) return
  clearTimeout(timer)
  shownStars.value = 0
  step.value = 0
  if (reduced() || flat.value || !ready.value) { finish(); return }
  playing.value = true
  scene.value?.clear()
  scene.value?.start(b.value.army, shown.value)
  const next = () => {
    if (!b.value) return
    if (step.value >= b.value.rounds.length) { playing.value = false; scene.value?.cheer(); dropStars(); return }
    step.value++
    const dur = 1100 / speed.value
    beat(step.value, dur)
    timer = window.setTimeout(next, dur)
  }
  timer = window.setTimeout(next, 900 / speed.value)
}
function finish() {
  clearTimeout(timer)
  if (!b.value) return
  const n = b.value.rounds.length
  if (ready.value && n) {
    if (step.value === 0) { scene.value?.clear(); scene.value?.start(b.value.army, shown.value) }
    step.value = n
    beat(n, 10)
    // The fight is over: no more swings or trembling behind the result.
    scene.value?.cheer()
  } else step.value = n
  playing.value = false
  dropStars()
}
function dropStars() {
  const total = b.value?.stars || 0
  const drop = (i: number) => { if (i > total) return; shownStars.value = i; if (i < total) timer = window.setTimeout(() => drop(i + 1), reduced() ? 0 : 420) }
  timer = window.setTimeout(() => drop(1), reduced() ? 0 : 500)
  nextTick(() => document.querySelector<HTMLElement>('.result .g-btn')?.focus())
}
function onReady() {
  ready.value = true
  if (session.gameOn) play()
  // Already showing the result (work mode): bring the scene to the end state too.
  else finish()
}

const eventText: Record<string, string> = { wall_breached: 'Стены пробиты', towers_down: 'Башни разрушены', townhall_down: 'Ратуша пала' }
function roundText(r: Battle['rounds'][number]) {
  // In a friendly duel nobody dies: they are out of this battle and come back home.
  const fell = friendly.value ? 'выбыли из боя' : 'погибли'
  const killed = Object.entries(r.killed).filter(([, n]) => n > 0).map(([k, n]) => `${game.troop(k)?.name || k} × ${n}`).join(', ')
  const parts = [`Башни нанесли ${r.tower_damage} урона${killed ? ` — ${fell}: ${killed}` : ''}.`]
  if (r.wall_damage) parts.push(`Урон стенам: ${r.wall_damage}.`)
  if (r.tower_hit) parts.push(`Урон башням: ${r.tower_hit}.`)
  if (r.base_damage) parts.push(`Урон зданиям: ${r.base_damage}.`)
  return parts.join(' ')
}
</script>

<template>
  <div class="page battle-page">
    <div v-if="!b" class="muted">Загрузка боя…</div>
    <template v-else>
      <section class="arena" aria-label="Повтор боя">
        <BattleScene3D v-if="city && !flat" ref="scene" :buildings="city.buildings" :grid="city.grid || game.config?.grid || 16" @ready="onReady" @unavailable="flat = true; finish()" />
        <div v-else class="flat">
          <CityScene v-if="city" :buildings="city.buildings" :grid="city.grid || game.config?.grid || 16" compact :ruined="ruinedTypes" :label="`Город ${b.defender}`" />
          <p v-else class="g-box no-city">Город соперника сейчас недоступен — показан итог боя.</p>
        </div>

        <div class="hud">
          <div class="g-box who">
            <div class="g-stroke s2 name">{{ b.attacker }} <span class="vs">против</span> {{ b.defender }}</div>
            <div class="sub">{{ friendly ? 'Дружеский бой' : 'Набег (архив)' }} · {{ dateTime(b.created_at) }}</div>
          </div>
          <div class="g-box score" :aria-label="`Разрушения ${destruction}%`">
            <div class="pct g-stroke">{{ destruction }}%</div>
            <div class="sub" aria-live="polite">{{ done ? 'бой окончен' : step === 0 ? 'подготовка' : `раунд ${step} из ${b.rounds.length}` }}</div>
          </div>
          <div class="ctl">
            <template v-if="playing">
              <button class="g-btn blue small" :class="{ on: speed === 2 }" :aria-pressed="speed === 2" @click="speed = speed === 1 ? 2 : 1"><span>×2</span></button>
              <button class="g-btn blue small" @click="finish"><span>Пропустить</span></button>
            </template>
            <RouterLink to="/fortress/duels" class="g-btn blue small"><span>К боям</span></RouterLink>
          </div>
        </div>

        <div class="bottom">
          <div class="g-box hp">
            <div v-for="h in hp" :key="h.key" class="hp-row">
              <span class="lbl">{{ h.name }}</span>
              <div class="g-bar hp" :class="{ low: h.now / h.max < 0.35 }"><i :style="{ width: (h.now / Math.max(1, h.max)) * 100 + '%' }" /></div>
            </div>
          </div>
          <div class="squad">
            <div v-for="(n, k) in b.army" :key="k" class="slot" :class="{ out: !alive[k] }" :title="`${game.troop(String(k))?.name}: ${alive[k] ?? n} из ${n}`">
              <Portrait3D :troop="String(k)" :size="52" :faded="!alive[k]" />
              <span class="sr-only">{{ game.troop(String(k))?.name }}: в строю</span>
              <b class="g-stroke s2">{{ alive[k] ?? n }}/{{ n }}</b>
            </div>
          </div>
        </div>

        <div v-if="done" class="result" role="dialog" aria-label="Итог боя">
          <div class="g-panel pop">
            <span class="g-ribbon" :class="{ gold: won, red: !won }"><b>{{ won ? (iAttacked ? 'Победа!' : 'Город взят') : (iAttacked ? 'Атака отбита' : 'Оборона выстояла') }}</b></span>
            <div class="stars" :aria-label="`Звёзд: ${b.stars} из 3`">
              <svg v-for="i in 3" :key="i" viewBox="0 0 24 24" class="star" :class="{ on: i <= shownStars, mid: i === 2 }" aria-hidden="true"><path d="m12 2 3.1 6.3 6.9 1-5 4.9 1.2 6.8L12 17.8 5.8 21 7 14.2l-5-4.9 6.9-1z" stroke="#1E1E1E" stroke-width="1.6" stroke-linejoin="round" /></svg>
            </div>
            <p class="dest">{{ b.destruction }}% разрушений</p>
            <div class="sums">
              <template v-if="friendly">
                <div v-if="iAttacked">+{{ b.xp }} опыта</div>
                <div>Потерь нет: все вернулись домой</div>
              </template>
              <template v-else>
                <div v-if="iAttacked"><span class="g-coin" aria-hidden="true" />+{{ num(b.gold) }}</div>
                <div v-if="iAttacked">+{{ b.xp }} опыта</div>
              </template>
            </div>
            <p class="note">Бои не влияют на рабочие баллы и не участвуют в оценке работы.</p>
            <div class="row-btns">
              <button class="g-btn blue" @click="play"><span>Повторить</span></button>
              <RouterLink to="/fortress/duels" class="g-btn"><span>Ещё бой</span></RouterLink>
            </div>
          </div>
        </div>
      </section>

      <details class="card panel log-card">
        <summary><h2>Журнал боя</h2></summary>
        <ol class="log">
          <li class="small">В бой вступили: {{ Object.entries(b.army).map(([k, n]) => `${game.troop(k)?.name} × ${n}`).join(', ') }}. Оборона: {{ b.defense.towers }} башен, {{ b.defense.walls }} стен.</li>
          <li v-for="r in b.rounds.slice(0, step)" :key="r.n" class="small">
            <strong>Раунд {{ r.n }}.</strong> {{ roundText(r) }}
            <span v-for="e in r.events" :key="e" class="badge warning">{{ eventText[e] }}</span>
          </li>
        </ol>
        <p class="xs muted">Случайный фактор ограничен ±{{ Math.round((game.config?.settings.random_spread || 0.1) * 100) }}% на каждый удар; зерно боя {{ b.seed }} сохранено, результат воспроизводим.</p>
      </details>
    </template>
  </div>
</template>

<style scoped>
.battle-page { max-width: none; padding-top: var(--space-4); }
.arena { position: relative; height: calc(100vh - 150px); min-height: 560px; border-radius: var(--radius-xl); overflow: hidden; border: 3px solid var(--g-ink); box-shadow: 0 8px 0 rgba(30, 30, 30, 0.2), var(--shadow-md); isolation: isolate; user-select: none; }
.flat { position: absolute; inset: 0; display: grid; place-items: center; padding: 100px 12px 150px; background: linear-gradient(#69bdf0, #b9e4fa 70%, #e3f4fb); }
.no-city { padding: 10px 16px; font: 700 14px var(--font-sans); }
.hud { position: absolute; top: 12px; left: 12px; right: 12px; z-index: 5; display: flex; justify-content: space-between; align-items: flex-start; gap: 10px; }
.who { padding: 8px 14px; max-width: 40%; }
.name { font-size: 18px; }
.vs { font-size: 13px; color: var(--g-gold); }
.sub { font: 700 12px var(--font-sans); color: #dde6ff; }
.score { padding: 6px 18px 8px; text-align: center; }
.pct { font-size: 34px; line-height: 1; color: var(--g-gold); }
.ctl { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; }
.bottom { position: absolute; left: 12px; right: 12px; bottom: 12px; z-index: 5; display: flex; justify-content: space-between; align-items: flex-end; gap: 12px; pointer-events: none; }
.hp { padding: 8px 12px; display: grid; gap: 6px; min-width: 260px; }
.hp-row { display: grid; grid-template-columns: 130px 1fr; gap: 8px; align-items: center; }
.lbl { font: 700 12px var(--font-sans); color: #dde6ff; }
.hp-row .g-bar { height: 12px; }
.squad { display: flex; gap: 8px; }
.slot { display: flex; flex-direction: column; align-items: center; padding: 4px 8px 6px; border: 2px solid var(--g-ink); border-radius: 14px; background: linear-gradient(#fff7de, #f5e7c6); box-shadow: 0 3px 0 var(--g-ink); }
.slot b { font-size: 15px; }
.slot.out { filter: grayscale(1); opacity: 0.7; }
.result { position: absolute; z-index: 8; left: 50%; top: 50%; transform: translate(-50%, -50%); width: min(440px, calc(100% - 24px)); text-align: center; }
.result::before { content: ""; position: fixed; inset: 0; z-index: -1; background: rgba(10, 12, 34, 0.35); pointer-events: none; }
.stars { display: flex; justify-content: center; align-items: flex-end; gap: 6px; margin: 4px 0; }
.star { width: 64px; height: 64px; fill: #56608a; }
.star.mid { width: 80px; height: 80px; }
.star.on { fill: #ffd323; animation: stamp 0.5s cubic-bezier(0.34, 1.56, 0.64, 1); }
@keyframes stamp { 0% { transform: translateY(-60px) scale(1.4) rotate(-14deg); opacity: 0; } 55% { transform: translateY(0) scale(1.4); opacity: 1; } 100% { transform: scale(1); } }
.dest { font: 900 20px var(--font-game); color: #3a2410; }
.sums { display: flex; justify-content: center; gap: 10px; margin: 10px 0; flex-wrap: wrap; }
.sums div { display: flex; align-items: center; gap: 6px; padding: 8px 12px; border-radius: 12px; background: rgba(107, 70, 38, 0.12); font: 800 15px var(--font-game); color: #3a2410; }
.note { text-align: center; font: 600 13px var(--font-sans); color: #8a6844; }
.row-btns { display: flex; gap: 12px; justify-content: center; margin-top: 10px; }
.log-card { margin-top: var(--space-5); }
.log-card summary { cursor: pointer; list-style: none; }
.log-card summary h2 { display: inline; font-size: 18px; }
.log { margin: 12px 0 0; padding-left: 18px; display: flex; flex-direction: column; gap: var(--space-2); }
@media (max-width: 767px) {
  .arena { height: calc(100vh - 210px); min-height: 520px; border-radius: 22px; }
  .who { max-width: 55%; }
  .name { font-size: 14px; }
  .pct { font-size: 24px; }
  .bottom { flex-direction: column; align-items: stretch; }
  .hp { min-width: 0; }
  .hp-row { grid-template-columns: 100px 1fr; }
  .squad { justify-content: center; }
  .ctl .g-btn { font-size: 12px; padding: 5px 9px; }
}
@media (prefers-reduced-motion: reduce) { .star.on { animation: none; } }
</style>
