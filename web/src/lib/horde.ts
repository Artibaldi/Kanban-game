// Where the horde stands and how a finished wave is replayed. Pure logic: the 3D scene only draws what this decides.
import type { Enemy } from '../types.ts'

/** Lane depth (scene z) for priorities P1…P4: P1 is closest to the walls. */
export const LANES = [-3.4, -5.6, -9.2, -11.6]
/** Where an overdue enemy stands to hit the wall. */
export const WALL_Z = -2.4
const WALL_SLOTS = [-3.2, 3.2, -5.2, 5.2, -2.2, 2.2, -6.2, 6.2]
const MAX_UNITS = 14

export interface Spot { id: string; x: number; z: number; mode: 'idle' | 'bash' }

/** layoutHorde: lane by priority, spread across the lane in the given order; overdue enemies go to the wall. */
export function layoutHorde(enemies: Pick<Enemy, 'task_id' | 'row' | 'overdue' | 'at_gates'>[]): Spot[] {
  const lanes: Record<number, typeof enemies> = { 1: [], 2: [], 3: [], 4: [] }
  const out: Spot[] = []
  let slot = 0
  for (const e of enemies) {
    if (e.overdue) {
      const x = WALL_SLOTS[slot % WALL_SLOTS.length]
      out.push({ id: e.task_id, x, z: WALL_Z - Math.floor(slot / WALL_SLOTS.length) * 0.9, mode: 'bash' })
      slot++
    } else lanes[Math.min(4, Math.max(1, e.row))].push(e)
  }
  for (const row of [1, 2, 3, 4]) {
    const list = lanes[row], n = list.length
    list.forEach((e, i) => {
      const x = n === 1 ? 1.2 : -6 + (12 * i) / (n - 1)
      // A crowded lane gets a second rank behind the first.
      const z = LANES[row - 1] - (n > 7 && i % 2 ? 0.9 : 0) + (e.at_gates ? 0.7 : 0)
      out.push({ id: e.task_id, x: Math.round(x * 100) / 100, z, mode: 'idle' })
    })
  }
  return out
}

/** chunkPoints: split points into enemy-sized pieces (8, 5, 3, 2, 1), never more than `max` pieces. */
export function chunkPoints(points: number, max = MAX_UNITS): number[] {
  if (points <= 0 || max <= 0) return []
  const out: number[] = []
  let rest = points
  for (const size of [8, 5, 3, 2, 1]) while (rest >= size) { out.push(size); rest -= size }
  if (out.length <= max) return out
  const base = Math.floor(points / max), extra = points % max
  return Array.from({ length: max }, (_, i) => base + (i < extra ? 1 : 0))
}

export type Kind = 'goblin' | 'knight' | 'giant' | 'dragon'
/** kindFor: the same size rule as the server's EnemyKind. */
export const kindFor = (points: number): Kind => (points <= 2 ? 'goblin' : points === 3 ? 'knight' : points <= 5 ? 'giant' : 'dragon')

export interface Unit { points: number; kind: Kind; lane: number; killed: boolean; overdue: boolean }

/** waveRoster: a horde that adds up to the wave — defeated units are exactly the closed points. */
export function waveRoster(w: { planned: number; closed: number; overdue: number }): Unit[] {
  if (w.planned <= 0) return []
  const closed = Math.min(w.closed, w.planned), open = w.planned - closed
  const killedCap = closed <= 0 ? 0 : open <= 0 ? MAX_UNITS : Math.min(MAX_UNITS - 1, Math.max(1, Math.round((MAX_UNITS * closed) / w.planned)))
  const killed = chunkPoints(closed, killedCap)
  const alive = chunkPoints(open, MAX_UNITS - killed.length)
  const units: Unit[] = []
  killed.forEach((p, i) => units.push({ points: p, kind: kindFor(p), lane: (i % 4) + 1, killed: true, overdue: false }))
  alive.forEach((p, i) => units.push({ points: p, kind: kindFor(p), lane: ((i + killed.length) % 4) + 1, killed: false, overdue: i < w.overdue }))
  return units
}

/**
 * crownTargets: for each crown the wave earned, how many defeated points win it. Crowns 1 and 2 come at the
 * thresholds of the plan, crown 3 with the last defeated point. Points, not kill order: enemies fall in any order.
 */
export function crownTargets(roster: Unit[], crowns: number, thresholds: number[]): number[] {
  const planned = roster.reduce((n, u) => n + u.points, 0)
  const killed = roster.filter(u => u.killed).reduce((n, u) => n + u.points, 0)
  if (!killed || crowns <= 0) return []
  const out: number[] = []
  for (let c = 0; c < Math.min(3, crowns); c++) out.push(c === 2 ? killed : Math.min(killed, thresholds[c] * planned))
  return out
}
