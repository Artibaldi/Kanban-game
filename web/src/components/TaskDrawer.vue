<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Modal from '../ui/Modal.vue'
import Icon from '../ui/Icon.vue'
import ReasonDialog from './ReasonDialog.vue'
import { api } from '../api'
import { columns, moveBlock, pointsScale, priorityName, rewardPreview, stateName } from '../lib/board'
import { date, dateTime, isOverdue, signed } from '../lib/format'
import { moveTask } from '../lib/taskActions'
import { useSession } from '../stores/session'
import { useGame } from '../stores/game'
import { useToasts } from '../stores/toast'
import type { TaskDetail, TaskState, TeamUser } from '../types'

const props = defineProps<{ taskId: string | null; users: TeamUser[] }>()
const emit = defineEmits<{ close: []; changed: [] }>()
const session = useSession()
const game = useGame()
const toasts = useToasts()
const detail = ref<TaskDetail | null>(null)
const loading = ref(false)
const tab = ref<'comments' | 'history' | 'rewards'>('comments')
const comment = ref('')
const busy = ref('')
const reopenTo = ref<TaskState | null>(null)
const returnOpen = ref(false)
const edit = ref({ title: '', description: '' })
const editing = ref(false)

async function load() {
  if (!props.taskId) { detail.value = null; return }
  loading.value = true
  try {
    detail.value = await api.get<TaskDetail>(`/tasks/${props.taskId}`)
    edit.value = { title: detail.value.task.title, description: detail.value.task.description }
  } catch (e) { toasts.error(e); emit('close') } finally { loading.value = false }
}
watch(() => props.taskId, () => { editing.value = false; tab.value = 'comments'; load() }, { immediate: true })

const t = computed(() => detail.value?.task)
const viewer = computed(() => ({ id: session.me?.id || '', role: session.me?.role || 'user', team_id: session.me?.team_id || null }))
const isLead = computed(() => !!t.value && (session.me?.role === 'admin' || (session.me?.role === 'lead' && session.me.team_id === t.value.team_id)))
const mine = computed(() => !!t.value && (t.value.assignee_id === session.me?.id || t.value.created_by_id === session.me?.id))
const canEdit = computed(() => isLead.value || mine.value)
const teamUsers = computed(() => props.users.filter(u => u.team_id === t.value?.team_id))
const preview = computed(() => {
  if (!t.value || !game.config) return null
  const onTime = !t.value.due_date || !isOverdue(t.value.due_date)
  return rewardPreview(t.value.points, t.value.priority, game.config.settings, onTime && !!t.value.due_date)
})
const moves = computed(() => {
  if (!t.value) return []
  return columns.filter(c => c.state !== t.value!.state).map(c => ({ ...c, block: moveBlock(t.value!, c.state, viewer.value) }))
})

async function patch(body: Record<string, unknown>) {
  if (!t.value) return
  busy.value = 'patch'
  try {
    detail.value = await api.patch<TaskDetail>(`/tasks/${t.value.id}`, { ...body, version: t.value.version })
    edit.value = { title: detail.value.task.title, description: detail.value.task.description }
    editing.value = false
    emit('changed')
  } catch (e) { toasts.error(e); load() } finally { busy.value = '' }
}

async function agree(points?: number) {
  if (!t.value) return
  busy.value = 'agree'
  try {
    detail.value = await api.post<TaskDetail>(`/tasks/${t.value.id}/estimate`, { points: points || t.value.points })
    toasts.push({ kind: 'success', title: `Сложность ${detail.value.task.points} согласована` })
    emit('changed')
  } catch (e) { toasts.error(e) } finally { busy.value = '' }
}

async function move(to: TaskState) {
  if (!t.value) return
  if (t.value.state === 'done') { reopenTo.value = to; return }
  if (t.value.state === 'review' && to === 'in_progress' && isLead.value && t.value.assignee_id !== session.me?.id) { returnOpen.value = true; return }
  busy.value = to
  const r = await moveTask(t.value, to)
  busy.value = ''
  if (r) { await load(); emit('changed') }
}
async function confirmReason(to: TaskState, text: string) {
  if (!t.value) return
  reopenTo.value = null
  returnOpen.value = false
  const r = await moveTask(t.value, to, { comment: text })
  if (r) { await load(); emit('changed') }
}

async function addComment() {
  if (!t.value || !comment.value.trim()) return
  busy.value = 'comment'
  try {
    detail.value = await api.post<TaskDetail>(`/tasks/${t.value.id}/comments`, { body: comment.value })
    comment.value = ''
  } catch (e) { toasts.error(e) } finally { busy.value = '' }
}

const historyText = (h: { kind: string; from: string | null; to: string | null }) => {
  const st = (v: string | null) => (v && v in stateName ? stateName[v as TaskState] : v || '—')
  switch (h.kind) {
    case 'created': return `Создана в «${st(h.to)}»`
    case 'state': return `${st(h.from)} → ${st(h.to)}`
    case 'approved': return 'Подтверждена руководителем'
    case 'reopened': return `Возвращена: ${st(h.from)} → ${st(h.to)}`
    case 'points': return `Сложность ${h.from} → ${h.to}`
    case 'points_agreed': return `Сложность ${h.to} согласована`
    case 'assignee': return `Исполнитель: ${h.from || 'нет'} → ${h.to || 'нет'}`
    case 'priority': return `Приоритет ${h.from} → ${h.to}`
    case 'due_date': return `Срок: ${date(h.from)} → ${date(h.to)}`
    case 'title': return 'Изменено название'
    default: return 'Изменено описание'
  }
}
const kindName: Record<string, string> = { task_reward: 'Награда', task_reward_reversal: 'Отмена награды' }
</script>

<template>
  <Modal :open="!!taskId" :title="t ? `${t.number} · ${t.project_name}` : 'Задача'" side @close="emit('close')">
    <div v-if="loading && !t" class="muted">Загрузка…</div>
    <div v-else-if="t" class="drawer">
      <div class="state-row">
        <span class="badge" :class="{ success: t.state === 'done', warning: t.state === 'review', info: t.state === 'in_progress' }">{{ stateName[t.state] }}</span>
        <span v-if="t.approved_at" class="small muted">Подтвердил(а) {{ t.approved_by }}, {{ dateTime(t.approved_at) }}</span>
      </div>

      <div v-if="!editing" class="title-block">
        <h2>{{ t.title }}</h2>
        <p v-if="t.description" class="desc">{{ t.description }}</p>
        <p v-else class="desc muted">Описания нет.</p>
        <button v-if="canEdit" class="btn ghost sm" @click="editing = true">Изменить текст</button>
      </div>
      <form v-else class="stack" @submit.prevent="patch({ title: edit.title, description: edit.description })">
        <label class="field">Название<input v-model="edit.title" class="input" maxlength="200" /></label>
        <label class="field">Описание<textarea v-model="edit.description" class="textarea" /></label>
        <div class="row"><button class="btn primary sm" :aria-busy="busy === 'patch'">Сохранить</button><button type="button" class="btn sm" @click="editing = false">Отмена</button></div>
      </form>

      <dl class="props">
        <div><dt>Исполнитель</dt><dd>
          <select v-if="t.state !== 'done' && (isLead || !t.assignee_id || t.assignee_id === session.me?.id)" class="select" :value="t.assignee_id || ''"
            @change="patch({ assignee_id: ($event.target as HTMLSelectElement).value || null })" aria-label="Исполнитель">
            <option value="">Не назначен</option>
            <option v-for="u in teamUsers" :key="u.id" :value="u.id" :disabled="!isLead && u.id !== session.me?.id">{{ u.display_name }}</option>
          </select>
          <span v-else>{{ t.assignee_name || 'Не назначен' }}</span>
        </dd></div>
        <div><dt>Приоритет</dt><dd>
          <select v-if="canEdit && t.state !== 'done'" class="select" :value="t.priority" aria-label="Приоритет" @change="patch({ priority: ($event.target as HTMLSelectElement).value })">
            <option v-for="(n, k) in priorityName" :key="k" :value="k">{{ k }} — {{ n }}</option>
          </select>
          <span v-else>{{ t.priority }} — {{ priorityName[t.priority] }}</span>
        </dd></div>
        <div><dt>Срок</dt><dd>
          <input v-if="canEdit && t.state !== 'done'" type="date" class="input" :value="t.due_date || ''" aria-label="Срок" @change="patch({ due_date: ($event.target as HTMLInputElement).value || null })" />
          <span v-else :class="{ late: isOverdue(t.due_date) && t.state !== 'done' }">{{ date(t.due_date) }}</span>
        </dd></div>
        <div><dt>Автор</dt><dd>{{ t.created_by }}</dd></div>
      </dl>

      <section class="estimate" :class="{ unagreed: !t.points_approved }" aria-labelledby="est-h">
        <div class="row wrap">
          <h3 id="est-h" class="grow">Сложность: {{ t.points }} {{ t.points === 1 ? 'балл' : t.points < 5 ? 'балла' : 'баллов' }}</h3>
          <span v-if="t.state === 'done'" class="badge"><Icon name="lock" :size="12" />Зафиксирована</span>
          <span v-else-if="t.points_approved" class="badge success"><Icon name="check" :size="12" />Согласовал(а) {{ t.points_approved_by }}</span>
          <span v-else class="badge warning"><Icon name="alert" :size="12" />Ждёт согласования</span>
        </div>
        <div v-if="t.state !== 'done' && (isLead || (!t.points_approved && canEdit))" class="seg" role="group" aria-label="Изменить сложность">
          <button v-for="p in pointsScale" :key="p" class="pt" :aria-pressed="p === t.points" :disabled="!!busy" @click="isLead ? agree(p) : patch({ points: p })">{{ p }}</button>
        </div>
        <p class="small muted">
          <template v-if="t.state === 'done'">После подтверждения сложность не меняется. Чтобы изменить её, руководитель возвращает задачу в работу — награда при этом отменяется.</template>
          <template v-else-if="isLead">Выбор значения согласует сложность. Каждое изменение попадает в историю и на график объёма бэклога.</template>
          <template v-else-if="t.points_approved">Изменить согласованную сложность может только руководитель.</template>
          <template v-else>Предложите оценку — руководитель согласует её до начала работы.</template>
        </p>
        <button v-if="isLead && !t.points_approved && t.state !== 'done'" class="btn primary sm" :aria-busy="busy === 'agree'" @click="agree()">Согласовать {{ t.points }}</button>
        <div v-if="session.gameOn" class="preview game-only">
          <template v-if="t.reward">
            <span class="eyebrow">Начислено</span>
            <span class="g">+{{ t.reward.gold }} золота</span><span class="e">+{{ t.reward.energy }} энергии</span><span class="x">+{{ t.reward.xp }} опыта</span>
          </template>
          <template v-else-if="preview">
            <span class="eyebrow">Награда после подтверждения</span>
            <span class="g">≈{{ preview.gold }} золота</span><span class="e">{{ preview.energy }} энергии</span><span class="x">{{ preview.xp }} опыта</span>
          </template>
        </div>
      </section>

      <section class="moves" aria-label="Переместить">
        <span class="eyebrow">Переместить</span>
        <div class="row wrap">
          <button v-for="m in moves" :key="m.state" class="btn sm" :class="{ primary: m.state === 'done' && !m.block, danger: t.state === 'done' && !m.block }"
            :disabled="!!m.block || !!busy" :title="m.block || ''" :aria-busy="busy === m.state" @click="move(m.state)">
            <Icon v-if="t.state === 'done'" name="undo" :size="14" />{{ m.state === 'done' ? 'Подтвердить выполнение' : m.name }}
          </button>
        </div>
        <p v-if="t.state === 'review' && t.assignee_id === session.me?.id && isLead" class="small muted">Это ваша задача: подтвердить её может другой руководитель или администратор.</p>
      </section>

      <div class="tabs" role="tablist">
        <button role="tab" :aria-selected="tab === 'comments'" @click="tab = 'comments'">Комментарии ({{ detail!.comments.length }})</button>
        <button role="tab" :aria-selected="tab === 'history'" @click="tab = 'history'">История ({{ detail!.history.length }})</button>
        <button role="tab" :aria-selected="tab === 'rewards'" @click="tab = 'rewards'">Начисления ({{ detail!.rewards.length }})</button>
      </div>

      <section v-if="tab === 'comments'" class="list">
        <p v-if="!detail!.comments.length" class="muted small">Комментариев пока нет.</p>
        <article v-for="c in detail!.comments" :key="c.id" class="comment">
          <div class="row small"><strong>{{ c.author }}</strong><span class="muted">{{ dateTime(c.created_at) }}</span></div>
          <p>{{ c.body }}</p>
        </article>
        <form class="stack" @submit.prevent="addComment">
          <label class="field">Новый комментарий<textarea v-model="comment" class="textarea" maxlength="4000" /></label>
          <div><button class="btn primary sm" :disabled="!comment.trim()" :aria-busy="busy === 'comment'"><Icon name="message" :size="14" />Отправить</button></div>
        </form>
      </section>
      <ol v-else-if="tab === 'history'" class="list timeline">
        <li v-for="h in detail!.history" :key="h.id">
          <div class="row small"><strong>{{ historyText(h) }}</strong></div>
          <div class="xs muted">{{ h.actor }} · {{ dateTime(h.created_at) }}</div>
          <p v-if="h.note" class="small">{{ h.note }}</p>
        </li>
      </ol>
      <section v-else class="list">
        <p class="small muted">Каждое начисление связано с задачей. Отмена — отдельная запись, исходная помечается отменённой.</p>
        <p v-if="!detail!.rewards.length" class="muted small">Начислений нет: награда появится после подтверждения.</p>
        <article v-for="l in detail!.rewards" :key="l.id" class="ledger" :class="{ reversed: l.reversed }">
          <div class="row small wrap"><strong>{{ kindName[l.kind] || l.kind }} №{{ l.id }}</strong><span class="muted">{{ l.user }} · {{ dateTime(l.created_at) }}</span>
            <span v-if="l.reversed" class="badge">отменена</span><span v-if="l.reverses_id" class="badge info">отменяет №{{ l.reverses_id }}</span></div>
          <div class="row small num wrap"><span class="g">{{ signed(l.gold) }} золота</span><span class="e">{{ signed(l.energy) }} энергии</span><span class="x">{{ signed(l.xp) }} опыта</span></div>
          <p v-if="l.gold_formula !== null && l.gold_formula !== l.gold" class="xs muted">По формуле {{ l.gold_formula }} золота, не поместилось в хранилище: {{ l.gold_formula - l.gold }}.</p>
          <p class="xs muted">{{ l.note }}</p>
        </article>
      </section>
    </div>
  </Modal>
  <ReasonDialog :open="!!reopenTo" title="Вернуть подтверждённую задачу" :text="`Задача вернётся в «${reopenTo ? stateName[reopenTo] : ''}». Награда исполнителя и баллы рейтинга будут отменены отдельной записью в журнале. Если ресурсы уже потрачены, у исполнителя появится долг, который погасится следующими наградами.`"
    action="Вернуть и отменить награду" destructive @close="reopenTo = null" @confirm="c => confirmReason(reopenTo!, c)" />
  <ReasonDialog :open="returnOpen" title="Вернуть на доработку" text="Задача вернётся в «В работе». Комментарий увидит исполнитель." action="Вернуть на доработку"
    @close="returnOpen = false" @confirm="c => confirmReason('in_progress', c)" />
</template>

<style scoped>
.drawer { display: flex; flex-direction: column; gap: var(--space-5); }
.state-row { display: flex; align-items: center; gap: var(--space-3); flex-wrap: wrap; }
.title-block { display: flex; flex-direction: column; gap: var(--space-2); align-items: flex-start; }
.title-block h2 { font-size: var(--text-xl); }
.desc { white-space: pre-wrap; }
.props { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-3) var(--space-4); margin: 0; }
.props dt { font-size: var(--text-xs); font-weight: 700; color: var(--color-text-muted); text-transform: uppercase; letter-spacing: 0.04em; margin-bottom: 4px; }
.props dd { margin: 0; }
.late { color: var(--color-danger); font-weight: 700; }
.estimate { display: flex; flex-direction: column; gap: var(--space-3); padding: var(--space-4); border-radius: var(--radius-md); background: var(--color-surface-2); border: 1px solid var(--color-border); }
.estimate.unagreed { border-color: var(--color-warning); background: var(--color-warning-bg); }
.seg { display: flex; gap: var(--space-2); flex-wrap: wrap; }
.pt { width: 44px; height: 38px; border-radius: var(--radius-sm); border: 1px solid var(--color-border-strong); background: var(--color-surface); font-weight: 800; cursor: pointer; }
.pt:hover:not(:disabled) { background: var(--color-action-secondary-hover); }
.pt[aria-pressed="true"] { background: var(--color-action-primary); color: var(--color-action-primary-text); border-color: var(--color-action-primary); }
.pt:disabled { opacity: 0.5; }
.preview { display: flex; flex-wrap: wrap; gap: var(--space-3); align-items: center; font-weight: 700; font-size: var(--text-sm); padding-top: var(--space-2); border-top: 1px dashed var(--color-border-strong); }
.g { color: var(--color-gold); } .e { color: var(--color-energy); } .x { color: var(--color-xp); }
.moves { display: flex; flex-direction: column; gap: var(--space-2); }
.list { display: flex; flex-direction: column; gap: var(--space-3); padding: 0; margin: 0; }
.comment, .ledger { padding: var(--space-3); border-radius: var(--radius-sm); background: var(--color-surface-2); }
.comment p { white-space: pre-wrap; margin-top: 4px; }
.ledger.reversed { opacity: 0.7; }
.timeline { list-style: none; border-left: 2px solid var(--color-border); padding-left: var(--space-4); }
.timeline li { position: relative; }
.timeline li::before { content: ""; position: absolute; left: calc(-1 * var(--space-4) - 6px); top: 6px; width: 10px; height: 10px; border-radius: 50%; background: var(--color-action-primary); }
@media (max-width: 500px) { .props { grid-template-columns: 1fr; } }
</style>
