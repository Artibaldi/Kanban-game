import { test } from 'node:test'
import assert from 'node:assert/strict'
import { avatarHue, boardStats, dropZoneAccepts, initials, undoFor, waitText } from '../src/lib/work.ts'
import type { Task } from '../src/types.ts'

const t = (state: string, points = 3, due: string | null = null) => ({ id: state + points, state, points, due_date: due }) as unknown as Task

test('initials take the first letters of the first two words', () => {
  assert.equal(initials('Алексей К.'), 'АК')
  assert.equal(initials('наталья петрова'), 'НП')
  assert.equal(initials('Ирина'), 'И')
  assert.equal(initials('  '), '?')
  assert.equal(initials(null), '?')
})

test('avatar hue is stable and in range', () => {
  assert.equal(avatarHue('Мария С.'), avatarHue('Мария С.'))
  assert.notEqual(avatarHue('Мария С.'), avatarHue('Олег Т.'))
  for (const n of ['a', 'Дмитрий Л.', '']) { const h = avatarHue(n); assert.ok(h >= 0 && h < 360) }
})

test('board stats count the plan, the work, the queue and the delivered points', () => {
  const now = new Date('2026-10-18T12:00:00')
  const s = boardStats([t('backlog', 5), t('ready', 2, '2026-10-17'), t('in_progress', 3), t('in_progress', 8, '2026-10-30'), t('review', 5), t('done', 3, '2026-10-01'), t('done', 2)], now)
  assert.deepEqual(s, { planned: 6, inProgress: 2, review: 1, donePoints: 5, overdue: 1 })
})

test('the drop zone takes cards that can be sent to review', () => {
  assert.equal(dropZoneAccepts('ready'), true)
  assert.equal(dropZoneAccepts('in_progress'), true)
  assert.equal(dropZoneAccepts('backlog'), false)
  assert.equal(dropZoneAccepts('review'), false)
  assert.equal(dropZoneAccepts('done'), false)
})

test('undo is the reverse move; an approval is undone by a reopen with a reason', () => {
  assert.deepEqual(undoFor('ready', t('in_progress')), { to: 'ready' })
  assert.deepEqual(undoFor('in_progress', t('review')), { to: 'in_progress' })
  const u = undoFor('review', t('done'))
  assert.equal(u?.to, 'review')
  assert.ok(u?.comment && u.comment.length > 0)
  assert.equal(undoFor('done', t('in_progress')), null, 'a reopen is not undone in one tap: it would pay the reward again')
  assert.equal(undoFor('review', t('review')), null, 'a reorder needs no undo')
})

test('review wait reads like a person would say it', () => {
  const now = new Date('2026-10-18T12:00:00Z')
  assert.equal(waitText('2026-10-18T11:59:30Z', now), 'только что')
  assert.equal(waitText('2026-10-18T11:20:00Z', now), 'ждёт 40 мин')
  assert.equal(waitText('2026-10-18T07:00:00Z', now), 'ждёт 5 ч')
  assert.equal(waitText('2026-10-16T12:00:00Z', now), 'ждёт 2 дн')
})

test('card reward is the paid gold for a done task and the preview otherwise', async () => {
  const { cardReward } = await import('../src/lib/work.ts')
  const s = { gold_per_point: 20, energy_per_point: 5, xp_per_point: 10, priority_mult: { P1: 1.5, P2: 1.2, P3: 1, P4: 0.8 }, on_time_bonus: 0.1 }
  const now = new Date('2026-10-18T12:00:00')
  const paid = { state: 'done', points: 3, priority: 'P3', due_date: null, reward: { gold: 77, energy: 1, xp: 1 } } as unknown as Task
  assert.equal(cardReward(paid, s, now), 77)
  const open = { state: 'in_progress', points: 3, priority: 'P3', due_date: '2026-10-20', reward: null } as unknown as Task
  assert.equal(cardReward(open, s, now), 66, 'on time: 20 × 3 × 1 × 1.1')
  assert.equal(cardReward({ ...open, due_date: '2026-10-01' }, s, now), 60, 'late: no on-time bonus')
  assert.equal(cardReward(open, null, now), null, 'no config yet: no chip')
  assert.equal(cardReward({ ...open, state: 'backlog' } as Task, s, now), null, 'backlog is not in the plan: no chip')
})

test('drop position: project-local index, optimistic rank and no-op detection', async () => {
  const { projectIndex, rankAt, sameSpot } = await import('../src/lib/work.ts')
  const c = (id: string, project_id: string, rank: number) => ({ id, project_id, rank, state: 'ready' }) as unknown as Task
  const col = [c('a', 'P', 1024), c('b', 'Q', 1024), c('m', 'P', 2048), c('d', 'P', 3072)]
  const moving = col[2]
  // Others in view: a(P), b(Q), d(P). Dropping before d (index 2) is position 1 among P's other tasks.
  assert.equal(projectIndex(col, moving, 2), 1)
  assert.equal(projectIndex(col, moving, 0), 0)
  assert.equal(projectIndex(col, moving, 3), 2)
  assert.equal(projectIndex(col, c('x', 'R', 0), 1), 0, 'a project with no other cards here starts at 0')
  // The optimistic rank sits between the neighbours, so the card shows where it was dropped.
  assert.ok(rankAt(col, moving, 0) < 1024)
  const r = rankAt(col, moving, 2)
  assert.ok(r > 1024 && r < 3072)
  assert.ok(rankAt(col, moving, 3) > 3072)
  // Dropping a card where it already is changes nothing and must not touch the server.
  assert.equal(sameSpot(col, moving, 2), true, 'm is the third card: after a and b, before d')
  assert.equal(sameSpot(col, moving, 1), false)
})
