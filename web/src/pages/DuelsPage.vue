<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from '../ui/Icon.vue'
import CityScene from '../components/CityScene.vue'
import CityScene3D from '../components/CityScene3D.vue'
import Portrait3D from '../components/Portrait3D.vue'
import GameOff from '../components/GameOff.vue'
import { api, newIdempotencyKey } from '../api'
import { ago, dateTime, num } from '../lib/format'
import { useGame } from '../stores/game'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import type { Battle, City, Duelist } from '../types'

interface DuelList { items: Duelist[]; today: number; per_day: number; friendly_xp: number }
interface BattleRow { id: string; kind: 'raid' | 'friendly'; role: 'attack' | 'defense'; attacker: string; defender: string; destruction: number; stars: number; created_at: string }
const game = useGame()
const session = useSession()
const toasts = useToasts()
const router = useRouter()
const route = useRoute()
/** Answering a challenge: the opponent and the city come from it, the attack goes through it. */
const challengeId = ref(typeof route.query.challenge === 'string' ? route.query.challenge : '')
const list = ref<DuelList | null>(null)
const troops = ref<Record<string, number>>({})
const squad = ref<Record<string, number>>({})
const selected = ref<Duelist | null>(null)
const preview = ref<City | null>(null)
const history = ref<BattleRow[]>([])
const busy = ref(false)
const flat = ref(false)
let attackKey = ''

async function load() {
  try {
    const [o, a, b] = await Promise.all([
      api.get<DuelList>('/duels'),
      api.get<{ troops: Record<string, number> }>('/army'),
      api.get<{ items: BattleRow[] }>('/battles'),
    ])
    list.value = o
    troops.value = a.troops
    squad.value = Object.fromEntries(Object.entries(a.troops).map(([k, v]) => [k, v]))
    history.value = b.items
    if (challengeId.value) await pickChallenge(o)
    else if (!selected.value) {
      const wanted = typeof route.query.opponent === 'string' ? o.items.find(x => x.user_id === route.query.opponent) : null
      const first = wanted || o.items.find(x => x.available)
      if (first) pick(first)
    }
  } catch (e) { toasts.error(e) }
}
onMounted(async () => { await game.loadConfig(); load() })
// Another challenge opened while on this page: the component is reused, so follow the address.
watch(() => [route.query.challenge, route.query.opponent], ([c]) => {
  challengeId.value = typeof c === 'string' ? c : ''
  selected.value = null
  preview.value = null
  load()
})

async function pickChallenge(o: DuelList) {
  try {
    const c = await api.get<City & { opponent_id: string; opponent_name: string; defense: Duelist['defense'] }>(`/challenges/${challengeId.value}/city`)
    attackKey = newIdempotencyKey()
    const left = o.per_day - o.today
    selected.value = { user_id: c.opponent_id, display_name: c.opponent_name, townhall: c.townhall, available: left > 0, reason: left > 0 ? '' : 'Лимит дружеских боёв на сегодня исчерпан', defense: c.defense }
    preview.value = c
  } catch (e) { toasts.error(e); challengeId.value = '' }
}
async function pick(o: Duelist) {
  challengeId.value = ''
  selected.value = o
  preview.value = null
  attackKey = newIdempotencyKey()
  try {
    const c = await api.get<City>(`/duels/${o.user_id}/city`)
    // A slow answer for a colleague no longer selected must not replace the current one.
    if (selected.value === o) preview.value = c
  } catch (e) { if (selected.value === o) toasts.error(e) }
}
const housing = computed(() => Object.entries(squad.value).reduce((n, [k, v]) => n + (game.troop(k)?.housing || 1) * (v || 0), 0))
const total = computed(() => Object.values(squad.value).reduce((n, v) => n + (v || 0), 0))
const blocker = computed(() => {
  if (!selected.value) return 'Выберите коллегу.'
  if (!selected.value.available) return selected.value.reason
  if (total.value === 0) return 'Выберите войска.'
  return ''
})
function step(type: string, d: number) { squad.value[type] = Math.min(troops.value[type] || 0, Math.max(0, (squad.value[type] || 0) + d)) }
async function attack() {
  if (!selected.value) return
  busy.value = true
  try {
    // One key per prepared duel: a double click or a retried request cannot start two battles.
    const b = challengeId.value
      ? await api.post<Battle>(`/challenges/${challengeId.value}/attack`, { army: squad.value }, attackKey)
      : await api.post<Battle>('/battles/friendly', { defender_id: selected.value.user_id, army: squad.value }, attackKey)
    game.refresh()
    router.push(`/battles/${b.id}`)
  } catch (e) { toasts.error(e); attackKey = newIdempotencyKey() } finally { busy.value = false }
}
</script>

<template>
  <GameOff v-if="!session.gameOn" />
  <div v-else class="page duels-page">
    <div class="page-head">
      <div>
        <p class="eyebrow"><RouterLink to="/fortress">Крепость</RouterLink> · своя команда</p>
        <h1>Дружеские<span class="sub">бои</span></h1>
        <p class="muted">Бой ничего не стоит: войска не гибнут, трофеи не меняются. Награда — {{ list?.friendly_xp ?? 5 }} опыта.</p>
      </div>
      <span v-if="list" class="g-box today g-stroke s2">Сегодня: {{ list.today }} из {{ list.per_day }}</span>
    </div>

    <div class="layout">
      <section class="opps" aria-label="Коллеги">
        <div v-if="challengeId" class="g-panel empty">
          <span class="g-ribbon gold"><b>Бой по вызову</b></span>
          <p>Соперник согласился на бой. Выберите отряд справа — после боя он увидит повтор и сможет ответить вызовом.</p>
          <RouterLink to="/fortress?arena=1" class="g-btn blue wide"><span>Назад в арену</span></RouterLink>
        </div>
        <div v-if="list && !list.items.length && !challengeId" class="g-panel empty">
          <span class="g-ribbon"><b>Бой — по вызову</b></span>
          <p>Без вызова сейчас никто не принимает бои. Вызовите коллегу или бросьте открытый вызов команде — это в крепости, в «Арене».</p>
          <RouterLink to="/fortress?arena=1" class="g-btn yellow wide"><span>Открыть арену</span></RouterLink>
        </div>
        <button v-for="o in list?.items || []" :key="o.user_id" class="opp" :class="{ on: selected?.user_id === o.user_id, off: !o.available }" :aria-pressed="selected?.user_id === o.user_id" @click="pick(o)">
          <span class="th g-stroke s2" :title="`Ратуша ${o.townhall}`">{{ o.townhall }}</span>
          <span class="grow">
            <strong>{{ o.display_name }}</strong>
            <span class="meta">башни {{ o.defense.towers }} · стены {{ o.defense.walls }}</span>
            <span v-if="!o.available" class="reason">{{ o.reason }}</span>
          </span>
        </button>
      </section>

      <section v-if="selected" class="g-panel target">
        <span class="g-ribbon" :class="{ gold: challengeId }"><b>{{ challengeId ? 'Вызов: ' : '' }}{{ selected.display_name }}</b></span>
        <div class="preview">
          <CityScene3D v-if="preview && !flat" :buildings="preview.buildings" :grid="game.config?.grid || 12" :selected-id="null" :placing="null" :now="Date.now()" readonly @unavailable="flat = true" />
          <CityScene v-else-if="preview" :buildings="preview.buildings" :grid="game.config?.grid" compact :show-levels="true" :label="`Город коллеги ${selected.display_name}`" />
          <p v-else class="loading">Загрузка города…</p>
          <span class="th-chip g-stroke s2">Ратуша {{ selected.townhall }}</span>
        </div>
        <div class="defense">
          <span class="g-chip">Стены {{ num(selected.defense.wall_hp) }}</span>
          <span class="g-chip">Башни {{ num(selected.defense.tower_hp) }}</span>
          <span class="g-chip">Урон башен {{ selected.defense.tower_dps }}/залп</span>
          <span class="g-chip">Ратуша {{ num(selected.defense.townhall_hp) }}</span>
        </div>

        <h3>Отряд</h3>
        <div class="squad">
          <div v-for="t in game.config?.troops || []" :key="t.type" class="unit" :class="{ none: !troops[t.type] }">
            <Portrait3D :troop="t.type" :size="72" :faded="!troops[t.type]" />
            <strong>{{ t.name }}</strong>
            <span class="have">готово {{ troops[t.type] || 0 }}</span>
            <div class="stepper" role="group" :aria-label="`${t.name} в отряде`">
              <button class="g-btn blue small" aria-label="Меньше" :disabled="!(squad[t.type] > 0)" @click="step(t.type, -1)"><span>−</span></button>
              <span class="val g-stroke s2" aria-live="polite">{{ squad[t.type] || 0 }}</span>
              <button class="g-btn blue small" aria-label="Больше" :disabled="(squad[t.type] || 0) >= (troops[t.type] || 0)" @click="step(t.type, 1)"><span>+</span></button>
            </div>
          </div>
        </div>
        <p class="sum">В отряде {{ total }} бойцов ({{ housing }} мест). Бой бесплатный, все вернутся домой.</p>
        <button class="g-btn yellow big wide" :disabled="!!blocker || busy" :aria-busy="busy" @click="attack"><span>В бой!</span></button>
        <p v-if="blocker" class="reason center">{{ blocker }}</p>
      </section>
    </div>

    <section class="card panel stack history">
      <h2>Последние бои</h2>
      <p v-if="!history.length" class="muted small">Боёв ещё не было.</p>
      <div class="table-scroll" v-else>
        <table class="table">
          <thead><tr><th>Бой</th><th>Разрушения</th><th>Звёзды</th><th>Когда</th><th /></tr></thead>
          <tbody>
            <tr v-for="h in history" :key="h.id">
              <td>{{ h.kind === 'raid' ? 'Набег (архив)' : 'Дружеский' }} · {{ h.role === 'attack' ? h.defender : h.attacker }}</td>
              <td class="num">{{ h.destruction }}%</td>
              <td><span class="stars" :aria-label="`${h.stars} из 3`"><Icon v-for="i in 3" :key="i" name="star" :size="14" :class="{ lit: i <= h.stars }" /></span></td>
              <td :title="dateTime(h.created_at)">{{ ago(h.created_at) }}</td>
              <td><RouterLink :to="`/battles/${h.id}`">Смотреть</RouterLink></td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<style scoped>
.duels-page { max-width: none; }
.today { padding: 8px 14px; font-size: 15px; }
.layout { display: grid; grid-template-columns: minmax(0, 360px) minmax(0, 1fr); gap: var(--space-6); align-items: start; padding-top: 26px; }
.opps { display: flex; flex-direction: column; gap: 10px; }
.opp { display: flex; align-items: center; gap: 12px; width: 100%; text-align: left; padding: 10px 12px; border: 3px solid var(--g-ink); border-radius: 16px; background: var(--g-panel); color: #3a2410; cursor: pointer; box-shadow: inset 0 0 0 2px #fff6dd, 0 4px 0 #3e2914, 0 4px 0 2px var(--g-ink); transition: transform 0.15s; }
.opp:hover { transform: translateY(-2px); }
.opp.on { background: linear-gradient(#ffec8a, #ffd323); }
.opp.off { filter: grayscale(0.5); opacity: 0.85; }
.opp .grow { display: flex; flex-direction: column; }
.opp strong { font: 800 16px var(--font-game); }
.meta { font: 600 12px var(--font-sans); color: #7a5530; }
.th { display: grid; place-items: center; width: 44px; height: 44px; border: 2px solid var(--g-ink); border-radius: 12px; background: linear-gradient(#7db8ff, #3e8ef0 55%, #2f78da); font-size: 20px; flex: none; }
.empty p { font: 600 14px var(--font-sans); color: #5a3a1e; text-align: center; margin: 4px 0 12px; }
.target { padding-top: 44px; display: flex; flex-direction: column; gap: 12px; }
.preview { position: relative; height: 320px; border: 3px solid var(--g-ink); border-radius: 16px; overflow: hidden; background: linear-gradient(#69bdf0, #b9e4fa 70%, #e3f4fb); display: grid; place-items: center; }
.loading { font: 700 14px var(--font-sans); color: #2b3566; }
.th-chip { position: absolute; top: 8px; left: 8px; z-index: 3; padding: 4px 10px; border: 2px solid var(--g-ink); border-radius: 10px; background: rgba(24, 30, 72, 0.85); font-size: 14px; }
.defense { display: flex; flex-wrap: wrap; gap: 8px; }
.target h3 { font: 900 18px var(--font-game); color: #3a2410; }
.squad { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; }
.unit { display: flex; flex-direction: column; align-items: center; gap: 2px; padding: 8px; border: 2px solid var(--g-panel-b); border-radius: 14px; background: #fff8e6; color: #3a2410; }
.unit strong { font: 800 15px var(--font-game); }
.unit.none { opacity: 0.7; }
.have { font: 600 12px var(--font-sans); color: #7a5530; }
.stepper { display: flex; align-items: center; gap: 6px; margin-top: 4px; }
.stepper .g-btn { width: 36px; padding: 4px 0; margin-bottom: 6px; }
.val { min-width: 30px; text-align: center; font-size: 20px; }
.sum { font: 600 13px var(--font-sans); color: #5a3a1e; text-align: center; }
.wide { width: 100%; }
.reason { font: 700 12px var(--font-sans); color: #9b1f1f; }
.center { text-align: center; }
.history { margin-top: var(--space-8); }
.stars { display: inline-flex; gap: 2px; color: var(--color-border-strong); }
.stars .lit { color: #e3a800; fill: #ffd323; }
@media (max-width: 900px) { .layout { grid-template-columns: 1fr; } .preview { height: 260px; } }
@media (max-width: 520px) { .squad { grid-template-columns: 1fr; } }
</style>
