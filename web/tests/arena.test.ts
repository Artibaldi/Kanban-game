import { test } from 'node:test'
import assert from 'node:assert/strict'
import { arenaBadge, hoursLeft, memberAction, myOpen } from '../src/lib/arena.ts'
import type { ArenaMember, ArenaState, Challenge } from '../src/types.ts'

const ch = (x: Partial<Challenge>) => ({ id: 'c', from_id: 'me', from_name: 'Я', to_id: 'm', to_name: 'Мария', open: false, status: 'pending', created_at: '', expires_at: '2026-10-19T12:00:00Z', attacked: false, ...x }) as Challenge
const member = (x: Partial<ArenaMember> = {}) => ({ user_id: 'm', display_name: 'Мария', townhall: 3, accepts_friendly: false, defense: {} as never, ...x }) as ArenaMember

test('each colleague shows the one action that makes sense now', () => {
  assert.deepEqual(memberAction(member(), []), { kind: 'challenge' })
  assert.deepEqual(memberAction(member(), [ch({ status: 'pending' })]), { kind: 'waiting', id: 'c' })
  assert.deepEqual(memberAction(member(), [ch({ status: 'accepted' })]), { kind: 'fight', id: 'c' })
  assert.deepEqual(memberAction(member(), [ch({ status: 'declined' })]), { kind: 'challenge' }, 'after «not now» a new challenge is fine')
  assert.deepEqual(memberAction(member({ accepts_friendly: true }), []), { kind: 'duel' }, 'colleagues who accept duels can be fought at once')
  assert.deepEqual(memberAction(member(), [ch({ status: 'pending', to_id: 'other' })]), { kind: 'challenge' })
})

test('my live open challenge, if any', () => {
  assert.equal(myOpen([ch({ open: true, to_id: null, status: 'closed' })]), null)
  assert.equal(myOpen([ch({ open: true, to_id: null, status: 'open', id: 'o' })])?.id, 'o')
})

test('time left reads in hours', () => {
  const now = new Date('2026-10-18T12:00:00Z')
  assert.equal(hoursLeft('2026-10-19T12:00:00Z', now), '24 ч')
  assert.equal(hoursLeft('2026-10-18T14:30:00Z', now), '2 ч')
  assert.equal(hoursLeft('2026-10-18T12:40:00Z', now), 'меньше часа')
})

test('the badge counts what waits for me: challenges to answer and accepted ones to fight', () => {
  const s = { incoming: [ch({ status: 'pending' }), ch({ status: 'accepted' })], outgoing: [ch({ status: 'accepted' }), ch({ status: 'pending' })], open: [ch({ open: true, attacked: false }), ch({ open: true, attacked: true })] } as ArenaState
  assert.equal(arenaBadge(s), 3, '1 to answer + 1 to fight + 1 open city not yet answered')
  assert.equal(arenaBadge(null), 0)
})
