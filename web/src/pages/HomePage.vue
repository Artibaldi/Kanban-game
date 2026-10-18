<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import Icon from '../ui/Icon.vue'
import TaskCard from '../components/TaskCard.vue'
import CityScene from '../components/CityScene.vue'
import ResourceBar from '../components/ResourceBar.vue'
import StatRow from '../components/StatRow.vue'
import { api } from '../api'
import { ago, dateTime, plural } from '../lib/format'
import { stateName } from '../lib/board'
import { useGame } from '../stores/game'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import { useWork } from '../stores/work'
import type { City, FeedItem, FortressState, MyProgress, Task } from '../types'

const session = useSession()
const game = useGame()
const toasts = useToasts()
const router = useRouter()
const work = useWork()
const tasks = ref<Task[]>([])
const feed = ref<FeedItem[]>([])
const city = ref<City | null>(null)
const week = ref<MyProgress | null>(null)
const fort = ref<FortressState | null>(null)

onMounted(async () => {
  try {
    const [t, f, w] = await Promise.all([
      api.get<{ items: Task[] }>('/tasks?assignee=me'),
      api.get<{ items: FeedItem[] }>('/events?limit=12'),
      session.me?.team_id ? api.get<MyProgress>('/progress/me') : Promise.resolve(null),
    ])
    tasks.value = t.items
    feed.value = f.items
    week.value = w
    if (session.gameOn) {
      city.value = await api.get<City>('/city'); game.setPlayer(city.value.player); game.loadConfig()
      fort.value = await api.get<FortressState>('/fortress')
    }
  } catch (e) { toasts.error(e) }
})
const active = computed(() => tasks.value.filter(t => t.state !== 'done' && t.state !== 'backlog').sort((a, b) => ['in_progress', 'review', 'ready'].indexOf(a.state) - ['in_progress', 'review', 'ready'].indexOf(b.state)))
const feedIcon = (a: string) => (a.startsWith('wave') ? 'flag' : a.startsWith('battle') ? 'swords' : a.startsWith('building') ? 'hammer' : a.startsWith('army') ? 'shield' : a === 'task.approved' ? 'check' : a === 'task.reopened' ? 'undo' : 'board')
const statItems = computed(() => [
  { icon: 'flame', value: active.value.length, label: 'моих задач в работе' },
  ...(week.value ? [
    { icon: 'trophy', value: week.value.this_week, label: 'очков за неделю' },
    { icon: 'chart', value: week.value.best_week, label: 'лучшая неделя' },
  ] : []),
  ...(session.isLead ? [{ icon: 'eye', value: work.reviewCount, label: 'ждут вашей проверки' }] : []),
])
const hello = computed(() => { const h = new Date().getHours(); return h < 12 ? 'Доброе утро' : h < 18 ? 'Добрый день' : 'Добрый вечер' })
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>{{ hello }},<span class="sub">{{ session.me?.display_name.split(' ')[0] }}</span></h1>
      </div>
      <div class="row">
        <RouterLink v-if="session.isLead" to="/review" class="btn" :class="{ primary: work.reviewCount > 0 }"><Icon name="review" :size="16" />На проверке: {{ work.reviewCount }}</RouterLink>
        <RouterLink to="/board" class="btn"><Icon name="board" :size="16" />Доска</RouterLink>
      </div>
    </div>
    <StatRow :items="statItems" class="home-stats" />

    <div class="grid">
      <section class="card panel stack main">
        <div class="row"><h2 class="grow">Мои задачи</h2><RouterLink to="/board?mine=1" class="small">Все мои</RouterLink></div>
        <p v-if="!active.length" class="muted small">Активных задач нет. Возьмите задачу из колонки «К выполнению» на доске.</p>
        <div class="tasks">
          <TaskCard v-for="t in active.slice(0, 6)" :key="t.id" :task="t" @open="router.push(`/board?task=${t.id}`)" />
        </div>
        <RouterLink v-if="week" to="/progress" class="small weekline">Мой прогресс и график бэклога · всего {{ week.total }} {{ plural(week.total, 'очко', 'очка', 'очков') }}</RouterLink>
      </section>

      <section v-if="session.gameOn" class="card city-card game-only">
        <div class="row panel-head"><h2 class="grow">Город</h2><RouterLink to="/city" class="btn sm primary"><Icon name="castle" :size="14" />Открыть</RouterLink></div>
        <ResourceBar v-if="game.player" :player="game.player" class="res" />
        <RouterLink to="/city" class="scene-link" aria-label="Открыть город">
          <CityScene v-if="city" :buildings="city.buildings" compact :show-levels="false" :now="Date.now()" label="Ваш город" />
        </RouterLink>
        <div class="row wrap chips small">
          <span>Ратуша {{ city?.townhall }}</span><span>Строители {{ game.player?.builders_busy }}/{{ game.player?.builders }}</span>
          <span>Армия {{ game.player?.army_housing }}/{{ game.player?.army_capacity }}</span>
        </div>
        <RouterLink v-if="fort" to="/fortress" class="fort-line small">
          <Icon name="flag" :size="14" />Крепость: {{ fort.fortress.hp }}% · волна {{ fort.wave.closed }}/{{ fort.wave.planned }} баллов
        </RouterLink>
      </section>

      <section class="card panel stack feed">
        <h2>Последние события</h2>
        <ol class="events">
          <li v-for="e in feed" :key="e.id">
            <span class="ic"><Icon :name="feedIcon(e.action)" :size="14" /></span>
            <div class="grow"><p class="small"><strong>{{ e.actor }}</strong> — {{ e.summary }}</p><p class="xs muted" :title="dateTime(e.created_at)">{{ ago(e.created_at) }}</p></div>
          </li>
        </ol>
        <p v-if="!feed.length" class="muted small">Событий пока нет.</p>
      </section>
    </div>
    <p class="sr-only">Колонки: {{ Object.values(stateName).join(', ') }}</p>
  </div>
</template>

<style scoped>
.grid { display: grid; grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr); grid-template-areas: "main city" "main feed"; gap: var(--space-4); align-items: start; }
.main { grid-area: main; }
.city-card { grid-area: city; padding: var(--space-4); display: flex; flex-direction: column; gap: var(--space-3); background: linear-gradient(180deg, var(--art-sky), var(--color-surface) 75%); overflow: hidden; }
.feed { grid-area: feed; }
body.work-mode .grid, .grid:not(:has(.city-card)) { grid-template-areas: "main feed"; }
.tasks { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: var(--space-3); }
.weekline { padding-top: var(--space-3); border-top: 1px solid var(--color-border); }
.home-stats { margin: 0 0 var(--space-5); }
.res { flex-wrap: wrap; }
.fort-line { display: inline-flex; align-items: center; gap: 6px; font-weight: 650; }
.scene-link { display: block; max-width: 460px; margin: 0 auto; width: 100%; }
.chips span:not(.badge) { background: var(--color-surface); padding: 2px 10px; border-radius: var(--radius-pill); border: 1px solid var(--color-border); }
.events { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: var(--space-3); }
.events li { display: flex; gap: var(--space-3); }
.ic { display: grid; place-items: center; width: 26px; height: 26px; border-radius: 50%; background: var(--color-surface-sunken); color: var(--color-text-muted); flex: none; }
@media (max-width: 900px) { .grid, body.work-mode .grid { grid-template-columns: 1fr; grid-template-areas: "main" "city" "feed"; } }
</style>
