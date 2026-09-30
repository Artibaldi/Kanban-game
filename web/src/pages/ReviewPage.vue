<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Icon from '../ui/Icon.vue'
import TaskDrawer from '../components/TaskDrawer.vue'
import ReasonDialog from '../components/ReasonDialog.vue'
import { api } from '../api'
import { date, isOverdue } from '../lib/format'
import { pointsScale, rewardPreview } from '../lib/board'
import { moveTask } from '../lib/taskActions'
import { useGame } from '../stores/game'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import type { Task, TeamUser } from '../types'

const session = useSession()
const game = useGame()
const toasts = useToasts()
const review = ref<Task[]>([])
const estimates = ref<Task[]>([])
const users = ref<TeamUser[]>([])
const open = ref<string | null>(null)
const busy = ref('')
const returning = ref<Task | null>(null)

async function load() {
  try {
    const r = await api.get<{ review: Task[]; estimates: Task[] }>('/review')
    review.value = r.review
    estimates.value = r.estimates
  } catch (e) { toasts.error(e) }
}
onMounted(async () => {
  load()
  game.loadConfig().catch(() => {})
  users.value = (await api.get<{ items: TeamUser[] }>('/users')).items
})
async function approve(t: Task) {
  busy.value = t.id
  if (await moveTask(t, 'done')) await load()
  busy.value = ''
}
async function sendBack(comment: string) {
  const t = returning.value
  returning.value = null
  if (t && await moveTask(t, 'in_progress', { comment })) load()
}
async function agree(t: Task, points: number) {
  busy.value = t.id
  try {
    await api.post(`/tasks/${t.id}/estimate`, { points })
    toasts.push({ kind: 'success', title: `${t.number}: сложность ${points} согласована` }, 2500)
    load()
  } catch (e) { toasts.error(e) } finally { busy.value = '' }
}
const preview = (t: Task) => (game.config ? rewardPreview(t.points, t.priority, game.config.settings, !!t.due_date && !isOverdue(t.due_date)) : null)
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <p class="eyebrow">Панель руководителя</p>
        <h1>Проверка задач</h1>
        <p class="muted">Подтверждение начисляет исполнителю награду и баллы рейтинга ровно один раз. Свою задачу подтвердить нельзя.</p>
      </div>
    </div>

    <section class="stack">
      <h2>На проверке <span class="badge">{{ review.length }}</span></h2>
      <p v-if="!review.length" class="card panel muted">Очередь пуста.</p>
      <article v-for="t in review" :key="t.id" class="card item">
        <div class="grow stack" style="gap: 4px">
          <div class="row small wrap"><span class="mono">{{ t.number }}</span><span class="badge">{{ t.priority }}</span><span class="badge info">{{ t.points }} балл.</span>
            <span v-if="t.due_date" :class="{ late: isOverdue(t.due_date) }">срок {{ date(t.due_date) }}</span></div>
          <button class="linkish" @click="open = t.id">{{ t.title }}</button>
          <p class="small muted">Исполнитель: <strong>{{ t.assignee_name }}</strong>
            <span v-if="session.gameOn && preview(t)" class="game-only"> · награда ≈ {{ preview(t)!.gold }} золота, {{ preview(t)!.energy }} энергии, {{ preview(t)!.xp }} опыта</span></p>
        </div>
        <div class="actions">
          <template v-if="t.assignee_id === session.me?.id">
            <span class="small muted self"><Icon name="lock" :size="14" />Ваша задача — подтвердит другой руководитель</span>
          </template>
          <template v-else>
            <button class="btn" :disabled="!!busy" @click="returning = t"><Icon name="undo" :size="14" />Вернуть</button>
            <button class="btn primary" :disabled="!!busy" :aria-busy="busy === t.id" @click="approve(t)"><Icon name="check" :size="14" />Подтвердить</button>
          </template>
        </div>
      </article>
    </section>

    <section class="stack" style="margin-top: 28px">
      <h2>Оценки на согласование <span class="badge">{{ estimates.length }}</span></h2>
      <p class="small muted">Сложность согласуется до начала работы. Предложенную оценку можно принять или заменить — изменение увидит команда в истории задачи.</p>
      <p v-if="!estimates.length" class="card panel muted">Все оценки согласованы.</p>
      <article v-for="t in estimates" :key="t.id" class="card item">
        <div class="grow stack" style="gap: 4px">
          <div class="row small"><span class="mono">{{ t.number }}</span><span class="muted">предложил(а) {{ t.created_by }}</span></div>
          <button class="linkish" @click="open = t.id">{{ t.title }}</button>
        </div>
        <div class="actions seg" role="group" :aria-label="`Сложность для ${t.number}`">
          <button v-for="p in pointsScale" :key="p" class="pt" :class="{ proposed: p === t.points }" :disabled="!!busy" :title="p === t.points ? 'Согласовать предложенную оценку' : `Согласовать ${p}`" @click="agree(t, p)">{{ p }}</button>
        </div>
      </article>
    </section>
    <TaskDrawer :task-id="open" :users="users" @close="open = null" @changed="load" />
    <ReasonDialog :open="!!returning" title="Вернуть на доработку" text="Задача вернётся в «В работе». Комментарий увидит исполнитель." action="Вернуть на доработку" @close="returning = null" @confirm="sendBack" />
  </div>
</template>

<style scoped>
.item { display: flex; align-items: center; gap: var(--space-4); padding: var(--space-4); flex-wrap: wrap; }
.actions { display: flex; gap: var(--space-2); flex-wrap: wrap; }
.mono { font-family: var(--font-mono); color: var(--color-text-muted); }
.linkish { background: none; border: 0; padding: 0; text-align: left; font-weight: 650; font-size: var(--text-md); cursor: pointer; color: var(--color-text); }
.linkish:hover { color: var(--color-action-primary); text-decoration: underline; }
.late { color: var(--color-danger); font-weight: 700; }
.self { display: inline-flex; align-items: center; gap: 6px; }
.pt { width: 42px; height: 38px; border-radius: var(--radius-sm); border: 1px solid var(--color-border-strong); background: var(--color-surface); font-weight: 800; cursor: pointer; }
.pt:hover:not(:disabled) { background: var(--color-selected); border-color: var(--color-action-primary); }
.pt.proposed { border: 2px solid var(--color-warning); }
h2 .badge { vertical-align: middle; }
</style>
