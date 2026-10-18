// How a recorded duel is shown: which part of the city each round hits, what is in ruins, how many figures stand.
// The server's rounds are the truth; this only maps them onto the scene.

export type Part = 'wall' | 'tower' | 'townhall' | 'other'
interface HP { wall_hp: number; tower_hp: number; townhall_hp: number; other_hp: number }

const order: [Part, keyof HP][] = [['wall', 'wall_hp'], ['tower', 'tower_hp'], ['townhall', 'townhall_hp'], ['other', 'other_hp']]

/** targetOf: the first part that lost hit points between two states — the simulation strikes in this order. */
export function targetOf(before: HP, after: HP): Part | null {
  for (const [part, key] of order) if (after[key] < before[key]) return part
  return null
}

const eventPart: Record<string, Part> = { wall_breached: 'wall', towers_down: 'tower', townhall_down: 'townhall' }

/**
 * ruinedAt: what lies in ruins. Walls, towers and the town hall fall by the server's events (it decides on exact
 * hit points and only rounds them for display); other buildings have no event and fall at zero.
 */
export function ruinedAt(start: HP, now: HP, events: string[]): Set<Part> {
  const out = new Set<Part>()
  for (const e of events) if (eventPart[e]) out.add(eventPart[e])
  if (start.other_hp > 0 && now.other_hp <= 0) out.add('other')
  return out
}

/** destructionAt: the server's formula — towers, town hall and other buildings count, walls do not. */
export function destructionAt(start: HP, now: HP): number {
  const total = start.tower_hp + start.townhall_hp + start.other_hp
  if (total <= 0) return 0
  return Math.floor(((total - (now.tower_hp + now.townhall_hp + now.other_hp)) / total) * 100)
}

/** shownCount: figures on the field for a troop type (a crowd is represented, not drawn in full). */
export const shownCount = (n: number, max = 8) => Math.min(n, max)

/** aliveShown: how many of the shown figures still stand; one survivor keeps one figure. */
export function aliveShown(alive: number, total: number, shown: number) {
  if (alive <= 0 || total <= 0) return 0
  return Math.max(1, Math.round((alive / total) * shown))
}
