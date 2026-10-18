/** What the market pays for `amount` of `give`, the same way the server counts it (game.Exchange). 0 = nothing. */
export function marketQuote(give: 'gold' | 'energy', amount: number, s: { gold_per_point: number; energy_per_point: number }, feePct: number): number {
  if (!(amount > 0) || !Number.isFinite(amount)) return 0
  const rate = s.gold_per_point / s.energy_per_point
  const keep = (100 - feePct) / 100
  const got = Math.floor((give === 'gold' ? amount / rate : amount * rate) * keep + 1e-9)
  return Math.max(0, got)
}
