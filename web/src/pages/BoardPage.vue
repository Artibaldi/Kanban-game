<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from '../ui/Icon.vue'
import TaskCard from '../components/TaskCard.vue'
import TaskDrawer from '../components/TaskDrawer.vue'
import CreateTaskDialog from '../components/CreateTaskDialog.vue'
import ReasonDialog from '../components/ReasonDialog.vue'
import { api } from '../api'
import { columns, moveBlock, nextStep, stateName } from '../lib/board'
import { moveTask } from '../lib/taskActions'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import type { Project, Task, TaskState, TeamUser } from '../types'

const session = useSession()
const toasts = useToasts()
const route = useRoute()
const router = useRouter()
const tasks = ref<Task[]>([])
const projects = ref<Project[]>([])
const users = ref<TeamUser[]>([])
const loading = ref(true)
const project = ref(typeof route.query.project === 'string' ? route.query.project : '')
const onlyMine = ref(route.query.mine === '1')
const creating = ref(false)
const dragId = ref('')
const dropAt = ref<{ state: TaskState; index: number } | null>(null)
const pendingReopen = ref<{ task: Task; to: TaskState; index: number } | null>(null)
const openId = computed(() => (typeof route.query.task === 'string' ? route.query.task : null))

async function load() {
  try {
    const q = new URLSearchParams()
    if (project.value) q.set('project', project.value)
    if (onlyMine.value) q.set('assignee', 'me')
    tasks.value = (await api.get<{ items: Task[] }>(`/tasks?${q}`)).items
  } catch (e) { toasts.error(e) } finally { loading.value = false }
}
onMounted(async () => {
  try {
    const [p, u] = await Promise.all([api.get<{ items: Project[] }>('/projects'), api.get<{ items: TeamUser[] }>('/users')])
    projects.value = p.items
    users.value = u.items
  } catch (e) { toasts.error(e) }
  load()
})
watch([project, onlyMine], () => {
  router.replace({ query: { ...route.query, project: project.value || undefined, mine: onlyMine.value ? '1' : undefined } })
  load()
})

const viewer = computed(() => ({ id: session.me?.id || '', role: session.me?.role || 'user', team_id: session.me?.team_id || null }))
const byColumn = computed(() => {
  const m: Record<string, Task[]> = {}
  for (const c of columns) m[c.state] = []
  for (const t of tasks.value) m[t.state]?.push(t)
  for (const c of columns) m[c.state].sort((a, b) => a.rank - b.rank)
  return m
})
const pointsIn = (s: string) => byColumn.value[s].reduce((n, t) => n + t.points, 0)

function open(t: Task) { router.replace({ query: { ...route.query, task: t.id } }) }
function close() { const q = { ...route.query }; delete q.task; router.replace({ query: q }) }

// Drag and drop is one path; the step button on each card is the keyboard and touch path.
function onDragStart(e: DragEvent) {
  const el = (e.target as HTMLElement).closest('[data-id]') as HTMLElement | null
  if (!el) return
  dragId.value = el.dataset.id || ''
  e.dataTransfer?.setData('text/plain', dragId.value)
  if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
}
function onDragOver(e: DragEvent, state: TaskState) {
  e.preventDefault()
  const list = (e.currentTarget as HTMLElement).querySelectorAll<HTMLElement>('[data-id]')
  let index = 0
  for (const el of list) {
    if (el.dataset.id === dragId.value) continue
    const r = el.getBoundingClientRect()
    if (e.clientY > r.top + r.height / 2) index++
  }
  dropAt.value = { state, index }
}
function onDragEnd() { dragId.value = ''; dropAt.value = null }
async function onDrop(state: TaskState) {
  const t = tasks.value.find(x => x.id === dragId.value)
  const index = dropAt.value?.index ?? 0
  onDragEnd()
  if (!t) return
  await tryMove(t, state, index)
}

async function tryMove(t: Task, to: TaskState, index?: number) {
  const block = moveBlock(t, to, viewer.value)
  if (block) { toasts.push({ kind: 'error', title: block }); return }
  if (t.state === 'done' && to !== 'done') { pendingReopen.value = { task: t, to, index: index ?? 0 }; return }
  // Optimistic: place the card, then reconcile with the server's answer.
  const prev = tasks.value
  tasks.value = tasks.value.map(x => (x.id === t.id ? { ...x, state: to, rank: index !== undefined ? index * 1024 - 1 : 1e12 } : x))
  const r = await moveTask(t, to, { index })
  if (!r) tasks.value = prev
  load()
}
async function confirmReopen(comment: string) {
  const p = pendingReopen.value
  pendingReopen.value = null
  if (!p) return
  if (await moveTask(p.task, p.to, { index: p.index, comment })) load()
}
function stepFor(t: Task) {
  const to = nextStep(t)
  if (!to) return null
  const block = moveBlock(t, to, viewer.value)
  const hidden = block && to === 'done' && session.me?.role === 'user'
  return hidden ? null : { to, block, label: to === 'done' ? 'Подтвердить' : `В «${stateName[to]}»` }
}
</script>

<template>
  <div class="page board-page">
    <div class="page-head">
      <div>
        <p class="eyebrow">{{ session.me?.team_name || 'Все команды' }}</p>
        <h1>Доска задач</h1>
      </div>
      <div class="row wrap filters">
        <label class="sr-only" for="proj">Проект</label>
        <select id="proj" v-model="project" class="select">
          <option value="">Все проекты</option>
          <option v-for="p in projects" :key="p.id" :value="p.id">{{ p.key }} — {{ p.name }}</option>
        </select>
        <label class="toggle"><input v-model="onlyMine" type="checkbox" />Только мои</label>
        <button class="btn primary" @click="creating = true"><Icon name="plus" :size="16" />Задача</button>
      </div>
    </div>

    <div v-if="loading" class="muted">Загрузка доски…</div>
    <div v-else class="columns">
      <section v-for="c in columns" :key="c.state" class="column" :class="{ over: dropAt?.state === c.state }" :aria-labelledby="`col-${c.state}`"
        @dragover="onDragOver($event, c.state)" @dragleave.self="dropAt = null" @drop.prevent="onDrop(c.state)">
        <header class="col-head">
          <h2 :id="`col-${c.state}`">{{ c.name }}</h2>
          <span class="count" :title="`${byColumn[c.state].length} задач, ${pointsIn(c.state)} баллов`">{{ byColumn[c.state].length }} · {{ pointsIn(c.state) }} б.</span>
        </header>
        <p class="hint xs muted">{{ c.hint }}</p>
        <div class="cards" @dragstart="onDragStart" @dragend="onDragEnd">
          <template v-for="(t, i) in byColumn[c.state]" :key="t.id">
            <div v-if="dropAt?.state === c.state && dropAt.index === i && dragId !== t.id" class="drop-line" />
            <TaskCard :task="t" :draggable="true" :class="{ dragging: dragId === t.id }"
              :step-label="stepFor(t)?.label" :step-disabled="stepFor(t)?.block" @open="open(t)" @step="tryMove(t, stepFor(t)!.to)" />
          </template>
          <div v-if="dropAt?.state === c.state && dropAt.index >= byColumn[c.state].filter(x => x.id !== dragId).length" class="drop-line" />
          <p v-if="!byColumn[c.state].length" class="empty xs muted">Перетащите карточку сюда</p>
        </div>
      </section>
    </div>
    <TaskDrawer :task-id="openId" :users="users" @close="close" @changed="load" />
    <CreateTaskDialog :open="creating" :projects="projects" :users="users" :project="project" @close="creating = false" @created="creating = false; load()" />
    <ReasonDialog :open="!!pendingReopen" title="Вернуть подтверждённую задачу"
      :text="`Задача вернётся в «${pendingReopen ? stateName[pendingReopen.to] : ''}». Награда и баллы рейтинга исполнителя будут отменены отдельной записью. Если ресурсы уже потрачены, появится долг.`"
      action="Вернуть и отменить награду" destructive @close="pendingReopen = null" @confirm="confirmReopen" />
  </div>
</template>

<style scoped>
.board-page { max-width: none; }
.filters .select { width: auto; min-width: 200px; }
.toggle { display: inline-flex; align-items: center; gap: 6px; font-size: var(--text-sm); font-weight: 600; padding: 0 var(--space-2); cursor: pointer; }
.toggle input { width: 18px; height: 18px; accent-color: var(--color-action-primary); }
.columns { display: grid; grid-template-columns: repeat(5, minmax(240px, 1fr)); gap: var(--space-3); overflow-x: auto; padding-bottom: var(--space-3); align-items: start; }
.column { background: var(--color-surface-sunken); border-radius: var(--radius-md); padding: var(--space-3); min-height: 240px; border: 2px solid transparent; transition: border-color var(--duration-fast); }
.column.over { border-color: var(--color-action-primary); }
.col-head { display: flex; align-items: center; justify-content: space-between; gap: var(--space-2); }
.col-head h2 { font-size: var(--text-sm); font-weight: 750; text-transform: uppercase; letter-spacing: 0.04em; }
.count { font-size: var(--text-xs); font-weight: 700; color: var(--color-text-muted); background: var(--color-surface); padding: 1px 8px; border-radius: var(--radius-pill); white-space: nowrap; }
.hint { margin: 2px 0 var(--space-3); }
.cards { display: flex; flex-direction: column; gap: var(--space-2); min-height: 120px; }
.dragging { opacity: 0.4; }
.drop-line { height: 4px; border-radius: 2px; background: var(--color-action-primary); }
.empty { text-align: center; padding: var(--space-6) 0; border: 1px dashed var(--color-border-strong); border-radius: var(--radius-sm); }
@media (max-width: 767px) {
  .columns { grid-template-columns: repeat(5, 82vw); scroll-snap-type: x mandatory; }
  .column { scroll-snap-align: start; }
  .filters { width: 100%; }
  .filters .select { flex: 1; min-width: 0; }
}
</style>
