// The fortress scene: renderer, camera, the live horde from the server, picking, labels, and the wave replay.
import * as THREE from 'three'
import { crownSchedule, kindFor, layoutHorde, LANES, waveRoster, WALL_Z, type Kind, type Unit } from '../lib/horde'
import type { Enemy } from '../types'
import { Kit, eIn, eIO, eOut, lerp, rand } from './kit'
import { buildFortress, type FortressWorld, type Tower } from './fortressWorld'
import { animateBody, KIND, makeBody, setFlash, type Body } from './enemies'

const ELEV = 0.64, YAW = Math.PI / 4
const camRight = new THREE.Vector3(1, 0, -1).normalize()
const camUp = new THREE.Vector3(-Math.sin(ELEV) * Math.SQRT1_2, Math.cos(ELEV), -Math.sin(ELEV) * Math.SQRT1_2)
const PRI = ['', 'Срочно', 'Высокий', 'Обычный', 'Низкий']

interface Live { id: string; body: Body; enemy: Enemy; alive: boolean; badge: HTMLElement; walk?: { from: number; to: number; t: number; dur: number }; scr?: [number, number] }
interface BattleUnit { unit: Unit; body: Body; alive: boolean; hp: number; pending: number; delay: number; startZ: number; targetZ: number; walking: boolean; killIndex: number }

export interface WaveInput { planned: number; closed: number; overdue: number; crowns: number; thresholds: number[] }
export interface WaveEvents {
  onProgress(points: number): void
  onCrown(n: number, screen: [number, number]): void
  onWallHit(): void
  onDone(): void
}

export class FortressStage {
  private renderer: THREE.WebGLRenderer
  private scene = new THREE.Scene()
  private cam = new THREE.OrthographicCamera(-1, 1, 1, -1, 0.1, 300)
  private kit: Kit
  private f: FortressWorld
  private live = new Map<string, Live>()
  private laneEls: { d: HTMLElement; v: THREE.Vector3; w?: number }[] = []
  private fogEl: HTMLElement
  private selRing: THREE.Mesh
  private selected: string | null = null
  private W = 1; private H = 1
  private T = 0
  private raf = 0
  private clock = new THREE.Clock()
  private running = true
  private pan = new THREE.Vector3()
  private down: { x: number; y: number; px: THREE.Vector3; moved: boolean } | null = null
  private ro: ResizeObserver
  private io: IntersectionObserver
  private visible = true
  private shakeSeg = 0
  private battle: {
    t: number; speed: number; stop: number; slow: number; shake: number; push: number; camK: number
    units: BattleUnit[]; proj: { u: BattleUnit; from: THREE.Vector3; t: number; m: THREE.Mesh; g: THREE.Sprite }[]
    kills: number; points: number; crowns: number; schedule: number[]; bashes: number; endAt?: number; done: boolean; ev: WaveEvents; hasOverdue: boolean
  } | null = null
  private tmp = new THREE.Vector3()
  onPick: (id: string | null) => void = () => {}

  constructor(private canvas: HTMLCanvasElement, private labels: HTMLElement, reduced: boolean) {
    this.kit = new Kit(reduced)
    this.renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true })
    this.renderer.setPixelRatio(Math.min(devicePixelRatio, 2))
    this.renderer.outputColorSpace = THREE.SRGBColorSpace
    this.renderer.setClearColor(0x000000, 0)
    this.kit.lights(this.scene)
    this.f = buildFortress(this.kit, this.scene)
    this.selRing = this.kit.mk(this.kit.geo('sel', () => new THREE.RingGeometry(1.15, 1.42, 40).rotateX(-Math.PI / 2)), this.kit.MB(0xffffff, { transparent: true, opacity: 0.95, depthWrite: false }), 0, 0.06, 0)
    this.selRing.visible = false
    this.laneEls = LANES.map((z, i) => {
      const d = document.createElement('div')
      d.className = 'g3-lane'
      d.innerHTML = `P${i + 1}<small>${PRI[i + 1]}</small>`
      labels.appendChild(d)
      return { d, v: new THREE.Vector3(-8.4, 0.2, z) }
    })
    this.fogEl = document.createElement('div')
    this.fogEl.className = 'g3-fog'
    labels.appendChild(this.fogEl)
    this.ro = new ResizeObserver(() => this.resize())
    this.ro.observe(canvas.parentElement!)
    this.io = new IntersectionObserver(([e]) => { this.visible = e.isIntersecting })
    this.io.observe(canvas)
    canvas.addEventListener('pointerdown', this.onDown)
    addEventListener('pointermove', this.onMove)
    addEventListener('pointerup', this.onUp)
    document.addEventListener('visibilitychange', this.onVis)
    this.resize()
    this.loop()
  }

  get reduced() { return this.kit.reduced }

  /** setFortress: hit points drive the cracks; fog is the number of tasks still in the backlog. */
  setFortress(hp: number, fog: number) {
    this.f.setDamage(Math.max(0, Math.min(1, (100 - hp) / 70)))
    this.fogEl.textContent = fog ? `Бэклог: ${fog} в тумане` : 'Туман рассеялся: бэклог пуст'
  }

  /**
   * sync: places the horde from the server. Enemies not in `known` walk out of the fog;
   * enemies that disappeared fall with a poof, and `onGone` gets their screen position (for coins).
   */
  sync(enemies: Enemy[], known: Set<string> | null, onGone?: (e: Enemy, screen: [number, number]) => void) {
    const spots = new Map(layoutHorde(enemies).map(s => [s.id, s]))
    const ids = new Set(enemies.map(e => e.task_id))
    for (const [id, l] of this.live) if (!ids.has(id) && l.alive) this.defeat(l, onGone)
    for (const e of enemies) {
      const s = spots.get(e.task_id)!
      const was = this.live.get(e.task_id)
      if (was && was.alive && was.enemy.kind === e.kind && was.enemy.overdue === e.overdue) {
        was.enemy = e
        was.body.holder.position.x = s.x
        if (!was.walk) was.body.holder.position.z = s.z
        this.badge(was)
        continue
      }
      if (was) this.remove(was)
      const body = makeBody(this.kit, this.f.world, e.kind as Kind, e.points, s.x, s.z, e.overdue)
      const badge = document.createElement('div')
      this.labels.appendChild(badge)
      const l: Live = { id: e.task_id, body, enemy: e, alive: true, badge }
      body.root.traverse(o => { o.userData.enemyId = e.task_id })
      this.badge(l)
      if (known && !known.has(e.task_id) && !this.kit.reduced) {
        // Out of the fog, toward its lane.
        body.holder.position.z = -15.5
        l.walk = { from: -15.5, to: s.z, t: -rand() * 0.6, dur: 1.6 + rand() * 0.6 }
      }
      this.live.set(e.task_id, l)
    }
    this.select(this.selected && this.live.get(this.selected)?.alive ? this.selected : null)
  }

  private badge(l: Live) {
    const e = l.enemy
    l.badge.className = 'g3-badge' + (e.overdue ? ' due' : '') + (e.state === 'review' ? ' review' : '')
    l.badge.innerHTML = `<b>${e.overdue ? '!' : e.points}</b>` + (e.overdue ? '<i>Просрочена</i>' : e.state === 'review' ? '<i>На проверке</i>' : '')
  }

  private remove(l: Live) {
    this.f.world.remove(l.body.holder)
    l.badge.remove()
    this.live.delete(l.id)
  }

  /** defeat: flash, knockback, fall, poof — then the enemy is gone. */
  private defeat(l: Live, onGone?: (e: Enemy, screen: [number, number]) => void) {
    l.alive = false
    l.badge.style.display = 'none'
    const scr = l.scr || this.toScreen(l.body.holder.getWorldPosition(new THREE.Vector3()))
    const k = this.kit, b = l.body, z0 = b.holder.position.z
    setFlash(k, b, true)
    k.wait(0.06, () => setFlash(k, b, false))
    k.tween(0.1, q => { b.holder.position.z = z0 - 0.25 * eOut(q) })
    k.wait(0.1, () => k.tween(0.2, q => { b.root.rotation.x = (-Math.PI / 2) * eIn(q); b.root.position.y = b.kind === 'dragon' ? -1.2 * q : 0 }))
    k.wait(0.3, () => {
      const p = b.holder.position.clone()
      p.y = 0.5 + b.h * 0.25
      k.poof(this.f.world, p, b.kind === 'dragon' ? 1.8 : b.kind === 'giant' ? 1.3 : 1, b.kind === 'goblin' ? 3 : 5)
      k.sparks(this.f.world, p, 6)
      this.remove(l)
      onGone?.(l.enemy, scr)
    })
  }

  select(id: string | null) {
    this.selected = id
    const l = id ? this.live.get(id) : null
    this.selRing.visible = !!l
    if (l) { l.body.holder.add(this.selRing); const s = KIND[l.body.kind].sh; this.selRing.scale.set(s, 1, s) }
    for (const x of this.live.values()) x.badge.classList.toggle('sel', x.id === id)
  }

  /** screenOf: where an enemy is on screen (for panels and coins). */
  screenOf(id: string): [number, number] | null { return this.live.get(id)?.scr ?? null }

  // —— pointer: tap picks, drag pans ——
  private onDown = (ev: PointerEvent) => { this.down = { x: ev.clientX, y: ev.clientY, px: this.pan.clone(), moved: false } }
  private onMove = (ev: PointerEvent) => {
    const d = this.down
    if (!d) return
    const dx = ev.clientX - d.x, dy = ev.clientY - d.y
    if (Math.hypot(dx, dy) > 6) d.moved = true
    if (!d.moved) return
    const k = (this.cam.top - this.cam.bottom) / this.H
    this.pan.copy(d.px).addScaledVector(camRight, -dx * k).add(new THREE.Vector3(-1, 0, -1).normalize().multiplyScalar((dy * k) / Math.sin(ELEV)))
    this.pan.clampLength(0, 9)
  }
  private onUp = (ev: PointerEvent) => {
    const d = this.down
    this.down = null
    if (!d || d.moved || this.battle) return
    const r = this.canvas.getBoundingClientRect()
    const id = this.pick(ev.clientX - r.left, ev.clientY - r.top)
    this.onPick(id)
  }
  private pick(x: number, y: number): string | null {
    const ray = new THREE.Raycaster(), ndc = new THREE.Vector2((x / this.W) * 2 - 1, -(y / this.H) * 2 + 1)
    ray.setFromCamera(ndc, this.cam)
    const roots = [...this.live.values()].filter(l => l.alive).map(l => l.body.root)
    const hit = ray.intersectObjects(roots, true)[0]
    if (hit) return hit.object.userData.enemyId as string
    // Generous fallback for small enemies on touch screens: the nearest badge within 60 px.
    let best: string | null = null, bd = 60
    for (const l of this.live.values()) { if (!l.alive || !l.scr) continue; const d = Math.hypot(l.scr[0] - x, l.scr[1] - y); if (d < bd) { bd = d; best = l.id } }
    return best
  }
  private onVis = () => { this.running = document.visibilityState === 'visible'; if (this.running) this.clock.getDelta() }

  private resize() {
    const p = this.canvas.parentElement!
    this.W = Math.max(1, p.clientWidth); this.H = Math.max(1, p.clientHeight)
    this.renderer.setSize(this.W, this.H, false)
    this.f.world.rotation.y = this.portrait ? Math.PI / 4 + 0.14 : Math.PI / 2 - 0.2
    this.laneEls.forEach(l => (l.w = undefined))
  }
  private get portrait() { return this.W / this.H < 0.9 }

  private toScreen(v: THREE.Vector3): [number, number] {
    this.tmp.copy(v).project(this.cam)
    return [((this.tmp.x + 1) / 2) * this.W, ((1 - this.tmp.y) / 2) * this.H]
  }

  private aim(shake: number) {
    const portrait = this.portrait
    const local = new THREE.Vector3(portrait ? -1.2 : -1.4, 0, -7.0)
    let viewH = portrait ? 14 / (this.W / this.H) : Math.max(19, 30 / (this.W / this.H))
    const B = this.battle
    if (B && !B.done) {
      local.lerpVectors(new THREE.Vector3(local.x, 0, -10), local.clone(), eIO(B.camK))
      if (B.push > 0) viewH /= 1 + 0.08 * Math.sin(Math.min(1, B.push / 0.6) * Math.PI)
    }
    this.f.world.updateMatrixWorld()
    const tg = this.f.world.localToWorld(local).add(this.pan)
    const a = this.W / this.H, c = this.cam
    c.left = (-viewH * a) / 2; c.right = (viewH * a) / 2; c.top = viewH / 2; c.bottom = -viewH / 2
    const d = 80
    c.position.set(tg.x + Math.sin(YAW) * Math.cos(ELEV) * d, tg.y + Math.sin(ELEV) * d, tg.z + Math.cos(YAW) * Math.cos(ELEV) * d)
    c.lookAt(tg)
    c.updateProjectionMatrix()
    if (shake) { const px = ((c.top - c.bottom) / this.H) * shake; c.position.addScaledVector(camRight, (rand() - 0.5) * 2 * px).addScaledVector(camUp, (rand() - 0.5) * 2 * px) }
  }

  private loop = () => {
    this.raf = requestAnimationFrame(this.loop)
    if (!this.running || !this.visible) return
    const dt = Math.min(this.clock.getDelta(), 0.05)
    if (!this.kit.reduced) this.T += dt
    const T = this.T
    this.kit.update(dt, T)
    if (this.battle) this.battleStep(dt)
    for (const l of this.live.values()) {
      if (!l.alive) continue
      let walking = false
      if (l.walk) {
        l.walk.t += dt
        const q = Math.max(0, Math.min(1, l.walk.t / l.walk.dur))
        l.body.holder.position.z = lerp(l.walk.from, l.walk.to, eOut(q))
        walking = q < 1
        if (q >= 1) l.walk = undefined
      }
      if (animateBody(l.body, T, dt, walking, this.kit.reduced) && !this.battle) this.wallHit(false)
    }
    if (this.battle) for (const u of this.battle.units) if (u.alive && animateBody(u.body, T, dt, u.walking, this.kit.reduced)) this.wallHit(true)
    const seg = this.f.hitSeg
    if (this.shakeSeg > 0) { this.shakeSeg -= dt; seg.position.x = this.f.hitSegX + Math.sin(T * 90) * 0.05 } else seg.position.x = this.f.hitSegX
    this.f.water.offset.x = (T * 0.05) % 1
    for (const c of this.f.clouds) c.g.position.x = c.x + Math.sin(T * 0.25 + c.ph) * 0.4
    for (const s of this.f.cshadows) { s.position.x += dt * 0.45; if (s.position.x > 22) s.position.x = -22 }
    this.f.torches.forEach((o, i) => { const fl = 1 + Math.sin(T * 17 + i) * 0.08 + Math.sin(T * 31 + i * 2) * 0.05; o.scale.setScalar(((o as THREE.Sprite).isSprite ? 0.9 : 1) * fl) })
    const B = this.battle
    if (B) { if (B.shake > 0) B.shake -= dt; if (B.push > 0) B.push -= dt }
    this.aim(B && B.shake > 0 ? 5 : 0)
    this.renderer.render(this.scene, this.cam)
    this.placeLabels()
  }

  private placeLabels() {
    for (const l of this.live.values()) {
      if (!l.alive) continue
      this.tmp.set(0, l.body.h + 0.35, 0)
      l.body.holder.localToWorld(this.tmp)
      const [x, y] = this.toScreen(this.tmp)
      l.scr = [x, y + 20]
      l.badge.style.transform = `translate(${x}px,${y}px) translate(-50%,-100%)`
      l.badge.style.display = this.battle ? 'none' : ''
    }
    for (const l of this.laneEls) {
      this.tmp.copy(l.v)
      this.f.world.localToWorld(this.tmp)
      const [x0, y] = this.toScreen(this.tmp)
      const x = Math.max(x0, (l.w ||= l.d.offsetWidth) / 2 + 6)
      l.d.style.transform = `translate(${x}px,${y}px) translate(-50%,-50%)`
    }
    this.tmp.set(-7, 3.2, -14.5)
    this.f.world.localToWorld(this.tmp)
    const [fx, fy] = this.toScreen(this.tmp)
    this.fogEl.style.transform = `translate(${Math.max(fx, this.fogEl.offsetWidth / 2 + 6)}px,${fy}px) translate(-50%,-50%)`
  }

  private wallHit(inBattle: boolean) {
    this.shakeSeg = 0.16
    this.kit.debris(this.f.world, new THREE.Vector3(-3, 1.2, -0.6), inBattle ? 4 : 3)
    if (inBattle && this.battle) { this.battle.bashes++; this.battle.ev.onWallHit() }
  }

  // —— the wave replay ——

  /** playWave: replays a finished wave from its numbers. The live horde steps aside until endWave(). */
  playWave(w: WaveInput, ev: WaveEvents) {
    this.endWave(false)
    for (const l of this.live.values()) l.body.holder.visible = false
    this.labels.classList.add('battle')
    this.pan.set(0, 0, 0)
    const roster = waveRoster(w)
    const schedule = crownSchedule(roster, w.crowns, w.thresholds)
    let killIndex = 0, slot = 0
    const units: BattleUnit[] = roster.map((unit, i) => {
      const laneZ = LANES[unit.lane - 1]
      const x = -6 + ((i * 37) % 13)
      const targetZ = unit.overdue ? WALL_Z : unit.killed ? -2.3 - rand() * 0.4 : laneZ
      const wallX = unit.overdue ? [-3.2, 3.2, -5.2, 5.2][slot++ % 4] : x
      const startZ = -15.4 - unit.lane * 0.5 - rand() * 0.6
      const body = makeBody(this.kit, this.f.world, unit.kind, unit.points, wallX, startZ, unit.overdue)
      return { unit, body, alive: true, hp: KIND[unit.kind].hp, pending: 0, delay: unit.overdue ? 1.0 : 1.2 + (unit.lane - 1) * 0.9 + rand() * 0.5, startZ, targetZ, walking: false, killIndex: unit.killed ? killIndex++ : -1 }
    })
    for (const t of this.f.towers) { t.glow.material.opacity = 0; t.cd = 0.4 + t.idx * 0.3 }
    this.battle = { t: 0, speed: 1, stop: 0, slow: 0, shake: 0, push: 0, camK: 0, units, proj: [], kills: 0, points: 0, crowns: 0, schedule, bashes: 0, done: false, ev, hasOverdue: units.some(u => u.unit.overdue) }
    if (this.kit.reduced) this.finishWave()
  }

  setSpeed(x: number) { if (this.battle) this.battle.speed = x }

  private battleStep(rdt: number) {
    const B = this.battle!
    if (B.done) return
    if (B.stop > 0) { B.stop -= rdt; return }
    const dt = rdt * B.speed * (B.slow > 0 ? 0.5 : 1)
    B.slow -= rdt
    B.t += dt
    const t = B.t
    B.camK = Math.min(1, t / 1.5)
    for (const u of B.units) {
      if (!u.alive || t < u.delay) continue
      const z = u.body.holder.position.z
      if (z < u.targetZ) { u.body.holder.position.z = Math.min(u.targetZ, z + KIND[u.unit.kind].sp * dt * (t < 3 ? 0.6 : 1)); u.walking = true }
      else u.walking = false
    }
    // Towers shoot at the enemies the team defeated in this wave, nearest to the walls first.
    for (const tw of this.f.towers) {
      tw.cd -= dt
      if (tw.cd > 0 || t < 3) continue
      let best: BattleUnit | null = null, bz = -12.2
      for (const u of B.units) {
        if (!u.alive || !u.unit.killed || t < u.delay || u.pending >= u.hp) continue
        const z = u.body.holder.position.z
        if (z > bz) { bz = z; best = u }
      }
      if (!best) continue
      tw.cd = 1.15
      best.pending++
      const from = tw.top.clone()
      const m = this.kit.mk(this.kit.sph(0.17, 10, 8), this.kit.MB(0xffe14a), from.x, from.y, from.z, this.f.world)
      const g = this.kit.glow(0xffb020, 1.1)
      this.f.world.add(g)
      B.proj.push({ u: best, from, t: 0, m, g })
    }
    for (let i = B.proj.length - 1; i >= 0; i--) {
      const p = B.proj[i], u = p.u
      p.t += dt / 0.6
      const to = u.body.holder.position.clone()
      to.y = u.body.h * 0.5
      const k = Math.min(1, p.t)
      p.m.position.lerpVectors(p.from, to, k)
      p.m.position.y += Math.sin(k * Math.PI) * 2.6
      p.g.position.copy(p.m.position)
      if (k < 1) continue
      this.f.world.remove(p.m); this.f.world.remove(p.g)
      B.proj.splice(i, 1)
      u.pending--
      if (!u.alive) continue
      this.kit.sparks(this.f.world, p.m.position, 4, 0xffe27a, 2)
      u.hp--
      setFlash(this.kit, u.body, true)
      this.kit.wait(0.06, () => setFlash(this.kit, u.body, false))
      if (u.hp > 0) continue
      this.killUnit(u)
    }
    const left = B.units.filter(u => u.alive && u.unit.killed).length
    if (!left && (!B.hasOverdue || B.bashes >= 2) && B.endAt === undefined) B.endAt = t + 0.8
    if ((B.endAt !== undefined && t >= B.endAt) || t > 24) this.finishWave()
  }

  private killUnit(u: BattleUnit) {
    const B = this.battle!
    u.alive = false
    B.kills++
    B.points += u.unit.points
    B.stop = 0.05
    if (!this.kit.reduced) B.shake = 0.12
    if (u.unit.points >= 5) { B.slow = 0.6; B.push = 0.6 }
    const b = u.body, k = this.kit, z0 = b.holder.position.z
    k.tween(0.1, q => { b.holder.position.z = z0 - 0.25 * eOut(q) })
    k.wait(0.1, () => k.tween(0.2, q => { b.root.rotation.x = (-Math.PI / 2) * eIn(q); b.root.position.y = b.kind === 'dragon' ? -1.2 * q : 0 }))
    k.wait(0.3, () => {
      const p = b.holder.position.clone()
      p.y = 0.5 + b.h * 0.25
      k.poof(this.f.world, p, b.kind === 'dragon' ? 1.8 : b.kind === 'giant' ? 1.3 : 1, b.kind === 'goblin' ? 3 : 5)
      k.sparks(this.f.world, p, 6)
      this.f.world.remove(b.holder)
    })
    B.ev.onProgress(B.points)
    while (B.crowns < B.schedule.length && B.schedule[B.crowns] <= u.killIndex) this.earnCrown(this.f.towers[[0, 2, 1][B.crowns]])
  }

  private earnCrown(tw: Tower) {
    const B = this.battle!
    B.crowns++
    const k = this.kit
    k.tween(0.8, q => {
      tw.glow.material.opacity = Math.sin(q * Math.PI) * 0.9 + (q > 0.5 ? 0.25 : 0)
      const w = Math.sin(q * Math.PI * 3) * (1 - q)
      tw.roof.scale.set(1 + w * 0.06, 1 - w * 0.08, 1 + w * 0.06)
    })
    const p = tw.top.clone()
    p.y += 3
    k.sparks(this.f.world, p, 12, 0xffe27a, 4)
    B.ev.onCrown(B.crowns, this.toScreen(this.f.world.localToWorld(p.clone())))
  }

  /** finishWave: skip to the end — every defeated enemy falls, the crowns are all in. */
  finishWave() {
    const B = this.battle
    if (!B || B.done) return
    for (const p of B.proj) { this.f.world.remove(p.m); this.f.world.remove(p.g) }
    B.proj = []
    for (const u of B.units) {
      if (u.unit.overdue) { u.body.holder.position.z = u.targetZ; continue }
      if (!u.unit.killed) { u.body.holder.position.z = u.targetZ; continue }
      if (u.alive) { u.alive = false; this.f.world.remove(u.body.holder); B.kills++; B.points += u.unit.points }
    }
    B.ev.onProgress(B.points)
    while (B.crowns < B.schedule.length) this.earnCrown(this.f.towers[[0, 2, 1][B.crowns]])
    B.done = true
    B.ev.onDone()
  }

  /** endWave: clears the replay and brings the live horde back. */
  endWave(_restore = true) {
    const B = this.battle
    if (!B) return
    for (const p of B.proj) { this.f.world.remove(p.m); this.f.world.remove(p.g) }
    for (const u of B.units) this.f.world.remove(u.body.holder)
    for (const t of this.f.towers) t.glow.material.opacity = 0
    this.battle = null
    this.labels.classList.remove('battle')
    for (const l of this.live.values()) l.body.holder.visible = true
  }

  dispose() {
    cancelAnimationFrame(this.raf)
    this.ro.disconnect()
    this.io.disconnect()
    this.canvas.removeEventListener('pointerdown', this.onDown)
    removeEventListener('pointermove', this.onMove)
    removeEventListener('pointerup', this.onUp)
    document.removeEventListener('visibilitychange', this.onVis)
    this.labels.replaceChildren()
    this.kit.dispose()
    this.renderer.dispose()
    this.renderer.forceContextLoss()
  }
}

/** webglAvailable: whether the 3D scene can run at all (otherwise the flat SVG scene is used). */
export function webglAvailable() {
  try {
    const c = document.createElement('canvas')
    return !!(c.getContext('webgl2') || c.getContext('webgl'))
  } catch { return false }
}

export { kindFor }
