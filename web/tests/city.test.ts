import { test } from 'node:test'
import assert from 'node:assert/strict'
import { placeBlock } from '../src/lib/city.ts'
import type { BuildingDef, City, Player } from '../src/types.ts'

const def = (x: Partial<BuildingDef>) => ({ type: 'wall', name: 'Стена', description: '', max_count: [10], levels: [{ level: 1, cost: 5, seconds: 3, requires_th: 1, hp: 100 }], defense: true, ...x }) as BuildingDef
const city = { townhall: 1, buildings: [], limits: { wall: { built: 0, allowed: 10 }, road: { built: 0, allowed: 40 } } } as unknown as City
const busy = { gold: 500, builders_busy: 2, debt: false } as Player

test('decor needs no builder; real buildings wait for one', () => {
  assert.match(placeBlock(def({}), city, busy, 2), /строители/)
  assert.equal(placeBlock(def({ type: 'road', decor: true, levels: [{ level: 1, cost: 5, seconds: 0, requires_th: 1, hp: 0 }] }), city, busy, 2), '')
})

test('the defence line names every visible defence', async () => {
  const { defenseLine } = await import('../src/lib/city.ts')
  const d = { wall_hp: 0, tower_hp: 0, tower_dps: 0, towers: 3, walls: 12, townhall_hp: 0, other_hp: 0, townhall_level: 4 }
  assert.equal(defenseLine(d), 'башни 3 · стены 12')
  assert.equal(defenseLine({ ...d, mortars: 1, crossbows: 2 }), 'башни 3 · мортиры 1 · арбалеты 2 · стены 12')
  assert.equal(defenseLine({ ...d, traps: 2 }), 'башни 3 · стены 12 · ловушки 2')
})
