// City buildings by level: the material and the silhouette say the level at a glance —
// wood and straw, then stone and tile, then metal, then gold and crystal.
import * as THREE from 'three'
import type { BuildingType } from '../types'
import { Kit, rand } from './kit'
import { GOLD, ROOF, STONE, STONE_D, STONE_L, WOOD, WOOD_D } from './fortressWorld'

export const STRAW = 0xe2b65a, STRAW_D = 0xc9973d, TILE = 0xd0603f, METAL = 0x7d8ba3, BARRACKS_RED = 0xc8463a

/** Animated parts of a building: chimney markers (smoke rises from them) and the citadel crystal. */
export interface Live { smoke: THREE.Object3D[]; crystal?: { c: THREE.Object3D; gl: THREE.Sprite } }

const trimOf = (level: number) => (level >= 5 ? GOLD : level >= 4 ? METAL : null)

function gable(k: Kit, w: number, h: number, L: number) {
  return k.geo(`gab${w},${h},${L}`, () => {
    const s = new THREE.Shape()
    s.moveTo(-w / 2, 0); s.lineTo(w / 2, 0); s.lineTo(0, h); s.closePath()
    return new THREE.ExtrudeGeometry(s, { depth: L, bevelEnabled: true, bevelThickness: 0.06, bevelSize: 0.06, bevelSegments: 2 }).translate(0, 0, -L / 2)
  })
}

/** Town hall: hut (1), house (2), stone hall (3), hall with a turret (4), citadel with a crystal (5). */
function townhall(k: Kit, g: THREE.Group, level: number, live: Live) {
  const H = k.grp(g)
  H.rotation.y = -0.42
  if (level <= 2) {
    const w = level === 1 ? 1.5 : 1.9, d = level === 1 ? 1.3 : 1.5, wh = level === 1 ? 0.78 : 0.9
    k.mk(k.rb(w + 0.2, 0.18, d + 0.2, 0.06), k.M(0xa79e8a), 0, 0.14, 0, H)
    k.mk(k.rb(w, wh, d, 0.1), k.M(WOOD), 0, 0.23 + wh / 2, 0, H)
    for (let y = 0.4; y < wh; y += 0.2) k.mk(k.rb(w + 0.02, 0.05, d + 0.02, 0.02), k.M(WOOD_D), 0, 0.23 + y, 0, H)
    for (const x of [-w / 2, w / 2]) for (const z of [-d / 2, d / 2]) k.mk(k.rb(0.14, wh + 0.05, 0.14, 0.04), k.M(WOOD_D), x, 0.23 + wh / 2, z, H)
    k.mk(k.rb(0.42, 0.62, 0.08, 0.05), k.M(0x5a3418), 0, 0.54, d / 2 + 0.02, H)
    k.mk(k.rb(0.3, 0.26, 0.08, 0.04), k.M(0x2e2420), w * 0.3, 0.3 + wh * 0.6, d / 2 + 0.02, H)
    if (level === 1) {
      k.mk(k.cone(1.3, 0.95, 4), k.M(STRAW), 0, 0.23 + wh + 0.47, 0, H).rotation.y = Math.PI / 4
      k.mk(k.cone(1.0, 0.62, 4), k.M(0xedc877), 0, 0.23 + wh + 0.74, 0, H).rotation.y = Math.PI / 4
      k.mk(k.sph(0.1, 8, 6), k.M(STRAW_D), 0, 0.23 + wh + 1.0, 0, H)
    } else {
      k.mk(gable(k, w + 0.5, 1.05, d + 0.5), k.M(STRAW), 0, 0.23 + wh, 0, H).rotation.y = Math.PI / 2
      k.mk(gable(k, w + 0.1, 0.8, d + 0.56), k.M(0xedc877), 0, 0.23 + wh + 0.22, 0, H).rotation.y = Math.PI / 2
      k.mk(k.rb(0.3, 0.7, 0.3, 0.05), k.M(0xa79e8a), -0.5, 0.23 + wh + 0.6, -0.3, H)
      live.smoke.push(k.grp(H, -0.5, 0.23 + wh + 1.0, -0.3))
      k.mk(k.rb(0.9, 0.06, 0.5, 0.02), k.M(STRAW), 0, 0.9, d / 2 + 0.3, H)
      for (const x of [-0.4, 0.4]) k.mk(k.rb(0.07, 0.7, 0.07, 0.02), k.M(WOOD_D), x, 0.55, d / 2 + 0.5, H)
    }
    return
  }
  const w = level === 3 ? 2.0 : 2.1, d = level === 3 ? 1.7 : 1.8, wh = level === 3 ? 1.05 : 1.15
  k.mk(k.rb(w + 0.25, 0.25, d + 0.25, 0.08), k.M(STONE_D), 0, 0.14, 0, H)
  k.mk(k.rb(w, wh, d, 0.12), k.M(STONE), 0, 0.25 + wh / 2, 0, H)
  for (let i = 0; i < 8; i++) k.mk(k.rb(0.3, 0.16, 0.05, 0.03), k.M(rand() > 0.5 ? STONE_L : STONE_D), (rand() - 0.5) * (w - 0.4), 0.45 + rand() * (wh - 0.4), d / 2 + 0.01, H)
  k.mk(k.rb(0.5, 0.72, 0.1, 0.06), k.M(WOOD), 0, 0.6, d / 2 + 0.03, H)
  k.mk(k.rb(0.6, 0.12, 0.14, 0.04), k.M(level >= 4 ? METAL : STONE_D), 0, 0.98, d / 2 + 0.04, H)
  for (const x of [-0.65, 0.65]) {
    k.mk(k.rb(0.32, 0.38, 0.1, 0.05), k.M(0x2e2420), x, 0.25 + wh * 0.62, d / 2 + 0.03, H)
    k.mk(k.rb(0.42, 0.08, 0.14, 0.03), k.M(level === 5 ? GOLD : level === 4 ? METAL : WOOD), x, 0.25 + wh * 0.62 - 0.22, d / 2 + 0.04, H)
  }
  const trim = trimOf(level)
  if (trim) {
    for (const x of [-w / 2, w / 2]) for (const z of [-d / 2, d / 2]) k.mk(k.rb(0.18, wh + 0.1, 0.18, 0.05), k.M(trim), x, 0.25 + wh / 2, z, H)
    k.mk(k.rb(w + 0.08, 0.12, d + 0.08, 0.04), k.M(trim), 0, 0.25 + wh, 0, H)
  }
  const rh = level === 3 ? 1.05 : 1.2
  k.mk(gable(k, w + 0.5, rh, d + 0.4), k.M(level === 3 ? TILE : ROOF), 0, 0.25 + wh + 0.04, 0, H)
  for (let i = 1; i < 4; i++) k.mk(gable(k, w + 0.5 - i * 0.28, 0.05, d + 0.44), k.M(level === 3 ? 0xb04a30 : 0x335c98), 0, 0.25 + wh + 0.04 + (rh * i) / 4.2, 0, H)
  k.mk(k.rb(0.12, 0.12, d + 0.5, 0.05), k.M(level === 5 ? GOLD : level === 4 ? METAL : 0xb04a30), 0, 0.25 + wh + rh + 0.04, 0, H)
  if (level === 3) {
    k.mk(k.rb(0.3, 0.62, 0.3, 0.05), k.M(STONE_D), 0.6, 0.25 + wh + 0.7, -0.3, H)
    live.smoke.push(k.grp(H, 0.6, 0.25 + wh + 1.1, -0.3))
  }
  if (level >= 4) {
    for (const x of level === 5 ? [-1.25, 1.25] : [1.2]) {
      k.mk(k.cyl(0.36, 0.4, wh + 0.9, 12), k.M(STONE), x, 0.25 + (wh + 0.9) / 2, -0.35, H)
      k.mk(k.cyl(0.42, 0.42, 0.1, 12), k.M(trim!), x, 0.25 + wh + 0.9, -0.35, H)
      k.mk(k.cone(0.56, 0.9, 10), k.M(ROOF), x, 0.25 + wh + 1.35, -0.35, H)
      k.mk(k.sph(0.08, 8, 6), k.M(trim!), x, 0.25 + wh + 1.82, -0.35, H)
      k.mk(k.rb(0.16, 0.26, 0.08, 0.04), k.M(0x2e2420), x, 0.25 + wh * 0.8, 0.03, H)
    }
    k.flag(H, level === 4 ? 1.2 : -1.25, 0.25 + wh + 1.7, -0.35, 0.7, 0.38, 0x3e8ef0)
  }
  if (level === 5) {
    const cr = k.grp(H, 0, 0.25 + wh + rh + 0.5, 0)
    k.mk(k.cyl(0.2, 0.3, 0.25, 8), k.M(GOLD), 0, -0.3, 0, cr)
    const c = k.mk(k.geo('oct', () => new THREE.OctahedronGeometry(0.36, 0)), k.M(0x63f0ff, { emissive: 0x2ab8e0, emissiveIntensity: 0.7 }), 0, 0.1, 0, cr)
    c.scale.y = 1.4
    const gl = k.glow(0x6af2ff, 2.2, 0.8)
    gl.position.set(0, 0.1, 0)
    cr.add(gl)
    live.crystal = { c, gl }
    for (const x of [-0.4, 0.4]) k.mk(k.sph(0.08, 8, 6), k.M(GOLD), x, 0.25 + wh + rh * 0.5, d / 2 + 0.2, H)
  }
}

/** Barracks: a long house with a red roof and a war banner; a training dummy from level 3. */
function barracks(k: Kit, g: THREE.Group, level: number) {
  const H = k.grp(g)
  H.rotation.y = 0.3
  const stone = level >= 2, w = 1.7 + level * 0.08, d = 1.15, wh = 0.75 + level * 0.06
  k.mk(k.rb(w + 0.2, 0.18, d + 0.2, 0.06), k.M(stone ? STONE_D : 0xa79e8a), 0, 0.12, 0, H)
  k.mk(k.rb(w, wh, d, 0.1), k.M(stone ? STONE : WOOD), 0, 0.2 + wh / 2, 0, H)
  if (!stone) for (let y = 0.35; y < wh; y += 0.2) k.mk(k.rb(w + 0.02, 0.05, d + 0.02, 0.02), k.M(WOOD_D), 0, 0.2 + y, 0, H)
  k.mk(k.rb(0.4, 0.55, 0.08, 0.05), k.M(0x5a3418), -0.35, 0.48, d / 2 + 0.02, H)
  for (const x of [0.25, 0.65]) k.mk(k.rb(0.24, 0.22, 0.08, 0.04), k.M(0x2e2420), x, 0.2 + wh * 0.62, d / 2 + 0.02, H)
  const trim = trimOf(level)
  if (trim) k.mk(k.rb(w + 0.08, 0.1, d + 0.08, 0.04), k.M(trim), 0, 0.2 + wh, 0, H)
  k.mk(gable(k, d + 0.45, 0.75, w + 0.35), k.M(BARRACKS_RED), 0, 0.2 + wh + 0.03, 0, H).rotation.y = Math.PI / 2
  // Crossed spears over the door.
  for (const s of [-1, 1]) { const sp = k.mk(k.cyl(0.025, 0.025, 0.9, 6), k.M(WOOD_D), -0.35, 0.95, d / 2 + 0.1, H); sp.rotation.z = s * 0.6; k.mk(k.cone(0.06, 0.16, 6), k.M(0xdde4ee), 0, 0.5, 0, sp) }
  k.flag(H, w / 2 - 0.1, 0.2 + wh + 0.2, -d / 2 + 0.1, 0.55, 0.32, BARRACKS_RED, GOLD)
  if (level >= 3) {
    const D = k.grp(g, 0.95, 0, 0.75)
    k.mk(k.cyl(0.04, 0.04, 0.8, 6), k.M(WOOD_D), 0, 0.4, 0, D)
    k.mk(k.cyl(0.18, 0.15, 0.4, 8), k.M(STRAW), 0, 0.7, 0, D)
    k.mk(k.sph(0.12, 8, 6), k.M(STRAW), 0, 1.0, 0, D)
    k.mk(k.rb(0.5, 0.06, 0.06, 0.02), k.M(WOOD_D), 0, 0.75, 0, D)
  }
}

/** Gold storage: a vault with a growing heap of coins; iron bands, then gold bands. */
function storage(k: Kit, g: THREE.Group, level: number) {
  const r = 0.62 + level * 0.05, h = 0.7 + level * 0.12
  k.mk(k.cyl(r + 0.12, r + 0.18, 0.2, 14), k.M(STONE_D), 0, 0.1, 0, g)
  k.mk(k.cyl(r, r + 0.05, h, 14), k.M(level >= 2 ? STONE : WOOD), 0, 0.2 + h / 2, 0, g)
  const band = level >= 4 ? GOLD : level >= 3 ? METAL : WOOD_D
  for (const y of [0.35, h - 0.05]) k.mk(k.cyl(r + 0.03, r + 0.03, 0.08, 14), k.M(band), 0, 0.2 + y, 0, g)
  k.mk(k.cyl(r + 0.08, r + 0.02, 0.12, 14), k.M(band), 0, 0.2 + h + 0.06, 0, g)
  // The coin heap on top.
  const heap = k.grp(g, 0, 0.2 + h + 0.12, 0)
  k.mk(k.sph(r * 0.85, 14, 8), k.M(GOLD), 0, -r * 0.45 + level * 0.06, 0, heap).scale.y = 0.55
  for (let i = 0; i < 5 + level * 2; i++) {
    const a = rand() * Math.PI * 2, rr = rand() * r * 0.6
    k.mk(k.cyl(0.11, 0.11, 0.035, 10), k.M(0xffe27a), Math.cos(a) * rr, 0.05 + rand() * 0.15 + level * 0.04, Math.sin(a) * rr, heap).rotation.set(rand() * 0.6, 0, rand() * 0.6)
  }
  k.mk(k.rb(0.36, 0.46, 0.08, 0.05), k.M(0x5a3418), 0, 0.45, r + 0.02, g).rotation.y = 0
}

/** Wall: a block that joins its neighbours — palisade logs, then stone, then capped and gilded stone. */
function wall(k: Kit, g: THREE.Group, level: number, joins: { e: boolean; s: boolean }) {
  const col = level === 1 ? WOOD : level >= 4 ? 0x8f8676 : STONE, cap = level >= 5 ? GOLD : level >= 4 ? METAL : level >= 2 ? STONE_L : WOOD_D
  const h = 0.75 + level * 0.07
  if (level === 1) {
    for (let i = 0; i < 3; i++) for (let j = 0; j < 3; j++) if ((i === 1 || j === 1)) {
      const lg = k.mk(k.cyl(0.13, 0.13, h, 7), k.M(i === 1 && j === 1 ? WOOD : WOOD_D), (i - 1) * 0.22, h / 2, (j - 1) * 0.22, g)
      k.mk(k.cone(0.13, 0.18, 7), k.M(WOOD), 0, h / 2 + 0.09, 0, lg)
    }
  } else {
    k.mk(k.rb(0.82, h, 0.82, 0.1), k.M(col), 0, h / 2, 0, g)
    k.mk(k.rb(0.9, 0.12, 0.9, 0.05), k.M(cap), 0, h + 0.04, 0, g)
  }
  // Links toward the east and south neighbours (CELL = 1.8 apart): each pair of walls is joined once.
  for (const [on, rot] of [[joins.e, 0], [joins.s, Math.PI / 2]] as const) {
    if (!on) continue
    const L = k.grp(g)
    L.rotation.y = -rot
    if (level === 1) for (let i = 1; i <= 3; i++) { const lg = k.mk(k.cyl(0.11, 0.11, h * 0.9, 7), k.M(WOOD), i * 0.45, (h * 0.9) / 2, 0, L); k.mk(k.cone(0.11, 0.15, 7), k.M(WOOD), 0, (h * 0.9) / 2 + 0.07, 0, lg) }
    else { k.mk(k.rb(1.2, h * 0.86, 0.6, 0.08), k.M(col), 0.9, (h * 0.86) / 2, 0, L); k.mk(k.rb(1.2, 0.08, 0.66, 0.03), k.M(cap), 0.9, h * 0.86 + 0.02, 0, L) }
  }
}

/** Defence tower: wood, then stone with a blue roof, a metal crown at 4, a golden one at 5. */
function tower(k: Kit, g: THREE.Group, level: number) {
  const r = 0.42 + level * 0.03, h = 1.3 + level * 0.22, stone = level >= 2
  k.mk(k.cyl(r + 0.16, r + 0.24, 0.3, 12), k.M(STONE_D), 0, 0.15, 0, g)
  k.mk(k.cyl(r, r + 0.06, h, 12), k.M(stone ? STONE : WOOD), 0, 0.3 + h / 2, 0, g)
  if (!stone) for (const y of [0.5, 1.0]) k.mk(k.cyl(r + 0.02, r + 0.02, 0.06, 12), k.M(WOOD_D), 0, 0.3 + y, 0, g)
  const trim = trimOf(level) ?? (stone ? STONE_D : WOOD_D)
  k.mk(k.cyl(r + 0.22, r + 0.08, 0.3, 12), k.M(trim), 0, 0.3 + h + 0.1, 0, g)
  for (let i = 0; i < 7; i++) { const a = (i / 7) * Math.PI * 2; k.mk(k.rb(0.2, 0.22, 0.16, 0.05), k.M(stone ? STONE : WOOD), Math.sin(a) * (r + 0.12), 0.3 + h + 0.36, Math.cos(a) * (r + 0.12), g).rotation.y = a }
  const roofH = 0.7 + level * 0.12
  k.mk(k.cone(r + 0.4, roofH, 12), k.M(stone ? ROOF : STRAW), 0, 0.3 + h + 0.4 + roofH / 2, 0, g)
  k.mk(k.sph(0.09, 8, 6), k.M(trimOf(level) ?? GOLD), 0, 0.3 + h + 0.42 + roofH, 0, g)
  k.mk(k.rb(0.2, 0.32, 0.08, 0.05), k.M(0x2e2420), 0, 0.3 + h * 0.6, r + 0.02, g)
  if (level >= 3) k.flag(g, 0, 0.3 + h + 0.4 + roofH - 0.05, 0, 0.5, 0.28)
}

/**
 * buildingModel: a building of the given level, its footprint fitted to one grid cell (CELL units wide).
 * `joins` connects walls to neighbouring walls.
 */
export function buildingModel(k: Kit, type: BuildingType, level: number, joins: { e: boolean; s: boolean }, live: Live): THREE.Group {
  const g = new THREE.Group(), inner = k.grp(g)
  const L = Math.max(1, level)
  switch (type) {
    case 'townhall': townhall(k, inner, Math.min(5, L), live); inner.scale.setScalar(0.78); break
    case 'barracks': barracks(k, inner, Math.min(4, L)); inner.scale.setScalar(0.8); break
    case 'storage': storage(k, inner, Math.min(4, L)); inner.scale.setScalar(0.92); break
    case 'wall': wall(k, inner, Math.min(5, L), joins); break
    case 'tower': tower(k, inner, Math.min(5, L)); break
  }
  if (type !== 'wall') k.outline(inner, 0.03)
  k.blob(g, 0, 0, type === 'wall' ? 1.2 : 2.0, type === 'wall' ? 1.2 : 1.7, 0.4)
  return g
}

/** scaffold: wooden poles and planks around a building under construction. */
export function scaffold(k: Kit, h: number): THREE.Group {
  const g = new THREE.Group()
  for (const x of [-0.85, 0.85]) for (const z of [-0.85, 0.85]) k.mk(k.cyl(0.04, 0.04, h, 6), k.M(0xc8964f), x, h / 2, z, g)
  for (const y of [h * 0.45, h * 0.9]) for (const [x, z, r] of [[0, 0.85, 0], [0, -0.85, 0], [0.85, 0, Math.PI / 2], [-0.85, 0, Math.PI / 2]]) k.mk(k.rb(1.8, 0.06, 0.16, 0.02), k.M(0xb07a45), x, y, z, g).rotation.y = r
  const sign = k.grp(g, 0.95, h * 0.55, 0.95)
  k.mk(k.rb(0.5, 0.36, 0.05, 0.03), k.M(0xffd323), 0, 0, 0, sign).rotation.y = Math.PI / 4
  return g
}
