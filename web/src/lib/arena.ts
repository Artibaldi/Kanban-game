// The Arena in the fortress: challenges to friendly duels. Pure helpers; the server enforces the rules.
import type { ArenaMember, ArenaState, Challenge } from '../types.ts'

export type MemberAction = { kind: 'challenge' } | { kind: 'waiting'; id: string } | { kind: 'fight'; id: string } | { kind: 'duel' }

/** memberAction: the one thing to do with a colleague now — challenge, wait for the answer, fight, or duel at once. */
export function memberAction(m: ArenaMember, outgoing: Challenge[]): MemberAction {
  const mine = outgoing.find(c => c.to_id === m.user_id && (c.status === 'pending' || c.status === 'accepted'))
  if (mine?.status === 'accepted') return { kind: 'fight', id: mine.id }
  if (mine) return { kind: 'waiting', id: mine.id }
  return m.accepts_friendly ? { kind: 'duel' } : { kind: 'challenge' }
}

/** myOpen: my open challenge to the team while it is live. */
export const myOpen = (outgoing: Challenge[]) => outgoing.find(c => c.open && c.status === 'open') ?? null

/** hoursLeft: how long a challenge still waits. */
export function hoursLeft(expires: string, now = new Date()) {
  const h = Math.floor((new Date(expires).getTime() - now.getTime()) / 3600000)
  return h < 1 ? 'меньше часа' : `${h} ч`
}

/** arenaBadge: what waits for me — challenges to answer, accepted ones to fight, open cities not yet answered. */
export function arenaBadge(s: Pick<ArenaState, 'incoming' | 'outgoing' | 'open'> | null) {
  if (!s) return 0
  return s.incoming.filter(c => c.status === 'pending').length + s.outgoing.filter(c => c.status === 'accepted').length + s.open.filter(c => !c.attacked).length
}
