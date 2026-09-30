import { api } from '../api'
import { useGame } from '../stores/game'
import { useSession } from '../stores/session'
import { useToasts } from '../stores/toast'
import { stateName } from './board'
import type { ReversalResult, RewardResult, Task, TaskState } from '../types'

export interface MoveResult { task: Task; reward?: RewardResult; reversal?: ReversalResult }

/** moveTask performs a transition and tells the user what happened to the reward. */
export async function moveTask(task: Task, to: TaskState, opts: { index?: number; comment?: string } = {}): Promise<MoveResult | null> {
  const toasts = useToasts()
  const game = useGame()
  const session = useSession()
  try {
    const r = await api.post<MoveResult>(`/tasks/${task.id}/transition`, { to, index: opts.index, comment: opts.comment, version: task.version })
    if (r.reward) {
      toasts.push({
        kind: session.gameOn ? 'reward' : 'success',
        title: `${task.number} подтверждена`,
        text: `${r.reward.assignee}: +${r.reward.points} балл. в рейтинг, +${r.reward.gold} золота, +${r.reward.energy} энергии, +${r.reward.xp} опыта` +
          (r.reward.gold < r.reward.gold_formula ? ' (часть не поместилась в хранилище)' : ''),
      }, 7000)
    } else if (r.reversal) {
      toasts.push({
        kind: 'info',
        title: `${task.number} возвращена в «${stateName[to]}»`,
        text: `Награда отменена: −${r.reversal.gold} золота, −${r.reversal.energy} энергии, −${r.reversal.points} балл.` + (r.reversal.debt ? ' Ресурсы были потрачены — у исполнителя долг.' : ''),
      }, 7000)
    } else if (task.state !== to) {
      toasts.push({ kind: 'success', title: `${task.number}: ${stateName[to]}` }, 2500)
    }
    if (session.plays) game.refresh()
    return r
  } catch (e) {
    toasts.error(e)
    return null
  }
}
