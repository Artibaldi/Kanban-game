import { test } from 'node:test'
import assert from 'node:assert/strict'
import { moveBlock, nextStep, rewardPreview } from '../src/lib/board.ts'
import type { Task } from '../src/types.ts'

const base = { id: 't', team_id: 'team', state: 'review', assignee_id: 'u1', points_approved: true, points: 5, priority: 'P2' } as unknown as Task
const member = { id: 'u1', role: 'user', team_id: 'team' }
const lead = { id: 'l1', role: 'lead', team_id: 'team' }

test('only a lead of the team confirms, and never their own task', () => {
  assert.match(moveBlock(base, 'done', member), /руководитель/)
  assert.equal(moveBlock(base, 'done', lead), '')
  assert.match(moveBlock({ ...base, assignee_id: 'l1' }, 'done', lead), /собственную/)
  assert.match(moveBlock({ ...base, state: 'in_progress' }, 'done', lead), /на проверке/)
})

test('work starts only after the estimate is agreed', () => {
  const t = { ...base, state: 'ready', points_approved: false } as Task
  assert.match(moveBlock(t, 'in_progress', member), /согласует/)
})

test('a done task is reopened only by a lead', () => {
  const t = { ...base, state: 'done' } as Task
  assert.notEqual(moveBlock(t, 'in_progress', member), '')
  assert.equal(moveBlock(t, 'in_progress', lead), '')
})

test('next step and reward preview', () => {
  assert.equal(nextStep({ ...base, state: 'ready' } as Task), 'in_progress')
  assert.equal(nextStep({ ...base, state: 'done' } as Task), null)
  const s = { gold_per_point: 40, energy_per_point: 10, xp_per_point: 25, priority_mult: { P2: 1.1 }, on_time_bonus: 0.1 }
  assert.deepEqual(rewardPreview(5, 'P2', s, true), { gold: 242, energy: 50, xp: 125 })
})
