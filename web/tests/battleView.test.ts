import { test } from 'node:test'
import assert from 'node:assert/strict'
import { aliveShown, destructionAt, ruinedAt, shownCount, targetOf } from '../src/lib/battleView.ts'

const defense = { wall_hp: 400, tower_hp: 600, townhall_hp: 1000, other_hp: 500 }
const round = (wall: number, tower: number, th: number, other: number) => ({ wall_hp: wall, tower_hp: tower, townhall_hp: th, other_hp: other })

test('the target of a round is the first category that lost hit points, in the order the simulation strikes', () => {
  const start = round(400, 600, 1000, 500)
  assert.equal(targetOf(start, round(300, 600, 1000, 500)), 'wall')
  assert.equal(targetOf(round(300, 600, 1000, 500), round(0, 550, 1000, 500)), 'wall', 'walls still took the first hits')
  assert.equal(targetOf(round(0, 550, 1000, 500), round(0, 200, 1000, 500)), 'tower')
  assert.equal(targetOf(round(0, 0, 1000, 500), round(0, 0, 700, 500)), 'townhall')
  assert.equal(targetOf(round(0, 0, 0, 500), round(0, 0, 0, 100)), 'other')
  assert.equal(targetOf(round(0, 0, 0, 100), round(0, 0, 0, 100)), null, 'nothing was hit')
})

test('walls, towers and the town hall fall by the server events, not by rounded hit points', () => {
  // The server rounds hit points for display but decides on exact values: 0.3 left shows as 0 and is not down.
  assert.deepEqual([...ruinedAt(defense, round(0, 0, 0, 500), [])], [], 'no event yet: nothing in ruins')
  assert.deepEqual([...ruinedAt(defense, round(0, 0, 0, 500), ['wall_breached', 'towers_down'])].sort(), ['tower', 'wall'])
  assert.deepEqual([...ruinedAt(defense, round(0, 0, 0, 0), ['wall_breached', 'towers_down', 'townhall_down'])].sort(), ['other', 'tower', 'townhall', 'wall'])
  assert.deepEqual([...ruinedAt({ ...defense, other_hp: 0 }, round(0, 0, 0, 0), [])], [], 'nothing to ruin is not a ruin')
})

test('destruction matches the server: walls do not count', () => {
  assert.equal(destructionAt(defense, round(0, 600, 1000, 500)), 0)
  assert.equal(destructionAt(defense, round(400, 0, 1000, 500)), 28, '600 of 2100, floored')
  assert.equal(destructionAt(defense, round(0, 0, 0, 0)), 100)
  assert.equal(destructionAt({ wall_hp: 0, tower_hp: 0, townhall_hp: 0, other_hp: 0 }, round(0, 0, 0, 0)), 0)
})

test('figures on the field stand for the troops, and the last one falls only with the last soldier', () => {
  assert.equal(shownCount(3), 3)
  assert.equal(shownCount(30), 8)
  assert.equal(aliveShown(14, 14, 8), 8)
  assert.equal(aliveShown(1, 14, 8), 1, 'one soldier left still shows one figure')
  assert.equal(aliveShown(0, 14, 8), 0)
  assert.equal(aliveShown(7, 14, 8), 4)
})
