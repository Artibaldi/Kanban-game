<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import Icon from '../ui/Icon.vue'
import GameOff from '../components/GameOff.vue'
import FortressScene from '../components/FortressScene.vue'
import { api, type ApiError } from '../api'
import { crownsText, daysLeftText, enemyName, enemyRows, waveProgress } from '../lib/fortress'
import { dateTime } from '../lib/format'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import type { Enemy, FortressState, WaveSummary } from '../types'

const session = useSession()
const toasts = useToasts()
const router = useRouter()
const state = ref<FortressState | null>(null)
const waves = ref<WaveSummary[]>([])
const noTeam = ref(false)
const selected = ref<Enemy | null>(null)

onMounted(async () => {
  if (!session.gameOn) return
  try {
    const [f, w] = await Promise.all([api.get<FortressState>('/fortress'), api.get<{ items: WaveSummary[] }>('/waves')])
    state.value = f
    waves.value = w.items
  } catch (e) {
    if ((e as ApiError).code === 'not_a_player') noTeam.value = true
    else toasts.error(e)
  }
})

const rows = computed(() => enemyRows(state.value?.enemies || []))
const progress = computed(() => (state.value ? Math.round(waveProgress(state.value.wave) * 100) : 0))
</script>

<template>
  <GameOff v-if="!session.gameOn" />
  <div v-else class="page fortress-page">
    <div class="page-head">
      <div>
        <p class="eyebrow">{{ session.me?.team_name }}</p>
        <h1>Крепость команды</h1>
        <p class="muted">Задачи, взятые в план, — враги. Подтверждённая задача — побеждённый враг. Просроченные бьют по стенам.</p>
      </div>
      <RouterLink to="/fortress/duels" class="btn"><Icon name="swords" :size="16" />Дружеские бои</RouterLink>
    </div>

    <p v-if="noTeam" class="card panel muted">Крепость есть у команд. Попросите администратора добавить вас в команду.</p>

    <template v-else-if="state">
      <div class="stats" role="list">
        <div class="stat" role="listitem" :class="{ danger: state.fortress.siege }">
          <span class="xs muted">Прочность</span>
          <strong class="num">{{ state.fortress.hp }}%</strong>
          <span v-if="state.fortress.siege" class="badge danger">Осада</span>
        </div>
        <div class="stat wave" role="listitem">
          <span class="xs muted">Волна</span>
          <strong class="num">{{ state.wave.closed }} из {{ state.wave.planned }} баллов</strong>
          <span class="bar" role="progressbar" :aria-valuenow="progress" aria-valuemin="0" aria-valuemax="100" aria-label="Закрыто от плана волны"><span :style="{ width: progress + '%' }" /></span>
        </div>
        <div class="stat" role="listitem">
          <span class="xs muted">До конца волны</span>
          <strong>{{ daysLeftText(state.wave.days_left) }}</strong>
          <span class="xs muted">сейчас: {{ crownsText(state.wave.crowns_now) }}</span>
        </div>
        <div class="stat" role="listitem">
          <span class="xs muted">Прошлая волна</span>
          <strong>{{ state.last_wave ? crownsText(state.last_wave.crowns) : '—' }}</strong>
        </div>
      </div>

      <div class="layout">
        <section class="card scene-card" aria-label="Поле боя">
          <FortressScene :fortress="state.fortress" :rows="rows" :fog="state.fog" :selected-id="selected?.task_id" @select="selected = $event" />
          <p v-if="!state.enemies.length" class="calm muted">Тишина на границе. Возьмите задачи из бэклога в план.</p>
          <p class="legend xs muted">Ряды — приоритет: P1 ближе всего к стенам. Красное кольцо — просрочено, жёлтое — срок через 2 дня или меньше.</p>
        </section>

        <aside class="side">
          <section v-if="selected" class="card panel stack" aria-live="polite">
            <div class="row"><h2 class="grow">{{ enemyName(selected.kind) }}</h2><span class="badge">{{ selected.priority }}</span></div>
            <p><strong>{{ selected.number }}</strong> {{ selected.title }}</p>
            <p class="small muted">{{ selected.points }} балл.<template v-if="selected.due_date"> · срок {{ selected.due_date }}</template></p>
            <p v-if="selected.overdue" class="small danger-text">Просрочена: каждый рабочий день бьёт по стенам.</p>
            <button class="btn primary" @click="router.push(`/board?task=${selected.task_id}`)"><Icon name="board" :size="16" />Открыть на доске</button>
          </section>

          <section class="card panel stack">
            <h2>Волны</h2>
            <ol v-if="waves.length" class="waves">
              <li v-for="w in waves" :key="w.id">
                <span class="crowns" :aria-label="crownsText(w.crowns)"><Icon v-for="i in 3" :key="i" name="star" :size="14" :class="{ on: i <= w.crowns }" /></span>
                <span class="grow small">{{ w.summary }}<span class="xs muted"> · {{ dateTime(w.ends_at) }}</span></span>
              </li>
            </ol>
            <p v-else class="small muted">Первая волна завершится в конце недели.</p>
          </section>
        </aside>
      </div>
    </template>
  </div>
</template>

<style scoped>
.stats { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: var(--space-3); margin-bottom: var(--space-4); }
.stat { display: flex; flex-direction: column; gap: 4px; padding: var(--space-3) var(--space-4); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-md); }
.stat strong { font-size: var(--text-lg); }
.stat.danger { border-color: var(--color-danger); }
.layout { display: grid; grid-template-columns: minmax(0, 1fr) 340px; gap: var(--space-5); align-items: start; }
.scene-card { padding: var(--space-3); overflow: hidden; }
.calm { text-align: center; margin: var(--space-2) 0 0; }
.legend { text-align: center; margin: var(--space-2) 0 0; }
.side { display: flex; flex-direction: column; gap: var(--space-4); }
.danger-text { color: var(--color-danger); }
.waves { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: var(--space-3); }
.waves li { display: flex; gap: var(--space-2); align-items: flex-start; }
.crowns { display: inline-flex; flex: none; color: var(--color-border-strong); padding-top: 2px; }
.crowns .on { color: var(--color-gold); }
@media (max-width: 1024px) { .layout { grid-template-columns: 1fr; } }
@media (max-width: 720px) { .stats { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
