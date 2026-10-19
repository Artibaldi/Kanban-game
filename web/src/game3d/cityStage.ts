// The player's city in 3D: an island with a grid, buildings by level, construction, placing and selection.
import * as THREE from 'three'
import type { BuildingType } from '../types'
import { Kit, eOutBack, rand } from './kit'
import { buildingModel, scaffold, type Live } from './cityModels'

const ELEV = 0.64, YAW = Math.PI / 4
const camRight = new THREE.Vector3(1, 0, -1).normalize()
export const CELL = 1.8

export interface CityItem { id: string; type: BuildingType; level: number; x: number; y: number; busy: boolean; name: string }

interface Placed { item: CityItem; holder: THREE.Group; model: THREE.Group; live: Live; scaffold?: THREE.Group; joins: string; badge: HTMLElement; height: number }

export class CityStage {
  private renderer: THREE.WebGLRenderer
  private scene = new THREE.Scene()
  private cam = new THREE.OrthographicCamera(-1, 1, 1, -1, 0.1, 300)
  private kit: Kit
  private world: THREE.Group
  private placed = new Map<string, Placed>()
  private selRing: THREE.Mesh
  private cursor: THREE.Mesh
  private freeMarks: THREE.InstancedMesh
  private placing = false
  private occupied = new Set<string>()
  private selected: string | null = null
  private W = 1; private H = 1; private T = 0
  private zoom = 1
  /** Where the camera looks and how much it shows: the built-up part of the grid, with a margin. */
  private fit = { c: new THREE.Vector3(), span: 8 }
  private pan = new THREE.Vector3()
  private down: { x: number; y: number; px: THREE.Vector3; moved: boolean } | null = null
  private raf = 0
  private clock = new THREE.Clock()
  private ro: ResizeObserver
  private running = true
  private puffs: { m: THREE.Mesh; life: number }[] = []
  private smokeT = 0
  private ground: THREE.Mesh
  private groundTex!: THREE.CanvasTexture
  /** The player's land: the centred square they may build on; the rest of the island waits under a haze. */
  private land = { start: 0, size: 0 }
  private tmp = new THREE.Vector3()
  onPick: (id: string | null) => void = () => {}
  onCell: (x: number, y: number) => void = () => {}
  onLost: () => void = () => {}
  onHoverCell: (c: { x: number; y: number }) => void = () => {}

  /** Per-frame hook for layers drawn on top of the city (the duel replay). */
  onFrame: (dt: number, t: number) => void = () => {}

  /** `interactive: false` makes a read-only view: no picking, no placing, still pans. */
  constructor(private canvas: HTMLCanvasElement, private labels: HTMLElement, reduced: boolean, private N: number, private interactive = true) {
    this.kit = new Kit(reduced)
    this.renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true })
    this.renderer.setPixelRatio(Math.min(devicePixelRatio, 2))
    this.renderer.outputColorSpace = THREE.SRGBColorSpace
    this.renderer.setClearColor(0x000000, 0)
    this.kit.lights(this.scene)
    const k = this.kit
    this.world = k.grp(this.scene)
    // The island: a checkered meadow on a slab of earth, with a margin for trees.
    const size = N * CELL, m = CELL * 1.6
    const tex = (this.groundTex = k.canvasTex(N * 8, N * 8, () => {}))
    tex.magFilter = THREE.NearestFilter
    this.paintGround()
    k.mk(k.rb(size + m * 2, 0.6, size + m * 2, 0.25), k.M(0x8f5e33), 0, -0.4, 0, this.world)
    k.mk(k.rb(size + m * 2 - 0.2, 0.5, size + m * 2 - 0.2, 0.22), k.M(0xb07a45), 0, -0.05, 0, this.world)
    k.mk(k.rb(size + m * 2, 0.16, size + m * 2, 0.08), k.M(0x6db83a), 0, 0.12, 0, this.world)
    this.ground = k.mk(new THREE.PlaneGeometry(size, size), k.M(0xffffff, { map: tex }), 0, 0.205, 0, this.world)
    this.ground.rotation.x = -Math.PI / 2
    const edge = size / 2 + m / 2
    for (let i = 0; i < 22; i++) {
      const side = i % 4, t = (rand() - 0.5) * (size + m)
      const x = side === 0 ? -edge : side === 1 ? edge : t, z = side === 2 ? -edge : side === 3 ? edge : t
      const g = k.grp(this.world, x, 0.2, z), s = 0.7 + rand() * 0.5
      g.scale.setScalar(s)
      k.mk(k.cyl(0.13, 0.2, 0.9, 7), k.M(0x9a6232), 0, 0.45, 0, g)
      if (i % 3) for (const [r, h, y] of [[0.95, 1.0, 1.15], [0.75, 0.9, 1.7], [0.5, 0.75, 2.2]]) k.mk(k.cone(r, h, 8), k.M(0x3f8f2e), 0, y, 0, g)
      else { k.mk(k.sph(0.62, 12, 10), k.M(0x5da530), 0, 1.35, 0, g); k.mk(k.sph(0.44, 12, 10), k.M(0x4e9a2e), -0.36, 1.15, -0.1, g) }
      k.blob(this.world, x, z, 1.6 * s, 1.4 * s, 0.4).position.y = 0.21
    }
    this.selRing = k.mk(k.geo('csel', () => new THREE.RingGeometry(0.95, 1.12, 4, 1).rotateX(-Math.PI / 2).rotateY(Math.PI / 4)), k.MB(0xffffff, { transparent: true, opacity: 0.95, depthWrite: false }), 0, 0.23, 0, this.world)
    this.selRing.visible = false
    this.cursor = k.mk(k.geo('ccur', () => new THREE.PlaneGeometry(CELL * 0.92, CELL * 0.92).rotateX(-Math.PI / 2)), k.MB(0x5cc02e, { transparent: true, opacity: 0.75, depthWrite: false }, true), 0, 0.22, 0, this.world)
    this.cursor.visible = false
    this.freeMarks = new THREE.InstancedMesh(k.geo('cfree', () => new THREE.PlaneGeometry(CELL * 0.86, CELL * 0.86).rotateX(-Math.PI / 2)), k.MB(0xffffff, { transparent: true, opacity: 0.32, depthWrite: false }), N * N)
    this.freeMarks.visible = false
    this.world.add(this.freeMarks)
    this.ro = new ResizeObserver(() => this.resize())
    this.ro.observe(canvas.parentElement!)
    canvas.addEventListener('pointerdown', this.onDown)
    canvas.addEventListener('pointermove', this.onHover)
    canvas.addEventListener('wheel', this.onWheel, { passive: false })
    canvas.addEventListener('webglcontextlost', this.onContextLost)
    addEventListener('pointermove', this.onMove)
    addEventListener('pointerup', this.onUp)
    document.addEventListener('visibilitychange', this.onVis)
    this.resize()
    this.loop()
  }

  /** For layers drawn on the city: the scene group, the kit, the grid size and the placed buildings. */
  get group() { return this.world }
  get toolkit() { return this.kit }
  get size() { return this.N * CELL }
  get time() { return this.T }
  /** The built-up part of the grid the camera frames: its centre and side. */
  get bounds() { return { c: this.fit.c.clone(), span: this.fit.span } }
  entries() { return [...this.placed.values()].map(p => ({ id: p.item.id, type: p.item.type, holder: p.holder, model: p.model, height: p.height })) }
  /** project: a scene point to canvas pixels (for floating numbers). */
  project(v: THREE.Vector3): [number, number] { return this.toScreen(this.world.localToWorld(v.clone())) }

  /** setLand: which part of the island is the player's; redraws the meadow. */
  setLand(start: number, size: number) {
    if (this.land.start === start && this.land.size === size) return
    this.land = { start, size }
    this.paintGround()
    if (this.placing) this.showFree()
  }
  private inLand(x: number, y: number) {
    const { start, size } = this.land
    return size === 0 || (x >= start && y >= start && x < start + size && y < start + size)
  }
  private paintGround() {
    const c = this.groundTex.image as HTMLCanvasElement, g = c.getContext('2d')!, s = c.width / this.N
    for (let y = 0; y < this.N; y++) for (let x = 0; x < this.N; x++) {
      const mine = this.inLand(x, y)
      g.fillStyle = mine ? ((x + y) % 2 ? '#7CC242' : '#86C94C') : ((x + y) % 2 ? '#A9BC8E' : '#B2C497')
      g.fillRect(x * s, y * s, s, s)
    }
    const { start, size } = this.land
    if (size && size < this.N) {
      g.strokeStyle = 'rgba(255,255,255,0.85)'
      g.lineWidth = 2
      g.setLineDash([6, 4])
      g.strokeRect(start * s + 1, start * s + 1, size * s - 2, size * s - 2)
    }
    this.groundTex.needsUpdate = true
  }

  /** cellPos: the centre of a grid cell in the scene. */
  cellPos(x: number, y: number) { return new THREE.Vector3((x - (this.N - 1) / 2) * CELL, 0.21, (y - (this.N - 1) / 2) * CELL) }

  /** sync: places the buildings; new ones drop in, upgraded ones jump, construction gets scaffolding. */
  sync(items: CityItem[], animate: boolean) {
    const k = this.kit
    const walls = new Set(items.filter(i => i.type === 'wall').map(i => `${i.x},${i.y}`))
    const roads = new Set(items.filter(i => i.type === 'road').map(i => `${i.x},${i.y}`))
    this.occupied = new Set(items.map(i => `${i.x},${i.y}`))
    if (items.length) {
      const xs = items.map(i => i.x), ys = items.map(i => i.y)
      const x0 = Math.min(...xs), x1 = Math.max(...xs), y0 = Math.min(...ys), y1 = Math.max(...ys)
      this.fit.c = this.cellPos((x0 + x1) / 2, (y0 + y1) / 2)
      this.fit.span = Math.min(this.N, Math.max(6, Math.max(x1 - x0, y1 - y0) + 3)) * CELL
    }
    const ids = new Set(items.map(i => i.id))
    for (const [id, p] of this.placed) if (!ids.has(id)) this.removePlaced(p)
    for (const it of items) {
      // Walls and roads join their east/south neighbours of the same kind; trees alternate pine and leafy.
      const same = it.type === 'wall' ? walls : it.type === 'road' ? roads : null
      const joins = same ? { e: same.has(`${it.x + 1},${it.y}`), s: same.has(`${it.x},${it.y + 1}`) } : { e: false, s: it.type === 'tree' && (it.x + it.y) % 2 === 0 }
      const jk = `${joins.e}${joins.s}`
      const was = this.placed.get(it.id)
      const isNew = !was
      const levelUp = !!was && was.item.level !== it.level
      const moved = !!was && (was.item.x !== it.x || was.item.y !== it.y)
      if (was && !levelUp && !moved && was.joins === jk) {
        if (was.item.busy !== it.busy) this.setScaffold(was, it.busy)
        was.item = it
        continue
      }
      if (was) this.removePlaced(was)
      const live: Live = { smoke: [] }
      const model = buildingModel(k, it.type, it.level, joins, live)
      const holder = k.grp(this.world)
      holder.position.copy(this.cellPos(it.x, it.y))
      holder.add(model)
      model.traverse(o => { o.userData.buildingId = it.id })
      const badge = document.createElement('div')
      badge.className = 'g3-badge city'
      this.labels.appendChild(badge)
      holder.updateMatrixWorld(true)
      const height = new THREE.Box3().setFromObject(model).max.y - holder.position.y - this.world.position.y
      const p: Placed = { item: it, holder, model, live, joins: jk, badge, height }
      this.placed.set(it.id, p)
      if (it.busy) this.setScaffold(p, true)
      if (animate && isNew && !k.reduced) {
        // Drop in from above, squash, puff of dust.
        model.position.y = 4
        k.tween(0.4, q => { model.position.y = 4 * (1 - q * q) }, () => {
          k.poof(this.world, holder.position.clone().setY(0.5), 0.6, 4)
          k.tween(0.25, q => { const s = Math.sin(q * Math.PI); model.scale.set(1 + 0.18 * s, 1 - 0.18 * s, 1 + 0.18 * s) })
        })
      } else if (animate && levelUp && !k.reduced) {
        k.tween(0.5, q => { model.scale.setScalar(0.6 + 0.4 * eOutBack(q, 2.4)) })
        k.sparks(this.world, holder.position.clone().setY(1.6), 16, 0xffe27a, 4)
      }
    }
    this.select(this.selected && this.placed.has(this.selected) ? this.selected : null)
    if (this.placing) this.showFree()
  }

  private setScaffold(p: Placed, on: boolean) {
    if (p.scaffold) { p.holder.remove(p.scaffold); p.scaffold = undefined }
    if (!on) return
    p.scaffold = scaffold(this.kit, Math.max(1, p.height + 0.1))
    p.holder.add(p.scaffold)
  }

  private removePlaced(p: Placed) {
    this.world.remove(p.holder)
    // Owned materials (glows) are freed now; shared ones stay cached in the kit.
    p.holder.traverse(o => { const m = (o as THREE.Mesh).material; if (m && !Array.isArray(m)) this.kit.release(m) })
    p.badge.remove()
    this.placed.delete(p.item.id)
  }

  /** setTimers: the countdown over each building under construction (id → text). */
  setTimers(t: Record<string, string>) {
    for (const p of this.placed.values()) {
      const text = t[p.item.id]
      const sel = p.item.id === this.selected
      p.badge.innerHTML = text ? `<span class="timer">${text}</span>` : sel ? `<b>${p.item.level}</b>` : ''
      p.badge.style.display = text || sel ? '' : 'none'
    }
  }

  select(id: string | null) {
    const changed = id !== this.selected
    this.selected = id
    const p = id ? this.placed.get(id) : null
    this.selRing.visible = !!p
    if (p) {
      this.selRing.position.copy(p.holder.position).setY(0.23)
      // The little hop only when the choice changes, not on every reload.
      if (changed && !this.kit.reduced) this.kit.tween(0.25, q => { p.model.position.y = Math.sin(q * Math.PI) * 0.35 })
    }
  }

  /** setPlacing: shows the free cells; clicks then go to onCell. */
  setPlacing(on: boolean) {
    this.placing = on
    this.freeMarks.visible = on
    this.cursor.visible = false
    if (on) this.showFree()
  }
  private showFree() {
    const mtx = new THREE.Matrix4()
    let n = 0
    for (let y = 0; y < this.N; y++) for (let x = 0; x < this.N; x++) {
      if (this.occupied.has(`${x},${y}`) || !this.inLand(x, y)) continue
      const p = this.cellPos(x, y)
      mtx.makeTranslation(p.x, 0.215, p.z)
      this.freeMarks.setMatrixAt(n++, mtx)
    }
    this.freeMarks.count = n
    this.freeMarks.instanceMatrix.needsUpdate = true
  }
  /** setCursor: the highlighted cell while placing (also driven by the keyboard). */
  setCursor(c: { x: number; y: number } | null) {
    if (!c || !this.placing) { this.cursor.visible = false; return }
    const free = !this.occupied.has(`${c.x},${c.y}`) && this.inLand(c.x, c.y)
    this.cursor.position.copy(this.cellPos(c.x, c.y)).setY(0.22)
    ;(this.cursor.material as THREE.MeshBasicMaterial).color.setHex(free ? 0x5cc02e : 0xf04848)
    this.cursor.visible = true
  }

  // —— pointer ——
  private ray(clientX: number, clientY: number) {
    const r = this.canvas.getBoundingClientRect()
    const ray = new THREE.Raycaster()
    ray.setFromCamera(new THREE.Vector2(((clientX - r.left) / this.W) * 2 - 1, -((clientY - r.top) / this.H) * 2 + 1), this.cam)
    return ray
  }
  private cellAt(clientX: number, clientY: number): { x: number; y: number } | null {
    const hit = this.ray(clientX, clientY).intersectObject(this.ground)[0]
    if (!hit) return null
    const local = this.world.worldToLocal(hit.point.clone())
    const x = Math.floor(local.x / CELL + this.N / 2), y = Math.floor(local.z / CELL + this.N / 2)
    return x >= 0 && y >= 0 && x < this.N && y < this.N ? { x, y } : null
  }
  private onDown = (ev: PointerEvent) => {
    if (!ev.isPrimary || ev.button !== 0) return
    this.down = { x: ev.clientX, y: ev.clientY, px: this.pan.clone(), moved: false } }
  private onMove = (ev: PointerEvent) => {
    const d = this.down
    if (!d) return
    const dx = ev.clientX - d.x, dy = ev.clientY - d.y
    if (Math.hypot(dx, dy) > 6) d.moved = true
    if (!d.moved) return
    const k = (this.cam.top - this.cam.bottom) / this.H
    this.pan.copy(d.px).addScaledVector(camRight, -dx * k).add(new THREE.Vector3(-1, 0, -1).normalize().multiplyScalar((dy * k) / Math.sin(ELEV)))
    this.pan.clampLength(0, this.N * CELL * 0.45)
  }
  private onHover = (ev: PointerEvent) => {
    if (!this.placing || this.down?.moved) return
    const c = this.cellAt(ev.clientX, ev.clientY)
    this.setCursor(c)
    if (c) this.onHoverCell(c)
  }
  private onUp = (ev: PointerEvent) => {
    const d = this.down
    this.down = null
    if (!d || d.moved || !this.interactive) return
    if (this.placing) {
      const c = this.cellAt(ev.clientX, ev.clientY)
      if (c && !this.occupied.has(`${c.x},${c.y}`) && this.inLand(c.x, c.y)) this.onCell(c.x, c.y)
      return
    }
    const hit = this.ray(ev.clientX, ev.clientY).intersectObjects([...this.placed.values()].map(p => p.model), true)[0]
    if (hit) { this.onPick(hit.object.userData.buildingId as string); return }
    // A tap on the cell of a building selects it too: small walls are hard to hit.
    const c = this.cellAt(ev.clientX, ev.clientY)
    const p = c && [...this.placed.values()].find(q => q.item.x === c.x && q.item.y === c.y)
    this.onPick(p ? p.item.id : null)
  }
  private onWheel = (ev: WheelEvent) => { ev.preventDefault(); this.zoom = Math.min(1.6, Math.max(0.6, this.zoom * (ev.deltaY > 0 ? 1.08 : 1 / 1.08))) }
  private onVis = () => { this.running = document.visibilityState === 'visible'; if (this.running) this.clock.getDelta() }
  private onContextLost = (e: Event) => { e.preventDefault(); cancelAnimationFrame(this.raf); this.onLost() }

  private resize() {
    const p = this.canvas.parentElement!
    this.W = Math.max(1, p.clientWidth); this.H = Math.max(1, p.clientHeight)
    this.renderer.setSize(this.W, this.H, false)
  }

  private toScreen(v: THREE.Vector3): [number, number] {
    this.tmp.copy(v).project(this.cam)
    return [((this.tmp.x + 1) / 2) * this.W, ((1 - this.tmp.y) / 2) * this.H]
  }

  private loop = () => {
    this.raf = requestAnimationFrame(this.loop)
    if (!this.running) return
    const dt = Math.min(this.clock.getDelta(), 0.05)
    if (!this.kit.reduced) this.T += dt
    this.kit.update(dt, this.T)
    this.onFrame(dt, this.T)
    // Chimney smoke and the citadel crystal.
    if (!this.kit.reduced) {
      this.smokeT += dt
      if (this.smokeT > 2.4) {
        this.smokeT = 0
        for (const p of this.placed.values()) for (const s of p.live.smoke) {
          const pos = s.getWorldPosition(new THREE.Vector3())
          const m = this.kit.mk(this.kit.sph(0.16, 8, 6), this.kit.MB(0xf2f2f2, { transparent: true, opacity: 0.9 }, true), pos.x, pos.y, pos.z, this.scene)
          this.puffs.push({ m, life: 0 })
        }
      }
      for (let i = this.puffs.length - 1; i >= 0; i--) {
        const q = this.puffs[i]
        q.life += dt
        q.m.position.y += dt * 0.5; q.m.position.x += dt * 0.15
        q.m.scale.setScalar(1 + q.life * 0.9)
        ;(q.m.material as THREE.MeshBasicMaterial).opacity = Math.max(0, 0.9 - q.life / 2.2)
        if (q.life > 2.2) { this.scene.remove(q.m); this.kit.release(q.m.material as THREE.Material); this.puffs.splice(i, 1) }
      }
      for (const p of this.placed.values()) if (p.live.crystal) { const c = p.live.crystal; c.c.rotation.y = this.T * 1.2; c.c.position.y = 0.1 + Math.sin(this.T * 2) * 0.08; c.gl.material.opacity = 0.6 + Math.sin(this.T * 3) * 0.25 }
      if (this.selRing.visible) (this.selRing.material as THREE.MeshBasicMaterial).opacity = 0.7 + Math.sin(this.T * 5) * 0.25
    }
    const size = this.fit.span
    const aspect = this.W / this.H
    // On a tall screen the island's width is the limit; zoom in closer there, the rest is a drag away.
    const viewH = (aspect < 0.9 ? (size * 0.95) / aspect : Math.max(size * 0.8, (size * 1.3) / aspect)) * this.zoom
    const tg = this.fit.c.clone().setY(0.5).add(this.pan), c = this.cam, a = aspect, d = 80
    c.left = (-viewH * a) / 2; c.right = (viewH * a) / 2; c.top = viewH / 2; c.bottom = -viewH / 2
    c.position.set(tg.x + Math.sin(YAW) * Math.cos(ELEV) * d, tg.y + Math.sin(ELEV) * d, tg.z + Math.cos(YAW) * Math.cos(ELEV) * d)
    c.lookAt(tg)
    c.updateProjectionMatrix()
    this.renderer.render(this.scene, c)
    for (const p of this.placed.values()) {
      if (p.badge.style.display === 'none') continue
      const [x, y] = this.toScreen(new THREE.Vector3(p.holder.position.x, p.holder.position.y + p.height + 0.3, p.holder.position.z))
      p.badge.style.transform = `translate(${x}px,${y}px) translate(-50%,-100%)`
    }
  }

  dispose() {
    cancelAnimationFrame(this.raf)
    this.ro.disconnect()
    this.canvas.removeEventListener('pointerdown', this.onDown)
    this.canvas.removeEventListener('pointermove', this.onHover)
    this.canvas.removeEventListener('wheel', this.onWheel)
    this.canvas.removeEventListener('webglcontextlost', this.onContextLost)
    removeEventListener('pointermove', this.onMove)
    removeEventListener('pointerup', this.onUp)
    document.removeEventListener('visibilitychange', this.onVis)
    this.labels.replaceChildren()
    this.kit.dispose()
    this.renderer.dispose()
    this.renderer.forceContextLoss()
  }
}

