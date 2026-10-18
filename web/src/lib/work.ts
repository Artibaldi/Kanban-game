// Work-layer helpers: what the board and the review queue show around the tasks. No game rules here.
import { isOverdue } from './format.ts'
import { rewardPreview } from './board.ts'
import type { Task, TaskState } from '../types.ts'

/** initials: the first letters of the first two words, for avatars. */
export function initials(name: string | null | undefined) {
  const words = (name || '').trim().split(/\s+/).filter(Boolean)
  if (!words.length) return '?'
  return words.slice(0, 2).map(w => w[0].toUpperCase()).join('')
}

/** avatarHue: a stable colour per person, so the same person looks the same on every card. */
export function avatarHue(name: string) {
  let h = 7
  for (const ch of name) h = (h * 31 + ch.codePointAt(0)!) % 3607
  return (h * 47) % 360
}

export interface BoardStats { planned: number; inProgress: number; review: number; donePoints: number; overdue: number }

/** boardStats: the four counters above the board, plus how many open tasks are past their date. */
export function boardStats(tasks: Task[], now = new Date()): BoardStats {
  const s: BoardStats = { planned: 0, inProgress: 0, review: 0, donePoints: 0, overdue: 0 }
  for (const t of tasks) {
    if (t.state !== 'backlog') s.planned++
    if (t.state === 'in_progress') s.inProgress++
    if (t.state === 'review') s.review++
    if (t.state === 'done') s.donePoints += t.points
    else if (t.state !== 'backlog' && isOverdue(t.due_date, now)) s.overdue++
  }
  return s
}

/** dropZoneAccepts: the zone under the board sends a card to review. */
export const dropZoneAccepts = (from: TaskState) => from === 'ready' || from === 'in_progress'

export const undoApprovalReason = 'Подтверждение отменено сразу после нажатия кнопкой «Отменить».'

/**
 * undoFor: the move that puts a task back where it was. An approval is undone by a reopen with a reason,
 * which the server records as a separate reversal. A reopen is not undone in one tap: that would pay again.
 */
export function undoFor(from: TaskState, task: Task): { to: TaskState; comment?: string } | null {
  if (from === task.state || from === 'done') return null
  if (task.state === 'done') return { to: from, comment: undoApprovalReason }
  return { to: from }
}

/** waitText: how long a task has been waiting in review. */
export function waitText(since: string, now = new Date()) {
  const min = Math.floor((now.getTime() - new Date(since).getTime()) / 60000)
  if (min < 1) return 'только что'
  if (min < 60) return `ждёт ${min} мин`
  const h = Math.floor(min / 60)
  if (h < 24) return `ждёт ${h} ч`
  return `ждёт ${Math.floor(h / 24)} дн`
}

type RewardSettings = Parameters<typeof rewardPreview>[2]

/** cardReward: the gold chip on a card — what was paid for a done task, the server's formula preview before that. */
export function cardReward(t: Task, s: RewardSettings | null | undefined, now = new Date()): number | null {
  if (t.reward) return t.reward.gold
  if (!s || t.state === 'backlog' || t.state === 'done') return null
  return rewardPreview(t.points, t.priority, s, !!t.due_date && !isOverdue(t.due_date, now)).gold
}
