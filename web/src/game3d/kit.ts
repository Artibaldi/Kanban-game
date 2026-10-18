// Toy-like 3D kit: toon materials, rounded geometry built in code, soft blob shadows, outlines, particles and tweens.
// Geometry and materials are cached per kit; a scene disposes its kit when it unmounts.
import * as THREE from 'three'
import { RoundedBoxGeometry } from 'three/examples/jsm/geometries/RoundedBoxGeometry.js'

export const rand = (() => { let s = 7; return () => (s = (s * 16807) % 2147483647) / 2147483647 })()
export const lerp = (a: number, b: number, k: number) => a + (b - a) * k
export const eOut = (x: number) => 1 - Math.pow(1 - x, 3)
export const eIn = (x: number) => x * x * x
export const eIO = (x: number) => (x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2)
export const eOutBack = (x: number, s = 1.70158) => 1 + (s + 1) * Math.pow(x - 1, 3) + s * Math.pow(x - 1, 2)

export class Kit {
  readonly reduced: boolean
  private mats = new Map<string, THREE.Material>()
  private geos = new Map<string, THREE.BufferGeometry>()
  private textures: THREE.Texture[] = []
  private extra = new Set<{ dispose(): void }>()
  private grad: THREE.DataTexture
  readonly blobTex: THREE.Texture
  readonly glowTex: THREE.Texture
  private tweens: { t: number; dur: number; fn: (k: number) => void; done?: () => void; tag?: string }[] = []
  private parts: { m: THREE.Mesh; v: THREE.Vector3; life: number; max: number; s0: number; s1: number; drag: number; g: number; fade: boolean; spin: boolean }[] = []
  private flags: { g: THREE.BufferGeometry; base: Float32Array; w: number; e: THREE.Object3D; ph: number }[] = []

  constructor(reduced: boolean) {
    this.reduced = reduced
    // Three-step ramp: the hard light bands of a cartoon.
    this.grad = new THREE.DataTexture(new Uint8Array([110, 190, 255]), 3, 1, THREE.RedFormat)
    this.grad.minFilter = this.grad.magFilter = THREE.NearestFilter
    this.grad.generateMipmaps = false
    this.grad.needsUpdate = true
    this.textures.push(this.grad)
    this.blobTex = this.canvasTex(64, 64, g => {
      const r = g.createRadialGradient(32, 32, 2, 32, 32, 32)
      r.addColorStop(0, 'rgba(30,30,70,.75)'); r.addColorStop(0.55, 'rgba(30,30,70,.35)'); r.addColorStop(1, 'rgba(30,30,70,0)')
      g.fillStyle = r; g.fillRect(0, 0, 64, 64)
    })
    this.glowTex = this.canvasTex(128, 128, g => {
      const r = g.createRadialGradient(64, 64, 0, 64, 64, 64)
      r.addColorStop(0, 'rgba(255,255,255,1)'); r.addColorStop(0.3, 'rgba(255,255,255,.5)'); r.addColorStop(1, 'rgba(255,255,255,0)')
      g.fillStyle = r; g.fillRect(0, 0, 128, 128)
    })
    this.glowTex.colorSpace = THREE.NoColorSpace
  }

  /** Toon material, shared by colour and options. */
  M(color: number, ex?: THREE.MeshToonMaterialParameters): THREE.MeshToonMaterial {
    const k = 't' + color + (ex ? JSON.stringify(ex, (key, v) => (key === 'map' ? (v as THREE.Texture).uuid : v)) : '')
    let m = this.mats.get(k)
    if (!m) { m = new THREE.MeshToonMaterial({ color, gradientMap: this.grad, ...ex }); this.mats.set(k, m) }
    return m as THREE.MeshToonMaterial
  }
  /** Unlit material (not cached when it is changed per object, e.g. opacity). */
  MB(color: number, ex?: THREE.MeshBasicMaterialParameters, own = false): THREE.MeshBasicMaterial {
    if (own) { const m = new THREE.MeshBasicMaterial({ color, ...ex }); this.extra.add(m); return m }
    const k = 'b' + color + (ex ? JSON.stringify(ex, (key, v) => (key === 'map' ? (v as THREE.Texture).uuid : v)) : '')
    let m = this.mats.get(k)
    if (!m) { m = new THREE.MeshBasicMaterial({ color, ...ex }); this.mats.set(k, m) }
    return m as THREE.MeshBasicMaterial
  }
  geo<T extends THREE.BufferGeometry>(k: string, f: () => T): T {
    let g = this.geos.get(k)
    if (!g) { g = f(); this.geos.set(k, g) }
    return g as T
  }
  rb(w: number, h: number, d: number, r = 0.08, s = 2) {
    const q = (v: number) => Math.round(v * 100) / 100
    return this.geo(`rb${q(w)},${q(h)},${q(d)},${r},${s}`, () => new RoundedBoxGeometry(w, h, d, s, Math.min(r, Math.min(w, h, d) / 2 - 0.001)))
  }
  sph(r: number, ws = 16, hs = 12) { return this.geo(`s${r},${ws},${hs}`, () => new THREE.SphereGeometry(r, ws, hs)) }
  cyl(rt: number, rb: number, h: number, s = 12) { return this.geo(`c${rt},${rb},${h},${s}`, () => new THREE.CylinderGeometry(rt, rb, h, s)) }
  cone(r: number, h: number, s = 10) { return this.geo(`k${r},${h},${s}`, () => new THREE.ConeGeometry(r, h, s)) }
  caps(r: number, l: number) { return this.geo(`p${r},${l}`, () => new THREE.CapsuleGeometry(r, l, 4, 12)) }

  mk(g: THREE.BufferGeometry, m: THREE.Material, x = 0, y = 0, z = 0, p?: THREE.Object3D) {
    const o = new THREE.Mesh(g, m)
    o.position.set(x, y, z)
    p?.add(o)
    return o
  }
  grp(p?: THREE.Object3D, x = 0, y = 0, z = 0) {
    const g = new THREE.Group()
    g.position.set(x, y, z)
    p?.add(g)
    return g
  }
  canvasTex(w: number, h: number, draw: (g: CanvasRenderingContext2D, w: number, h: number) => void, rep?: number) {
    const c = document.createElement('canvas')
    c.width = w; c.height = h
    draw(c.getContext('2d')!, w, h)
    const t = new THREE.CanvasTexture(c)
    t.colorSpace = THREE.SRGBColorSpace
    if (rep) { t.wrapS = t.wrapT = THREE.RepeatWrapping; t.repeat.set(rep, rep); t.magFilter = THREE.NearestFilter }
    this.textures.push(t)
    return t
  }
  /** Soft round shadow under an object (cheaper and softer than real shadows). */
  blob(p: THREE.Object3D, x: number, z: number, sx: number, sz = sx, op = 0.5) {
    const m = new THREE.Mesh(this.geo('blob', () => new THREE.PlaneGeometry(1, 1).rotateX(-Math.PI / 2)), this.MB(0xffffff, { map: this.blobTex, transparent: true, depthWrite: false, opacity: op }))
    m.position.set(x, 0.03, z)
    m.scale.set(sx, 1, sz)
    m.renderOrder = 1
    p.add(m)
    return m
  }
  glow(color: number, size: number, op = 1) {
    const mat = new THREE.SpriteMaterial({ map: this.glowTex, color, transparent: true, opacity: op, blending: THREE.AdditiveBlending, depthWrite: false })
    this.extra.add(mat)
    const s = new THREE.Sprite(mat)
    s.scale.setScalar(size)
    return s
  }
  /** Dark inverted-hull outline: the cartoon edge around characters. */
  outline(root: THREE.Object3D, t = 0.035) {
    const list: THREE.Mesh[] = []
    root.traverse(o => { if ((o as THREE.Mesh).isMesh && !o.userData.noOL) list.push(o as THREE.Mesh) })
    for (const o of list) {
      const g = o.geometry
      if (!g.boundingSphere) g.computeBoundingSphere()
      const bs = g.boundingSphere!, r = bs.radius * Math.max(o.scale.x, o.scale.y, o.scale.z)
      if (r < 0.06 || bs.center.length() > bs.radius * 0.3) continue
      const col = (o.material as THREE.MeshToonMaterial).color.getHex()
      const mat = this.MB(new THREE.Color(col).multiplyScalar(0.28).getHex(), { side: THREE.BackSide })
      const h = new THREE.Mesh(g, mat)
      h.scale.setScalar(1 + t / r)
      h.userData.hull = 1
      o.add(h)
    }
  }
  /** Waving cloth flag on a pole. */
  flag(p: THREE.Object3D, x: number, y: number, z: number, w = 1, h = 0.62, color = 0x3e8ef0, emblem = 0xffd323) {
    this.mk(this.cyl(0.035, 0.045, h * 2.2, 6), this.M(0x6b4626), x, y + h * 1.1, z, p)
    this.mk(this.sph(0.07, 8, 6), this.M(0xffd323), x, y + h * 2.2 + 0.05, z, p)
    const g = new THREE.PlaneGeometry(w, h, 10, 3)
    g.translate(w / 2, 0, 0)
    this.extra.add(g)
    const m = this.mk(g, this.M(color, { side: THREE.DoubleSide }), x, y + h * 2.2 - h / 2 - 0.05, z, p)
    const e = this.mk(this.rb(h * 0.34, h * 0.34, 0.03, 0.02), this.M(emblem), w * 0.5, 0, 0.02, m)
    e.rotation.z = Math.PI / 4
    this.flags.push({ g, base: (g.attributes.position.array as Float32Array).slice(), w, e, ph: rand() * 6 })
  }
  lights(scene: THREE.Scene, k = 1) {
    scene.add(new THREE.AmbientLight(0x3b3a6b, 1.6 * k))
    scene.add(new THREE.HemisphereLight(0xfff3d6, 0x3b3a6b, 1.0 * k))
    const d = new THREE.DirectionalLight(0xfff3d6, 2.7 * k)
    d.position.set(-6, 14, 9)
    scene.add(d)
  }

  /** tween: runs fn(0…1) over `dur` seconds; without motion it jumps to the end. A tag lets a group be cancelled. */
  tween(dur: number, fn: (k: number) => void, done?: () => void, tag?: string) {
    if (this.reduced || dur <= 0) { fn(1); done?.(); return }
    this.tweens.push({ t: 0, dur, fn, done, tag })
  }
  wait(s: number, f: () => void, tag?: string) {
    if (this.reduced) { f(); return }
    this.tweens.push({ t: 0, dur: s, fn: () => {}, done: f, tag })
  }
  /** cancel: drops the tweens of one group (e.g. a finished wave replay) without running them. */
  cancel(tag: string) { this.tweens = this.tweens.filter(w => w.tag !== tag) }
  /** release: frees an owned material or geometry now instead of at unmount. */
  release(x: { dispose(): void } | null | undefined) { if (x && this.extra.delete(x)) x.dispose() }

  poof(p: THREE.Object3D, pos: THREE.Vector3, s = 1, n = 5) {
    if (this.reduced) return
    for (let i = 0; i < n; i++) {
      const m = this.mk(this.sph(0.3, 10, 8), this.MB(0xffffff, { transparent: true, opacity: 1 }, true), pos.x, pos.y, pos.z, p)
      const a = (i / n) * Math.PI * 2 + rand()
      this.parts.push({ m, v: new THREE.Vector3(Math.cos(a) * 1.6 * s, 1 + rand() * 1.2, Math.sin(a) * 1.6 * s), life: 0, max: 0.55 + rand() * 0.2, s0: 0.4 * s, s1: 1.2 * s, drag: 3, g: 0, fade: true, spin: false })
    }
  }
  debris(p: THREE.Object3D, pos: THREE.Vector3, n = 4, color = 0xa99d85) {
    if (this.reduced) return
    for (let i = 0; i < n; i++) {
      const m = this.mk(this.geo('deb', () => new THREE.DodecahedronGeometry(0.12, 0)), this.M(color), pos.x, pos.y, pos.z, p)
      this.parts.push({ m, v: new THREE.Vector3((rand() - 0.5) * 3, 2.5 + rand() * 1.5, -1 - rand() * 1.5), life: 0, max: 0.4, s0: 1, s1: 1, drag: 0, g: 14, fade: false, spin: true })
    }
  }
  sparks(p: THREE.Object3D, pos: THREE.Vector3, n = 8, color = 0xffe27a, sp = 3) {
    if (this.reduced) return
    for (let i = 0; i < n; i++) {
      const mat = this.MB(color, { transparent: true }, true)
      const m = this.mk(this.geo('spk', () => new THREE.OctahedronGeometry(0.09, 0)), mat, pos.x, pos.y, pos.z, p)
      this.parts.push({ m, v: new THREE.Vector3((rand() - 0.5) * sp, rand() * sp * 0.9 + 1, (rand() - 0.5) * sp), life: 0, max: 0.6 + rand() * 0.3, s0: 1.2, s1: 0.2, drag: 1.5, g: 4, fade: true, spin: true })
    }
  }

  /** update: advances tweens, particles and flags. */
  update(dt: number, t: number) {
    for (let i = this.tweens.length - 1; i >= 0; i--) {
      const w = this.tweens[i]
      w.t += dt
      const k = Math.min(1, w.t / w.dur)
      w.fn(k)
      if (k >= 1) { this.tweens.splice(i, 1); w.done?.() }
    }
    for (let i = this.parts.length - 1; i >= 0; i--) {
      const q = this.parts[i]
      q.life += dt
      const k = q.life / q.max
      if (k >= 1) { q.m.parent?.remove(q.m); if (q.fade) this.release(q.m.material as THREE.Material); this.parts.splice(i, 1); continue }
      q.v.y -= q.g * dt
      q.v.multiplyScalar(1 - Math.min(1, q.drag * dt))
      q.m.position.addScaledVector(q.v, dt)
      q.m.scale.setScalar(lerp(q.s0, q.s1, eOut(k)))
      if (q.fade) (q.m.material as THREE.MeshBasicMaterial).opacity = 1 - k * k
      if (q.spin) { q.m.rotation.x += dt * 8; q.m.rotation.y += dt * 6 }
    }
    if (this.reduced) return
    for (const f of this.flags) {
      const a = f.g.attributes.position as THREE.BufferAttribute, b = f.base
      for (let i = 0; i < a.count; i++) {
        const x = b[i * 3]
        a.array[i * 3 + 2] = Math.sin(x * 3.2 - t * 2.6 + f.ph) * 0.11 * (x / f.w)
        a.array[i * 3 + 1] = b[i * 3 + 1] - x * x * 0.03
      }
      a.needsUpdate = true
      f.g.computeVertexNormals()
      f.e.position.z = Math.sin(f.w * 0.5 * 3.2 - t * 2.6 + f.ph) * 0.11 * 0.5 + 0.02
    }
  }
  clearTweens() { this.tweens.length = 0 }

  dispose() {
    for (const m of this.mats.values()) m.dispose()
    for (const g of this.geos.values()) g.dispose()
    for (const t of this.textures) t.dispose()
    for (const x of this.extra) x.dispose()
    this.mats.clear(); this.geos.clear(); this.extra.clear(); this.tweens.length = 0; this.parts.length = 0; this.flags.length = 0
  }
}
