<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import Icon from '../ui/Icon.vue'
import TroopIcon from '../components/TroopIcon.vue'
import CityScene from '../components/CityScene.vue'
import GameOff from '../components/GameOff.vue'
import { api, newIdempotencyKey } from '../api'
import { ago, dateTime, num } from '../lib/format'
import { useGame } from '../stores/game'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import type { Battle, City, Opponent } from '../types'

interface OppList { items: Opponent[]; scope: string; attack_energy: number; shield_minutes: number; rematch_minutes: number }
interface BattleRow { id: string; role: 'attack' | 'defense'; attacker: string; defender: string; destruction: number; stars: number; gold: number; attacker_trophies: number; defender_trophies: number; created_at: string }
const game = useGame()
const session = useSession()
const toasts = useToasts()
const router = useRouter()
const list = ref<OppList | null>(null)
const troops = ref<Record<string, number>>({})
const squad = ref<Record<string, number>>({})
const selected = ref<Opponent | null>(null)
const preview = ref<City | null>(null)
const history = ref<BattleRow[]>([])
const busy = ref(false)
let attackKey = ''

async function load() {
  try {
    const [o, a, b] = await Promise.all([
      api.get<OppList>('/opponents'),
      api.get<{ troops: Record<string, number> }>('/army'),
      api.get<{ items: BattleRow[] }>('/battles'),
    ])
    list.value = o
    troops.value = a.troops
    squad.value = Object.fromEntries(Object.entries(a.troops).map(([k, v]) => [k, v]))
    history.value = b.items
    if (!selected.value) {
      const first = o.items.find(x => x.available)
      if (first) pick(first)
    }
  } catch (e) { toasts.error(e) }
}
onMounted(async () => { await game.loadConfig(); load() })

async function pick(o: Opponent) {
  selected.value = o
  preview.value = null
  attackKey = newIdempotencyKey()
  try { preview.value = await api.get<City>(`/opponents/${o.user_id}/city`) } catch (e) { toasts.error(e) }
}
const housing = computed(() => Object.entries(squad.value).reduce((n, [k, v]) => n + (game.troop(k)?.housing || 1) * (v || 0), 0))
const total = computed(() => Object.values(squad.value).reduce((n, v) => n + (v || 0), 0))
const blocker = computed(() => {
  if (!selected.value) return 'Выберите соперника.'
  if (!selected.value.available) return selected.value.reason
  if (!game.player) return ''
  if (game.player.debt) return 'Сначала погасите долг.'
  if (game.player.energy < (list.value?.attack_energy || 0)) return `Для атаки нужно ${list.value?.attack_energy} энергии.`
  if (total.value === 0) return 'Выберите войска.'
  return ''
})
async function attack() {
  if (!selected.value) return
  busy.value = true
  try {
    // One key per prepared attack: a double click or a retried request cannot start two battles.
    const b = await api.post<Battle>('/battles', { defender_id: selected.value.user_id, army: squad.value }, attackKey)
    game.refresh()
    router.push(`/battles/${b.id}`)
  } catch (e) { toasts.error(e); attackKey = newIdempotencyKey() } finally { busy.value = false }
}
</script>

<template>
  <GameOff v-if="!session.gameOn" />
  <div v-else class="page">
    <div class="page-head">
      <div>
        <p class="eyebrow">Асинхронные бои · {{ list?.scope === 'all' ? 'все команды' : 'своя команда' }}</p>
        <h1>Выбор соперника</h1>
        <p class="muted">Соперник не обязан быть в сети. Защитник не теряет ни зданий, ни ресурсов — награду создаёт система.</p>
      </div>
    </div>
    <div class="layout">
      <section class="stack">
        <h2 class="sr-only">Соперники</h2>
        <p v-if="list && !list.items.length" class="card panel muted">В вашей области боёв нет других игроков.</p>
        <button v-for="o in list?.items || []" :key="o.user_id" class="opp card" :class="{ on: selected?.user_id === o.user_id, off: !o.available }" :aria-pressed="selected?.user_id === o.user_id" @click="pick(o)">
          <span class="th" :title="`Ратуша ${o.townhall}`">{{ o.townhall }}</span>
          <span class="grow">
            <strong>{{ o.display_name }}</strong>
            <span class="xs muted">{{ o.team }} · башни {{ o.defense.towers }}, стены {{ o.defense.walls }}</span>
            <span v-if="!o.available" class="xs reason">{{ o.reason }}</span>
          </span>
          <span class="trophies"><Icon name="trophy" :size="14" />{{ o.trophies }}</span>
        </button>
        <p class="xs muted">Подбор по уровню ратуши ±1. После атаки у защитника щит на {{ list?.shield_minutes }} мин, повторная атака того же игрока — через {{ list?.rematch_minutes }} мин.</p>
      </section>

      <section v-if="selected" class="card panel stack target">
        <div class="row wrap"><h2 class="grow">{{ selected.display_name }}</h2><span class="badge">Ратуша {{ selected.townhall }}</span></div>
        <div class="preview">
          <CityScene v-if="preview" :buildings="preview.buildings" :grid="game.config?.grid" compact :show-levels="true" :label="`Город соперника ${selected.display_name}`" />
          <p v-else class="muted small">Загрузка города…</p>
        </div>
        <dl class="defense small">
          <div><dt>Стены</dt><dd>{{ num(selected.defense.wall_hp) }}</dd></div>
          <div><dt>Башни</dt><dd>{{ num(selected.defense.tower_hp) }}</dd></div>
          <div><dt>Урон башен</dt><dd>{{ selected.defense.tower_dps }}/залп</dd></div>
          <div><dt>Ратуша</dt><dd>{{ num(selected.defense.townhall_hp) }}</dd></div>
        </dl>

        <h3>Отряд</h3>
        <div v-for="t in game.config?.troops || []" :key="t.type" class="squad-row">
          <TroopIcon :type="t.type" :size="40" :faded="!troops[t.type]" />
          <label class="grow small" :for="`sq-${t.type}`"><strong>{{ t.name }}</strong><span class="muted"> · готово {{ troops[t.type] || 0 }}</span></label>
          <input :id="`sq-${t.type}`" v-model.number="squad[t.type]" type="range" min="0" :max="troops[t.type] || 0" :disabled="!troops[t.type]" />
          <span class="num val">{{ squad[t.type] || 0 }}</span>
        </div>
        <p class="small muted">В отряде {{ total }} бойцов ({{ housing }} мест). Атака стоит {{ list?.attack_energy }} энергии, погибшие бойцы не возвращаются.</p>
        <button class="btn primary big" :disabled="!!blocker || busy" :aria-busy="busy" @click="attack"><Icon name="swords" />Атаковать</button>
        <p v-if="blocker" class="small reason">{{ blocker }}</p>
      </section>
    </div>

    <section class="card panel stack" style="margin-top: 20px">
      <h2>Последние бои</h2>
      <p v-if="!history.length" class="muted small">Боёв ещё не было.</p>
      <div class="table-scroll" v-else>
        <table class="table">
          <thead><tr><th>Бой</th><th>Разрушения</th><th>Звёзды</th><th>Трофеи</th><th>Когда</th><th /></tr></thead>
          <tbody>
            <tr v-for="b in history" :key="b.id">
              <td>{{ b.role === 'attack' ? `Атака · ${b.defender}` : `Защита · ${b.attacker}` }}</td>
              <td class="num">{{ b.destruction }}%</td>
              <td>{{ b.stars }}</td>
              <td class="num">{{ b.role === 'attack' ? b.attacker_trophies : b.defender_trophies }}</td>
              <td :title="dateTime(b.created_at)">{{ ago(b.created_at) }}</td>
              <td><RouterLink :to="`/battles/${b.id}`">Журнал</RouterLink></td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<style scoped>
.layout { display: grid; grid-template-columns: minmax(0, 380px) minmax(0, 1fr); gap: var(--space-5); align-items: start; }
.opp { display: flex; align-items: center; gap: var(--space-3); width: 100%; text-align: left; padding: var(--space-3); cursor: pointer; border-width: 1px; }
.opp:hover { border-color: var(--color-action-primary); }
.opp.on { border-color: var(--color-action-primary); background: var(--color-selected); }
.opp.off { opacity: 0.8; }
.opp .grow { display: flex; flex-direction: column; }
.th { display: grid; place-items: center; width: 36px; height: 36px; border-radius: var(--radius-sm); background: var(--art-roof-hall); color: #fff; font-weight: 800; flex: none; }
.trophies { display: inline-flex; align-items: center; gap: 4px; font-weight: 750; color: var(--color-trophy); }
.reason { color: var(--color-warning); }
.preview { background: linear-gradient(180deg, var(--art-sky), transparent); border-radius: var(--radius-md); padding: var(--space-2); }
.defense { display: grid; grid-template-columns: repeat(4, 1fr); gap: var(--space-2); margin: 0; }
.defense div { background: var(--color-surface-2); padding: var(--space-2); border-radius: var(--radius-sm); }
.defense dt { color: var(--color-text-muted); font-size: var(--text-xs); }
.defense dd { margin: 0; font-weight: 750; }
.squad-row { display: flex; align-items: center; gap: var(--space-3); }
.squad-row input[type="range"] { width: 40%; accent-color: var(--color-action-primary); }
.val { width: 28px; text-align: right; font-weight: 800; }
.big { min-height: 48px; font-size: var(--text-md); }
@media (max-width: 900px) { .layout { grid-template-columns: 1fr; } .defense { grid-template-columns: 1fr 1fr; } }
</style>
