<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import Icon from '../ui/Icon.vue'
import StatRow from '../components/StatRow.vue'
import TaskDrawer from '../components/TaskDrawer.vue'
import ReasonDialog from '../components/ReasonDialog.vue'
import { api } from '../api'
import { date, isOverdue, plural } from '../lib/format'
import { pointsScale, priorityName } from '../lib/board'
import { avatarHue, cardReward, initials, waitText } from '../lib/work'
import { moveTask } from '../lib/taskActions'
import { ease, flyCoins, reduced } from '../ui/motion'
import { useGame } from '../stores/game'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import { useWork } from '../stores/work'
import type { Task, TeamUser } from '../types'

const session = useSession()
const game = useGame()
const work = useWork()
const toasts = useToasts()
const review = ref<Task[]>([])
const estimates = ref<Task[]>([])
const approved = ref<Task[]>([])
const users = ref<TeamUser[]>([])
const open = ref<string | null>(null)
const busy = ref('')
const returning = ref<Task | null>(null)
const now = ref(new Date())

async function load() {
  try {
    const r = await api.get<{ review: Task[]; estimates: Task[] }>('/review')
    review.value = r.review
    estimates.value = r.estimates
    approved.value = approved.value.filter(a => !r.review.some(t => t.id === a.id))
    now.value = new Date()
    work.reviewCount = r.review.length
  } catch (e) { toasts.error(e) }
}
onMounted(async () => {
  load()
  game.loadConfig().catch(() => {})
  users.value = (await api.get<{ items: TeamUser[] }>('/users')).items
})

const cards = computed(() => [...review.value.map(t => ({ t, done: false })), ...approved.value.map(t => ({ t, done: true }))])
const statItems = computed(() => [
  { icon: 'eye', value: review.value.length, label: 'ждут проверки' },
  { icon: 'layers', value: estimates.value.length, label: 'оценок на согласование' },
  { icon: 'check', value: approved.value.length, label: 'подтверждено сейчас' },
])
const reward = (t: Task) => cardReward(t, game.config?.settings)

async function approve(t: Task, ev: Event) {
  const el = (ev.currentTarget as HTMLElement).closest<HTMLElement>('.rcard')!
  const chip = el.querySelector('.gold-chip')
  const avatar = el.querySelector('.r-meta .av')
  busy.value = t.id
  const r = await moveTask(t, 'done', { undoable: true, after: load })
  busy.value = ''
  if (!r) return
  // The card stays in place, stamped, until the page is reloaded; the queue itself is the server's.
  review.value = review.value.filter(x => x.id !== t.id)
  approved.value = [{ ...r.task }, ...approved.value]
  work.reviewCount = review.value.length
  await nextTick()
  const card = document.querySelector<HTMLElement>(`.rcard[data-id="${t.id}"]`)
  if (!card || reduced()) return
  const stamp = card.querySelector('.stamp'), ink = card.querySelector('.ink'), R = 'translate(-50%,-50%) '
  if (!session.gameOn) {
    stamp?.animate([{ opacity: 0, transform: R + 'rotate(-6deg) scale(1.1)' }, { opacity: 1, transform: R + 'rotate(-6deg) scale(1)' }], { duration: 200, easing: ease.out })
    return
  }
  stamp?.animate([
    { opacity: 0, transform: R + 'rotate(-12deg) scale(1.4)', offset: 0 },
    { opacity: 1, transform: R + 'rotate(-12deg) scale(1.4)', offset: 0.25, easing: ease.slam },
    { opacity: 1, transform: R + 'rotate(-6deg) scale(.95)', offset: 0.62, easing: ease.out },
    { opacity: 1, transform: R + 'rotate(-6deg) scale(1)', offset: 1 }], { duration: 320 })
  card.animate([{ transform: 'none' }, { transform: 'translateX(-3px)' }, { transform: 'translateX(3px)' }, { transform: 'translateX(-3px)' }, { transform: 'translateX(3px)' }, { transform: 'none' }], { duration: 120, delay: 200 })
  ink?.animate([{ opacity: 0.5, transform: 'rotate(-6deg) scale(.6)' }, { opacity: 0, transform: 'rotate(-6deg) scale(1.3)' }], { duration: 240, delay: 200, easing: ease.out, fill: 'backwards' })
  // The gold goes to the assignee, so the coins fly to their avatar, not to the reviewer's counter.
  if (chip && avatar) setTimeout(() => flyCoins(chip, avatar, () => {}), 450)
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
</script>

<template>
  <div class="page">
    <section class="hero">
      <h1>Очередь<span class="sub">проверки</span></h1>
    </section>
    <StatRow :items="statItems" />
    <p class="rv-bar glass">
      <span class="av" :style="{ '--hue': avatarHue(session.me?.display_name || '') }" aria-hidden="true">{{ initials(session.me?.display_name) }}</span>
      <span>Вы проверяете как руководитель · <b>{{ session.me?.display_name }}</b> — подтверждённые задачи уходят в «Выполнено». Свою задачу подтвердить нельзя.</span>
    </p>

    <div v-if="cards.length" class="rgrid">
      <article v-for="{ t, done } in cards" :key="t.id" class="rcard" :class="{ approved: done }" :data-id="t.id" :aria-label="`${t.number} ${t.title}${done ? ', подтверждена' : ''}`">
        <div class="r-body">
          <div class="r-top"><Icon name="calendar" :size="14" />
            <span :class="{ late: isOverdue(t.due_date) && !done }">{{ t.due_date ? `до ${date(t.due_date)}` : 'без срока' }}</span><span>·</span><span>{{ t.number }}</span>
            <span class="prio" :title="priorityName[t.priority]">{{ t.priority }}</span>
            <span class="wait"><Icon name="clock" :size="14" />{{ waitText(t.updated_at, now) }}</span>
          </div>
          <h3 class="r-title"><button class="linkish" @click="open = t.id">{{ t.title }}</button></h3>
          <div class="r-meta">
            <span v-if="t.assignee_name" class="av" :style="{ '--hue': avatarHue(t.assignee_name), '--fill': '#1e1e1e' }" aria-hidden="true">{{ initials(t.assignee_name) }}</span>
            {{ t.assignee_name }} · {{ t.project_name }}
          </div>
          <div class="r-mets">
            <div><b>{{ t.points }}</b><span>{{ plural(t.points, 'очко', 'очка', 'очков') }}</span></div>
            <div><b>{{ t.priority }}</b><span>{{ priorityName[t.priority].toLowerCase() }}</span></div>
            <div><b>{{ t.due_date ? date(t.due_date) : '—' }}</b><span>{{ t.due_date && isOverdue(t.due_date) ? 'просрочен' : 'срок' }}</span></div>
          </div>
          <div v-if="session.gameOn && reward(t) !== null" class="r-rew game-only">
            <span class="gold-chip"><svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><circle cx="12" cy="12" r="9" fill="#FFD75A" stroke="#7A4A00" stroke-width="2"/><path d="M12 8v8" stroke="#7A4A00" stroke-width="2" stroke-linecap="round"/></svg>+{{ reward(t) }}</span>
            <span>{{ done ? 'начислено исполнителю' : 'награда исполнителю после подтверждения' }}</span>
          </div>
        </div>
        <div v-if="done" class="r-done"><Icon name="check" :size="18" />Подтверждено — задача в «Выполнено»</div>
        <p v-else-if="t.assignee_id === session.me?.id" class="r-self"><Icon name="lock" :size="16" />Ваша задача — подтвердит другой руководитель</p>
        <div v-else class="r-act">
          <button class="btn ghost-dark" :disabled="!!busy" @click="returning = t"><Icon name="undo" :size="16" />Вернуть</button>
          <button class="btn ok" :disabled="!!busy" :aria-busy="busy === t.id" @click="approve(t, $event)"><Icon name="check" :size="16" />Подтвердить</button>
        </div>
        <div class="ink" aria-hidden="true" /><div class="stamp" aria-hidden="true">ПОДТВЕРЖДЕНО</div>
      </article>
    </div>
    <div v-else class="rempty glass"><b>Очередь проверки пуста</b>Когда команда сдаст задачу, она появится здесь.</div>

    <section class="est">
      <h2>Оценки на согласование <span class="cnt">{{ estimates.length }}</span></h2>
      <p class="small muted">Сложность согласуется до начала работы. Предложенную оценку можно принять или заменить — изменение увидит команда в истории задачи.</p>
      <p v-if="!estimates.length" class="rempty glass small">Все оценки согласованы.</p>
      <article v-for="t in estimates" :key="t.id" class="eitem glass">
        <div class="grow">
          <p class="xs muted">{{ t.number }} · предложил(а) {{ t.created_by }}</p>
          <button class="linkish etitle" @click="open = t.id">{{ t.title }}</button>
        </div>
        <div class="seg" role="group" :aria-label="`Сложность для ${t.number}`">
          <button v-for="p in pointsScale" :key="p" class="pt" :class="{ proposed: p === t.points }" :disabled="!!busy"
            :title="p === t.points ? 'Согласовать предложенную оценку' : `Согласовать ${p}`" @click="agree(t, p)">{{ p }}</button>
        </div>
      </article>
    </section>
    <TaskDrawer :task-id="open" :users="users" @close="open = null" @changed="load" />
    <ReasonDialog :open="!!returning" title="Вернуть на доработку" text="Задача вернётся в «В работе». Комментарий увидит исполнитель." action="Вернуть на доработку" @close="returning = null" @confirm="sendBack" />
  </div>
</template>

<style scoped>
.hero { margin-top: 8px; }
.rv-bar { display: flex; align-items: center; gap: 10px; margin-top: 20px; padding: 8px 16px 8px 8px; border-radius: var(--radius-pill); width: fit-content; max-width: 100%; font-size: 13px; color: var(--color-text-muted); }
.rv-bar b { color: var(--color-text); }
.rgrid { display: grid; grid-template-columns: repeat(auto-fill, minmax(310px, 1fr)); gap: 16px; margin-top: 16px; }
.rcard { position: relative; isolation: isolate; overflow: hidden; padding: 20px; border-radius: 24px; background: #1e1e1e; color: #fff; box-shadow: 0 2px 4px rgba(30, 20, 10, 0.1), 0 18px 36px rgba(30, 20, 10, 0.18); }
.rcard::after { content: ""; position: absolute; inset: auto -40px -80px auto; width: 220px; height: 220px; border-radius: 50%; background: radial-gradient(closest-side, rgba(242, 207, 91, 0.16), transparent); z-index: -1; }
.r-body { transition: opacity 0.35s var(--ease-out), filter 0.35s; }
.r-top { display: flex; align-items: center; gap: 8px; font-size: 12px; color: #b8b8b8; flex-wrap: wrap; }
.r-top .late { color: #ffb59e; font-weight: 600; }
.prio { font-size: 11px; font-weight: 700; padding: 2px 7px; border-radius: var(--radius-pill); background: rgba(255, 255, 255, 0.14); color: #fff; }
.wait { margin-left: auto; display: inline-flex; align-items: center; gap: 5px; }
.r-title { font-size: 20px; line-height: 1.2; font-weight: 700; margin-top: 14px; letter-spacing: -0.02em; }
.linkish { all: unset; cursor: pointer; }
.linkish:hover { text-decoration: underline; text-underline-offset: 4px; }
.linkish:focus-visible { outline: 3px solid var(--color-focus); outline-offset: 2px; border-radius: 4px; }
.r-meta { display: flex; align-items: center; gap: 8px; font-size: 13px; color: #b8b8b8; margin-top: 8px; }
.r-mets { display: grid; grid-template-columns: repeat(3, 1fr); margin-top: 16px; padding: 12px 0; border-radius: 16px; background: rgba(255, 255, 255, 0.08); }
.r-mets div { padding: 0 14px; min-width: 0; }
.r-mets div + div { border-left: 1px solid rgba(255, 255, 255, 0.14); }
.r-mets b { display: block; font-size: 18px; font-weight: 700; white-space: nowrap; }
.r-mets span { font-size: 11px; color: #b8b8b8; }
.r-rew { display: flex; align-items: center; gap: 10px; margin-top: 14px; font-size: 12px; color: #b8b8b8; }
.r-act { display: flex; gap: 8px; margin-top: 18px; }
.r-act .btn { flex: 1; min-height: 46px; font-size: 14px; min-width: 0; }
.r-act .ok { flex: 1.3; background: #fff; color: #1e1e1e; box-shadow: none; }
.r-act .ok:hover:not(:disabled) { background: #d9eedf; }
.ghost-dark { background: transparent; color: #fff; box-shadow: inset 0 0 0 1.5px rgba(255, 255, 255, 0.32); }
.ghost-dark:hover:not(:disabled) { background: rgba(255, 255, 255, 0.08); }
.ok:focus-visible, .ghost-dark:focus-visible { outline-color: #fff; }
.r-self, .r-done { display: flex; align-items: center; gap: 8px; margin-top: 18px; min-height: 46px; font-size: 13px; font-weight: 500; color: #b8b8b8; }
.r-done { color: #d7ebdf; }
.rcard.approved .r-body { opacity: 0.42; filter: saturate(0.6); }
.stamp { position: absolute; left: 50%; top: 44%; z-index: 3; transform: translate(-50%, -50%) rotate(-6deg); padding: 8px 16px; border: 3px solid #7fe0b0; border-radius: 12px; color: #7fe0b0; font: 900 21px/1 var(--font-game); letter-spacing: 0.08em; opacity: 0; pointer-events: none; white-space: nowrap; background: rgba(30, 30, 30, 0.55); box-shadow: inset 0 0 0 2px #1e1e1e, inset 0 0 0 4px rgba(127, 224, 176, 0.55); }
.rcard.approved .stamp { opacity: 1; }
.ink { position: absolute; left: 50%; top: 44%; z-index: 2; width: 260px; height: 90px; margin: -45px 0 0 -130px; border-radius: 24px; border: 3px solid #7fe0b0; opacity: 0; pointer-events: none; transform: rotate(-6deg); }
.rempty { margin-top: 16px; padding: 40px 24px; border-radius: var(--radius-xl); text-align: center; color: var(--color-text-muted); }
.rempty b { display: block; color: var(--color-text); font-size: 18px; margin-bottom: 6px; }
.est { margin-top: 36px; display: flex; flex-direction: column; gap: 10px; }
.est h2 { font-size: 20px; display: flex; align-items: center; gap: 8px; }
.cnt { font-size: 12px; font-weight: 500; color: var(--color-text-muted); background: var(--glass-strong); border-radius: var(--radius-pill); padding: 2px 8px; }
.est .rempty { padding: 20px; }
.eitem { display: flex; align-items: center; gap: 16px; padding: 14px 16px; border-radius: var(--radius-lg); flex-wrap: wrap; }
.etitle { font-weight: 700; font-size: 15px; margin-top: 2px; }
.seg { display: flex; gap: 6px; }
.pt { width: 44px; height: 40px; border: 0; border-radius: 12px; background: var(--glass-strong); font-weight: 800; cursor: pointer; box-shadow: inset 0 0 0 1px var(--color-border); }
.pt:hover:not(:disabled) { background: var(--color-action-primary); color: var(--color-action-primary-text); }
.pt.proposed { box-shadow: inset 0 0 0 2px var(--color-action-primary); }
@media (max-width: 767px) { .rgrid { grid-template-columns: 1fr; } .rv-bar { border-radius: var(--radius-lg); align-items: flex-start; } }
</style>
