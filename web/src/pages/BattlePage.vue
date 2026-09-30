<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import Icon from '../ui/Icon.vue'
import TroopIcon from '../components/TroopIcon.vue'
import CityScene from '../components/CityScene.vue'
import { api } from '../api'
import { dateTime, num } from '../lib/format'
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
let timer = 0

onMounted(async () => {
  await game.loadConfig()
  try {
    battle.value = await api.get<Battle>(`/battles/${route.params.id}`)
    const mine = battle.value.attacker_id === session.me?.id
    city.value = await api.get<City>(mine ? `/opponents/${battle.value.defender_id}/city` : '/city').catch(() => null)
    if (session.gameOn && !window.matchMedia('(prefers-reduced-motion: reduce)').matches) play()
    else step.value = battle.value.rounds.length
  } catch (e) { toasts.error(e) }
})
onUnmounted(() => clearInterval(timer))

function play() {
  if (!battle.value) return
  clearInterval(timer)
  step.value = 0
  playing.value = true
  timer = window.setInterval(() => {
    if (!battle.value || step.value >= battle.value.rounds.length) { playing.value = false; clearInterval(timer); return }
    step.value++
  }, 1100)
}
function skip() { clearInterval(timer); playing.value = false; if (battle.value) step.value = battle.value.rounds.length }

const b = computed(() => battle.value)
const cur = computed(() => (b.value && step.value > 0 ? b.value.rounds[step.value - 1] : null))
const hp = computed(() => {
  const d = b.value?.defense
  if (!d) return []
  const r = cur.value
  return [
    { key: 'wall', name: 'Стены', max: d.wall_hp, now: r ? r.wall_hp : d.wall_hp },
    { key: 'tower', name: 'Башни', max: d.tower_hp, now: r ? r.tower_hp : d.tower_hp },
    { key: 'townhall', name: 'Ратуша', max: d.townhall_hp, now: r ? r.townhall_hp : d.townhall_hp },
    { key: 'other', name: 'Казармы и хранилища', max: d.other_hp, now: r ? r.other_hp : d.other_hp },
  ]
})
const ruined = computed<BuildingType[]>(() => {
  const out: BuildingType[] = []
  for (const h of hp.value) {
    if (h.max > 0 && h.now <= 0) {
      if (h.key === 'other') out.push('barracks', 'storage')
      else out.push(h.key as BuildingType)
    }
  }
  return out
})
const alive = computed(() => (cur.value ? cur.value.alive : b.value?.army || {}))
const done = computed(() => !!b.value && step.value >= b.value.rounds.length)
const won = computed(() => (b.value?.stars || 0) > 0)
const iAttacked = computed(() => b.value?.attacker_id === session.me?.id)
const eventText: Record<string, string> = { wall_breached: 'Стены пробиты', towers_down: 'Башни разрушены', townhall_down: 'Ратуша пала' }
function roundText(r: Battle['rounds'][number]) {
  const killed = Object.entries(r.killed).filter(([, n]) => n > 0).map(([k, n]) => `${game.troop(k)?.name || k} × ${n}`).join(', ')
  const parts = [`Башни нанесли ${r.tower_damage} урона${killed ? ` — погибли: ${killed}` : ''}.`]
  if (r.wall_damage) parts.push(`Урон стенам: ${r.wall_damage}.`)
  if (r.tower_hit) parts.push(`Урон башням: ${r.tower_hit}.`)
  if (r.base_damage) parts.push(`Урон зданиям: ${r.base_damage}.`)
  return parts.join(' ')
}
</script>

<template>
  <div class="page">
    <RouterLink to="/opponents" class="back small"><Icon name="left" :size="14" />К соперникам</RouterLink>
    <div v-if="!b" class="muted">Загрузка боя…</div>
    <template v-else>
      <div class="page-head">
        <div>
          <p class="eyebrow">{{ dateTime(b.created_at) }}</p>
          <h1>Бой: {{ b.attacker }} против {{ b.defender }}</h1>
        </div>
        <div class="row">
          <button class="btn" :disabled="playing" @click="play"><Icon name="history" :size="16" />Повторить</button>
          <button class="btn" :disabled="!playing" @click="skip">Показать итог</button>
        </div>
      </div>

      <div class="layout">
        <section class="card arena">
          <div class="round-label">{{ step === 0 ? 'Подготовка' : done ? 'Бой окончен' : `Раунд ${step} из ${b.rounds.length}` }}</div>
          <div class="scene-box" :class="{ shaking: playing && step > 0 }">
            <CityScene v-if="city" :buildings="city.buildings" :grid="game.config?.grid" compact :ruined="ruined" :label="`Город ${b.defender}`" />
          </div>
          <div class="hp">
            <div v-for="h in hp" :key="h.key" class="hp-row" v-show="h.max > 0">
              <span class="small">{{ h.name }}</span>
              <div class="bar"><span :class="{ low: h.now / h.max < 0.35 }" :style="{ width: (h.now / Math.max(1, h.max)) * 100 + '%' }" /></div>
              <span class="num xs">{{ num(h.now) }}/{{ num(h.max) }}</span>
            </div>
          </div>
          <div class="army">
            <div v-for="(n, k) in b.army" :key="k" class="unit">
              <TroopIcon :type="String(k)" :size="44" :faded="!alive[k]" />
              <span class="num"><strong>{{ alive[k] ?? n }}</strong>/{{ n }}</span>
              <span class="xs muted">{{ game.troop(String(k))?.name }}</span>
            </div>
          </div>
        </section>

        <aside class="stack">
          <section class="card panel result" :class="{ win: won, lose: !won }" v-if="done">
            <p class="eyebrow">{{ iAttacked ? 'Результат атаки' : 'Результат защиты' }}</p>
            <h2>{{ won ? (iAttacked ? 'Победа' : 'Город взят') : (iAttacked ? 'Атака отбита' : 'Оборона выстояла') }}</h2>
            <div class="stars" :aria-label="`Звёзд: ${b.stars} из 3`">
              <Icon v-for="i in 3" :key="i" name="star" :size="30" :class="{ lit: i <= b.stars }" />
            </div>
            <p class="big num">{{ b.destruction }}% разрушений</p>
            <dl class="rew small">
              <template v-if="iAttacked">
                <div><dt>Золото</dt><dd class="g">+{{ b.gold }}</dd></div>
                <div><dt>Опыт</dt><dd class="x">+{{ b.xp }}</dd></div>
                <div><dt>Трофеи</dt><dd>{{ b.attacker_trophies > 0 ? '+' : '' }}{{ b.attacker_trophies }}</dd></div>
              </template>
              <div v-else><dt>Трофеи</dt><dd>{{ b.defender_trophies > 0 ? '+' : '' }}{{ b.defender_trophies }}</dd></div>
              <div><dt>Потери</dt><dd>{{ Object.entries(b.losses).filter(([, n]) => n > 0).map(([k, n]) => `${game.troop(k)?.name} × ${n}`).join(', ') || 'нет' }}</dd></div>
            </dl>
            <p class="xs muted">Награду создаёт система: у защитника не отнимаются ни ресурсы, ни здания. Бои не влияют на рабочий рейтинг.</p>
          </section>
          <section class="card panel stack">
            <h2>Журнал боя</h2>
            <ol class="log">
              <li class="small">В бой вступили: {{ Object.entries(b.army).map(([k, n]) => `${game.troop(k)?.name} × ${n}`).join(', ') }}. Оборона: {{ b.defense.towers }} башен, {{ b.defense.walls }} стен.</li>
              <li v-for="r in b.rounds.slice(0, step)" :key="r.n" class="small">
                <strong>Раунд {{ r.n }}.</strong> {{ roundText(r) }}
                <span v-for="e in r.events" :key="e" class="badge warning">{{ eventText[e] }}</span>
              </li>
            </ol>
            <p class="xs muted">Случайный фактор ограничен ±{{ Math.round((game.config?.settings.random_spread || 0.1) * 100) }}% на каждый удар; зерно боя {{ b.seed }} сохранено, результат воспроизводим.</p>
          </section>
        </aside>
      </div>
    </template>
  </div>
</template>

<style scoped>
.back { display: inline-flex; align-items: center; gap: 4px; margin-bottom: var(--space-3); text-decoration: none; font-weight: 600; }
.layout { display: grid; grid-template-columns: minmax(0, 1fr) 380px; gap: var(--space-5); align-items: start; }
.arena { padding: var(--space-4); display: flex; flex-direction: column; gap: var(--space-3); background: linear-gradient(180deg, var(--art-sky), var(--color-surface) 60%); }
.round-label { align-self: center; font-weight: 800; letter-spacing: 0.06em; text-transform: uppercase; font-size: var(--text-sm); background: var(--color-surface); padding: 4px 14px; border-radius: var(--radius-pill); box-shadow: var(--shadow-sm); }
.scene-box { max-width: 640px; margin: 0 auto; width: 100%; }
.shaking { animation: shake 1.1s ease-in-out infinite; }
@keyframes shake { 10%, 30% { transform: translateX(-2px); } 20%, 40% { transform: translateX(2px); } 50% { transform: none; } }
.hp { display: grid; gap: var(--space-2); }
.hp-row { display: grid; grid-template-columns: 170px 1fr 90px; gap: var(--space-3); align-items: center; }
.hp-row .bar span { background: var(--color-success); }
.hp-row .bar span.low { background: var(--color-danger); }
.army { display: flex; justify-content: center; gap: var(--space-5); flex-wrap: wrap; }
.unit { display: flex; flex-direction: column; align-items: center; gap: 2px; }
.result { text-align: center; display: flex; flex-direction: column; align-items: center; gap: var(--space-2); border-width: 2px; }
.result.win { border-color: var(--color-gold); }
.result h2 { font-family: var(--font-display); font-size: 2.25rem; }
.stars { display: flex; gap: 4px; color: var(--color-border-strong); }
.stars .lit { color: var(--art-gold); fill: var(--art-gold); }
.big { font-size: var(--text-lg); font-weight: 750; }
.rew { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-2); width: 100%; margin: 0; text-align: left; }
.rew div { background: var(--color-surface-2); padding: var(--space-2); border-radius: var(--radius-sm); }
.rew dt { color: var(--color-text-muted); font-size: var(--text-xs); }
.rew dd { margin: 0; font-weight: 750; }
.g { color: var(--color-gold); } .x { color: var(--color-xp); }
.log { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: var(--space-2); }
@media (max-width: 1000px) { .layout { grid-template-columns: 1fr; } .hp-row { grid-template-columns: 110px 1fr 80px; } }
</style>
