<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from '../ui/Icon.vue'
import TaskCard from '../components/TaskCard.vue'
import MoveMenu from '../components/MoveMenu.vue'
import StatRow from '../components/StatRow.vue'
import TaskDrawer from '../components/TaskDrawer.vue'
import CreateTaskDialog from '../components/CreateTaskDialog.vue'
import ReasonDialog from '../components/ReasonDialog.vue'
import { api } from '../api'
import { columns, moveBlock, stateName } from '../lib/board'
import { plural } from '../lib/format'
import { boardStats, dropZoneAccepts } from '../lib/work'
import { moveTask } from '../lib/taskActions'
import { ease, reduced } from '../ui/motion'
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
const pendingReopen = ref<{ task: Task; to: TaskState; index: number } | null>(null)
const pendingReturn = ref<Task | null>(null)
const menu = ref<{ task: Task; anchor: HTMLElement } | null>(null)
const doneAll = ref(false)
const openId = computed(() => (typeof route.query.task === 'string' ? route.query.task : null))
const DONE_SHOWN = 6

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
  m.done.sort((a, b) => (b.approved_at || '').localeCompare(a.approved_at || ''))
  return m
})
const shown = (s: string) => (s === 'done' && !doneAll.value ? byColumn.value.done.slice(0, DONE_SHOWN) : byColumn.value[s])
const pointsIn = (s: string) => byColumn.value[s].reduce((n, t) => n + t.points, 0)
const stats = computed(() => boardStats(tasks.value))
const statItems = computed(() => [
  { icon: 'layers', value: stats.value.planned, label: `${plural(stats.value.planned, 'задача', 'задачи', 'задач')} в плане`, badge: stats.value.overdue ? `просрочено: ${stats.value.overdue}` : '', warn: true },
  { icon: 'flame', value: stats.value.inProgress, label: 'в работе' },
  { icon: 'eye', value: stats.value.review, label: 'ждут проверки' },
  { icon: 'trophy', value: stats.value.donePoints, label: `${plural(stats.value.donePoints, 'очко', 'очка', 'очков')} сдано` },
])

function open(t: Task) { menu.value = null; router.replace({ query: { ...route.query, task: t.id } }) }
function close() { const q = { ...route.query }; delete q.task; router.replace({ query: q }) }

async function tryMove(t: Task, to: TaskState, index?: number) {
  const block = moveBlock(t, to, viewer.value)
  if (block) { toasts.push({ kind: 'error', title: block }); return }
  if (t.state === 'done' && to !== 'done') { pendingReopen.value = { task: t, to, index: index ?? 0 }; return }
  if (t.state === 'review' && to === 'in_progress' && t.assignee_id !== session.me?.id && session.isLead) { pendingReturn.value = t; return }
  // Optimistic: place the card, then reconcile with the server's answer.
  const prev = tasks.value
  tasks.value = tasks.value.map(x => (x.id === t.id ? { ...x, state: to, rank: index !== undefined ? index * 1024 - 1 : 1e12 } : x))
  const r = await moveTask(t, to, { index, undoable: true, after: load })
  if (!r) tasks.value = prev
  load()
}
async function confirmReopen(comment: string) {
  const p = pendingReopen.value
  pendingReopen.value = null
  if (p && await moveTask(p.task, p.to, { index: p.index, comment })) load()
}
async function confirmReturn(comment: string) {
  const t = pendingReturn.value
  pendingReturn.value = null
  if (t && await moveTask(t, 'in_progress', { comment })) load()
}
function onMenuMove(to: TaskState) {
  const m = menu.value
  menu.value = null
  if (!m) return
  m.anchor.focus({ preventScroll: true })
  tryMove(m.task, to)
}
function closeMenu(refocus: boolean) {
  const a = menu.value?.anchor
  menu.value = null
  if (refocus) a?.focus()
}

// Drag and drop with pointer events: mouse starts after a few pixels, touch after a long press.
// The «⋯» menu is the keyboard path for the same moves.
const boardEl = ref<HTMLElement>()
const zoneEl = ref<HTMLElement>()
const drag = ref<{ id: string; from: TaskState } | null>(null)
const dropAt = ref<{ state: TaskState; index: number } | null>(null)
const zoneHot = ref(false)
let pend: { id: string; el: HTMLElement; sx: number; sy: number; x: number; y: number; pid: number; touch: boolean; started: boolean; timer?: number; ox: number; oy: number; ghost?: HTMLElement } | null = null
let suppressClick = false
let raf = 0

const dragTask = computed(() => tasks.value.find(t => t.id === drag.value?.id))
const canDrop = (s: TaskState) => !!dragTask.value && (s === dragTask.value.state || !moveBlock(dragTask.value, s, viewer.value))
const zoneShown = computed(() => !!dragTask.value && dropZoneAccepts(dragTask.value.state) && !moveBlock(dragTask.value, 'review', viewer.value))

function onPointerDown(e: PointerEvent) {
  const el = (e.target as HTMLElement).closest<HTMLElement>('.tcard')
  if (!el || (e.target as HTMLElement).closest('.c-more') || e.button !== 0 || pend) return
  pend = { id: el.dataset.id!, el, sx: e.clientX, sy: e.clientY, x: e.clientX, y: e.clientY, pid: e.pointerId, touch: e.pointerType !== 'mouse', started: false, ox: 0, oy: 0 }
  if (pend.touch) pend.timer = window.setTimeout(() => { if (pend && !pend.started) { startDrag(); navigator.vibrate?.(12) } }, 260)
}
function onPointerMove(e: PointerEvent) {
  if (!pend || e.pointerId !== pend.pid) return
  pend.x = e.clientX; pend.y = e.clientY
  if (!pend.started) {
    const d = Math.hypot(e.clientX - pend.sx, e.clientY - pend.sy)
    if (pend.touch) { if (d > 10) { clearTimeout(pend.timer); pend = null } return }
    if (d < 6) return
    startDrag()
  }
  moveDrag()
}
function onPointerUp(e: PointerEvent) {
  if (!pend || e.pointerId !== pend.pid) return
  clearTimeout(pend.timer)
  if (pend.started) { endDrag(false); suppressClick = true; setTimeout(() => (suppressClick = false), 350) }
  pend = null
}
function onPointerCancel(e: PointerEvent) {
  if (!pend || e.pointerId !== pend.pid) return
  clearTimeout(pend.timer)
  if (pend.started) endDrag(true)
  pend = null
}
function onTouchMove(e: TouchEvent) { if (pend?.started) e.preventDefault() }
function onClickCapture(e: MouseEvent) { if (suppressClick) { e.stopPropagation(); e.preventDefault() } }

function startDrag() {
  const p = pend!, r = p.el.getBoundingClientRect()
  const t = tasks.value.find(x => x.id === p.id)
  if (!t) return
  p.started = true; p.ox = p.x - r.left; p.oy = p.y - r.top
  const g = document.createElement('div')
  g.className = 'drag-ghost'
  const clone = p.el.cloneNode(true) as HTMLElement
  clone.style.width = r.width + 'px'
  clone.querySelectorAll('button').forEach(b => b.setAttribute('tabindex', '-1'))
  g.appendChild(clone)
  g.style.transform = `translate(${r.left}px,${r.top}px)`
  document.body.appendChild(g)
  requestAnimationFrame(() => g.classList.add('lifted'))
  p.ghost = g
  drag.value = { id: t.id, from: t.state }
  document.body.classList.add('dragging')
  const loop = () => { if (!pend?.started) return; edgeScroll(); raf = requestAnimationFrame(loop) }
  raf = requestAnimationFrame(loop)
}
function edgeScroll() {
  const b = boardEl.value
  if (!b || b.scrollWidth <= b.clientWidth || !pend) return
  if (pend.x < 36) b.scrollLeft -= 9
  else if (pend.x > innerWidth - 36) b.scrollLeft += 9
  else return
  moveDrag()
}
function moveDrag() {
  const p = pend!
  if (!p.ghost) return
  p.ghost.style.transform = `translate(${p.x - p.ox}px,${p.y - p.oy}px)`
  const hit = document.elementFromPoint(p.x, p.y)
  const zr = zoneEl.value?.getBoundingClientRect()
  const onZone = zoneShown.value && !!zr && p.x > zr.left - 48 && p.x < zr.right + 48 && p.y > zr.top - 48 && p.y < zr.bottom + 48
  if (onZone && !zoneHot.value && !reduced()) zoneEl.value?.querySelector('.dz-ico')?.animate([{ transform: 'none' }, { transform: 'translateY(-6px)', offset: 0.45 }, { transform: 'none' }], { duration: 200, easing: ease.spring })
  zoneHot.value = onZone
  const col = !onZone ? hit?.closest<HTMLElement>('[data-col]') : null
  const state = col?.dataset.col as TaskState | undefined
  if (!col || !state || !canDrop(state)) { dropAt.value = null; return }
  let index = 0
  for (const el of col.querySelectorAll<HTMLElement>('.tcard')) {
    if (el.dataset.id === p.id) continue
    const r = el.getBoundingClientRect()
    if (p.y > r.top + r.height / 2) index++
  }
  dropAt.value = { state, index }
}
async function endDrag(cancel: boolean) {
  const p = pend!
  cancelAnimationFrame(raf)
  document.body.classList.remove('dragging')
  const t = tasks.value.find(x => x.id === p.id)
  const target: { state: TaskState; index?: number } | null = cancel ? null : zoneHot.value ? { state: 'review' } : dropAt.value
  drag.value = null; dropAt.value = null; zoneHot.value = false
  const g = p.ghost
  if (t && target && (target.state !== t.state || target.index !== undefined)) tryMove(t, target.state, target.index)
  await nextTick()
  const dest = boardEl.value?.querySelector<HTMLElement>(`.tcard[data-id="${p.id}"]`)
  if (!g) return
  if (!dest || reduced()) { g.remove(); return }
  // The ghost lands where the card now is, then hands over to it.
  dest.style.visibility = 'hidden'
  const r = dest.getBoundingClientRect()
  g.classList.remove('lifted'); g.classList.add('landing')
  const a = g.animate([{ transform: g.style.transform }, { transform: `translate(${r.left}px,${r.top}px)` }], { duration: 220, easing: ease.out, fill: 'forwards' })
  a.onfinish = () => { g.remove(); dest.style.visibility = ''; dest.animate([{ transform: 'scale(.96)' }, { transform: 'none' }], { duration: 120, easing: ease.out }) }
}
onMounted(() => {
  addEventListener('pointermove', onPointerMove)
  addEventListener('pointerup', onPointerUp)
  addEventListener('pointercancel', onPointerCancel)
  document.addEventListener('touchmove', onTouchMove, { passive: false })
})
onUnmounted(() => {
  removeEventListener('pointermove', onPointerMove)
  removeEventListener('pointerup', onPointerUp)
  removeEventListener('pointercancel', onPointerCancel)
  document.removeEventListener('touchmove', onTouchMove)
  document.body.classList.remove('dragging')
  document.querySelectorAll('.drag-ghost').forEach(g => g.remove())
})

// Phone: one column at a time, with stage tabs above.
const currentStage = ref<TaskState>('ready')
function goStage(s: TaskState) {
  currentStage.value = s
  boardEl.value?.querySelector<HTMLElement>(`[data-col="${s}"]`)?.scrollIntoView({ behavior: reduced() ? 'auto' : 'smooth', inline: 'start', block: 'nearest' })
}
function onBoardScroll() {
  const b = boardEl.value
  if (!b || b.scrollWidth <= b.clientWidth) return
  const cols = [...b.querySelectorAll<HTMLElement>('[data-col]')]
  const i = cols.findIndex(c => c.offsetLeft + c.offsetWidth / 2 > b.scrollLeft + 16)
  if (i >= 0) currentStage.value = cols[i].dataset.col as TaskState
}
</script>

<template>
  <div class="page board-page">
    <section class="hero">
      <h1>Доска<span class="sub">задач</span></h1>
      <div class="hero-r">
        <label class="chip glass sel"><Icon name="filter" :size="16" /><span class="sr-only">Проект</span>
          <select v-model="project">
            <option value="">Все проекты</option>
            <option v-for="p in projects" :key="p.id" :value="p.id">{{ p.key }} — {{ p.name }}</option>
          </select>
        </label>
        <button class="chip glass" :aria-pressed="onlyMine" @click="onlyMine = !onlyMine"><Icon :name="onlyMine ? 'check' : 'user'" :size="16" />Только мои</button>
        <button class="btn primary add" @click="creating = true"><Icon name="plus" :size="16" />Задача</button>
      </div>
    </section>
    <StatRow :items="statItems" />

    <div v-if="loading" class="muted loading">Загрузка доски…</div>
    <template v-else>
      <nav class="stabs" aria-label="Этапы">
        <button v-for="c in columns" :key="c.state" class="stab" :class="'s-' + c.state" :aria-current="currentStage === c.state || undefined" @click="goStage(c.state)">
          <span class="sw" aria-hidden="true" />{{ c.name }}<em>{{ byColumn[c.state].length }}</em>
        </button>
      </nav>
      <div ref="boardEl" class="board" :class="{ dragging: !!drag }" @pointerdown="onPointerDown" @click.capture="onClickCapture" @contextmenu="e => (e.target as HTMLElement).closest('.tcard') && e.preventDefault()" @scroll.passive="onBoardScroll">
        <section v-for="c in columns" :key="c.state" class="col" :class="['s-' + c.state, { hot: dropAt?.state === c.state, nodrop: drag && !canDrop(c.state) }]" :data-col="c.state" :aria-labelledby="`col-${c.state}`">
          <header class="col-h">
            <span class="sw" aria-hidden="true" />
            <h2 :id="`col-${c.state}`">{{ c.name }}</h2>
            <span class="cnt">{{ byColumn[c.state].length }}</span>
            <span class="sum">{{ pointsIn(c.state) }} {{ plural(pointsIn(c.state), 'очко', 'очка', 'очков') }}</span>
          </header>
          <div class="list">
            <template v-for="(t, i) in shown(c.state)" :key="t.id">
              <div v-if="dropAt?.state === c.state && dropAt.index === i && drag?.id !== t.id" class="drop-line" />
              <TaskCard :task="t" menu :class="{ placeholder: drag?.id === t.id }" @open="open(t)" @menu="a => (menu = menu?.task.id === t.id ? null : { task: t, anchor: a })" />
            </template>
            <div v-if="dropAt?.state === c.state && dropAt.index >= shown(c.state).filter(x => x.id !== drag?.id).length" class="drop-line" />
            <p v-if="!byColumn[c.state].length" class="empty">{{ c.state === 'done' ? 'Подтверждённые задачи появятся здесь' : 'Перетащите карточку сюда' }}</p>
            <button v-if="c.state === 'done' && byColumn.done.length > DONE_SHOWN" class="more-btn" @click="doneAll = !doneAll">
              {{ doneAll ? 'Свернуть' : `Показать все (${byColumn.done.length})` }}
            </button>
          </div>
        </section>
      </div>
    </template>

    <div ref="zoneEl" class="dropzone" :class="{ show: zoneShown, hot: zoneHot }" aria-hidden="true">
      <span class="dz-ico"><Icon name="send" :size="24" /></span>
      <div><div class="dz-t">Отпустите, чтобы сдать на проверку</div><div class="dz-s">Руководитель подтвердит задачу, и она уйдёт в «Выполнено»</div></div>
    </div>
    <button class="fab" aria-label="Новая задача" @click="creating = true"><Icon name="plus" :size="24" /></button>

    <MoveMenu v-if="menu" :task="menu.task" :anchor="menu.anchor" :viewer="viewer" can-open @open="open(menu.task)" @move="onMenuMove" @close="closeMenu" />
    <TaskDrawer :task-id="openId" :users="users" @close="close" @changed="load" />
    <CreateTaskDialog :open="creating" :projects="projects" :users="users" :project="project" @close="creating = false" @created="creating = false; load()" />
    <ReasonDialog :open="!!pendingReopen" title="Вернуть подтверждённую задачу"
      :text="`Задача вернётся в «${pendingReopen ? stateName[pendingReopen.to] : ''}». Награда исполнителя будет отменена отдельной записью. Если ресурсы уже потрачены, появится долг.`"
      action="Вернуть и отменить награду" destructive @close="pendingReopen = null" @confirm="confirmReopen" />
    <ReasonDialog :open="!!pendingReturn" title="Вернуть на доработку" text="Задача вернётся в «В работе». Комментарий увидит исполнитель." action="Вернуть на доработку"
      @close="pendingReturn = null" @confirm="confirmReturn" />
  </div>
</template>

<style scoped>
.board-page { padding-bottom: 140px; }
.hero { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; flex-wrap: wrap; margin-top: 8px; }
.hero-r { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.chip { display: inline-flex; align-items: center; gap: 8px; height: 40px; padding: 0 14px; border-radius: var(--radius-pill); font-weight: 500; font-size: 13px; color: var(--color-text); cursor: pointer; }
.chip[aria-pressed="true"] { background: var(--color-action-primary); color: var(--color-action-primary-text); border-color: transparent; }
.sel select { appearance: none; border: 0; background: transparent; font-weight: 600; font-size: 13px; padding-right: 4px; cursor: pointer; max-width: 220px; }
.sel:focus-within { outline: 3px solid var(--color-focus); outline-offset: 2px; }
.loading { margin-top: 24px; }

.board { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 14px; align-items: start; margin-top: 20px; }
.col { border-radius: var(--radius-lg); padding: 10px; transition: background 0.25s, box-shadow 0.25s, opacity 0.25s; }
.col-h { display: flex; align-items: center; gap: 8px; padding: 6px 6px 12px; }
.sw { width: 12px; height: 12px; border-radius: 50%; background: var(--fill); box-shadow: inset 0 0 0 1px rgba(30, 30, 30, 0.18); flex: none; }
.col-h h2 { font-size: 14px; font-weight: 700; white-space: nowrap; }
.cnt { font-size: 12px; font-weight: 500; color: var(--color-text-muted); background: var(--glass-strong); border-radius: var(--radius-pill); padding: 2px 8px; }
.sum { margin-left: auto; font-size: 12px; color: var(--color-text-muted); white-space: nowrap; }
.list { display: flex; flex-direction: column; gap: 10px; min-height: 72px; }
.col.hot { background: var(--glass-strong); box-shadow: 0 0 0 3px rgba(30, 30, 30, 0.14), var(--shadow-md); }
.board.dragging .col.nodrop { opacity: 0.55; }
.board :deep(.tcard) { cursor: grab; touch-action: pan-x pan-y; }
.board :deep(.tcard.placeholder) { opacity: 0.4; transform: none; box-shadow: none; }
.drop-line { height: 4px; border-radius: 2px; background: var(--color-action-primary); }
.empty { font-size: 12px; color: var(--color-text-muted); text-align: center; padding: 22px 8px; border: 1.5px dashed var(--color-border-strong); border-radius: var(--radius-md); }
.more-btn { height: 40px; border: 0; border-radius: var(--radius-pill); background: var(--glass-strong); font-weight: 600; font-size: 13px; cursor: pointer; }
.more-btn:hover { background: var(--color-surface); }

.dropzone { position: fixed; left: 50%; bottom: 22px; width: min(1180px, calc(100vw - 64px)); height: 104px; z-index: 60; transform: translate(-50%, 24px); opacity: 0; pointer-events: none; display: flex; align-items: center; justify-content: center; gap: 16px; border-radius: 24px; border: 2px dashed rgba(47, 107, 87, 0.5); background: var(--glass-strong); -webkit-backdrop-filter: blur(18px); backdrop-filter: blur(18px); box-shadow: var(--shadow-md); transition: transform 0.28s var(--ease-emphasized), opacity 0.2s var(--ease-emphasized), border-color 0.2s; }
.dropzone.show { transform: translate(-50%, 0); opacity: 1; }
.dropzone.hot { border-color: #2f6b57; background: linear-gradient(rgba(47, 107, 87, 0.1), rgba(47, 107, 87, 0.1)), var(--glass-strong); }
.dz-ico { display: grid; place-items: center; width: 56px; height: 56px; border-radius: 50%; background: #2f6b57; color: #fff; box-shadow: 0 8px 18px rgba(47, 107, 87, 0.35); }
.dz-t { font-weight: 700; font-size: 16px; }
.dz-s { font-size: 12px; color: var(--color-text-muted); margin-top: 3px; }

.stabs, .fab { display: none; }
@media (max-width: 1100px) {
  .board { display: flex; overflow-x: auto; scroll-snap-type: x mandatory; gap: 12px; margin: 12px calc(-1 * var(--space-8)) 0; padding: 0 var(--space-8) 8px; scroll-padding: 0 var(--space-8); scrollbar-width: none; overscroll-behavior-x: contain; }
  .board::-webkit-scrollbar { display: none; }
  .col { flex: 0 0 min(340px, 82vw); scroll-snap-align: start; }
}
@media (max-width: 767px) {
  .hero-r .sel, .hero-r .add { display: none; }
  .board { margin: 12px -16px 0; padding: 0 16px 8px; scroll-padding: 0 16px; }
  .stabs { display: flex; gap: 8px; overflow-x: auto; margin: 16px -16px 0; padding: 2px 16px 4px; scrollbar-width: none; }
  .stabs::-webkit-scrollbar { display: none; }
  .stab { flex: none; display: inline-flex; align-items: center; gap: 7px; height: 36px; padding: 0 14px; border: 0; border-radius: var(--radius-pill); background: var(--glass-strong); font-weight: 500; font-size: 13px; color: var(--color-text); cursor: pointer; transition: background 0.3s, color 0.3s; }
  .stab .sw { width: 10px; height: 10px; }
  .stab em { font-style: normal; font-size: 12px; color: var(--color-text-muted); }
  .stab[aria-current="true"] { background: var(--color-action-primary); color: var(--color-action-primary-text); }
  .stab[aria-current="true"] em { color: inherit; opacity: 0.7; }
  .dropzone { width: calc(100vw - 32px); bottom: calc(96px + env(safe-area-inset-bottom)); height: 92px; }
  .fab { display: grid; place-items: center; position: fixed; right: 16px; bottom: calc(92px + env(safe-area-inset-bottom)); z-index: 45; width: 58px; height: 58px; border: 0; border-radius: 50%; background: #1e1e1e; color: #fff; box-shadow: 0 14px 30px rgba(30, 20, 10, 0.3); cursor: pointer; }
}
</style>

<style>
/* The drag ghost lives on <body>, outside the page's scoped styles. */
.drag-ghost { position: fixed; left: 0; top: 0; z-index: 100; pointer-events: none; will-change: transform; }
.drag-ghost .tcard { transition: transform 0.12s cubic-bezier(.2,0,0,1), box-shadow 0.12s cubic-bezier(.2,0,0,1); }
.drag-ghost.lifted .tcard { transform: scale(1.03) rotate(2deg); box-shadow: var(--shadow-drag); }
.drag-ghost.landing .tcard { transform: scale(0.96); box-shadow: var(--shadow-sm); }
body.dragging, body.dragging * { cursor: grabbing !important; }
</style>
