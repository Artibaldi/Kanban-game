import { test } from 'node:test'
import assert from 'node:assert/strict'
import { LANES, WALL_Z, chunkPoints, crownTargets, layoutHorde, waveRoster } from '../src/lib/horde.ts'
import type { Enemy } from '../src/types.ts'

const e = (id: string, row: number, extra: Partial<Enemy> = {}) => ({ task_id: id, row, points: 3, kind: 'knight', overdue: false, at_gates: false, ...extra }) as Enemy

test('enemies stand in their priority lane, spread across it, overdue ones at the wall', () => {
  const spots = layoutHorde([e('a', 1), e('b', 1), e('c', 3), e('d', 2, { overdue: true }), e('f', 4, { at_gates: true })])
  const by = Object.fromEntries(spots.map(s => [s.id, s]))
  assert.equal(by.a.z, LANES[0])
  assert.equal(by.c.z, LANES[2])
  assert.notEqual(by.a.x, by.b.x, 'two in one lane do not overlap')
  assert.equal(by.d.mode, 'bash')
  assert.equal(by.d.z, WALL_Z)
  assert.ok(by.f.z > LANES[3], 'due soon: a step closer to the walls')
  for (const s of spots) assert.ok(Math.abs(s.x) <= 7, 'inside the field')
})

test('layout is stable: the same enemies get the same places', () => {
  const list = [e('a', 1), e('b', 2), e('c', 2)]
  assert.deepEqual(layoutHorde(list), layoutHorde([...list]))
})

test('points split into enemy-sized chunks', () => {
  assert.deepEqual(chunkPoints(0), [])
  assert.deepEqual(chunkPoints(1), [1])
  assert.deepEqual(chunkPoints(13), [8, 5])
  assert.deepEqual(chunkPoints(17), [8, 8, 1])
  assert.equal(chunkPoints(400).length, 14, 'never more than 14 enemies')
  assert.equal(chunkPoints(400).reduce((a, b) => a + b, 0), 400, 'and the points still add up')
})

test('a wave roster defeats exactly the closed points and sends the overdue to the wall', () => {
  const r = waveRoster({ planned: 30, closed: 24, overdue: 1 })
  const sum = (xs: { points: number }[]) => xs.reduce((n, x) => n + x.points, 0)
  assert.equal(sum(r), 30)
  assert.equal(sum(r.filter(u => u.killed)), 24)
  assert.equal(r.filter(u => u.overdue).length, 1)
  assert.ok(r.every(u => !(u.killed && u.overdue)), 'a defeated enemy is not at the wall')
  assert.deepEqual(waveRoster({ planned: 0, closed: 0, overdue: 0 }), [])
})

test('crowns arrive when the defeated points cross each threshold, in whatever order enemies fall', () => {
  const r = waveRoster({ planned: 30, closed: 24, overdue: 1 })
  assert.deepEqual(crownTargets(r, 2, [0.5, 0.8]), [15, 24])
  const all = waveRoster({ planned: 10, closed: 10, overdue: 0 })
  assert.deepEqual(crownTargets(all, 3, [0.5, 0.8]), [5, 8, 10], 'the third crown needs every defeated point')
  assert.deepEqual(crownTargets(r, 0, [0.5, 0.8]), [])
  assert.deepEqual(crownTargets(waveRoster({ planned: 0, closed: 0, overdue: 0 }), 1, [0.5, 0.8]), [])
})
