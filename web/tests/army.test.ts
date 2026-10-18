import { test } from 'node:test'
import assert from 'node:assert/strict'
import { researchBlock, troopMult } from '../src/lib/army.ts'

test('each level above 1 adds 10%, like the server', () => {
  assert.equal(troopMult(1), 1)
  assert.equal(troopMult(0), 1)
  assert.equal(troopMult(3), 1.2)
})

const army = (x: object) => ({ levels: { guard: 1, raider: 2 }, research: null, max_troop_level: 2, research_costs: [{ level: 2, cost: 400, seconds: 20 }, { level: 3, cost: 900, seconds: 35 }, { level: 4, cost: 1600, seconds: 50 }], ...x })

test('why a troop cannot be raised now', () => {
  assert.equal(researchBlock('guard', army({}), 1000, false), '')
  assert.match(researchBlock('guard', army({ max_troop_level: 1 }), 1000, false), /мастерскую/)
  assert.match(researchBlock('raider', army({}), 1000, false), /Улучшите мастерскую/)
  assert.match(researchBlock('guard', army({ research: { type: 'raider', target_level: 3, started_at: '', finish_at: '' } }), 1000, false), /уже работает/)
  assert.match(researchBlock('guard', army({}), 100, false), /золота/)
  assert.match(researchBlock('guard', army({}), 1000, true), /долг/)
  assert.match(researchBlock('guard', army({ levels: { guard: 4 }, max_troop_level: 4 }), 9000, false), /максимальн/)
})
