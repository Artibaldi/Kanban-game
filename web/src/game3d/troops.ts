// The player's troops: Страж (shield and spear), Налётчик (two blades, red hood), Лучник (green hood, longbow),
// Всадник (a knight on a pony, lance and pennant), Таран (iron-headed ram on wheels).
// Same toy style as the horde, but in the team's blue and gold so friend and foe never look alike.
import * as THREE from 'three'
import { Kit, rand } from './kit'
import { GOLD, WOOD, WOOD_D } from './fortressWorld'
import type { Rig } from './enemies'

const BLUE = 0x3e8ef0, BLUE_D = 0x2c5fa8, SKIN = 0xf2c49b, STEEL = 0xc9d2de

function eyes(k: Kit, p: THREE.Object3D, x: number, y: number, z: number, r = 0.07) {
  const l: THREE.Mesh[] = []
  for (const s of [-1, 1]) {
    const e = k.mk(k.sph(r, 10, 8), k.M(0xffffff), s * x, y, z, p)
    k.mk(k.sph(r * 0.55, 8, 6), k.MB(0x1e1e1e), 0, 0, r * 0.7, e).userData.noOL = 1
    e.userData.noOL = 1
    l.push(e)
  }
  return l
}

function guard(k: Kit) {
  const g = new THREE.Group(), rig = k.grp(g)
  for (const s of [-1, 1]) { k.mk(k.caps(0.1, 0.18), k.M(0x4a4a58), s * 0.13, 0.17, 0, rig); k.mk(k.rb(0.2, 0.1, 0.26, 0.04), k.M(WOOD_D), s * 0.13, 0.05, 0.04, rig) }
  k.mk(k.rb(0.56, 0.52, 0.42, 0.14), k.M(BLUE), 0, 0.58, 0, rig)
  k.mk(k.rb(0.6, 0.08, 0.46, 0.03), k.M(WOOD_D), 0, 0.38, 0, rig)
  k.mk(k.rb(0.12, 0.12, 0.05, 0.03), k.M(GOLD), 0, 0.38, 0.23, rig)
  for (const s of [-1, 1]) k.mk(k.sph(0.13, 10, 8), k.M(STEEL), s * 0.32, 0.8, 0, rig)
  const head = k.grp(rig, 0, 1.06, 0)
  k.mk(k.sph(0.26), k.M(SKIN), 0, 0, 0, head)
  k.mk(k.sph(0.28, 16, 10), k.M(STEEL), 0, 0.08, -0.02, head).scale.set(1, 0.75, 1)
  k.mk(k.rb(0.08, 0.12, 0.34, 0.03), k.M(BLUE_D), 0, 0.3, -0.02, head)
  const ey = eyes(k, head, 0.09, 0.0, 0.21)
  k.mk(k.rb(0.12, 0.03, 0.03, 0.01), k.MB(0x7a4a2a), 0, -0.1, 0.24, head)
  // Shield on the left arm, spear in the right.
  const sh = k.grp(rig, -0.4, 0.6, 0.16)
  sh.rotation.y = -0.25
  k.mk(k.cyl(0.28, 0.28, 0.07, 18), k.M(BLUE_D), 0, 0, 0, sh).rotation.x = Math.PI / 2
  k.mk(k.cyl(0.2, 0.2, 0.08, 18), k.M(GOLD), 0, 0, 0.01, sh).rotation.x = Math.PI / 2
  k.mk(k.sph(0.07, 8, 6), k.M(STEEL), 0, 0, 0.06, sh)
  const arm = k.grp(rig, 0.38, 0.62, 0.1)
  k.mk(k.cyl(0.03, 0.03, 1.5, 6), k.M(WOOD), 0, 0.35, 0, arm)
  k.mk(k.cone(0.07, 0.24, 6), k.M(STEEL), 0, 1.2, 0, arm)
  k.mk(k.rb(0.12, 0.05, 0.05, 0.02), k.M(GOLD), 0, 1.06, 0, arm)
  g.userData = { rig, head, eyes: ey, arms: [arm], h: 1.45 } satisfies Rig
  return g
}

function raider(k: Kit) {
  const g = new THREE.Group(), rig = k.grp(g)
  for (const s of [-1, 1]) k.mk(k.caps(0.08, 0.16), k.M(0x3a2a20), s * 0.12, 0.15, 0, rig)
  k.mk(k.rb(0.46, 0.46, 0.34, 0.14), k.M(0xc8463a), 0, 0.52, 0, rig)
  k.mk(k.rb(0.5, 0.07, 0.38, 0.03), k.M(0x3a2a20), 0, 0.34, 0, rig)
  k.mk(k.cone(0.34, 0.5, 10), k.M(0xa8332a), 0, 0.5, -0.1, rig).rotation.x = -0.2
  const head = k.grp(rig, 0, 0.94, 0)
  k.mk(k.sph(0.24), k.M(SKIN), 0, 0, 0, head)
  k.mk(k.cone(0.29, 0.5, 12), k.M(0xc8463a), 0, 0.14, -0.04, head).rotation.x = -0.35
  k.mk(k.rb(0.34, 0.08, 0.06, 0.03), k.MB(0x1e1e1e), 0, -0.07, 0.2, head)
  const ey = eyes(k, head, 0.08, 0.03, 0.2, 0.06)
  const blades: THREE.Group[] = []
  for (const s of [-1, 1]) {
    const a = k.grp(rig, s * 0.32, 0.52, 0.12)
    a.rotation.z = s * -0.5
    k.mk(k.rb(0.05, 0.12, 0.05, 0.02), k.M(0x3a2a20), 0, 0, 0, a)
    k.mk(k.rb(0.12, 0.04, 0.06, 0.02), k.M(GOLD), 0, 0.07, 0, a)
    k.mk(k.rb(0.05, 0.5, 0.02, 0.015), k.M(0xe8eef5), 0, 0.33, 0, a)
    blades.push(a)
  }
  g.userData = { rig, head, eyes: ey, arms: blades, h: 1.3 } satisfies Rig
  return g
}

function archer(k: Kit) {
  const g = new THREE.Group(), rig = k.grp(g)
  const GREEN = 0x4f9a3a, GREEN_D = 0x356e27
  for (const s of [-1, 1]) k.mk(k.caps(0.08, 0.16), k.M(0x5a3f22), s * 0.12, 0.15, 0, rig)
  k.mk(k.rb(0.44, 0.46, 0.32, 0.14), k.M(GREEN), 0, 0.52, 0, rig)
  k.mk(k.rb(0.48, 0.07, 0.36, 0.03), k.M(WOOD_D), 0, 0.34, 0, rig)
  k.mk(k.rb(0.1, 0.1, 0.05, 0.02), k.M(GOLD), 0, 0.34, 0.19, rig)
  // Quiver on the back with fletched arrows.
  const q = k.grp(rig, 0.1, 0.62, -0.22)
  q.rotation.z = -0.3
  k.mk(k.cyl(0.08, 0.07, 0.42, 8), k.M(0x8a5a2b), 0, 0, 0, q)
  for (const x of [-0.03, 0.03]) k.mk(k.cone(0.04, 0.08, 4), k.M(0xf2f2f2), x, 0.26, 0, q)
  const head = k.grp(rig, 0, 0.94, 0)
  k.mk(k.sph(0.24), k.M(SKIN), 0, 0, 0, head)
  k.mk(k.sph(0.27, 14, 10), k.M(GREEN_D), 0, 0.06, -0.05, head).scale.set(1, 0.95, 1.05)
  k.mk(k.cone(0.12, 0.3, 8), k.M(GREEN_D), 0, 0.14, -0.28, head).rotation.x = -1.2
  const ey = eyes(k, head, 0.08, 0.02, 0.2, 0.06)
  // The bow in the left hand, the drawing arm on the right.
  const bow = k.grp(rig, -0.34, 0.62, 0.16)
  k.mk(k.geo('bow', () => new THREE.TorusGeometry(0.42, 0.025, 6, 16, Math.PI)), k.M(WOOD), 0, 0, 0, bow).rotation.z = -Math.PI / 2
  k.mk(k.cyl(0.006, 0.006, 0.84, 4), k.MB(0xeeeeee), 0, 0, 0, bow).userData.noOL = 1
  const arm = k.grp(rig, 0.3, 0.6, 0.1)
  k.mk(k.rb(0.08, 0.26, 0.08, 0.03), k.M(GREEN), 0, 0.1, 0, arm)
  g.userData = { rig, head, eyes: ey, arms: [arm], h: 1.3 } satisfies Rig
  return g
}

function rider(k: Kit) {
  const g = new THREE.Group(), rig = k.grp(g)
  // The pony: a round body, four short legs, a neck and a mane; the rig carries both so they gallop together.
  const HORSE = 0x9a6436, HORSE_D = 0x6e4422
  k.mk(k.caps(0.26, 0.55), k.M(HORSE), 0, 0.55, 0, rig).rotation.x = Math.PI / 2
  for (const [x, z] of [[-0.16, 0.28], [0.16, 0.28], [-0.16, -0.28], [0.16, -0.28]]) k.mk(k.caps(0.07, 0.28), k.M(HORSE_D), x, 0.2, z, rig)
  const neck = k.grp(rig, 0, 0.78, 0.38)
  neck.rotation.x = 0.5
  k.mk(k.caps(0.13, 0.28), k.M(HORSE), 0, 0.12, 0, neck)
  k.mk(k.rb(0.2, 0.22, 0.34, 0.08), k.M(HORSE), 0, 0.32, 0.12, neck)
  k.mk(k.rb(0.06, 0.34, 0.12, 0.03), k.M(0x3a2414), 0, 0.2, -0.1, neck)
  k.mk(k.rb(0.5, 0.05, 0.5, 0.02), k.M(BLUE_D), 0, 0.82, -0.02, rig)
  // The rider.
  k.mk(k.rb(0.36, 0.38, 0.28, 0.12), k.M(BLUE), 0, 1.04, -0.05, rig)
  k.mk(k.rb(0.4, 0.06, 0.32, 0.02), k.M(GOLD), 0, 0.88, -0.05, rig)
  const head = k.grp(rig, 0, 1.42, -0.05)
  k.mk(k.sph(0.2), k.M(SKIN), 0, 0, 0, head)
  k.mk(k.sph(0.22, 14, 10), k.M(STEEL), 0, 0.06, -0.02, head).scale.set(1, 0.8, 1)
  k.mk(k.cone(0.05, 0.22, 6), k.M(0xd0453a), 0, 0.28, -0.04, head)
  const ey = eyes(k, head, 0.07, 0, 0.17, 0.05)
  const arm = k.grp(rig, 0.26, 1.05, 0.05)
  arm.rotation.x = 1.25
  k.mk(k.cyl(0.03, 0.03, 1.3, 6), k.M(WOOD), 0, 0.45, 0, arm)
  k.mk(k.cone(0.06, 0.26, 6), k.M(STEEL), 0, 1.2, 0, arm)
  const pen = k.mk(k.rb(0.02, 0.14, 0.22, 0.01), k.M(GOLD), 0, 0.95, -0.12, arm)
  pen.userData.noOL = 1
  g.userData = { rig, head, eyes: ey, arms: [arm], h: 1.7 } satisfies Rig
  return g
}

function ram(k: Kit) {
  const g = new THREE.Group(), rig = k.grp(g)
  // A covered frame on four wheels with a swinging iron-headed log.
  for (const x of [-0.45, 0.45]) for (const z of [-0.5, 0.5]) {
    const w = k.mk(k.cyl(0.22, 0.22, 0.12, 14), k.M(WOOD_D), x, 0.22, z, rig)
    w.rotation.z = Math.PI / 2
    k.mk(k.cyl(0.07, 0.07, 0.14, 8), k.M(GOLD), 0, 0, 0, w)
  }
  k.mk(k.rb(0.9, 0.12, 1.3, 0.04), k.M(WOOD), 0, 0.38, 0, rig)
  for (const x of [-0.4, 0.4]) for (const z of [-0.55, 0.55]) k.mk(k.rb(0.08, 0.7, 0.08, 0.02), k.M(WOOD_D), x, 0.75, z, rig)
  const roof = k.mk(k.geo('ramroof', () => {
    const s = new THREE.Shape(); s.moveTo(-0.6, 0); s.lineTo(0.6, 0); s.lineTo(0, 0.45); s.closePath()
    return new THREE.ExtrudeGeometry(s, { depth: 1.4, bevelEnabled: true, bevelThickness: 0.04, bevelSize: 0.04, bevelSegments: 1 }).translate(0, 0, -0.7)
  }), k.M(BLUE), 0, 1.08, 0, rig)
  roof.userData.roof = 1
  k.mk(k.rb(0.08, 0.06, 1.46, 0.02), k.M(GOLD), 0, 1.55, 0, rig)
  const log = k.grp(rig, 0, 0.72, 0)
  k.mk(k.cyl(0.13, 0.13, 1.6, 10), k.M(WOOD), 0, 0, 0.1, log).rotation.x = Math.PI / 2
  const tip = k.mk(k.cyl(0.18, 0.2, 0.3, 10), k.M(0x6f7780), 0, 0, 0.95, log)
  tip.rotation.x = Math.PI / 2
  k.mk(k.sph(0.12, 10, 8), k.M(0x6f7780), 0, 0, 1.12, log)
  for (const z of [-0.3, 0.4]) k.mk(k.cyl(0.03, 0.03, 0.36, 6), k.M(0x4a4a58), 0, 0.17, z, log)
  k.flag(rig, 0.4, 1.0, -0.6, 0.45, 0.26, BLUE, GOLD)
  g.userData = { rig, head: log, arms: [log], h: 1.6 } satisfies Rig
  return g
}

export type TroopType = 'guard' | 'raider' | 'archer' | 'rider' | 'ram'
const builders: Record<TroopType, (k: Kit) => THREE.Group> = { guard, raider, archer, rider, ram }

/** troopModel: one soldier with its outline; unknown types become guards. */
export function troopModel(k: Kit, type: string): THREE.Group {
  const g = (builders[type as TroopType] || guard)(k)
  k.outline(g)
  return g
}

/**
 * animateTroop: idle breathing and sway, a march bounce, and an attack swing.
 * `phase` keeps a crowd from moving in lockstep.
 */
export function animateTroop(g: THREE.Group, type: string, t: number, phase: number, mode: 'idle' | 'walk' | 'attack') {
  const u = g.userData as Rig
  if (type === 'ram') {
    const swing = mode === 'attack' ? Math.sin(t * 6 + phase) : Math.sin(t * 1.4 + phase) * 0.15
    u.head.position.z = swing * 0.25
    u.rig.position.y = mode === 'walk' ? Math.abs(Math.sin(t * 8 + phase)) * 0.04 : 0
    return
  }
  const walk = mode === 'walk'
  u.rig.position.y = Math.abs(Math.sin(t * (walk ? 9 : 2.4) * 0.5 + phase)) * (walk ? 0.1 : 0.03)
  const br = Math.sin(t * 2.4 + phase) * 0.03
  u.rig.scale.set(1 - br * 0.5, 1 + br, 1 - br * 0.5)
  u.rig.rotation.z = Math.sin(t * (walk ? 4.5 : 1.7) + phase) * (walk ? 0.07 : 0.04)
  u.head.rotation.y = Math.sin(t * 0.9 + phase) * 0.25
  const a = u.arms?.[u.arms.length - 1]
  // The rider's lance stays couched forward; a thrust instead of a swing.
  const rest = type === 'rider' ? 1.25 : 0
  if (a && mode === 'attack') a.rotation.x = rest - Math.abs(Math.sin(t * 7 + phase)) * (type === 'rider' ? 0.3 : 1.3)
  else if (a) a.rotation.x = rest
  if (u.eyes) u.eyes.forEach(e => (e.scale.y = Math.sin(t * 0.7 + phase * 3) > 0.985 ? 0.15 : 1))
}

export const randomPhase = () => rand() * 6
