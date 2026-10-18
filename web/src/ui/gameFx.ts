// DOM effects for game screens: coins that hop and fly into a counter, and floating numbers.
import { reduced } from './motion'

/** gameCoins: coins hop out at (x, y) (client coordinates) and fly into `to`; onLand fires once all have arrived. */
export function gameCoins(x: number, y: number, to: Element, n: number, onLand?: () => void) {
  if (reduced()) { onLand?.(); return }
  const r = to.getBoundingClientRect(), tx = r.left + Math.min(40, r.width / 2), ty = r.top + r.height / 2
  let arrived = 0
  const count = Math.max(1, Math.min(n, 6))
  for (let i = 0; i < count; i++) {
    const c = document.createElement('div')
    c.className = 'g-fly'
    c.innerHTML = '<div class="g-coin"></div>'
    document.body.appendChild(c)
    const sx = x + (i - count / 2) * 14, sy = y + ((i * 7) % 11) - 5, hop = sy - 20 - ((i * 5) % 14)
    const a = c.animate([
      { transform: `translate(${sx}px,${sy}px) scale(.4)` },
      { transform: `translate(${sx}px,${hop}px) scale(1.15)`, offset: 0.25 },
      { transform: `translate(${sx}px,${sy}px) scale(1)`, offset: 0.4 },
      { transform: `translate(${tx}px,${ty}px) scale(.8)` },
    ], { duration: 700, delay: i * 70, easing: 'linear', fill: 'forwards' })
    a.onfinish = () => {
      c.remove()
      to.animate([{ transform: 'scale(1)' }, { transform: 'scale(1.06)' }, { transform: 'scale(1)' }], { duration: 160 })
      if (++arrived === count) onLand?.()
    }
  }
}

/** floatText: a big stroked number that pops up and fades, e.g. «+5». */
export function floatText(x: number, y: number, text: string) {
  if (reduced()) return
  const d = document.createElement('div')
  d.className = 'g-float'
  d.textContent = text
  d.style.left = x + 'px'
  d.style.top = y + 'px'
  document.body.appendChild(d)
  setTimeout(() => d.remove(), 1100)
}
