// The army camp: a yard behind the barracks where trained troops stand in ranks by type.
// New recruits march in from the gate when their training finishes.
import * as THREE from 'three'
import { Kit, eOut, rand } from './kit'
import { buildingModel } from './cityModels'
import { animateTroop, troopModel } from './troops'
import { WOOD, WOOD_D } from './fortressWorld'

const ELEV = 0.62, YAW = Math.PI / 4
const ORDER = ['guard', 'raider', 'ram']
const MAX_SHOWN = 12

interface Soldier { type: string; g: THREE.Group; ph: number; walk?: { from: THREE.Vector3; to: THREE.Vector3; t: number } }

export class CampStage {
  private renderer: THREE.WebGLRenderer
  private scene = new THREE.Scene()
  private cam = new THREE.OrthographicCamera(-1, 1, 1, -1, 0.1, 300)
  private kit: Kit
  private yard: THREE.Group
  private soldiers: Soldier[] = []
  private fire: THREE.Sprite[] = []
  private counts: Record<string, number> = {}
  private tags = new Map<string, HTMLElement>()
  private W = 1; private H = 1; private T = 0
  private raf = 0
  private clock = new THREE.Clock()
  private ro: ResizeObserver
  private running = true
  private tmp = new THREE.Vector3()
  onLost: () => void = () => {}

  constructor(private canvas: HTMLCanvasElement, private labels: HTMLElement, reduced: boolean) {
    const k = (this.kit = new Kit(reduced))
    this.renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true })
    this.renderer.setPixelRatio(Math.min(devicePixelRatio, 2))
    this.renderer.outputColorSpace = THREE.SRGBColorSpace
    this.renderer.setClearColor(0x000000, 0)
    k.lights(this.scene)
    const y = (this.yard = k.grp(this.scene))
    // Meadow, a trodden yard, a palisade with a gate, the barracks, a campfire.
    k.mk(k.rb(26, 0.4, 20, 0.2), k.M(0x7cc242), 0, -0.2, 0, y)
    k.mk(k.rb(14, 0.06, 10, 0.05), k.M(0xc9a26a), 0, 0.02, 0.4, y)
    for (let i = 0; i < 40; i++) {
      const a = (i / 40) * Math.PI * 2, x = Math.cos(a) * 8.2, z = Math.sin(a) * 6.2 + 0.4
      if (z > 5 && Math.abs(x) < 1.6) continue
      const lg = k.mk(k.cyl(0.16, 0.16, 1.1 + (i % 3) * 0.12, 7), k.M(i % 2 ? WOOD : WOOD_D), x, 0.55, z, y)
      k.mk(k.cone(0.16, 0.22, 7), k.M(WOOD), 0, 0.66, 0, lg)
    }
    const b = buildingModel(k, 'barracks', 3, { e: false, s: false }, { smoke: [] })
    b.position.set(-1.2, 0, -5.4)
    b.scale.setScalar(1.6)
    y.add(b)
    const F = k.grp(y, 4.6, 0, -2.6)
    for (let i = 0; i < 6; i++) { const lg = k.mk(k.cyl(0.08, 0.08, 0.7, 6), k.M(WOOD_D), Math.cos(i) * 0.25, 0.12, Math.sin(i) * 0.25, F); lg.rotation.set(Math.sin(i) * 1.2, 0, Math.cos(i) * 1.2) }
    for (let i = 0; i < 7; i++) k.mk(k.geo('fstone', () => new THREE.DodecahedronGeometry(0.14, 0)), k.M(0x9e978a), Math.cos(i) * 0.5, 0.08, Math.sin(i) * 0.5, F)
    for (const [s, c] of [[1.6, 0xff8a2a], [1.0, 0xffd34a]] as const) { const f = k.glow(c, s, 0.9); f.position.set(0, 0.45, 0); F.add(f); this.fire.push(f) }
    for (const [x, z] of [[-6.6, -3], [6.8, 3.6], [-6.2, 4.2], [7.6, -4.4], [-8.4, 0.2]]) {
      const t = k.grp(y, x, 0, z)
      k.mk(k.cyl(0.13, 0.2, 0.9, 7), k.M(WOOD), 0, 0.45, 0, t)
      for (const [r, h, yy] of [[0.95, 1.0, 1.15], [0.75, 0.9, 1.7], [0.5, 0.75, 2.2]]) k.mk(k.cone(r, h, 8), k.M(0x3f8f2e), 0, yy, 0, t)
      k.blob(y, x, z, 1.6, 1.4, 0.4)
    }
    this.ro = new ResizeObserver(() => this.resize())
    this.ro.observe(canvas.parentElement!)
    canvas.addEventListener('webglcontextlost', this.onContextLost)
    document.addEventListener('visibilitychange', this.onVis)
    this.resize()
    this.loop()
  }

  /** setArmy: troops in ranks; new ones march in through the gate. Large counts show a "×N" tag. */
  setArmy(troops: Record<string, number>, animate: boolean) {
    const k = this.kit
    ORDER.forEach((type, row) => {
      const want = Math.min(MAX_SHOWN, troops[type] || 0)
      const mine = this.soldiers.filter(s => s.type === type)
      for (const s of mine.slice(want)) { this.yard.remove(s.g); this.soldiers.splice(this.soldiers.indexOf(s), 1) }
      for (let i = mine.length; i < want; i++) {
        const g = troopModel(k, type)
        g.scale.setScalar(type === 'ram' ? 0.95 : 1.15)
        const to = this.spot(type, row, i)
        const s: Soldier = { type, g, ph: rand() * 6 }
        k.blob(g, 0, 0, type === 'ram' ? 1.6 : 0.9, type === 'ram' ? 1.8 : 0.9, 0.45)
        this.yard.add(g)
        if (animate && !k.reduced && (this.counts[type] ?? 0) > 0) {
          const from = new THREE.Vector3(0, 0, 7.6)
          g.position.copy(from)
          s.walk = { from, to, t: -(i - mine.length) * 0.25 }
        } else g.position.copy(to)
        this.soldiers.push(s)
      }
      let tag = this.tags.get(type)
      if (!tag) { tag = document.createElement('div'); tag.className = 'g3-badge camp'; this.labels.appendChild(tag); this.tags.set(type, tag) }
      const n = troops[type] || 0
      tag.innerHTML = n ? `<b>×${n}</b>` : ''
      tag.style.display = n ? '' : 'none'
    })
    this.counts = { ...troops }
  }

  /** spot: ranks of four per type, rams at the back. */
  private spot(type: string, row: number, i: number) {
    const per = type === 'ram' ? 4 : 6, gap = type === 'ram' ? 2.0 : 1.2
    const col = i % per, line = Math.floor(i / per)
    const x = (col - (per - 1) / 2) * gap
    // Each type has its own block, front to back: guards, raiders, rams.
    const base = [3.4, 1.0, -1.6][row]
    const z = base - line * (type === 'ram' ? 1.9 : 1.0)
    return new THREE.Vector3(x - 1.2, 0, z)
  }

  private resize() {
    const p = this.canvas.parentElement!
    this.W = Math.max(1, p.clientWidth); this.H = Math.max(1, p.clientHeight)
    this.renderer.setSize(this.W, this.H, false)
  }
  private onVis = () => { this.running = document.visibilityState === 'visible'; if (this.running) this.clock.getDelta() }
  private onContextLost = (e: Event) => { e.preventDefault(); cancelAnimationFrame(this.raf); this.onLost() }

  private loop = () => {
    this.raf = requestAnimationFrame(this.loop)
    if (!this.running) return
    const dt = Math.min(this.clock.getDelta(), 0.05)
    if (!this.kit.reduced) this.T += dt
    this.kit.update(dt, this.T)
    for (const s of this.soldiers) {
      let mode: 'idle' | 'walk' = 'idle'
      if (s.walk) {
        s.walk.t += dt
        const q = Math.max(0, Math.min(1, s.walk.t / 1.6))
        s.g.position.lerpVectors(s.walk.from, s.walk.to, eOut(q))
        mode = q < 1 ? 'walk' : 'idle'
        if (q >= 1) s.walk = undefined
      }
      if (!this.kit.reduced) animateTroop(s.g, s.type, this.T, s.ph, mode)
    }
    this.fire.forEach((f, i) => f.scale.setScalar((i ? 1.0 : 1.6) * (1 + Math.sin(this.T * 17 + i) * 0.08 + Math.sin(this.T * 29) * 0.05)))
    const a = this.W / this.H, viewH = a < 0.9 ? 15 / a : Math.max(12.5, 19 / a), d = 80, c = this.cam
    const tg = new THREE.Vector3(0, 0.8, -0.2)
    c.left = (-viewH * a) / 2; c.right = (viewH * a) / 2; c.top = viewH / 2; c.bottom = -viewH / 2
    c.position.set(tg.x + Math.sin(YAW) * Math.cos(ELEV) * d, tg.y + Math.sin(ELEV) * d, tg.z + Math.cos(YAW) * Math.cos(ELEV) * d)
    c.lookAt(tg)
    c.updateProjectionMatrix()
    this.renderer.render(this.scene, c)
    ORDER.forEach((type, row) => {
      const tag = this.tags.get(type)
      if (!tag || tag.style.display === 'none') return
      this.tmp.copy(this.spot(type, row, 0)).setY(type === 'ram' ? 2.2 : 2.0).x -= 0.9
      this.tmp.project(c)
      tag.style.transform = `translate(${((this.tmp.x + 1) / 2) * this.W}px,${((1 - this.tmp.y) / 2) * this.H}px) translate(-50%,-100%)`
    })
  }

  dispose() {
    cancelAnimationFrame(this.raf)
    this.ro.disconnect()
    this.canvas.removeEventListener('webglcontextlost', this.onContextLost)
    document.removeEventListener('visibilitychange', this.onVis)
    this.labels.replaceChildren()
    this.kit.dispose()
    this.renderer.dispose()
    this.renderer.forceContextLoss()
  }
}
