// The horde: four enemy kinds by task size — goblin (1–2), knight (3), giant (5), dragon (8). Built from primitives.
import * as THREE from 'three'
import type { Kind } from '../lib/horde'
import { Kit, lerp, rand } from './kit'
import { GOLD, WOOD } from './fortressWorld'

const SKIN_G = 0xa77be0, RED = 0xf04848, STEEL = 0x8e9bb0

export interface Rig { rig: THREE.Group; head: THREE.Group; eyes?: THREE.Mesh[]; arms?: THREE.Group[]; wings?: THREE.Group[]; body?: THREE.Group; h: number }

function eyes(k: Kit, p: THREE.Object3D, x: number, y: number, z: number, r = 0.085) {
  const l: THREE.Mesh[] = []
  for (const s of [-1, 1]) {
    const e = k.mk(k.sph(r, 10, 8), k.M(0xffffff), s * x, y, z, p)
    k.mk(k.sph(r * 0.5, 8, 6), k.MB(0x1e1e1e), 0, 0, r * 0.75, e).userData.noOL = 1
    e.userData.noOL = 1
    l.push(e)
  }
  return l
}

function goblin(k: Kit) {
  const g = new THREE.Group(), rig = k.grp(g)
  for (const s of [-1, 1]) k.mk(k.caps(0.08, 0.12), k.M(0x5b3a86), s * 0.12, 0.14, 0, rig)
  k.mk(k.sph(0.3), k.M(SKIN_G), 0, 0.42, 0, rig).scale.set(1, 0.95, 0.88)
  k.mk(k.rb(0.56, 0.1, 0.48, 0.04), k.M(0x6b4626), 0, 0.34, 0, rig)
  k.mk(k.cone(0.3, 0.52, 10), k.M(RED), 0, 0.44, -0.12, rig).rotation.x = -0.25
  const head = k.grp(rig, 0, 0.8, 0)
  k.mk(k.sph(0.29), k.M(SKIN_G), 0, 0, 0, head)
  for (const s of [-1, 1]) k.mk(k.cone(0.1, 0.38, 6), k.M(SKIN_G), s * 0.32, 0.04, 0, head).rotation.z = -s * 1.25
  const ey = eyes(k, head, 0.11, 0.03, 0.23)
  for (const s of [-1, 1]) k.mk(k.rb(0.13, 0.035, 0.04, 0.015), k.MB(0x2a1a3a), s * 0.11, 0.14, 0.27, head).rotation.z = s * 0.35
  k.mk(k.sph(0.075, 8, 6), k.M(0x8a5fc8), 0, -0.06, 0.29, head)
  k.mk(k.rb(0.05, 0.055, 0.03, 0.01), k.M(0xffffff), 0.06, -0.15, 0.25, head)
  k.mk(k.cone(0.25, 0.34, 9), k.M(RED), 0, 0.24, -0.04, head).rotation.x = -0.35
  k.mk(k.sph(0.06, 6, 4), k.M(GOLD), 0, 0.4, -0.1, head)
  const d = k.grp(rig, 0.34, 0.42, 0.1)
  d.rotation.z = 0.3
  k.mk(k.rb(0.06, 0.32, 0.03, 0.015), k.M(0xdde4ee), 0, 0.18, 0, d)
  k.mk(k.rb(0.14, 0.045, 0.05, 0.015), k.M(0x6b4626), 0, 0.02, 0, d)
  g.userData = { rig, head, eyes: ey, h: 1.15 } satisfies Rig
  return g
}

function knight(k: Kit) {
  const g = new THREE.Group(), rig = k.grp(g)
  for (const s of [-1, 1]) k.mk(k.caps(0.1, 0.2), k.M(0x5e6878), s * 0.14, 0.18, 0, rig)
  k.mk(k.rb(0.62, 0.58, 0.44, 0.14), k.M(STEEL), 0, 0.62, 0, rig)
  k.mk(k.rb(0.44, 0.52, 0.06, 0.03), k.M(RED), 0, 0.6, 0.22, rig)
  k.mk(k.rb(0.09, 0.52, 0.07, 0.02), k.M(GOLD), 0, 0.6, 0.23, rig)
  k.mk(k.rb(0.66, 0.09, 0.48, 0.03), k.M(0x6b4626), 0, 0.38, 0, rig)
  for (const s of [-1, 1]) { k.mk(k.sph(0.15, 10, 8), k.M(0xa7b3c6), s * 0.35, 0.84, 0, rig); k.mk(k.caps(0.08, 0.22), k.M(STEEL), s * 0.38, 0.58, 0, rig) }
  const head = k.grp(rig, 0, 1.12, 0)
  k.mk(k.sph(0.28), k.M(0xa7b3c6), 0, 0, 0, head)
  k.mk(k.rb(0.42, 0.09, 0.1, 0.03), k.MB(0x1e1e2a), 0, 0, 0.22, head)
  for (const s of [-1, 1]) k.mk(k.sph(0.032, 6, 4), k.MB(0xffe14a), s * 0.08, 0, 0.275, head).userData.noOL = 1
  k.mk(k.rb(0.07, 0.14, 0.5, 0.03), k.M(STEEL), 0, 0.24, 0, head)
  for (const [x, y, z, r] of [[0, 0.36, -0.02, 0.1], [0, 0.33, -0.16, 0.09], [0, 0.25, -0.27, 0.08]]) k.mk(k.sph(r, 8, 6), k.M(RED), x, y, z, head)
  const sh = k.grp(rig, -0.44, 0.6, 0.14)
  sh.rotation.y = -0.3
  k.mk(k.rb(0.5, 0.6, 0.08, 0.1), k.M(GOLD), 0, 0, 0, sh)
  k.mk(k.rb(0.42, 0.5, 0.09, 0.08), k.M(RED), 0, 0, 0.01, sh)
  k.mk(k.rb(0.08, 0.36, 0.1, 0.02), k.M(0xffffff), 0, 0, 0.02, sh)
  const sw = k.grp(rig, 0.44, 0.55, 0.16)
  k.mk(k.rb(0.07, 0.66, 0.03, 0.02), k.M(0xe8eef5), 0, 0.38, 0, sw)
  k.mk(k.rb(0.24, 0.05, 0.07, 0.02), k.M(GOLD), 0, 0.04, 0, sw)
  k.mk(k.rb(0.05, 0.14, 0.05, 0.02), k.M(0x6b4626), 0, -0.05, 0, sw)
  g.userData = { rig, head, arms: [sw], h: 1.62 } satisfies Rig
  return g
}

function giant(k: Kit) {
  const g = new THREE.Group(), rig = k.grp(g), SK = 0xe58e5a
  for (const s of [-1, 1]) { k.mk(k.cyl(0.17, 0.2, 0.5, 10), k.M(SK), s * 0.23, 0.32, 0, rig); k.mk(k.rb(0.34, 0.18, 0.44, 0.07), k.M(0x6b4626), s * 0.23, 0.09, 0.05, rig) }
  k.mk(k.rb(0.84, 0.34, 0.62, 0.14), k.M(RED), 0, 0.62, 0, rig)
  k.mk(k.sph(0.56), k.M(SK), 0, 1.08, 0, rig).scale.set(1.12, 1, 0.88)
  k.mk(k.sph(0.36), k.M(0xf5ae80), 0, 0.98, 0.3, rig).scale.set(1, 0.9, 0.5)
  k.mk(k.rb(0.96, 0.13, 0.7, 0.05), k.M(0x6b4626), 0, 0.77, 0, rig)
  k.mk(k.rb(0.2, 0.17, 0.1, 0.04), k.M(GOLD), 0, 0.77, 0.35, rig)
  const head = k.grp(rig, 0, 1.72, 0.08)
  k.mk(k.sph(0.28), k.M(SK), 0, 0, 0, head)
  k.mk(k.cone(0.15, 0.26, 6), k.M(0x4a2a18), 0, 0.3, -0.02, head).rotation.x = -0.3
  const ey = eyes(k, head, 0.1, 0.04, 0.22, 0.07)
  for (const s of [-1, 1]) k.mk(k.rb(0.15, 0.05, 0.05, 0.02), k.MB(0x4a2a18), s * 0.1, 0.14, 0.26, head).rotation.z = s * 0.45
  k.mk(k.sph(0.08, 8, 6), k.M(0xd47a48), 0, -0.04, 0.28, head)
  k.mk(k.rb(0.18, 0.04, 0.03, 0.01), k.MB(0x5a2a1a), 0, -0.15, 0.24, head)
  const arms: THREE.Group[] = []
  for (const s of [-1, 1]) {
    k.mk(k.sph(0.24), k.M(SK), s * 0.62, 1.34, 0, rig)
    const a = k.grp(rig, s * 0.64, 1.34, 0)
    k.mk(k.caps(0.14, 0.44), k.M(SK), 0, -0.36, 0, a)
    k.mk(k.sph(0.22), k.M(SK), 0, -0.74, 0.04, a)
    k.mk(k.rb(0.32, 0.12, 0.32, 0.05), k.M(0x6b4626), 0, -0.56, 0, a)
    arms.push(a)
  }
  const club = k.grp(arms[1], 0, -0.74, 0.08)
  club.rotation.x = 1.3
  k.mk(k.cyl(0.12, 0.22, 1.1, 8), k.M(WOOD), 0, 0.55, 0, club)
  k.mk(k.cyl(0.24, 0.24, 0.08, 8), k.M(0x4a4a58), 0, 0.85, 0, club)
  g.userData = { rig, head, eyes: ey, arms, h: 2.1 } satisfies Rig
  return g
}

function dragon(k: Kit) {
  const g = new THREE.Group(), rig = k.grp(g), body = k.grp(rig, 0, 1.4, 0), R = 0xe8443f, RD = 0xa8282e
  k.mk(k.caps(0.5, 0.8), k.M(R), 0, 0, 0, body).rotation.x = Math.PI / 2
  k.mk(k.sph(0.46), k.M(0xffd36a), 0, -0.16, 0.15, body).scale.set(0.85, 0.7, 1.25)
  k.mk(k.caps(0.24, 0.4), k.M(R), 0, 0.42, 0.62, body).rotation.x = -0.7
  const head = k.grp(body, 0, 0.82, 0.88)
  k.mk(k.rb(0.62, 0.46, 0.56, 0.18), k.M(R), 0, 0, 0, head)
  k.mk(k.rb(0.44, 0.26, 0.42, 0.1), k.M(R), 0, -0.08, 0.4, head)
  for (const s of [-1, 1]) {
    k.mk(k.sph(0.035, 6, 4), k.MB(0x3a0a0a), s * 0.1, -0.02, 0.62, head)
    k.mk(k.cone(0.08, 0.36, 6), k.M(0xf5e3b5), s * 0.2, 0.3, -0.12, head).rotation.x = -0.7
    k.mk(k.cone(0.035, 0.1, 4), k.M(0xffffff), s * 0.14, -0.24, 0.52, head).rotation.x = Math.PI
  }
  const ey = eyes(k, head, 0.17, 0.1, 0.24, 0.1)
  for (const s of [-1, 1]) k.mk(k.rb(0.18, 0.05, 0.06, 0.02), k.MB(0x5a1010), s * 0.17, 0.22, 0.27, head).rotation.z = s * 0.4
  k.mk(k.cone(0.32, 1.5, 8), k.M(R), 0, -0.02, -1.35, body).rotation.x = -Math.PI / 2
  k.mk(k.cone(0.2, 0.32, 4), k.M(RD), 0, -0.02, -2.15, body).rotation.x = -Math.PI / 2
  for (let i = 0; i < 5; i++) k.mk(k.cone(0.08, 0.24, 5), k.M(0xf5e3b5), 0, 0.5 - i * 0.05, 0.45 - i * 0.38, body).rotation.x = -0.3
  for (const s of [-1, 1]) { k.mk(k.caps(0.1, 0.2), k.M(R), s * 0.3, -0.5, 0.3, body); k.mk(k.caps(0.1, 0.2), k.M(R), s * 0.3, -0.5, -0.35, body) }
  const wg = k.geo('wing', () => {
    const sh = new THREE.Shape()
    ;[[0, -0.3], [0.9, -0.6], [2.1, -0.25], [1.75, 0.35], [1.3, 0.12], [0.98, 0.62], [0.58, 0.36], [0, 0.6]].forEach(([x, y], i) => (i ? sh.lineTo(x, y) : sh.moveTo(x, y)))
    return new THREE.ExtrudeGeometry(sh, { depth: 0.04, bevelEnabled: true, bevelThickness: 0.02, bevelSize: 0.03, bevelSegments: 1 }).rotateX(-Math.PI / 2)
  })
  const wings: THREE.Group[] = []
  for (const s of [-1, 1]) {
    const w = k.grp(body, s * 0.32, 0.35, 0.1)
    w.scale.x = s
    k.mk(wg, k.M(RD, { side: THREE.DoubleSide }), 0, 0, 0, w).userData.noOL = 1
    const bone = k.mk(k.caps(0.045, 2.0), k.M(0xf5e3b5), 1.0, 0.04, -0.05, w)
    bone.rotation.z = Math.PI / 2; bone.rotation.y = -0.25; bone.userData.noOL = 1
    wings.push(w)
  }
  g.userData = { rig, body, head, eyes: ey, wings, h: 2.9 } satisfies Rig
  g.scale.setScalar(1.15)
  return g
}

const builders: Record<Kind, (k: Kit) => THREE.Group> = { goblin, knight, giant, dragon }
/** Shadow size and walking speed per kind. */
export const KIND = { goblin: { sh: 1, sp: 1.7, hp: 1 }, knight: { sh: 1.1, sp: 1.3, hp: 2 }, giant: { sh: 1.7, sp: 1.0, hp: 3 }, dragon: { sh: 2.6, sp: 1.45, hp: 4 } } as const

export interface Body {
  kind: Kind
  holder: THREE.Group
  root: THREE.Group
  rig: Rig
  meshes: [THREE.Mesh, THREE.Material][]
  h: number
  ph: number
  blinkT: number
  bash?: { t: number; last: number }
  ring?: THREE.Mesh
}

/** makeBody: an enemy with its outline and shadow, placed at (x, z); `points` 1 makes a smaller goblin. */
export function makeBody(k: Kit, parent: THREE.Object3D, kind: Kind, points: number, x: number, z: number, overdue: boolean): Body {
  const root = builders[kind](k)
  k.outline(root)
  root.scale.multiplyScalar(1.3 * (kind === 'goblin' && points === 1 ? 0.85 : 1))
  const holder = k.grp(parent, x, 0, z)
  holder.add(root)
  k.blob(holder, 0, 0, KIND[kind].sh * 1.3, KIND[kind].sh, 0.55)
  const meshes: Body['meshes'] = []
  root.traverse(o => { if ((o as THREE.Mesh).isMesh && !o.userData.hull) meshes.push([o as THREE.Mesh, (o as THREE.Mesh).material as THREE.Material]) })
  const rig = root.userData as Rig
  const b: Body = { kind, holder, root, rig, meshes, h: rig.h * root.scale.y, ph: rand() * 6, blinkT: 2 + rand() * 3 }
  if (overdue) {
    // A pulsing red ring: this one is past its date and hits the wall.
    b.ring = k.mk(k.geo('ring', () => new THREE.RingGeometry(0.95, 1.2, 32).rotateX(-Math.PI / 2)), k.MB(0xf04848, { transparent: true, opacity: 0.85, depthWrite: false }, true), 0, 0.05, 0, holder)
    k.mk(k.geo('ring2', () => new THREE.CircleGeometry(0.95, 32).rotateX(-Math.PI / 2)), k.MB(0xf04848, { transparent: true, opacity: 0.2, depthWrite: false }), 0, 0.045, 0, holder)
    b.bash = { t: rand(), last: 0 }
  }
  return b
}

export function setFlash(k: Kit, b: Body, on: boolean) {
  const white = k.MB(0xffffff)
  for (const [m, orig] of b.meshes) m.material = on ? white : orig
}

/**
 * animateBody: idle breathing, sway, blinking; walking bounce; the wall bash for overdue ones.
 * Returns true on the frame a bash lands, so the scene can shake the wall.
 */
export function animateBody(b: Body, t: number, dt: number, walking: boolean, reduced: boolean): boolean {
  const u = b.rig, ph = b.ph
  if (reduced) return false
  if (b.kind === 'dragon') {
    u.body!.position.y = 1.4 + Math.sin(t * 2 + ph) * 0.15
    const f = 0.35 + Math.sin(t * (walking ? 9 : 5)) * 0.5
    u.wings![1].rotation.z = f; u.wings![0].rotation.z = -f
    u.body!.rotation.z = Math.sin(t * 1.3) * 0.05
  } else {
    const sp = walking ? 9 : 2.4
    u.rig.position.y = Math.abs(Math.sin(t * sp * 0.5 + ph)) * (walking ? 0.12 : 0.04)
    const br = Math.sin(t * 2.4 + ph) * 0.03
    u.rig.scale.set(1 - br * 0.5, 1 + br, 1 - br * 0.5)
    u.rig.rotation.z = Math.sin(t * (walking ? 4.5 : 1.7) + ph) * (walking ? 0.08 : 0.045)
    u.head.rotation.y = Math.sin(t * 0.9 + ph) * 0.2
  }
  if (u.eyes) { b.blinkT -= dt; const s = b.blinkT < 0 ? 0.15 : 1; if (b.blinkT < -0.12) b.blinkT = 2 + rand() * 3; u.eyes.forEach(x => (x.scale.y = s)) }
  if (b.ring) { const s = 1 + Math.sin(t * 4) * 0.06; b.ring.scale.set(s, 1, s); (b.ring.material as THREE.MeshBasicMaterial).opacity = 0.65 + Math.sin(t * 4) * 0.25 }
  if (!b.bash || walking) return false
  // Wind-up, lunge, impact, recover: every 2.4 s.
  const c = (t + b.bash.t * 2.4) % 2.4, a = u.arms?.[u.arms.length - 1], r = b.root
  if (c < 0.2) { r.rotation.x = -0.17 * (c / 0.2); if (a) a.rotation.x = -1.4 * (c / 0.2) }
  else if (c < 0.28) { const q = (c - 0.2) / 0.08; r.rotation.x = lerp(-0.17, 0.2, q); if (a) a.rotation.x = lerp(-1.4, 0.6, q) }
  else { const q = Math.min(1, (c - 0.28) / 0.5); r.rotation.x = lerp(0.2, 0, q); if (a) a.rotation.x = lerp(0.6, 0, q) }
  const hit = b.bash.last < 0.28 && c >= 0.28
  b.bash.last = c
  return hit
}
