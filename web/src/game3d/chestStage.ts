// The wave chest: it drops onto a pedestal, shakes, and bursts open. Its contents are known before it opens.
import * as THREE from 'three'
import { Kit, eIn, eOut, eOutBack, lerp, rand } from './kit'
import { GOLD, STONE, WOOD, WOOD_D } from './fortressWorld'

const ELEV = 0.64, YAW = Math.PI / 4
const camRight = new THREE.Vector3(1, 0, -1).normalize()

export class ChestStage {
  private renderer: THREE.WebGLRenderer
  private scene = new THREE.Scene()
  private cam = new THREE.OrthographicCamera(-1, 1, 1, -1, 0.1, 300)
  private kit: Kit
  private g!: THREE.Group
  private lid!: THREE.Group
  private back!: THREE.Sprite
  private leak!: THREE.Sprite
  private burst!: THREE.Sprite
  private state: 'drop' | 'idle' | 'opening' | 'open' = 'drop'
  private W = 1; private H = 1; private T = 0; private shake = 0
  private raf = 0
  private clock = new THREE.Clock()
  private ro: ResizeObserver

  constructor(private canvas: HTMLCanvasElement, reduced: boolean) {
    this.kit = new Kit(reduced)
    this.renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true })
    this.renderer.setPixelRatio(Math.min(devicePixelRatio, 2))
    this.renderer.outputColorSpace = THREE.SRGBColorSpace
    this.renderer.setClearColor(0x000000, 0)
    this.kit.lights(this.scene, 1.05)
    this.build()
    this.ro = new ResizeObserver(() => this.resize())
    this.ro.observe(canvas.parentElement!)
    this.resize()
    this.loop()
    this.drop()
  }

  private build() {
    const k = this.kit, P = k.grp(this.scene)
    k.mk(k.cyl(2.3, 2.5, 0.4, 20), k.M(0x8d8370), 0, -0.2, 0, P)
    k.mk(k.cyl(1.9, 2.05, 0.45, 20), k.M(STONE), 0, 0.2, 0, P)
    k.mk(k.geo('ptor', () => new THREE.TorusGeometry(1.92, 0.07, 6, 32).rotateX(Math.PI / 2)), k.M(GOLD), 0, 0.43, 0, P)
    for (let i = 0; i < 8; i++) { const a = (i / 8) * Math.PI * 2; k.mk(k.sph(0.09, 8, 6), k.M(0xffe27a), Math.sin(a) * 2.1, 0.2, Math.cos(a) * 2.1, P) }
    k.blob(P, 0, 0, 3.4, 2.6, 0.55).position.y = 0.48
    this.back = k.glow(0xffc94a, 7, 0.55)
    this.back.position.set(0, 1.4, -1.5)
    P.add(this.back)
    const C = (this.g = k.grp(P, 0, 0.47, 0))
    const body = k.grp(C)
    k.mk(k.rb(2.3, 1.15, 1.5, 0.14), k.M(WOOD), 0, 0.58, 0, body)
    for (const x of [-0.4, 0.4]) k.mk(k.rb(0.05, 1.0, 0.05, 0.02), k.M(WOOD_D), x, 0.58, 0.76, body)
    k.mk(k.rb(2.36, 0.14, 1.56, 0.06), k.M(WOOD_D), 0, 0.1, 0, body)
    for (const x of [-0.78, 0.78]) k.mk(k.rb(0.24, 1.19, 1.56, 0.07), k.M(GOLD), x, 0.58, 0, body)
    for (const x of [-1.12, 1.12]) for (const z of [-0.72, 0.72]) k.mk(k.rb(0.2, 0.32, 0.2, 0.06), k.M(0xf2b800), x, 0.16, z, body)
    const lock = k.mk(k.rb(0.44, 0.52, 0.12, 0.08), k.M(GOLD), 0, 0.92, 0.78, body)
    k.mk(k.cyl(0.06, 0.06, 0.14, 8), k.MB(0x3a2410), 0, -0.02, 0.02, lock).rotation.x = Math.PI / 2
    k.mk(k.rb(2.1, 0.05, 1.3, 0.02), k.M(0xc98a2e), 0, 1.12, 0, body)
    for (let i = 0; i < 9; i++) k.mk(k.cyl(0.12, 0.12, 0.04, 10), k.M(GOLD), -0.7 + (i % 5) * 0.35, 1.17 + Math.floor(i / 5) * 0.05, -0.3 + (i % 3) * 0.3, body).rotation.set(rand() * 0.5, 0, rand() * 0.5)
    const lid = (this.lid = k.grp(C, 0, 1.15, -0.75))
    k.mk(k.geo('lid', () => new THREE.CylinderGeometry(0.75, 0.75, 2.3, 18, 1, false, 0, Math.PI).rotateZ(Math.PI / 2)), k.M(0xb07a40, { side: THREE.DoubleSide }), 0, 0, 0.75, lid)
    const lb = k.geo('lidb', () => new THREE.CylinderGeometry(0.79, 0.79, 0.24, 18, 1, false, 0, Math.PI).rotateZ(Math.PI / 2))
    for (const x of [-0.78, 0.78]) k.mk(lb, k.M(GOLD), x, 0, 0.75, lid)
    k.mk(k.rb(2.36, 0.14, 0.14, 0.06), k.M(GOLD), 0, 0.02, 1.5, lid)
    k.mk(k.rb(0.3, 0.3, 0.14, 0.06), k.M(GOLD), 0, -0.04, 1.54, lid)
    this.leak = k.glow(0xffe89a, 3.5, 0)
    this.leak.position.set(0, 1.2, 0.3)
    C.add(this.leak)
    this.burst = k.glow(0xfff0b0, 2, 0)
    this.burst.position.set(0, 1.6, 0.4)
    C.add(this.burst)
  }

  /** drop: the chest falls onto the pedestal and squashes. */
  private drop() {
    const k = this.kit, g = this.g
    this.state = 'drop'
    g.position.y = 0.47 + 3.2
    k.tween(0.45, q => { g.position.y = 0.47 + 3.2 * (1 - eIn(q)) }, () => {
      k.poof(this.scene, new THREE.Vector3(-1, 0.6, 0.6), 0.6, 3)
      k.poof(this.scene, new THREE.Vector3(1, 0.6, 0.6), 0.6, 3)
      k.tween(0.25, q => { const s = Math.sin(q * Math.PI); g.scale.set(1 + 0.15 * s, 1 - 0.15 * s, 1 + 0.15 * s) }, () => { this.state = 'idle' })
    })
  }

  /** open: shake, crack of light, the lid flies back. `onBurst` fires when the cards should fly out. */
  open(onBurst: () => void) {
    if (this.state === 'opening' || this.state === 'open') return
    const k = this.kit, g = this.g
    this.state = 'opening'
    if (k.reduced) { this.lid.rotation.x = -1.9; this.burst.material.opacity = 0.12; this.state = 'open'; onBurst(); return }
    // Escalating shake and a light leaking from under the lid.
    k.tween(0.75, q => {
      g.rotation.z = Math.sin(q * Math.PI * 8) * lerp(0.07, 0.105, q)
      g.scale.setScalar(1 + 0.08 * eOut(q))
      this.leak.material.opacity = q > 0.65 ? ((q - 0.65) / 0.35) * 0.9 : 0
    }, () => {
      g.rotation.z = 0
      k.tween(0.15, q => { this.lid.position.y = 1.15 + 0.5 * eOut(q); this.lid.rotation.x = -0.44 * q }, () => {
        k.tween(0.45, q => { this.lid.rotation.x = lerp(-0.44, -1.95, eOutBack(q, 2.2)); this.lid.position.y = lerp(1.65, 1.15, eOut(q)); this.leak.material.opacity = 0.9 * (1 - q) + 0.25 })
      })
      this.shake = 0.15
      k.tween(0.7, q => { this.burst.material.opacity = 0.9 * (1 - q) + 0.12; this.burst.scale.setScalar(2 + 5 * eOut(q)); g.scale.setScalar(lerp(1.08, 1, q)) })
      k.sparks(this.scene, new THREE.Vector3(0, 1.9, 0.3), 18, 0xffe27a, 5)
      k.wait(0.35, () => { this.state = 'open'; onBurst() })
    })
  }

  /** screen: where the chest mouth is on screen (cards fly out of it). */
  mouth(): [number, number] {
    const v = this.g.localToWorld(new THREE.Vector3(0, 1.4, 0)).project(this.cam)
    return [((v.x + 1) / 2) * this.W, ((1 - v.y) / 2) * this.H]
  }

  private resize() {
    const p = this.canvas.parentElement!
    this.W = Math.max(1, p.clientWidth); this.H = Math.max(1, p.clientHeight)
    this.renderer.setSize(this.W, this.H, false)
  }

  private loop = () => {
    this.raf = requestAnimationFrame(this.loop)
    const dt = Math.min(this.clock.getDelta(), 0.05)
    if (!this.kit.reduced) this.T += dt
    this.kit.update(dt, this.T)
    // Idle: a little hop every 1.2 s, as if something inside wants out.
    if (this.state === 'idle' && !this.kit.reduced) { const c = this.T % 1.2; this.g.rotation.z = c < 0.4 ? Math.sin((c / 0.4) * Math.PI * 2) * 0.052 : 0 }
    this.back.material.opacity = 0.45 + Math.sin(this.T * 2) * 0.1
    const portrait = this.W / this.H < 0.9
    const viewH = portrait ? Math.max(8.5, 7.6 / (this.W / this.H)) : 9.6
    const tg = new THREE.Vector3(0, portrait ? 1.0 : 1.2, 0), c = this.cam, a = this.W / this.H, d = 80
    c.left = (-viewH * a) / 2; c.right = (viewH * a) / 2; c.top = viewH / 2; c.bottom = -viewH / 2
    c.position.set(tg.x + Math.sin(YAW) * Math.cos(ELEV) * d, tg.y + Math.sin(ELEV) * d, tg.z + Math.cos(YAW) * Math.cos(ELEV) * d)
    c.lookAt(tg)
    c.updateProjectionMatrix()
    if (this.shake > 0) { this.shake -= dt; c.position.addScaledVector(camRight, (rand() - 0.5) * 2 * ((c.top - c.bottom) / this.H) * 4) }
    this.renderer.render(this.scene, c)
  }

  dispose() {
    cancelAnimationFrame(this.raf)
    this.ro.disconnect()
    this.kit.dispose()
    this.renderer.dispose()
    this.renderer.forceContextLoss()
  }
}
