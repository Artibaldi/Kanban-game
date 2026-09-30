// Board rules as the client shows them. The server enforces the same rules; this only decides what to offer.
import type { Task, TaskState } from '../types'

export const columns: { state: TaskState; name: string; hint: string }[] = [
  { state: 'backlog', name: 'Бэклог', hint: 'Идеи и задачи без срока' },
  { state: 'ready', name: 'К выполнению', hint: 'Сложность согласована, можно брать' },
  { state: 'in_progress', name: 'В работе', hint: 'Исполнитель работает' },
  { state: 'review', name: 'На проверке', hint: 'Ждёт подтверждения руководителя' },
  { state: 'done', name: 'Выполнено', hint: 'Подтверждено, награда начислена' },
]
export const stateName = Object.fromEntries(columns.map(c => [c.state, c.name])) as Record<TaskState, string>
export const priorityName: Record<string, string> = { P1: 'Критичный', P2: 'Высокий', P3: 'Обычный', P4: 'Низкий' }
export const pointsScale = [1, 2, 3, 5, 8]

export interface Viewer { id: string; role: string; team_id: string | null }

const isLead = (v: Viewer, t: Task) => v.role === 'admin' || (v.role === 'lead' && v.team_id === t.team_id)

/** Why a move is not allowed, or '' when it is. */
export function moveBlock(t: Task, to: TaskState, v: Viewer): string {
  if (t.state === to) return ''
  const lead = isLead(v, t)
  if (to === 'done') {
    if (t.state !== 'review') return 'Подтвердить можно только задачу на проверке.'
    if (!lead) return 'Выполнение подтверждает руководитель.'
    if (t.assignee_id === v.id) return 'Нельзя подтвердить собственную задачу.'
    return ''
  }
  if (t.state === 'done') return lead ? '' : 'Вернуть подтверждённую задачу может только руководитель.'
  if (!lead && t.assignee_id && t.assignee_id !== v.id) return 'Двигать чужую задачу может только руководитель.'
  if ((to === 'in_progress' || to === 'review') && !t.points_approved) return 'Сначала руководитель согласует сложность.'
  return ''
}

/** The single forward step for keyboard and touch users. */
export function nextStep(t: Task): TaskState | null {
  const order: TaskState[] = ['backlog', 'ready', 'in_progress', 'review', 'done']
  const i = order.indexOf(t.state)
  return i >= 0 && i < order.length - 1 ? order[i + 1] : null
}

/** Reward preview with the same formula as the server (for the card; the server's number is final). */
export function rewardPreview(points: number, priority: string, s: { gold_per_point: number; energy_per_point: number; xp_per_point: number; priority_mult: Record<string, number>; on_time_bonus: number }, onTime: boolean) {
  const mult = s.priority_mult[priority] || 1
  let gold = s.gold_per_point * points * mult
  if (onTime) gold *= 1 + s.on_time_bonus
  return { gold: Math.round(gold), energy: Math.round(s.energy_per_point * points), xp: Math.round(s.xp_per_point * points) }
}
