import { test } from 'node:test'
import assert from 'node:assert/strict'
import { marketQuote } from '../src/lib/market.ts'

const s = { gold_per_point: 40, energy_per_point: 10 } // 1 energy = 4 gold

test('the quote matches the server: rate from work rewards, minus the fee', () => {
  assert.equal(marketQuote('gold', 100, s, 30), 17)
  assert.equal(marketQuote('energy', 10, s, 30), 28)
  assert.equal(marketQuote('gold', 100, s, 10), 22)
})

test('nothing for nonsense amounts', () => {
  assert.equal(marketQuote('gold', 3, s, 30), 0)
  assert.equal(marketQuote('gold', 0, s, 30), 0)
  assert.equal(marketQuote('gold', -10, s, 30), 0)
  assert.equal(marketQuote('gold', Number.NaN, s, 30), 0)
})

test('whole units only: the server takes integers', () => {
  assert.equal(marketQuote('gold', 100.5, s, 30), 0)
})
