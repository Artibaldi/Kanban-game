import { api } from '../api'
import { useGame } from '../stores/game'
import { useSession } from '../stores/session'
import { useToasts, type ToastAction } from '../stores/toast'
import { useWork } from '../stores/work'
import { stateName } from './board'
import { undoFor } from './work'
import type { ReversalResult, RewardResult, Task, TaskState } from '../types'

export interface MoveResult { task: Task; reward?: RewardResult; reversal?: ReversalResult }
export interface MoveOptions {
  index?: number
  comment?: string
  /** Offer «Отменить» in the notice; `after` runs once the move or its undo has landed, to reload the view. */
  undoable?: boolean
  after?: () => void
  /** Keep quiet about plain moves (the caller shows its own feedback). */
  quiet?: boolean
}

/** moveTask performs a transition and tells the user what happened to the reward. */
export async function moveTask(task: Task, to: TaskState, opts: MoveOptions = {}): Promise<MoveResult | null> {
  const toasts = useToasts()
  const game = useGame()
  const session = useSession()
  const work = useWork()
  try {
    const r = await api.post<MoveResult>(`/tasks/${task.id}/transition`, { to, index: opts.index, comment: opts.comment, version: task.version })
    const back = opts.undoable ? undoFor(task.state, r.task) : null
    const action: ToastAction | undefined = back ? {
      label: 'Отменить',
      run: async () => {
        if (await moveTask(r.task, back.to, { comment: back.comment, quiet: true })) toasts.push({ kind: 'info', title: 'Действие отменено' }, 2500)
        opts.after?.()
      },
    } : undefined
    if (r.reward) {
      toasts.push({
        kind: session.gameOn ? 'reward' : 'success',
        title: session.gameOn ? `+${r.reward.gold} золота: ${r.reward.assignee}` : `${task.number} подтверждена · в «Выполнено»`,
        text: session.gameOn
          ? `${task.number} подтверждена: +${r.reward.energy} энергии, +${r.reward.xp} опыта` + (r.reward.gold < r.reward.gold_formula ? ' (часть не поместилась в хранилище)' : '')
          : `${r.reward.assignee}: +${r.reward.points} балл.`,
        action,
      }, 7000)
    } else if (r.reversal) {
      if (!opts.quiet) toasts.push({
        kind: 'info',
        title: `${task.number} возвращена в «${stateName[to]}»`,
        text: `Награда отменена: −${r.reversal.gold} золота, −${r.reversal.energy} энергии, −${r.reversal.points} балл.` + (r.reversal.debt ? ' Ресурсы были потрачены — у исполнителя долг.' : ''),
      }, 7000)
    } else if (task.state !== to && !opts.quiet) {
      toasts.push({ kind: 'success', title: to === 'review' ? `${task.number} отправлена на проверку` : `${task.number} → «${stateName[to]}»`, action }, 4500)
    }
    if (session.plays) game.refresh()
    if (session.isLead && (task.state === 'review' || to === 'review')) work.refreshReview()
    return r
  } catch (e) {
    toasts.error(e)
    return null
  }
}
