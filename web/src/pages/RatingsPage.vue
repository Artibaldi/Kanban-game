<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import Icon from '../ui/Icon.vue'
import BurndownChart from '../components/BurndownChart.vue'
import WeeklyBars from '../components/WeeklyBars.vue'
import Sparkline from '../components/Sparkline.vue'
import { api } from '../api'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import type { Project } from '../types'

interface WorkRow { user_id: string; display_name: string; role: string; points: number; tasks: number; week: number; month: number; all: number; weekly: number[] }
interface Work { period: string; items: WorkRow[]; weeks: string[]; team_weekly: number[]; explanation: string }
interface BattleRow { user_id: string; display_name: string; team: string; trophies: number; wins: number; attacks: number; defenses_held: number; townhall: number }
interface Burn { start_remaining: number; days: { date: string; added: number; reestimated: number; reopened: number; completed: number; remaining: number }[] }

const session = useSession()
const toasts = useToasts()
const tab = ref<'work' | 'battle'>('work')
const period = ref<'week' | 'month' | 'all'>('week')
const project = ref('')
const team = ref('')
const teams = ref<{ id: string; name: string }[]>([])
const projects = ref<Project[]>([])
const work = ref<Work | null>(null)
const battles = ref<BattleRow[]>([])
const burn = ref<Burn | null>(null)

const q = () => {
  const p = new URLSearchParams()
  if (project.value) p.set('project', project.value)
  if (team.value) p.set('team', team.value)
  return p
}
async function loadWork() {
  try {
    const p = q(); p.set('period', period.value)
    const [w, b] = await Promise.all([api.get<Work>(`/ratings/work?${p}`), api.get<Burn>(`/burndown?${q()}`)])
    work.value = w
    burn.value = b
  } catch (e) { toasts.error(e) }
}
onMounted(async () => {
  try {
    const [t, p] = await Promise.all([api.get<{ items: { id: string; name: string }[] }>('/teams'), api.get<{ items: Project[] }>('/projects')])
    teams.value = t.items
    projects.value = p.items
    if (!session.me?.team_id && t.items.length) team.value = t.items[0].id
  } catch (e) { toasts.error(e) }
  loadWork()
  if (session.plays) api.get<{ items: BattleRow[] }>('/ratings/battle').then(r => (battles.value = r.items)).catch(() => {})
})
watch([period, project, team], loadWork)
const visibleProjects = computed(() => projects.value.filter(p => !team.value || p.team_id === team.value))
const periodName = { week: 'эту неделю', month: 'этот месяц', all: 'всё время' }
const top = computed(() => work.value?.items[0])
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <p class="eyebrow">{{ teams.find(t => t.id === (team || session.me?.team_id))?.name || session.me?.team_name }}</p>
        <h1>Рейтинги</h1>
      </div>
      <div class="tabs" role="tablist" aria-label="Вид рейтинга">
        <button role="tab" :aria-selected="tab === 'work'" @click="tab = 'work'">По задачам</button>
        <button v-if="session.plays" role="tab" :aria-selected="tab === 'battle'" class="game-only" @click="tab = 'battle'">Боевой</button>
      </div>
    </div>

    <template v-if="tab === 'work'">
      <div class="explain notice info row"><Icon name="info" /><span>{{ work?.explanation }} Сравнение идёт внутри команды: у неё общая шкала сложности 1, 2, 3, 5, 8.</span></div>
      <div class="filters row wrap">
        <div class="tabs" role="tablist" aria-label="Период">
          <button role="tab" :aria-selected="period === 'week'" @click="period = 'week'">Неделя</button>
          <button role="tab" :aria-selected="period === 'month'" @click="period = 'month'">Месяц</button>
          <button role="tab" :aria-selected="period === 'all'" @click="period = 'all'">Всё время</button>
        </div>
        <select v-if="session.isAdmin" v-model="team" class="select narrow" aria-label="Команда"><option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</option></select>
        <select v-model="project" class="select narrow" aria-label="Проект">
          <option value="">Все проекты команды</option>
          <option v-for="p in visibleProjects" :key="p.id" :value="p.id">{{ p.key }} — {{ p.name }}</option>
        </select>
      </div>

      <div class="grid">
        <section class="card panel lead-card" v-if="top">
          <p class="eyebrow">Лидер за {{ periodName[period] }}</p>
          <p class="leader">{{ top.points > 0 ? top.display_name : 'Пока никого' }}</p>
          <p class="big num">{{ top.points }} <span class="muted small">баллов</span></p>
          <WeeklyBars v-if="work" :weeks="work.weeks" :values="work.team_weekly" label="Выполнено командой по неделям" />
          <p class="xs muted">Баллы команды по неделям, последние 8 недель.</p>
        </section>
        <section class="card panel table-card">
          <div class="table-scroll">
            <table class="table">
              <thead><tr><th>#</th><th>Участник</th><th class="r">Баллы</th><th class="r">Задач</th><th class="r">Неделя</th><th class="r">Месяц</th><th>Динамика</th></tr></thead>
              <tbody>
                <tr v-for="(r, i) in work?.items || []" :key="r.user_id" :class="{ me: r.user_id === session.me?.id }">
                  <td class="num">{{ i + 1 }}</td>
                  <td><strong>{{ r.display_name }}</strong><span v-if="r.role === 'lead'" class="badge" style="margin-left: 6px">руководитель</span></td>
                  <td class="r num"><strong>{{ r.points }}</strong></td>
                  <td class="r num">{{ r.tasks }}</td>
                  <td class="r num">{{ r.week }}</td>
                  <td class="r num">{{ r.month }}</td>
                  <td><Sparkline :values="r.weekly" /></td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>

      <section class="card panel stack" style="margin-top: 20px">
        <div class="row wrap"><h2 class="grow">Объём бэклога, баллы</h2><span class="small muted">30 дней · начало периода: {{ burn?.start_remaining }} баллов</span></div>
        <p class="small muted">Линия — сколько баллов осталось сделать. Столбики вверх — новые задачи, повышение оценок и возвраты; вниз — подтверждённая работа. Так видно, растёт объём работы или команда его сжигает.</p>
        <BurndownChart v-if="burn" :days="burn.days" />
      </section>
    </template>

    <section v-else class="card panel stack">
      <p class="notice info">Боевой рейтинг — трофеи за бои. Он не связан с рейтингом по задачам: победы не добавляют рабочих баллов, поражения их не отнимают.</p>
      <div class="table-scroll">
        <table class="table">
          <thead><tr><th>#</th><th>Игрок</th><th>Команда</th><th class="r">Трофеи</th><th class="r">Победы / атаки</th><th class="r">Удержанные защиты</th><th class="r">Ратуша</th></tr></thead>
          <tbody>
            <tr v-for="(r, i) in battles" :key="r.user_id" :class="{ me: r.user_id === session.me?.id }">
              <td class="num">{{ i + 1 }}</td><td><strong>{{ r.display_name }}</strong></td><td>{{ r.team }}</td>
              <td class="r num"><strong>{{ r.trophies }}</strong></td><td class="r num">{{ r.wins }} / {{ r.attacks }}</td><td class="r num">{{ r.defenses_held }}</td><td class="r num">{{ r.townhall }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<style scoped>
.explain { margin-bottom: var(--space-4); align-items: flex-start; }
.filters { margin-bottom: var(--space-4); }
.narrow { width: auto; min-width: 200px; }
.grid { display: grid; grid-template-columns: 340px minmax(0, 1fr); gap: var(--space-4); align-items: start; }
.lead-card { display: flex; flex-direction: column; gap: var(--space-2); }
.leader { font-family: var(--font-display); font-size: var(--text-xl); font-weight: 700; }
.big { font-size: var(--text-display); font-weight: 800; line-height: 1; color: var(--color-action-primary); }
.table-card { padding: var(--space-2); }
.r { text-align: right; }
@media (max-width: 900px) { .grid { grid-template-columns: 1fr; } .narrow { flex: 1; min-width: 0; } }
</style>
