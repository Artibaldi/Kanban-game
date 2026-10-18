import { test } from 'node:test'
import assert from 'node:assert/strict'
import { CAMP_ORDER, campCap, campSpot } from '../src/lib/camp.ts'

test('every troop type has a place in the camp, inside the palisade, without overlapping', () => {
  assert.deepEqual(CAMP_ORDER, ['guard', 'raider', 'archer', 'rider', 'ram'])
  const spots: { type: string; x: number; z: number }[] = []
  for (const type of CAMP_ORDER) for (let i = 0; i < campCap(type); i++) {
    const p = campSpot(type, i)
    assert.ok(Number.isFinite(p.x) && Number.isFinite(p.z), `${type} ${i}`)
    // The palisade is an ellipse 8.2 × 6.2 around z = 0.4; keep a margin, and stay in front of the barracks.
    assert.ok((p.x / 7.4) ** 2 + ((p.z - 0.4) / 5.6) ** 2 < 1, `${type} ${i} outside: ${p.x},${p.z}`)
    assert.ok(p.z > -4.4, `${type} ${i} in the barracks`)
    spots.push({ type, ...p })
  }
  const size = (t: string) => (t === 'ram' ? 0.95 : t === 'rider' ? 0.65 : 0.45)
  for (let a = 0; a < spots.length; a++) for (let b = a + 1; b < spots.length; b++) {
    const s = spots[a], t = spots[b]
    assert.ok(Math.hypot(s.x - t.x, s.z - t.z) >= size(s.type) + size(t.type), `${s.type} and ${t.type} overlap`)
  }
})
