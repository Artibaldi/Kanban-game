<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import Icon from '../ui/Icon.vue'
import BurndownChart from '../components/BurndownChart.vue'
import WeeklyBars from '../components/WeeklyBars.vue'
import { api } from '../api'
import { crownsText } from '../lib/fortress'
import { date } from '../lib/format'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import type { MyProgress, TeamProgress } from '../types'

interface Burn { start_remaining: number; days: { date: string; added: number; reestimated: number; reopened: number; completed: number; remaining: number }[] }

const session = useSession()
const toasts = useToasts()
const team = ref('')
const teams = ref<{ id: string; name: string }[]>([])
const mine = ref<MyProgress | null>(null)
const teamProgress = ref<TeamProgress | null>(null)
const burn = ref<Burn | null>(null)

async function loadTeam() {
  try {
    const q = team.value ? `?team=${team.value}` : ''
    const [t, b] = await Promise.all([api.get<TeamProgress>(`/progress/team${q}`), api.get<Burn>(`/burndown${q}`)])
    teamProgress.value = t
    burn.value = b
  } catch (e) { toasts.error(e) }
}
onMounted(async () => {
  try {
    if (session.me?.team_id) mine.value = await api.get<MyProgress>('/progress/me')
    else {
      teams.value = (await api.get<{ items: { id: string; name: string }[] }>('/teams')).items
      if (teams.value.length) team.value = teams.value[0].id
    }
  } catch (e) { toasts.error(e) }
  loadTeam()
})
watch(team, loadTeam)
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <p class="eyebrow">{{ teams.find(t => t.id === team)?.name || session.me?.team_name }}</p>
        <h1>Прогресс<span class="sub">мой и команды</span></h1>
      </div>
      <select v-if="session.isAdmin && teams.length" v-model="team" class="select narrow" aria-label="Команда">
        <option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</option>
      </select>
    </div>
    <p class="notice info row explain"><Icon name="info" /><span>Здесь только ваш личный прогресс и общий прогресс команды. Сравнения между коллегами нет. Игровые данные не используются для оценки работы.</span></p>

    <div class="grid">
      <section v-if="mine" class="card panel stack">
        <h2>Мой прогресс</h2>
        <div class="row wrap figures">
          <div><span class="big num">{{ mine.this_week }}</span><span class="small muted"> баллов на этой неделе</span></div>
          <div class="small muted">Лучшая неделя: <strong>{{ mine.best_week }}</strong> · всего <strong>{{ mine.total }}</strong></div>
        </div>
        <WeeklyBars :weeks="mine.weeks" :values="mine.weekly" label="Мои баллы по неделям" />
        <p class="xs muted">{{ mine.explanation }}</p>
      </section>

      <section v-if="teamProgress" class="card panel stack">
        <h2>Команда</h2>
        <div class="row wrap figures">
          <div><span class="big num">{{ teamProgress.team_weekly[teamProgress.team_weekly.length - 1] }}</span><span class="small muted"> баллов командой на этой неделе</span></div>
          <div class="small" :class="teamProgress.overdue_now ? 'danger-text' : 'muted'">Просрочено сейчас: <strong>{{ teamProgress.overdue_now }}</strong></div>
        </div>
        <WeeklyBars :weeks="teamProgress.weeks" :values="teamProgress.team_weekly" label="Выполнено командой по неделям" />
        <h3>Волны</h3>
        <ol v-if="teamProgress.waves.length" class="waves small">
          <li v-for="w in teamProgress.waves" :key="w.id"><strong>{{ crownsText(w.crowns) }}</strong> · {{ w.summary }} <span class="xs muted">неделя до {{ date(w.ends_at) }}</span></li>
        </ol>
        <p v-else class="small muted">Закрытых волн пока нет.</p>
      </section>
    </div>

    <section class="card panel stack" style="margin-top: 20px">
      <div class="row wrap"><h2 class="grow">Объём бэклога, баллы</h2><span class="small muted">30 дней · начало периода: {{ burn?.start_remaining }} баллов</span></div>
      <p class="small muted">Линия — сколько баллов осталось сделать. Столбики вверх — новые задачи, повышение оценок и возвраты; вниз — подтверждённая работа. Так видно, растёт объём работы или команда его сжигает.</p>
      <BurndownChart v-if="burn" :days="burn.days" />
    </section>
  </div>
</template>

<style scoped>
.explain { margin-bottom: var(--space-4); align-items: flex-start; }
.narrow { width: auto; min-width: 200px; }
.grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--space-4); align-items: start; }
.figures { justify-content: space-between; align-items: baseline; }
.big { font-size: var(--text-display); font-weight: 800; line-height: 1; color: var(--color-action-primary); }
.danger-text { color: var(--color-danger); }
.waves { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: var(--space-2); }
@media (max-width: 900px) { .grid { grid-template-columns: 1fr; } .narrow { flex: 1; min-width: 0; } }
</style>
