<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import Modal from '../ui/Modal.vue'
import Icon from '../ui/Icon.vue'
import ReasonDialog from './ReasonDialog.vue'
import MoveMenu from './MoveMenu.vue'
import { api } from '../api'
import { columns, moveBlock, pointsScale, priorityName, rewardPreview, stateName } from '../lib/board'
import { date, dateTime, isOverdue, plural, signed } from '../lib/format'
import { avatarHue, cardReward, initials } from '../lib/work'
import { countUp } from '../ui/motion'
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
  const r = await moveTask(t.value, to, { undoable: true, after: () => { load(); emit('changed') } })
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
const menuAnchor = ref<HTMLElement | null>(null)
const reward = computed(() => (t.value ? cardReward(t.value, game.config?.settings) : null))
const stageIndex = computed(() => columns.findIndex(c => c.state === t.value?.state))
const primary = computed(() => {
  if (!t.value) return null
  const map: Record<TaskState, [string, TaskState | null]> = {
    backlog: ['Взять в план', 'ready'], ready: ['Взять в работу', 'in_progress'], in_progress: ['Отправить на проверку', 'review'],
    review: isLead.value && t.value.assignee_id !== session.me?.id ? ['Подтвердить', 'done'] : ['Ждёт проверки', null], done: ['Задача сдана', null],
  }
  const [label, to] = map[t.value.state]
  return { label, to, block: to ? moveBlock(t.value, to, viewer.value) : '' }
})
// The big number rolls up when the card opens.
const bigEl = ref<HTMLElement>()
watch(() => [props.taskId, !!detail.value, session.gameOn] as const, async ([id, ready]) => {
  if (!id || !ready) return
  await nextTick()
  const to = session.gameOn && reward.value !== null ? reward.value : t.value?.points ?? 0
  countUp(bigEl.value, 0, to, 500)
})
function onMenuMove(to: TaskState) { menuAnchor.value?.focus(); menuAnchor.value = null; move(to) }
const kindName: Record<string, string> = { task_reward: 'Награда', task_reward_reversal: 'Отмена награды' }
</script>

<template>
  <Modal :open="!!taskId" :title="t ? `${t.number}: ${t.title}` : 'Задача'" sheet hide-head :tone="t ? 's-' + t.state : 's-backlog'"
    :origin="taskId ? `.tcard[data-id='${taskId}']` : undefined" @close="emit('close')">
    <div v-if="loading && !t" class="loading">Загрузка…</div>
    <div v-else-if="t" class="sheet-body">
      <div class="m-top">
        <Icon :name="t.state === 'done' ? 'check' : 'calendar'" :size="14" />
        <span :class="{ late: isOverdue(t.due_date) && t.state !== 'done' }">{{ t.state === 'done' ? `сдано ${date(t.approved_at)}` : !t.due_date ? 'без срока' : isOverdue(t.due_date) ? `просрочено · ${date(t.due_date)}` : `до ${date(t.due_date)}` }}</span>
        <span>·</span><span>{{ t.number }}</span><span class="prio">{{ t.priority }}</span>
        <button class="m-x" aria-label="Закрыть" @click="emit('close')"><Icon name="x" /></button>
      </div>

      <div v-if="!editing">
        <h2 class="m-title">{{ t.title }}</h2>
        <p class="m-proj">{{ t.project_name }} · сейчас: {{ stateName[t.state] }}<template v-if="t.approved_at"> · подтвердил(а) {{ t.approved_by }}</template></p>
      </div>
      <form v-else class="stack edit" @submit.prevent="patch({ title: edit.title, description: edit.description })">
        <label class="field">Название<input v-model="edit.title" class="input" maxlength="200" /></label>
        <label class="field">Описание<textarea v-model="edit.description" class="textarea" /></label>
        <div class="row"><button class="btn sm pr" :aria-busy="busy === 'patch'">Сохранить</button><button type="button" class="btn sm ol" @click="editing = false">Отмена</button></div>
      </form>

      <div class="m-big">
        <svg v-if="session.gameOn && reward !== null" class="m-coin game-only" viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="10" fill="#FFC42E" stroke="#7A4A00" stroke-width="1.6"/><circle cx="12" cy="12" r="6.5" fill="none" stroke="#B87A00" stroke-width="1.2"/><path d="M12 8.5v7" stroke="#7A4A00" stroke-width="1.8" stroke-linecap="round"/></svg>
        <span ref="bigEl" class="m-num" :class="{ game: session.gameOn && reward !== null }">{{ session.gameOn && reward !== null ? reward : t.points }}</span>
        <span class="m-unit">
          <template v-if="session.gameOn && reward !== null">золота{{ t.assignee_name ? ` · ${t.assignee_name}` : '' }}<br>{{ t.reward ? 'начислено' : 'после подтверждения' }}</template>
          <template v-else>{{ plural(t.points, 'очко', 'очка', 'очков') }} сложности<br>{{ t.points_approved ? 'оценка согласована' : 'ждёт согласования' }}</template>
        </span>
      </div>

      <div class="m-stages" role="img" :aria-label="`Этап ${stageIndex + 1} из 5: ${stateName[t.state]}`">
        <div v-for="(c, i) in columns" :key="c.state" :class="{ on: i <= stageIndex, cur: i === stageIndex }"><i /><span>{{ c.name }}</span></div>
      </div>

      <div class="m-mets">
        <div><b>{{ t.priority }}</b><span>{{ priorityName[t.priority].toLowerCase() }}</span></div>
        <div><b>{{ t.due_date ? date(t.due_date) : '—' }}</b><span>срок</span></div>
        <div class="who"><span v-if="t.assignee_name" class="av" :style="{ '--hue': avatarHue(t.assignee_name) }" aria-hidden="true">{{ initials(t.assignee_name) }}</span><span><b>{{ t.assignee_name || 'Нет' }}</b><span>исполнитель</span></span></div>
      </div>

      <div class="m-act">
        <button class="btn ol" aria-haspopup="menu" @click="menuAnchor = menuAnchor ? null : ($event.currentTarget as HTMLElement)">Переместить…</button>
        <button v-if="primary" class="btn pr" :disabled="!primary.to || !!primary.block || !!busy" :title="primary.block || undefined" :aria-busy="busy === primary.to"
          @click="primary.to && move(primary.to)">{{ primary.label }}<Icon v-if="primary.to" name="right" :size="16" /></button>
      </div>
      <p v-if="primary?.block" class="block-note">{{ primary.block }}</p>

      <section class="panel2">
        <div class="row"><h3 class="grow">Описание</h3><button v-if="canEdit && !editing" class="btn sm ol" @click="editing = true">Изменить текст</button></div>
        <p v-if="t.description" class="desc">{{ t.description }}</p>
        <p v-else class="desc mute">Описания нет.</p>

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
            <span v-else>{{ date(t.due_date) }}</span>
          </dd></div>
          <div><dt>Автор</dt><dd>{{ t.created_by }}</dd></div>
        </dl>
      </section>

      <section class="panel2" :class="{ unagreed: !t.points_approved }" aria-labelledby="est-h">
        <div class="row wrap">
          <h3 id="est-h" class="grow">Сложность: {{ t.points }} {{ plural(t.points, 'очко', 'очка', 'очков') }}</h3>
          <span v-if="t.state === 'done'" class="pill"><Icon name="lock" :size="12" />Зафиксирована</span>
          <span v-else-if="t.points_approved" class="pill"><Icon name="check" :size="12" />Согласовал(а) {{ t.points_approved_by }}</span>
          <span v-else class="pill warn"><Icon name="alert" :size="12" />Ждёт согласования</span>
        </div>
        <div v-if="t.state !== 'done' && (isLead || (!t.points_approved && canEdit))" class="seg" role="group" aria-label="Изменить сложность">
          <button v-for="p in pointsScale" :key="p" class="pt" :aria-pressed="p === t.points" :disabled="!!busy" @click="isLead ? agree(p) : patch({ points: p })">{{ p }}</button>
        </div>
        <p class="small mute">
          <template v-if="t.state === 'done'">После подтверждения сложность не меняется. Чтобы изменить её, руководитель возвращает задачу в работу — награда при этом отменяется.</template>
          <template v-else-if="isLead">Выбор значения согласует сложность. Каждое изменение попадает в историю и на график объёма бэклога.</template>
          <template v-else-if="t.points_approved">Изменить согласованную сложность может только руководитель.</template>
          <template v-else>Предложите оценку — руководитель согласует её до начала работы.</template>
        </p>
        <button v-if="isLead && !t.points_approved && t.state !== 'done'" class="btn sm pr" :aria-busy="busy === 'agree'" @click="agree()">Согласовать {{ t.points }}</button>
        <div v-if="session.gameOn" class="preview game-only">
          <template v-if="t.reward"><span class="lbl">Начислено</span><span>+{{ t.reward.gold }} золота</span><span>+{{ t.reward.energy }} энергии</span><span>+{{ t.reward.xp }} опыта</span></template>
          <template v-else-if="preview"><span class="lbl">После подтверждения</span><span>≈{{ preview.gold }} золота</span><span>{{ preview.energy }} энергии</span><span>{{ preview.xp }} опыта</span></template>
        </div>
      </section>

      <div class="m-tabs" role="tablist">
        <button role="tab" :aria-selected="tab === 'comments'" @click="tab = 'comments'">Комментарии · {{ detail!.comments.length }}</button>
        <button role="tab" :aria-selected="tab === 'history'" @click="tab = 'history'">История · {{ detail!.history.length }}</button>
        <button role="tab" :aria-selected="tab === 'rewards'" @click="tab = 'rewards'">Начисления · {{ detail!.rewards.length }}</button>
      </div>

      <section v-if="tab === 'comments'" class="list">
        <p v-if="!detail!.comments.length" class="mute small">Комментариев пока нет.</p>
        <article v-for="c in detail!.comments" :key="c.id" class="item">
          <div class="row small"><strong>{{ c.author }}</strong><span class="mute">{{ dateTime(c.created_at) }}</span></div>
          <p class="pre">{{ c.body }}</p>
        </article>
        <form class="stack" @submit.prevent="addComment">
          <label class="field">Новый комментарий<textarea v-model="comment" class="textarea" maxlength="4000" /></label>
          <div><button class="btn sm pr" :disabled="!comment.trim()" :aria-busy="busy === 'comment'"><Icon name="message" :size="14" />Отправить</button></div>
        </form>
      </section>
      <ol v-else-if="tab === 'history'" class="list timeline">
        <li v-for="h in detail!.history" :key="h.id">
          <div class="small"><strong>{{ historyText(h) }}</strong></div>
          <div class="xs mute">{{ h.actor }} · {{ dateTime(h.created_at) }}</div>
          <p v-if="h.note" class="small">{{ h.note }}</p>
        </li>
      </ol>
      <section v-else class="list">
        <p class="small mute">Каждое начисление связано с задачей. Отмена — отдельная запись, исходная помечается отменённой.</p>
        <p v-if="!detail!.rewards.length" class="mute small">Начислений нет: награда появится после подтверждения.</p>
        <article v-for="l in detail!.rewards" :key="l.id" class="item" :class="{ reversed: l.reversed }">
          <div class="row small wrap"><strong>{{ kindName[l.kind] || l.kind }} №{{ l.id }}</strong><span class="mute">{{ l.user }} · {{ dateTime(l.created_at) }}</span>
            <span v-if="l.reversed" class="pill">отменена</span><span v-if="l.reverses_id" class="pill">отменяет №{{ l.reverses_id }}</span></div>
          <div class="row small num wrap"><span>{{ signed(l.gold) }} золота</span><span>{{ signed(l.energy) }} энергии</span><span>{{ signed(l.xp) }} опыта</span></div>
          <p v-if="l.gold_formula !== null && l.gold_formula !== l.gold" class="xs mute">По формуле {{ l.gold_formula }} золота, не поместилось в хранилище: {{ l.gold_formula - l.gold }}.</p>
          <p class="xs mute">{{ l.note }}</p>
        </article>
      </section>
    </div>
  </Modal>
  <MoveMenu v-if="menuAnchor && t" :task="t" :anchor="menuAnchor" :viewer="viewer" @move="onMenuMove" @close="r => { const a = menuAnchor; menuAnchor = null; if (r) a?.focus() }" />
  <ReasonDialog :open="!!reopenTo" title="Вернуть подтверждённую задачу" :text="`Задача вернётся в «${reopenTo ? stateName[reopenTo] : ''}». Награда исполнителя и баллы будут отменены отдельной записью в журнале. Если ресурсы уже потрачены, у исполнителя появится долг, который погасится следующими наградами.`"
    action="Вернуть и отменить награду" destructive @close="reopenTo = null" @confirm="c => confirmReason(reopenTo!, c)" />
  <ReasonDialog :open="returnOpen" title="Вернуть на доработку" text="Задача вернётся в «В работе». Комментарий увидит исполнитель." action="Вернуть на доработку"
    @close="returnOpen = false" @confirm="c => confirmReason('in_progress', c)" />
</template>

<style scoped>
.loading { color: var(--mute); }
.sheet-body { display: flex; flex-direction: column; gap: 0; }
.m-top { display: flex; align-items: center; gap: 8px; font-size: 13px; color: var(--mute); }
.m-top .late { font-weight: 700; color: var(--ink); text-decoration: underline wavy; text-underline-offset: 4px; }
.prio { font-size: 11px; font-weight: 700; padding: 2px 7px; border-radius: var(--radius-pill); background: var(--soft); color: var(--ink); margin-left: 4px; }
.m-x { margin-left: auto; display: grid; place-items: center; width: 40px; height: 40px; border: 0; border-radius: 50%; background: var(--soft); color: var(--ink); cursor: pointer; transition: background 0.2s; }
.m-x:hover { background: var(--line); }
.m-title { font-size: 32px; line-height: 1.08; font-weight: 700; letter-spacing: -0.03em; margin-top: 12px; text-wrap: balance; }
.m-proj { font-size: 14px; color: var(--mute); margin-top: 8px; }
.edit { margin-top: 12px; }
.m-big { display: flex; align-items: flex-end; gap: 12px; margin-top: 22px; flex-wrap: wrap; }
.m-num { font-size: 96px; line-height: 0.8; font-weight: 700; letter-spacing: -0.06em; font-variant-numeric: tabular-nums; }
.m-num.game { font-family: var(--font-game); font-weight: 900; letter-spacing: -0.03em; }
.m-coin { width: 58px; height: 58px; align-self: center; filter: drop-shadow(0 3px 0 rgba(0, 0, 0, 0.2)); }
.m-unit { font-size: 14px; color: var(--mute); padding-bottom: 4px; line-height: 1.35; }
.m-stages { display: grid; grid-template-columns: repeat(5, 1fr); gap: 6px; margin-top: 24px; }
.m-stages i { display: block; height: 8px; border-radius: var(--radius-pill); background: var(--soft); position: relative; overflow: hidden; }
.m-stages i::after { content: ""; position: absolute; inset: 0; background: var(--ink); transform: scaleX(0); transform-origin: left; transition: transform 0.28s var(--ease-emphasized); }
.m-stages .on i::after { transform: none; }
.m-stages span { display: block; font-size: 11px; line-height: 1.25; color: var(--mute); margin-top: 7px; }
.m-stages .cur span { color: var(--ink); font-weight: 700; }
.m-mets { display: grid; grid-template-columns: repeat(3, 1fr); margin-top: 20px; background: var(--glass2); border-radius: 20px; padding: 14px 0; }
.m-mets > div { padding: 0 16px; min-width: 0; }
.m-mets > div + div { border-left: 1px solid var(--line); }
.m-mets b { display: block; font-size: 20px; font-weight: 700; letter-spacing: -0.02em; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.m-mets span { font-size: 12px; color: var(--mute); }
.m-mets .who { display: flex; align-items: center; gap: 10px; }
.m-mets .who b { font-size: 15px; }
.m-mets .who > span:last-child { min-width: 0; }
.m-mets .av { width: 34px; height: 34px; font-size: 11px; box-shadow: none; }
.m-act { display: flex; gap: 10px; margin-top: 22px; justify-content: flex-end; flex-wrap: wrap; }
.block-note { margin-top: 8px; font-size: 12px; color: var(--mute); text-align: right; }
.btn { min-height: 46px; padding: 0 20px; font-size: 14px; }
.btn.sm { min-height: 34px; padding: 0 14px; font-size: 13px; }
.btn.ol { background: transparent; color: var(--ink); box-shadow: inset 0 0 0 1.5px var(--line); }
.btn.ol:hover:not(:disabled) { background: var(--soft); }
.btn.pr { background: var(--pbg); color: var(--pink); box-shadow: none; }
.btn.pr:hover:not(:disabled) { box-shadow: 0 8px 20px rgba(0, 0, 0, 0.18); }
.panel2 { display: flex; flex-direction: column; gap: 12px; margin-top: 18px; padding: 16px 18px; border-radius: 20px; background: var(--glass2); }
.panel2 h3 { font-size: 15px; }
.panel2.unagreed { box-shadow: inset 0 0 0 2px var(--ink); }
.desc { white-space: pre-wrap; font-size: 14px; line-height: 1.55; }
.mute { color: var(--mute); }
.props { display: grid; grid-template-columns: 1fr 1fr; gap: 12px 16px; margin: 0; }
.props dt { font-size: 11px; font-weight: 600; color: var(--mute); text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 4px; }
.props dd { margin: 0; font-size: 14px; }
.input, .select, .textarea { background: var(--soft); color: var(--ink); border-color: var(--line); color-scheme: light; }
.s-in_progress .input, .s-in_progress .select, .s-review .input, .s-review .select, .s-done .input, .s-done .select { color-scheme: dark; }
.select option { color: #1e1e1e; background: #fff; }
.pill { display: inline-flex; align-items: center; gap: 4px; padding: 3px 9px; border-radius: var(--radius-pill); background: var(--soft); font-size: 12px; font-weight: 600; }
.pill.warn { background: var(--ink); color: var(--fill); }
.seg { display: flex; gap: 8px; flex-wrap: wrap; }
.pt { width: 46px; height: 40px; border: 0; border-radius: 12px; background: var(--soft); color: var(--ink); font-weight: 800; cursor: pointer; }
.pt:hover:not(:disabled) { background: var(--line); }
.pt[aria-pressed="true"] { background: var(--pbg); color: var(--pink); }
.pt:disabled { opacity: 0.5; }
.preview { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; font-weight: 700; font-size: 13px; padding-top: 10px; border-top: 1px dashed var(--line); }
.preview .lbl { font-weight: 500; color: var(--mute); }
.m-tabs { display: flex; gap: 2px; margin-top: 22px; padding: 4px; border-radius: var(--radius-pill); background: var(--soft); width: fit-content; max-width: 100%; overflow-x: auto; }
.m-tabs button { border: 0; background: transparent; color: var(--mute); padding: 7px 14px; border-radius: var(--radius-pill); font-size: 13px; font-weight: 600; white-space: nowrap; cursor: pointer; }
.m-tabs button[aria-selected="true"] { background: var(--pbg); color: var(--pink); }
.list { display: flex; flex-direction: column; gap: 10px; padding: 0; margin: 14px 0 0; }
.item { padding: 12px 14px; border-radius: 16px; background: var(--glass2); }
.item.reversed { opacity: 0.65; }
.pre { white-space: pre-wrap; margin-top: 4px; font-size: 14px; }
.timeline { list-style: none; border-left: 2px solid var(--line); padding-left: 16px; }
.timeline li { position: relative; }
.timeline li::before { content: ""; position: absolute; left: -23px; top: 5px; width: 12px; height: 12px; border-radius: 50%; background: var(--ink); box-shadow: 0 0 0 3px var(--fill); }
.field { color: var(--ink); }
@media (max-width: 767px) {
  .m-title { font-size: 26px; }
  .m-num { font-size: 76px; }
  .m-mets > div { padding: 0 10px; }
  .m-mets .av { display: none; }
  .m-act .btn { flex: 1; }
  .props { grid-template-columns: 1fr; }
}
</style>
