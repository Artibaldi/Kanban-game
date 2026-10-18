// City buildings by level: the material and the silhouette say the level at a glance —
// wood and straw, then stone and tile, then metal, then gold and crystal.
import * as THREE from 'three'
import type { BuildingType } from '../types'
import { Kit, rand } from './kit'
import { GOLD, ROOF, STONE, STONE_D, STONE_L, WOOD, WOOD_D } from './fortressWorld'

export const STRAW = 0xe2b65a, STRAW_D = 0xc9973d, TILE = 0xd0603f, METAL = 0x7d8ba3, BARRACKS_RED = 0xc8463a

/** Animated parts of a building: chimney markers (smoke rises from them) and the citadel crystal. */
export interface Live { smoke: THREE.Object3D[]; crystal?: { c: THREE.Object3D; gl: THREE.Sprite }; glows?: THREE.Sprite[] }

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

/** Tavern: a timber inn with a mug on its sign; stone ground floor and tiles at 2, a terrace with lanterns at 3. */
function tavern(k: Kit, g: THREE.Group, level: number) {
  const H = k.grp(g)
  H.rotation.y = -0.35
  const w = 1.6, d = 1.3, low = 0.62, up = 0.55
  k.mk(k.rb(w + 0.2, 0.16, d + 0.2, 0.06), k.M(level >= 2 ? STONE_D : 0xa79e8a), 0, 0.1, 0, H)
  k.mk(k.rb(w, low, d, 0.08), k.M(level >= 2 ? STONE : WOOD), 0, 0.18 + low / 2, 0, H)
  k.mk(k.rb(w + 0.16, up, d + 0.16, 0.08), k.M(0xf1e2c2), 0, 0.18 + low + up / 2, 0, H)
  for (const x of [-w / 2 - 0.08, 0, w / 2 + 0.08]) k.mk(k.rb(0.08, up, d + 0.2, 0.02), k.M(WOOD_D), x, 0.18 + low + up / 2, 0, H)
  k.mk(k.rb(w + 0.2, 0.07, d + 0.2, 0.02), k.M(WOOD_D), 0, 0.18 + low, 0, H)
  k.mk(gable(k, d + 0.5, 0.8, w + 0.5), k.M(level >= 2 ? TILE : STRAW), 0, 0.18 + low + up + 0.02, 0, H).rotation.y = Math.PI / 2
  k.mk(k.rb(0.42, 0.55, 0.08, 0.05), k.M(0x5a3418), -0.3, 0.46, d / 2 + 0.02, H)
  for (const x of [0.3, 0.6]) k.mk(k.rb(0.24, 0.24, 0.08, 0.04), k.M(0xffd36a, { emissive: 0xffa43a, emissiveIntensity: 0.5 }), x, 0.18 + low * 0.6, d / 2 + 0.02, H)
  for (const x of [-0.45, 0.45]) k.mk(k.rb(0.24, 0.24, 0.08, 0.04), k.M(0xffd36a, { emissive: 0xffa43a, emissiveIntensity: 0.5 }), x, 0.18 + low + up / 2, d / 2 + 0.1, H)
  // The sign: a golden mug on a bracket.
  const S = k.grp(H, w / 2 + 0.12, 0.18 + low + 0.05, d / 2 - 0.1)
  k.mk(k.rb(0.4, 0.04, 0.04, 0.01), k.M(WOOD_D), 0.18, 0.2, 0, S)
  const board = k.mk(k.rb(0.04, 0.34, 0.34, 0.03), k.M(WOOD), 0.34, 0, 0, S)
  k.mk(k.cyl(0.07, 0.06, 0.14, 10), k.M(GOLD), 0.03, 0, 0, board)
  for (const [x, z] of [[-w / 2 - 0.35, d / 2 + 0.1], [-w / 2 - 0.3, d / 2 - 0.3]]) { k.mk(k.cyl(0.14, 0.14, 0.3, 10), k.M(WOOD), x, 0.33, z, H); k.mk(k.cyl(0.15, 0.15, 0.04, 10), k.M(0x4a4a58), x, 0.4, z, H) }
  if (level >= 3) {
    k.mk(k.rb(0.9, 0.06, 0.6, 0.02), k.M(WOOD), 0.25, 0.2, d / 2 + 0.5, H)
    k.mk(k.cyl(0.18, 0.18, 0.04, 10), k.M(WOOD_D), 0.25, 0.42, d / 2 + 0.5, H)
    k.mk(k.cyl(0.03, 0.03, 0.22, 6), k.M(WOOD_D), 0.25, 0.31, d / 2 + 0.5, H)
    for (let i = 0; i < 5; i++) k.mk(k.sph(0.05, 6, 4), k.MB([0xff6fa8, 0xffe15a, 0x7fe0b0, 0x6fb0ff, 0xffa43a][i]), -0.6 + i * 0.3, 0.18 + low + 0.05 - Math.sin((i / 4) * Math.PI) * 0.08, d / 2 + 0.25, H)
    k.flag(H, -w / 2 + 0.1, 0.18 + low + up + 0.3, -d / 2 + 0.1, 0.5, 0.3, 0xffd323, 0xc8463a)
  }
}

/** Builder hut: a small plank cabin with a straw roof, a saw on the wall, a workbench and a stack of beams. */
function hut(k: Kit, g: THREE.Group) {
  const H = k.grp(g)
  H.rotation.y = 0.4
  const w = 1.1, d = 0.95, wh = 0.62
  k.mk(k.rb(w + 0.18, 0.14, d + 0.18, 0.05), k.M(0xa79e8a), 0, 0.08, 0, H)
  k.mk(k.rb(w, wh, d, 0.08), k.M(WOOD), 0, 0.15 + wh / 2, 0, H)
  for (let y = 0.3; y < wh; y += 0.18) k.mk(k.rb(w + 0.02, 0.04, d + 0.02, 0.02), k.M(WOOD_D), 0, 0.15 + y, 0, H)
  k.mk(gable(k, d + 0.35, 0.6, w + 0.3), k.M(STRAW), 0, 0.15 + wh + 0.02, 0, H).rotation.y = Math.PI / 2
  k.mk(k.rb(0.32, 0.46, 0.06, 0.04), k.M(0x5a3418), -0.2, 0.38, d / 2 + 0.02, H)
  // A saw and a hammer hung by the door.
  k.mk(k.rb(0.36, 0.1, 0.03, 0.01), k.M(0xdde4ee), 0.28, 0.55, d / 2 + 0.03, H)
  k.mk(k.rb(0.1, 0.12, 0.03, 0.01), k.M(WOOD_D), 0.1, 0.55, d / 2 + 0.03, H)
  // Workbench with a hammer, and beams waiting to be used.
  const B = k.grp(g, 0.8, 0, 0.55)
  k.mk(k.rb(0.6, 0.06, 0.3, 0.02), k.M(WOOD), 0, 0.36, 0, B)
  for (const x of [-0.24, 0.24]) k.mk(k.rb(0.06, 0.34, 0.24, 0.01), k.M(WOOD_D), x, 0.17, 0, B)
  k.mk(k.cyl(0.02, 0.02, 0.3, 6), k.M(WOOD_D), 0.05, 0.41, 0, B).rotation.z = Math.PI / 2
  k.mk(k.rb(0.1, 0.07, 0.07, 0.02), k.M(METAL), -0.1, 0.41, 0, B)
  for (let i = 0; i < 3; i++) k.mk(k.rb(0.9, 0.1, 0.1, 0.02), k.M(i % 2 ? WOOD : 0xc88b4f), -0.75, 0.07 + i * 0.1, 0.55 - (i % 2) * 0.05, g).rotation.y = 0.3
}

/** Market: a stall with a striped awning, crates of gold and energy crystals, and scales; stone base and a second stall at 2, gold trim at 3. */
function market(k: Kit, g: THREE.Group, level: number, live: Live) {
  const H = k.grp(g)
  H.rotation.y = -0.3
  const stall = (x: number, z: number, stripe: number) => {
    const S = k.grp(H, x, 0, z)
    k.mk(k.rb(1.1, 0.5, 0.6, 0.06), k.M(level >= 2 ? STONE : WOOD), 0, 0.35, 0, S)
    k.mk(k.rb(1.16, 0.07, 0.66, 0.03), k.M(WOOD_D), 0, 0.62, 0, S)
    for (const px of [-0.52, 0.52]) for (const pz of [-0.27, 0.27]) k.mk(k.cyl(0.03, 0.03, 0.75, 6), k.M(WOOD_D), px, 0.98, pz, S)
    // The awning: alternating stripes, tilted forward.
    const A = k.grp(S, 0, 1.38, 0)
    A.rotation.x = 0.28
    for (let i = 0; i < 6; i++) k.mk(k.rb(0.2, 0.05, 0.86, 0.02), k.M(i % 2 ? 0xfff3d6 : stripe), -0.5 + i * 0.2, 0, 0, A)
    if (level >= 3) k.mk(k.rb(1.24, 0.06, 0.06, 0.02), k.M(GOLD), 0, -0.03, 0.43, A)
    return S
  }
  const S = stall(0, 0, 0xd0453a)
  // Goods: a crate of coins and a crate of glowing crystals, and balance scales between them.
  k.mk(k.rb(0.3, 0.12, 0.3, 0.03), k.M(WOOD), -0.3, 0.71, 0, S)
  for (let i = 0; i < 5; i++) k.mk(k.cyl(0.07, 0.07, 0.03, 10), k.M(0xffe27a), -0.36 + (i % 3) * 0.07, 0.8 + Math.floor(i / 3) * 0.03, -0.05 + (i % 2) * 0.08, S)
  k.mk(k.rb(0.3, 0.12, 0.3, 0.03), k.M(WOOD), 0.3, 0.71, 0, S)
  for (const [x, z] of [[0.25, -0.04], [0.35, 0.06]]) k.mk(k.geo('oct', () => new THREE.OctahedronGeometry(0.36, 0)), k.M(0x8a6cff, { emissive: 0x6a4cff, emissiveIntensity: 0.5 }), x, 0.85, z, S).scale.setScalar(0.22)
  k.mk(k.cyl(0.02, 0.02, 0.3, 6), k.M(GOLD), 0, 0.83, 0, S)
  k.mk(k.rb(0.36, 0.02, 0.02, 0.01), k.M(GOLD), 0, 0.98, 0, S)
  for (const x of [-0.17, 0.17]) k.mk(k.cyl(0.07, 0.05, 0.03, 10), k.M(GOLD), x, 0.92, 0, S)
  if (level >= 2) {
    const S2 = stall(0.05, -0.85, 0x3e8ef0)
    for (let i = 0; i < 3; i++) k.mk(k.sph(0.09, 8, 6), k.M([0xf04848, 0x7cc242, 0xffb84a][i]), -0.25 + i * 0.25, 0.72, 0, S2)
  }
  const lamp = k.glow(0xffc46a, 0.9, 0.5)
  lamp.position.set(0, 1.2, 0.4)
  H.add(lamp)
  live.glows ??= []
  live.glows.push(lamp)
}

/** Crystal well: a stone ring holding a glowing energy crystal; taller crystal and more stones by level. */
function well(k: Kit, g: THREE.Group, level: number, live: Live) {
  const r = 0.62 + level * 0.04
  k.mk(k.cyl(r + 0.12, r + 0.18, 0.16, 16), k.M(STONE_D), 0, 0.08, 0, g)
  k.mk(k.cyl(r, r + 0.04, 0.42, 16), k.M(level >= 2 ? STONE_L : STONE), 0, 0.37, 0, g)
  k.mk(k.cyl(r - 0.12, r - 0.12, 0.05, 16), k.M(0x7a5cff, { emissive: 0x5a3cff, emissiveIntensity: 0.6 }), 0, 0.57, 0, g)
  for (let i = 0; i < 8; i++) { const a = (i / 8) * Math.PI * 2; k.mk(k.rb(0.2, 0.1, 0.14, 0.03), k.M(i % 2 ? STONE_L : STONE), Math.cos(a) * r, 0.62, Math.sin(a) * r, g).rotation.y = -a }
  const c = k.mk(k.geo('oct', () => new THREE.OctahedronGeometry(0.36, 0)), k.M(0xa58cff, { emissive: 0x6a4cff, emissiveIntensity: 0.75 }), 0, 1.0 + level * 0.08, 0, g)
  c.scale.set(0.8, 1.3 + level * 0.15, 0.8)
  for (let i = 0; i < level; i++) { const a = i * 2.1 + 0.4; const sm = k.mk(k.geo('oct', () => new THREE.OctahedronGeometry(0.36, 0)), k.M(0xc4b2ff, { emissive: 0x6a4cff, emissiveIntensity: 0.5 }), Math.cos(a) * 0.3, 0.72, Math.sin(a) * 0.3, g); sm.scale.set(0.3, 0.5, 0.3); sm.rotation.z = 0.3 }
  if (level >= 3) for (const x of [-r - 0.05, r + 0.05]) { k.mk(k.cyl(0.05, 0.05, 1.0, 6), k.M(METAL), x, 0.6, 0, g); k.mk(k.sph(0.07, 8, 6), k.M(GOLD), x, 1.12, 0, g) }
  const gl = k.glow(0x9c84ff, 1.8, 0.75)
  gl.position.set(0, 1.0 + level * 0.08, 0)
  g.add(gl)
  live.glows ??= []
  live.glows.push(gl)
}

/** Archery range: a fenced yard with straw targets and a weapon rack; a roofed stand from level 2, banners at 3. */
function archery(k: Kit, g: THREE.Group, level: number) {
  k.mk(k.rb(1.7, 0.08, 1.4, 0.04), k.M(0xc9b48c), 0, 0.05, 0, g)
  for (const [x, z, ry] of [[-0.85, 0, 0], [0.85, 0, 0], [0, -0.7, Math.PI / 2]] as const) {
    const F = k.grp(g, x, 0, z)
    F.rotation.y = ry
    for (const dz of [-0.55, 0, 0.55]) k.mk(k.cyl(0.03, 0.03, 0.4, 6), k.M(WOOD_D), 0, 0.2, dz, F)
    k.mk(k.rb(0.04, 0.04, 1.2, 0.01), k.M(WOOD), 0, 0.32, 0, F)
  }
  // Round straw targets with red rings.
  for (const x of [-0.4, 0.35]) {
    const T = k.grp(g, x, 0, -0.4)
    for (const s of [-1, 1]) k.mk(k.cyl(0.025, 0.025, 0.7, 5), k.M(WOOD_D), s * 0.15, 0.32, -0.08, T).rotation.x = 0.25
    k.mk(k.cyl(0.26, 0.26, 0.1, 16), k.M(STRAW), 0, 0.55, 0, T).rotation.x = Math.PI / 2 - 0.15
    k.mk(k.cyl(0.17, 0.17, 0.11, 16), k.M(0xf2f2f2), 0, 0.55, 0.005, T).rotation.x = Math.PI / 2 - 0.15
    k.mk(k.cyl(0.08, 0.08, 0.12, 12), k.M(0xd0453a), 0, 0.55, 0.01, T).rotation.x = Math.PI / 2 - 0.15
    k.mk(k.cyl(0.01, 0.01, 0.3, 4), k.M(WOOD), 0.05, 0.6, 0.15, T).rotation.x = Math.PI / 2
  }
  // A rack of bows.
  const R = k.grp(g, 0.3, 0, 0.45)
  k.mk(k.rb(0.7, 0.05, 0.08, 0.02), k.M(WOOD_D), 0, 0.55, 0, R)
  for (const x of [-0.3, 0.3]) k.mk(k.cyl(0.03, 0.03, 0.55, 5), k.M(WOOD_D), x, 0.27, 0, R)
  for (const x of [-0.18, 0, 0.18]) k.mk(k.geo('bow', () => new THREE.TorusGeometry(0.42, 0.025, 6, 16, Math.PI)), k.M(WOOD), x, 0.35, 0.05, R).scale.setScalar(0.45)
  if (level >= 2) {
    const S = k.grp(g, -0.45, 0, 0.4)
    for (const [x, z] of [[-0.3, -0.2], [0.3, -0.2], [-0.3, 0.2], [0.3, 0.2]]) k.mk(k.cyl(0.03, 0.03, 0.8, 6), k.M(WOOD_D), x, 0.4, z, S)
    k.mk(gable(k, 0.6, 0.3, 0.8), k.M(level >= 3 ? TILE : STRAW), 0, 0.8, 0, S)
  }
  if (level >= 3) for (const x of [-0.85, 0.85]) k.flag(g, x, 0.4, 0.6, 0.45, 0.26, 0x4f9a3a, GOLD)
}

/** Workshop: a stone smithy with a big gear on the wall, an anvil and a glowing forge; metal roof trim by level. */
function workshop(k: Kit, g: THREE.Group, level: number, live: Live) {
  const H = k.grp(g)
  H.rotation.y = 0.3
  const w = 1.5, d = 1.15, wh = 0.75
  k.mk(k.rb(w + 0.2, 0.16, d + 0.2, 0.06), k.M(STONE_D), 0, 0.1, 0, H)
  k.mk(k.rb(w, wh, d, 0.1), k.M(STONE), 0, 0.18 + wh / 2, 0, H)
  k.mk(gable(k, d + 0.4, 0.6, w + 0.3), k.M(level >= 3 ? METAL : TILE), 0, 0.18 + wh + 0.02, 0, H).rotation.y = Math.PI / 2
  k.mk(k.rb(0.5, 0.55, 0.08, 0.05), k.M(0x5a3418), 0.35, 0.46, d / 2 + 0.02, H)
  // The gear.
  const G = k.grp(H, -0.35, 0.62, d / 2 + 0.06)
  k.mk(k.cyl(0.22, 0.22, 0.06, 16), k.M(METAL), 0, 0, 0, G).rotation.x = Math.PI / 2
  for (let i = 0; i < 8; i++) { const a = (i / 8) * Math.PI * 2; k.mk(k.rb(0.08, 0.08, 0.06, 0.01), k.M(METAL), Math.cos(a) * 0.26, Math.sin(a) * 0.26, 0, G).rotation.z = a }
  k.mk(k.cyl(0.07, 0.07, 0.08, 10), k.M(GOLD), 0, 0, 0.01, G).rotation.x = Math.PI / 2
  // Chimney with smoke, and a forge glowing under it.
  k.mk(k.rb(0.26, 0.6, 0.26, 0.04), k.M(STONE_D), w / 2 - 0.25, 0.18 + wh + 0.4, -0.2, H)
  const chim = k.grp(H, w / 2 - 0.25, 0.18 + wh + 0.75, -0.2)
  live.smoke.push(chim)
  // Anvil outside.
  const A = k.grp(g, 0.95, 0, 0.55)
  k.mk(k.cyl(0.12, 0.16, 0.3, 8), k.M(WOOD_D), 0, 0.15, 0, A)
  k.mk(k.rb(0.36, 0.12, 0.16, 0.03), k.M(0x4a4a58), 0, 0.36, 0, A)
  k.mk(k.cone(0.06, 0.16, 6), k.M(0x4a4a58), 0.24, 0.36, 0, A).rotation.z = -Math.PI / 2
  const fire = k.glow(0xff8a3a, 0.9, 0.6)
  fire.position.set(0.95, 0.5, 0.55)
  g.add(fire)
  live.glows ??= []
  live.glows.push(fire)
}

/** Mortar: a squat stone emplacement with a fat iron barrel aimed at the sky; bands and a bigger barrel by level. */
function mortar(k: Kit, g: THREE.Group, level: number) {
  const r = 0.7
  k.mk(k.cyl(r + 0.1, r + 0.16, 0.2, 10), k.M(STONE_D), 0, 0.1, 0, g)
  k.mk(k.cyl(r, r + 0.04, 0.4, 10), k.M(level >= 2 ? STONE_L : STONE), 0, 0.4, 0, g)
  for (let i = 0; i < 10; i++) { const a = (i / 10) * Math.PI * 2; if (i % 2) k.mk(k.rb(0.22, 0.14, 0.16, 0.03), k.M(STONE), Math.cos(a) * r, 0.66, Math.sin(a) * r, g).rotation.y = -a }
  const B = k.grp(g, 0, 0.62, 0)
  B.rotation.x = -0.55
  const br = 0.22 + level * 0.03
  k.mk(k.cyl(br + 0.04, br + 0.1, 0.32, 12), k.M(0x3c3f48), 0, 0.05, 0, B)
  k.mk(k.cyl(br, br + 0.05, 0.62, 12), k.M(0x4a4e5a), 0, 0.42, 0, B)
  k.mk(k.cyl(br * 0.7, br * 0.7, 0.05, 12), k.MB(0x111111), 0, 0.74, 0, B)
  const band = level >= 3 ? GOLD : METAL
  for (const y of [0.2, 0.6]) k.mk(k.cyl(br + 0.05, br + 0.05, 0.06, 12), k.M(band), 0, y, 0, B)
  // A small stack of shells.
  for (const [x, z] of [[0.45, 0.35], [0.55, 0.22], [0.5, 0.3]]) k.mk(k.sph(0.09, 10, 8), k.M(0x2e2e36), x, 0.74, z, g)
}

/** Trap: a hatch of planks with spikes peeking through — only its owner ever sees it in the city. */
function trap(k: Kit, g: THREE.Group, level: number) {
  k.mk(k.rb(1.0, 0.06, 1.0, 0.03), k.M(0x8a6a44), 0, 0.04, 0, g)
  for (let i = 0; i < 4; i++) k.mk(k.rb(0.22, 0.03, 0.94, 0.01), k.M(i % 2 ? WOOD : WOOD_D), -0.36 + i * 0.24, 0.08, 0, g)
  const n = level >= 2 ? 3 : 2
  for (let i = 0; i < n * n; i++) k.mk(k.cone(0.06, 0.2, 5), k.M(STEEL_TRAP), -0.3 + (i % n) * (0.6 / (n - 1)), 0.18, -0.3 + Math.floor(i / n) * (0.6 / (n - 1)), g)
}
const STEEL_TRAP = 0xc9d2de

/** Stable: a long timber barn with a hay loft, a paddock fence and a horseshoe sign; stone and tiles at 2. */
function stable(k: Kit, g: THREE.Group, level: number) {
  const H = k.grp(g)
  H.rotation.y = -0.25
  const w = 1.8, d = 1.0, wh = 0.7
  k.mk(k.rb(w + 0.2, 0.14, d + 0.2, 0.05), k.M(level >= 2 ? STONE_D : 0xa79e8a), 0, 0.08, 0, H)
  k.mk(k.rb(w, wh, d, 0.08), k.M(level >= 2 ? STONE : WOOD), 0, 0.15 + wh / 2, 0, H)
  for (const x of [-0.55, 0, 0.55]) {
    k.mk(k.rb(0.4, 0.5, 0.06, 0.04), k.M(0x5a3418), x, 0.42, d / 2 + 0.02, H)
    k.mk(k.rb(0.4, 0.04, 0.07, 0.01), k.M(WOOD_D), x, 0.42, d / 2 + 0.03, H)
  }
  k.mk(gable(k, d + 0.4, 0.62, w + 0.3), k.M(level >= 2 ? TILE : STRAW), 0, 0.15 + wh + 0.02, 0, H).rotation.y = Math.PI / 2
  k.mk(k.rb(0.36, 0.3, 0.06, 0.04), k.M(STRAW_D), 0, 0.15 + wh + 0.22, d / 2 + 0.1, H)
  // Horseshoe over the middle door.
  k.mk(k.geo('shoe', () => new THREE.TorusGeometry(0.1, 0.025, 6, 12, Math.PI * 1.4)), k.M(GOLD), 0, 0.15 + wh - 0.06, d / 2 + 0.06, H).rotation.z = -Math.PI * 0.2
  // Paddock fence in front.
  const F = k.grp(g, 0, 0, 0.85)
  for (let i = 0; i < 5; i++) k.mk(k.cyl(0.03, 0.03, 0.36, 5), k.M(WOOD_D), -0.8 + i * 0.4, 0.18, 0, F)
  for (const y of [0.16, 0.3]) k.mk(k.rb(1.66, 0.04, 0.04, 0.01), k.M(WOOD), 0, y, 0, F)
  k.mk(k.cyl(0.16, 0.18, 0.12, 10), k.M(WOOD), 0.65, 0.06, -0.15, F)
  k.mk(k.cyl(0.13, 0.13, 0.03, 10), k.M(0x4db4f0), 0.65, 0.12, -0.15, F)
}

/** Crossbow tower: a square stone tower with a big crossbow on a turntable at the top; metal, then gold trim. */
function crossbow(k: Kit, g: THREE.Group, level: number) {
  const s = 0.82, h = 1.5 + level * 0.18
  k.mk(k.rb(s + 0.3, 0.3, s + 0.3, 0.06), k.M(STONE_D), 0, 0.15, 0, g)
  k.mk(k.rb(s, h, s, 0.08), k.M(STONE), 0, 0.3 + h / 2, 0, g)
  for (let i = 0; i < 6; i++) k.mk(k.rb(0.2, 0.14, 0.05, 0.03), k.M(STONE_L), (i % 2 ? 0.2 : -0.2), 0.6 + i * 0.22, s / 2 + 0.01, g)
  const trim = level >= 3 ? GOLD : METAL
  k.mk(k.rb(s + 0.24, 0.14, s + 0.24, 0.04), k.M(trim), 0, 0.3 + h + 0.07, 0, g)
  for (const [x, z] of [[-1, -1], [1, -1], [-1, 1], [1, 1]]) k.mk(k.rb(0.2, 0.24, 0.2, 0.04), k.M(STONE), x * (s / 2 + 0.02), 0.3 + h + 0.26, z * (s / 2 + 0.02), g)
  // The crossbow: a stock, a wide bow, a loaded bolt.
  const C = k.grp(g, 0, 0.3 + h + 0.3, 0)
  C.rotation.y = 0.6
  k.mk(k.cyl(0.18, 0.22, 0.12, 10), k.M(WOOD_D), 0, 0, 0, C)
  k.mk(k.rb(0.14, 0.12, 0.9, 0.03), k.M(WOOD), 0, 0.14, 0.1, C)
  k.mk(k.rb(1.0, 0.06, 0.08, 0.02), k.M(trim), 0, 0.16, 0.48, C)
  k.mk(k.cyl(0.006, 0.006, 0.95, 4), k.MB(0xeeeeee), 0, 0.16, 0.36, C).rotation.z = Math.PI / 2
  k.mk(k.rb(0.04, 0.04, 0.7, 0.01), k.M(WOOD_D), 0, 0.22, 0.25, C)
  k.mk(k.cone(0.05, 0.12, 4), k.M(STEEL_TRAP), 0, 0.22, 0.65, C).rotation.x = Math.PI / 2
  k.mk(k.rb(0.24, 0.34, 0.08, 0.05), k.M(0x2e2420), 0, 0.3 + h * 0.4, s / 2 + 0.02, g)
}

/** A grand hall for town halls 6 and 7: the citadel with golden obelisks at the corners; banners and a crown of crystals at 7. */
function grandHall(k: Kit, g: THREE.Group, level: number, live: Live) {
  townhall(k, g, 5, live)
  for (const [x, z] of [[-1.25, -1.0], [1.25, -1.0], [-1.25, 1.0], [1.25, 1.0]]) {
    k.mk(k.rb(0.26, 0.2, 0.26, 0.04), k.M(STONE_D), x, 0.1, z, g)
    k.mk(k.cyl(0.08, 0.12, 0.9, 4), k.M(GOLD), x, 0.65, z, g).rotation.y = Math.PI / 4
    k.mk(k.cone(0.1, 0.2, 4), k.M(GOLD), x, 1.2, z, g).rotation.y = Math.PI / 4
  }
  if (level >= 7) {
    for (const x of [-1.25, 1.25]) k.flag(g, x, 1.3, 1.0, 0.6, 0.34, 0x3e8ef0, GOLD)
    const R = k.grp(g, 0, 3.4, 0)
    for (let i = 0; i < 6; i++) { const a = (i / 6) * Math.PI * 2; const c = k.mk(k.geo('oct', () => new THREE.OctahedronGeometry(0.36, 0)), k.M(0x63f0ff, { emissive: 0x2ab8e0, emissiveIntensity: 0.7 }), Math.cos(a) * 0.55, 0, Math.sin(a) * 0.55, R); c.scale.setScalar(0.35) }
    const gl = k.glow(0x6af2ff, 2.6, 0.6)
    gl.position.set(0, 3.4, 0)
    g.add(gl)
    live.glows ??= []
    live.glows.push(gl)
  }
}

/** Team banner: a tall pole with the team's flag over a stone base and shields; a crown plaque and a second pennant by level. */
function banner(k: Kit, g: THREE.Group, level: number) {
  k.mk(k.cyl(0.55, 0.65, 0.22, 8), k.M(STONE_D), 0, 0.11, 0, g)
  k.mk(k.cyl(0.42, 0.5, 0.2, 8), k.M(level >= 2 ? STONE_L : STONE), 0, 0.32, 0, g)
  k.mk(k.cyl(0.05, 0.06, 2.6, 8), k.M(WOOD_D), 0, 1.6, 0, g)
  k.mk(k.sph(0.1, 10, 8), k.M(GOLD), 0, 2.95, 0, g)
  // The big team flag: blue with a gold stripe.
  const F = k.grp(g, 0.05, 2.45, 0)
  k.mk(k.rb(0.95, 0.62, 0.03, 0.01), k.M(0x3e8ef0), 0.5, 0, 0, F)
  k.mk(k.rb(0.95, 0.1, 0.035, 0.01), k.M(GOLD), 0.5, -0.12, 0, F)
  k.mk(k.cone(0.12, 0.18, 4), k.M(GOLD), 0.5, 0.08, 0.02, F).rotation.x = Math.PI / 2
  // Shields around the base.
  for (let i = 0; i < 3; i++) { const a = (i / 3) * Math.PI * 2 + 0.5; const sh = k.mk(k.cyl(0.18, 0.18, 0.05, 12), k.M(i % 2 ? 0x3e8ef0 : 0xd0453a), Math.cos(a) * 0.5, 0.42, Math.sin(a) * 0.5, g); sh.rotation.set(Math.PI / 2, 0, -a + Math.PI / 2) }
  if (level >= 2) k.flag(g, 0, 1.9, 0, 0.5, 0.26, 0xffd34a, 0x3e8ef0)
  if (level >= 3) { k.mk(k.rb(0.5, 0.3, 0.06, 0.03), k.M(GOLD), 0, 0.9, 0.08, g); for (const x of [-0.14, 0, 0.14]) k.mk(k.cone(0.05, 0.1, 4), k.M(GOLD), x, 1.1, 0.08, g) }
}

/** Team flags: two poles with pennants in the team's colours. */
function flags(k: Kit, g: THREE.Group) {
  for (const [x, c] of [[-0.35, 0x3e8ef0], [0.35, 0xffd34a]] as const) {
    k.mk(k.cyl(0.1, 0.12, 0.1, 8), k.M(STONE), x, 0.05, 0, g)
    k.mk(k.cyl(0.03, 0.03, 1.5, 6), k.M(WOOD_D), x, 0.8, 0, g)
    k.flag(g, x, 1.45, 0, 0.45, 0.26, c, c === 0x3e8ef0 ? GOLD : 0x3e8ef0)
  }
}

/** Triumphal arch: two stone piers, a rounded vault, a gold crown on top. */
function arch(k: Kit, g: THREE.Group) {
  for (const x of [-0.55, 0.55]) {
    k.mk(k.rb(0.36, 1.3, 0.5, 0.05), k.M(STONE_L), x, 0.65, 0, g)
    k.mk(k.rb(0.44, 0.12, 0.58, 0.03), k.M(STONE_D), x, 0.06, 0, g)
  }
  k.mk(k.geo('vault', () => new THREE.TorusGeometry(0.55, 0.16, 8, 16, Math.PI)), k.M(STONE_L), 0, 1.3, 0, g)
  k.mk(k.rb(1.5, 0.22, 0.56, 0.05), k.M(STONE), 0, 1.92, 0, g)
  k.mk(k.rb(1.6, 0.06, 0.6, 0.02), k.M(GOLD), 0, 2.05, 0, g)
  for (const x of [-0.16, 0, 0.16]) k.mk(k.cone(0.07, 0.18, 4), k.M(GOLD), x, 2.17, 0, g)
}

/** Fountain: a round basin, a column and a small upper bowl with water. */
function fountain(k: Kit, g: THREE.Group, live: Live) {
  k.mk(k.cyl(0.78, 0.84, 0.3, 18), k.M(STONE_L), 0, 0.15, 0, g)
  k.mk(k.cyl(0.66, 0.66, 0.06, 18), k.M(0x4db4f0), 0, 0.28, 0, g)
  k.mk(k.cyl(0.1, 0.14, 0.8, 10), k.M(STONE), 0, 0.6, 0, g)
  k.mk(k.cyl(0.36, 0.2, 0.16, 14), k.M(STONE_L), 0, 1.02, 0, g)
  k.mk(k.cyl(0.3, 0.3, 0.04, 14), k.M(0x7fd0ff), 0, 1.1, 0, g)
  k.mk(k.sph(0.08, 10, 8), k.M(0x9fe0ff), 0, 1.22, 0, g)
  const gl = k.glow(0xbfe9ff, 1.2, 0.45)
  gl.position.set(0, 0.9, 0)
  g.add(gl)
  live.glows ??= []
  live.glows.push(gl)
}

/** Statue: a defender with shield and spear on a stepped pedestal, all in pale stone. */
function statue(k: Kit, g: THREE.Group) {
  k.mk(k.rb(0.9, 0.2, 0.9, 0.04), k.M(STONE_D), 0, 0.1, 0, g)
  k.mk(k.rb(0.66, 0.5, 0.66, 0.05), k.M(STONE), 0, 0.45, 0, g)
  k.mk(k.rb(0.5, 0.06, 0.04, 0.01), k.M(GOLD), 0, 0.5, 0.34, g)
  const S = k.grp(g, 0, 0.7, 0), P = 0xe6e0d2
  for (const x of [-0.1, 0.1]) k.mk(k.caps(0.08, 0.18), k.M(P), x, 0.15, 0, S)
  k.mk(k.rb(0.42, 0.42, 0.3, 0.1), k.M(P), 0, 0.55, 0, S)
  k.mk(k.sph(0.2, 12, 10), k.M(P), 0, 0.92, 0, S)
  k.mk(k.sph(0.22, 12, 8), k.M(P), 0, 0.98, 0, S).scale.set(1, 0.7, 1)
  k.mk(k.cyl(0.22, 0.22, 0.05, 14), k.M(P), -0.3, 0.55, 0.12, S).rotation.x = Math.PI / 2
  k.mk(k.cyl(0.025, 0.025, 1.2, 6), k.M(P), 0.28, 0.75, 0.05, S)
  k.mk(k.cone(0.05, 0.16, 6), k.M(P), 0.28, 1.4, 0.05, S)
}

/** Decor: road pavers that join their neighbours, trees, a flower bed, a lantern, a pond. Looks only. */
function road(k: Kit, g: THREE.Group, joins: { e: boolean; s: boolean }) {
  k.mk(k.rb(1.5, 0.06, 1.5, 0.04), k.M(0xc9b48c), 0, 0.04, 0, g)
  // Paving stones in a loose grid; a link half-way toward each east/south neighbour.
  for (let i = 0; i < 9; i++) k.mk(k.rb(0.42, 0.05, 0.42, 0.06), k.M(i % 3 ? 0xd8cdb5 : 0xbfb29a), -0.48 + (i % 3) * 0.48, 0.08, -0.48 + Math.floor(i / 3) * 0.48, g)
  if (joins.e) k.mk(k.rb(0.4, 0.06, 1.3, 0.04), k.M(0xc9b48c), 0.9, 0.04, 0, g)
  if (joins.s) k.mk(k.rb(1.3, 0.06, 0.4, 0.04), k.M(0xc9b48c), 0, 0.04, 0.9, g)
}

function tree(k: Kit, g: THREE.Group, pine: boolean) {
  k.mk(k.cyl(0.13, 0.2, 0.9, 7), k.M(WOOD), 0, 0.45, 0, g)
  if (pine) for (const [r, h, y] of [[0.8, 0.9, 1.1], [0.62, 0.8, 1.6], [0.42, 0.65, 2.05]]) k.mk(k.cone(r, h, 8), k.M(0x3f8f2e), 0, y, 0, g)
  else {
    k.mk(k.sph(0.6, 12, 10), k.M(0x5da530), 0, 1.3, 0, g)
    k.mk(k.sph(0.45, 12, 10), k.M(0x6db83a), 0.34, 1.08, 0.18, g)
    k.mk(k.sph(0.4, 12, 10), k.M(0x4e9a2e), -0.32, 1.12, -0.1, g)
    for (const [x, y, z] of [[0.3, 1.5, 0.35], [-0.2, 1.2, 0.45], [0.45, 1.05, -0.2]]) k.mk(k.sph(0.06, 6, 4), k.M(0xf04848), x, y, z, g)
  }
}

function flowers(k: Kit, g: THREE.Group) {
  k.mk(k.rb(1.3, 0.16, 1.0, 0.08), k.M(0x8f5e33), 0, 0.08, 0, g)
  k.mk(k.rb(1.16, 0.06, 0.86, 0.04), k.M(0x6a4a2a), 0, 0.17, 0, g)
  const colours = [0xff6fa8, 0xffe15a, 0xffffff, 0xb98cff, 0xff8a4a]
  for (let i = 0; i < 14; i++) {
    const x = -0.48 + (i % 5) * 0.24 + ((i * 7) % 3) * 0.02, z = -0.3 + Math.floor(i / 5) * 0.3
    k.mk(k.cyl(0.015, 0.015, 0.2, 4), k.M(0x3f8f2e), x, 0.28, z, g)
    k.mk(k.sph(0.07, 8, 6), k.M(colours[i % colours.length]), x, 0.4, z, g)
  }
}

function lantern(k: Kit, g: THREE.Group, live: Live) {
  k.mk(k.cyl(0.16, 0.2, 0.12, 8), k.M(0x4a4a58), 0, 0.06, 0, g)
  k.mk(k.cyl(0.05, 0.06, 1.5, 8), k.M(0x4a4a58), 0, 0.8, 0, g)
  k.mk(k.rb(0.36, 0.06, 0.06, 0.02), k.M(0x4a4a58), 0.14, 1.5, 0, g)
  const lamp = k.mk(k.rb(0.2, 0.26, 0.2, 0.05), k.M(0xffe08a, { emissive: 0xffb84a, emissiveIntensity: 0.8 }), 0.28, 1.36, 0, g)
  k.mk(k.cone(0.16, 0.12, 4), k.M(0x4a4a58), 0, 0.18, 0, lamp).rotation.y = Math.PI / 4
  const glow = k.glow(0xffc46a, 1.4, 0.75)
  glow.position.set(0.28, 1.36, 0)
  g.add(glow)
  live.glows ??= []
  live.glows.push(glow)
}

function pond(k: Kit, g: THREE.Group) {
  k.mk(k.cyl(0.78, 0.82, 0.14, 18), k.M(0xa79e8a), 0, 0.07, 0, g)
  k.mk(k.cyl(0.66, 0.66, 0.06, 18), k.M(0x4db4f0), 0, 0.15, 0, g)
  for (let i = 0; i < 10; i++) { const a = (i / 10) * Math.PI * 2; k.mk(k.geo('pebble', () => new THREE.DodecahedronGeometry(0.1, 0)), k.M(i % 2 ? 0xbdb6a6 : 0x9e978a), Math.cos(a) * 0.78, 0.16, Math.sin(a) * 0.78, g) }
  for (const [x, z] of [[0.2, -0.15], [-0.25, 0.2]]) { k.mk(k.cyl(0.13, 0.13, 0.02, 10), k.M(0x5da530), x, 0.19, z, g); k.mk(k.sph(0.05, 6, 4), k.M(0xff8ac0), x, 0.22, z, g) }
  k.mk(k.cyl(0.03, 0.03, 0.5, 5), k.M(0x6a8a3a), -0.55, 0.35, -0.4, g)
  k.mk(k.caps(0.045, 0.14), k.M(0x7a5530), -0.55, 0.62, -0.4, g)
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
    case 'townhall': if (L >= 6) grandHall(k, inner, Math.min(7, L), live); else townhall(k, inner, L, live); inner.scale.setScalar(0.78); break
    case 'barracks': barracks(k, inner, Math.min(4, L)); inner.scale.setScalar(0.8); break
    case 'storage': storage(k, inner, Math.min(4, L)); inner.scale.setScalar(0.92); break
    case 'wall': wall(k, inner, Math.min(5, L), joins); break
    case 'tower': tower(k, inner, Math.min(5, L)); break
    case 'tavern': tavern(k, inner, Math.min(3, L)); inner.scale.setScalar(0.85); break
    case 'hut': hut(k, inner); inner.scale.setScalar(0.9); break
    case 'market': market(k, inner, Math.min(3, L), live); inner.scale.setScalar(1.05); break
    case 'well': well(k, inner, Math.min(3, L), live); break
    case 'archery': archery(k, inner, Math.min(3, L)); inner.scale.setScalar(0.95); break
    case 'workshop': workshop(k, inner, Math.min(3, L), live); inner.scale.setScalar(0.85); break
    case 'mortar': mortar(k, inner, Math.min(3, L)); break
    case 'trap': trap(k, inner, Math.min(2, L)); break
    case 'stable': stable(k, inner, Math.min(2, L)); inner.scale.setScalar(0.85); break
    case 'crossbow': crossbow(k, inner, Math.min(3, L)); break
    case 'banner': banner(k, inner, Math.min(3, L)); break
    case 'flags': flags(k, inner); break
    case 'arch': arch(k, inner); inner.scale.setScalar(0.85); break
    case 'fountain': fountain(k, inner, live); break
    case 'statue': statue(k, inner); break
    case 'road': road(k, inner, joins); break
    case 'tree': tree(k, inner, joins.s); break
    case 'flowers': flowers(k, inner); break
    case 'lantern': lantern(k, inner, live); break
    case 'pond': pond(k, inner); break
  }
  const flat = type === 'road' || type === 'pond' || type === 'flowers' || type === 'trap' || type === 'fountain'
  if (type !== 'wall' && !flat) k.outline(inner, 0.03)
  if (!flat) k.blob(g, 0, 0, type === 'wall' ? 1.2 : type === 'tree' || type === 'lantern' ? 1.3 : 2.0, type === 'wall' ? 1.2 : type === 'tree' || type === 'lantern' ? 1.1 : 1.7, 0.4)
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
