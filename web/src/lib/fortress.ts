// Pure helpers for the team fortress screen: enemy names, Russian plurals and the order of the horde.
import type { Enemy, WaveNow } from '../types'

const names: Record<Enemy['kind'], string> = { goblin: 'Гоблин', knight: 'Рыцарь', giant: 'Великан', dragon: 'Дракон' }

export function enemyName(kind: Enemy['kind']): string { return names[kind] }

function plural(n: number, one: string, few: string, many: string): string {
  const m10 = n % 10, m100 = n % 100
  if (m10 === 1 && m100 !== 11) return one
  if (m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14)) return few
  return many
}

export function crownsText(n: number): string { return `${n} ${plural(n, 'корона', 'короны', 'корон')}` }

export function daysLeftText(n: number): string {
  return n <= 0 ? 'последний день' : `${n} ${plural(n, 'день', 'дня', 'дней')}`
}

// enemyRows puts the horde into 4 rows by priority; within a row the overdue come first, then those at the gates,
// then by due date, and enemies without a due date last.
export function enemyRows(enemies: Enemy[]): Enemy[][] {
  const rows: Enemy[][] = [[], [], [], []]
  for (const e of enemies) rows[Math.min(4, Math.max(1, e.row)) - 1].push(e)
  const rank = (e: Enemy) => (e.overdue ? 0 : e.at_gates ? 1 : 2)
  for (const r of rows) {
    r.sort((a, b) => rank(a) - rank(b) || (a.due_date ?? '9999').localeCompare(b.due_date ?? '9999'))
  }
  return rows
}

export function waveProgress(w: Pick<WaveNow, 'planned' | 'closed'>): number {
  return w.planned > 0 ? Math.min(1, w.closed / w.planned) : 0
}
