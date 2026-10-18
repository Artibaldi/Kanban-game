import { test } from 'node:test'
import assert from 'node:assert/strict'
import { donateBlock } from '../src/lib/banner.ts'

const mate = { capacity: 6, held: 2 }

test('sending troops to a banner, like the server checks it', () => {
  assert.equal(donateBlock(mate, 1, 4, 10), '')
  assert.match(donateBlock(mate, 2, 3, 10), /места/)
  assert.match(donateBlock({ capacity: 0, held: 0 }, 1, 1, 10), /знамени/)
  assert.match(donateBlock(mate, 1, 5, 10), /места/)
  assert.match(donateBlock(mate, 1, 3, 2), /Готово только 2/)
  assert.match(donateBlock(mate, 1, 0, 10), /количество/)
  assert.match(donateBlock(mate, 1, 1.5, 10), /количество/)
})
