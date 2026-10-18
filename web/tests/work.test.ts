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
