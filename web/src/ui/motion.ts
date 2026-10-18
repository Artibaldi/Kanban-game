// Motion for the work layer: short, purposeful, and off when the system asks for reduced motion.
// Every effect here is decorative: state changes first, the animation only follows it.

export const ease = {
  out: 'cubic-bezier(.2,0,0,1)',
  inOut: 'cubic-bezier(.4,0,.2,1)',
  exit: 'cubic-bezier(.4,0,1,1)',
  spring: 'cubic-bezier(.34,1.56,.64,1)',
  slam: 'cubic-bezier(.7,0,.84,0)',
}

const query = typeof matchMedia === 'function' ? matchMedia('(prefers-reduced-motion: reduce)') : null
export const reduced = () => !!query?.matches

/** flipFrom: the keyframe that makes an element look like it starts at rect `from` (for grow-from-card). */
export function flipFrom(from: DOMRect, to: DOMRect, radius: number): Keyframe {
  const sx = from.width / to.width, sy = from.height / to.height
  return { transform: `translate(${from.left - to.left}px,${from.top - to.top}px) scale(${sx},${sy})`, borderRadius: `${radius / sx}px / ${radius / sy}px` }
}

const nf = new Intl.NumberFormat('ru-RU')

/** countUp: rolls a number in an element from `from` to `to`. */
export function countUp(el: HTMLElement | null | undefined, from: number, to: number, ms = 500) {
  if (!el) return
  if (reduced() || from === to) { el.textContent = nf.format(to); return }
  const t0 = performance.now()
  const step = (now: number) => {
    if (!el.isConnected) return
    const p = Math.min(1, (now - t0) / ms), e = 1 - Math.pow(1 - p, 3)
    el.textContent = nf.format(Math.round(from + (to - from) * e))
    if (p < 1) requestAnimationFrame(step)
  }
  requestAnimationFrame(step)
}

const coinSvg = '<svg viewBox="0 0 24 24" width="24" height="24" aria-hidden="true"><circle cx="12" cy="12" r="10" fill="#FFC42E" stroke="#7A4A00" stroke-width="2"/><circle cx="12" cy="12" r="6" fill="none" stroke="#B87A00" stroke-width="1.6"/><path d="M12 8.5v7" stroke="#7A4A00" stroke-width="2" stroke-linecap="round"/></svg>'

/**
 * flyCoins: a burst of coins from `from` that arcs into the gold counter. `onLand(i, n)` fires as each coin
 * arrives, so the counter can step up in time with them. Without motion it calls onLand once, at the end.
 */
export function flyCoins(from: Element, to: Element, onLand: (i: number, n: number) => void) {
  if (reduced()) { onLand(0, 1); return }
  const a = from.getBoundingClientRect(), b = to.getBoundingClientRect()
  const x0 = a.left + a.width / 2, y0 = a.top + a.height / 2, bx = b.left + 16, by = b.top + b.height / 2
  const n = 7, burst = 150, fly = 450, total = burst + fly, off = burst / total
  for (let i = 0; i < n; i++) {
    const ang = ((-90 + (i - (n - 1) / 2) * (300 / n)) * Math.PI) / 180
    const dist = 40 + ((i * 17) % 40)
    const ux = x0 + Math.cos(ang) * dist, uy = y0 + Math.sin(ang) * dist * 0.8
    const outer = document.createElement('div'), inner = document.createElement('div')
    outer.style.cssText = `position:fixed;left:${x0 - 12}px;top:${y0 - 12}px;z-index:400;pointer-events:none;width:24px;height:24px`
    inner.innerHTML = coinSvg
    outer.appendChild(inner)
    document.body.appendChild(outer)
    const delay = i * 50
    // X travels with the outer box, Y with the inner one, so the path is a real arc.
    outer.animate([{ transform: 'translateX(0) scale(.6)', easing: ease.out }, { transform: `translateX(${ux - x0}px) scale(1)`, offset: off, easing: 'linear' }, { transform: `translateX(${bx - x0}px) scale(.8)` }], { duration: total, delay, fill: 'both' })
    inner.animate([{ transform: 'translateY(0)', easing: ease.out }, { transform: `translateY(${uy - y0}px)`, offset: off, easing: 'cubic-bezier(.55,0,.75,.2)' }, { transform: `translateY(${by - y0}px)` }], { duration: total, delay, fill: 'both' })
      .onfinish = () => {
        outer.remove()
        to.animate([{ transform: 'scale(1)' }, { transform: 'scale(1.15)' }, { transform: 'scale(1)' }], { duration: 120, easing: ease.out })
        onLand(i, n)
      }
  }
}
