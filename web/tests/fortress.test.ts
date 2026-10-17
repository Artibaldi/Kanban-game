import { test } from 'node:test'
import assert from 'node:assert/strict'
import { crownsText, daysLeftText, enemyName, enemyRows, waveProgress } from '../src/lib/fortress.ts'
import type { Enemy, WaveNow } from '../src/types.ts'

const e = (o: Partial<Enemy>): Enemy => ({ task_id: 'x', number: 'T-1', title: 't', points: 3, priority: 'P2', due_date: null, kind: 'knight', row: 2, at_gates: false, overdue: false, ...o })

test('crowns and days in Russian', () => {
  assert.equal(crownsText(1), '1 корона')
  assert.equal(crownsText(3), '3 короны')
  assert.equal(crownsText(0), '0 корон')
  assert.equal(daysLeftText(0), 'последний день')
  assert.equal(daysLeftText(1), '1 день')
  assert.equal(daysLeftText(2), '2 дня')
  assert.equal(daysLeftText(5), '5 дней')
  assert.equal(enemyName('dragon'), 'Дракон')
})

test('enemy rows: overdue first, then at the gates, then by due date', () => {
  const rows = enemyRows([e({ number: 'a', row: 1, due_date: '2026-10-20' }), e({ number: 'n', row: 1 }), e({ number: 'b', row: 1, overdue: true }), e({ number: 'c', row: 1, at_gates: true }), e({ number: 'd', row: 4 })])
  assert.equal(rows.length, 4)
  assert.deepEqual(rows[0].map(x => x.number), ['b', 'c', 'a', 'n'])
  assert.deepEqual(rows[1], [])
  assert.deepEqual(rows[3].map(x => x.number), ['d'])
})

test('wave progress', () => {
  assert.equal(waveProgress({ planned: 30, closed: 24 } as WaveNow), 0.8)
  assert.equal(waveProgress({ planned: 0, closed: 0 } as WaveNow), 0)
  assert.equal(waveProgress({ planned: 5, closed: 9 } as WaveNow), 1)
})
