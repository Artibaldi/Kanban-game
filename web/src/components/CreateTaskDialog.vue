<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import Modal from '../ui/Modal.vue'
import { api } from '../api'
import { pointsScale, priorityName } from '../lib/board'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import type { Project, Task, TeamUser } from '../types'

const props = defineProps<{ open: boolean; projects: Project[]; users: TeamUser[]; project?: string }>()
const emit = defineEmits<{ close: []; created: [task: Task] }>()
const session = useSession()
const toasts = useToasts()
const busy = ref(false)
const touched = ref(false)
const form = reactive({ project_id: '', title: '', description: '', priority: 'P3', points: 3, due_date: '', assignee_id: '', state: 'backlog' })
watch(() => props.open, v => {
  if (v) Object.assign(form, { project_id: props.project || props.projects[0]?.id || '', title: '', description: '', priority: 'P3', points: 3, due_date: '', assignee_id: '', state: 'backlog' })
  touched.value = false
})
const teamUsers = computed(() => props.users.filter(u => u.team_id === props.projects.find(p => p.id === form.project_id)?.team_id))
async function submit() {
  touched.value = true
  if (!form.title.trim() || !form.project_id) return
  busy.value = true
  try {
    const t = await api.post<Task>('/tasks', { ...form, due_date: form.due_date || null, assignee_id: form.assignee_id || null })
    toasts.push({ kind: 'success', title: `Создана ${t.number}`, text: t.points_approved ? 'Сложность согласована.' : 'Сложность ждёт согласования руководителем.' })
    emit('created', t)
  } catch (e) { toasts.error(e) } finally { busy.value = false }
}
</script>

<template>
  <Modal :open="open" title="Новая задача" wide @close="emit('close')">
    <form class="form" @submit.prevent="submit">
      <label class="field full">Название
        <input v-model="form.title" class="input" maxlength="200" :aria-invalid="touched && !form.title.trim()" autofocus />
        <span v-if="touched && !form.title.trim()" class="hint" style="color: var(--color-danger)">Введите название.</span>
      </label>
      <label class="field full">Описание<textarea v-model="form.description" class="textarea" /></label>
      <label class="field">Проект
        <select v-model="form.project_id" class="select"><option v-for="p in projects" :key="p.id" :value="p.id">{{ p.key }} — {{ p.name }}</option></select>
      </label>
      <label class="field">Исполнитель
        <select v-model="form.assignee_id" class="select">
          <option value="">Не назначен</option>
          <option v-for="u in teamUsers" :key="u.id" :value="u.id" :disabled="!session.isLead && u.id !== session.me?.id">{{ u.display_name }}</option>
        </select>
      </label>
      <fieldset class="field full points">
        <legend>Сложность, баллы <span class="hint">— согласуется до начала работы</span></legend>
        <div class="seg" role="radiogroup">
          <label v-for="p in pointsScale" :key="p" :class="{ on: form.points === p }"><input v-model="form.points" type="radio" :value="p" class="sr-only" />{{ p }}</label>
        </div>
      </fieldset>
      <label class="field">Приоритет
        <select v-model="form.priority" class="select"><option v-for="(n, k) in priorityName" :key="k" :value="k">{{ k }} — {{ n }}</option></select>
      </label>
      <label class="field">Срок<input v-model="form.due_date" type="date" class="input" /></label>
      <label class="field">Колонка
        <select v-model="form.state" class="select"><option value="backlog">Бэклог</option><option value="ready">К выполнению</option></select>
      </label>
      <p class="full small muted">{{ session.isLead ? 'Вы руководитель: сложность будет согласована сразу.' : 'Руководитель согласует сложность перед началом работы. Дробить задачу ради рейтинга бессмысленно: оценку проверяет руководитель.' }}</p>
    </form>
    <template #footer>
      <button class="btn" @click="emit('close')">Отмена</button>
      <button class="btn primary" :aria-busy="busy" :disabled="busy" @click="submit">Создать задачу</button>
    </template>
  </Modal>
</template>

<style scoped>
.form { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-4); }
.full { grid-column: 1 / -1; }
fieldset { border: 0; padding: 0; margin: 0; }
legend { padding: 0; margin-bottom: var(--space-1); }
.seg { display: flex; gap: var(--space-2); }
.seg label { display: grid; place-items: center; width: 48px; height: 40px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); font-weight: 800; cursor: pointer; }
.seg label:hover { background: var(--color-action-secondary-hover); }
.seg label.on { background: var(--color-action-primary); border-color: var(--color-action-primary); color: var(--color-action-primary-text); }
.seg label:focus-within { outline: 2px solid var(--color-focus); outline-offset: 2px; }
@media (max-width: 600px) { .form { grid-template-columns: 1fr; } }
</style>
