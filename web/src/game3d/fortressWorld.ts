// The fortress map: meadow, path, river with a bridge, the walls, gate, towers and keep, trees, and the backlog fog.
import * as THREE from 'three'
import { LANES } from '../lib/horde'
import { Kit, rand } from './kit'

export const STONE = 0xc9bfa8, STONE_D = 0xa99d85, STONE_L = 0xddd3be, WOOD = 0x9a6232, WOOD_D = 0x6e4422, ROOF = 0x3f6fb0, GOLD = 0xffd323
export const RIVER = -7.4
export const pathX = (z: number) => Math.sin(z * 0.32) * 0.7

export interface Tower { g: THREE.Group; top: THREE.Vector3; roof: THREE.Mesh; glow: THREE.Sprite; idx: number; cd: number }
export interface FortressWorld {
  world: THREE.Group
  /** The wall section the overdue enemies hit; it shakes and cracks. */
  hitSeg: THREE.Group
  hitSegX: number
  towers: Tower[]
  water: THREE.Texture
  clouds: { g: THREE.Group; x: number; ph: number }[]
  cshadows: THREE.Mesh[]
  torches: THREE.Object3D[]
  /** damage: 0 (whole) … 1 (badly broken): cracks, a fallen merlon and rubble. */
  setDamage(d: number): void
}

function wallSeg(k: Kit, p: THREE.Object3D, cx: number, cz: number, L: number, rot: number, broken?: THREE.Object3D[]) {
  const g = k.grp(p, cx, 0, cz)
  g.rotation.y = rot
  const WH = 1.25, WT = 1.1
  k.mk(k.rb(L + 0.2, 0.32, WT + 0.28, 0.1), k.M(STONE_D), 0, 0.16, 0, g)
  k.mk(k.rb(L, WH, WT, 0.12), k.M(STONE), 0, WH / 2, 0, g)
  k.mk(k.rb(L - 0.1, 0.06, WT - 0.42, 0.02), k.M(0xb5aa92), 0, WH + 0.01, 0.14, g)
  const n = Math.max(2, Math.round(L / 0.78))
  for (let i = 0; i < n; i++) {
    const x = -L / 2 + (i + 0.5) * (L / n)
    const m = k.mk(k.rb(0.44, 0.36, 0.34, 0.08), k.M(STONE), x, WH + 0.18, -WT / 2 + 0.17, g)
    // On the section under attack two merlons can break: one falls off, one tilts.
    if (broken && i === 5) { m.userData.fall = 1; broken.push(m) }
    if (broken && i === 4) { m.userData.tilt = 1; broken.push(m) }
  }
  // Bricks on both faces.
  for (let i = 0; i < L * 1.6; i++) {
    const bw = 0.3 + Math.round(rand() * 4) * 0.05, y = 0.45 + Math.floor(rand() * 3) * 0.26
    k.mk(k.rb(bw, 0.2, 0.06, 0.03), k.M(rand() > 0.5 ? STONE_L : STONE_D), -L / 2 + 0.3 + rand() * (L - 0.6), y, WT / 2 + 0.01, g)
    k.mk(k.rb(bw, 0.2, 0.06, 0.03), k.M(rand() > 0.5 ? STONE_L : STONE_D), -L / 2 + 0.3 + rand() * (L - 0.6), y, -WT / 2 - 0.01, g)
  }
  return g
}

function tree(k: Kit, p: THREE.Object3D, x: number, z: number, s = 1, pine = false) {
  const g = k.grp(p, x, 0, z)
  g.scale.setScalar(s)
  k.mk(k.cyl(0.13, 0.2, 0.9, 7), k.M(WOOD), 0, 0.45, 0, g)
  if (pine) for (const [r, h, y] of [[0.95, 1.0, 1.15], [0.75, 0.9, 1.7], [0.5, 0.75, 2.2]]) k.mk(k.cone(r, h, 8), k.M(0x3f8f2e), 0, y, 0, g)
  else {
    k.mk(k.sph(0.62, 12, 10), k.M(0x5da530), 0, 1.35, 0, g)
    k.mk(k.sph(0.48, 12, 10), k.M(0x6db83a), 0.38, 1.1, 0.2, g)
    k.mk(k.sph(0.44, 12, 10), k.M(0x4e9a2e), -0.36, 1.15, -0.1, g)
    k.mk(k.sph(0.36, 12, 10), k.M(0x7cc242), 0.05, 1.75, 0.15, g)
  }
  k.blob(p, x, z, 1.7 * s, 1.5 * s, 0.45)
}

export function buildFortress(k: Kit, scene: THREE.Scene): FortressWorld {
  const world = k.grp(scene)
  // Ground: a fine checker of two greens, like a playing field.
  const groundTex = k.canvasTex(2, 2, g => { g.fillStyle = '#7CC242'; g.fillRect(0, 0, 2, 2); g.fillStyle = '#74BA3C'; g.fillRect(0, 0, 1, 1); g.fillRect(1, 1, 1, 1) }, 90)
  const ground = k.mk(new THREE.PlaneGeometry(180, 180), k.M(0xffffff, { map: groundTex }), 0, 0, 0, world)
  ground.rotation.x = -Math.PI / 2
  const fogTex = k.canvasTex(4, 128, g => { const r = g.createLinearGradient(0, 0, 0, 128); r.addColorStop(0, 'rgba(226,222,246,1)'); r.addColorStop(0.55, 'rgba(226,222,246,.9)'); r.addColorStop(1, 'rgba(226,222,246,0)'); g.fillStyle = r; g.fillRect(0, 0, 4, 128) })
  const fogGround = k.mk(new THREE.PlaneGeometry(80, 24), k.MB(0xffffff, { map: fogTex, transparent: true, depthWrite: false }), 0, 0.04, -24.5, world)
  fogGround.rotation.x = -Math.PI / 2
  for (const z of LANES) k.mk(k.rb(26, 0.03, 1.7, 0.01), k.MB(0xa6e06a, { transparent: true, opacity: 0.45, depthWrite: false }), -1, 0.015, z, world)
  for (let z = -0.4; z > -15; z -= 0.45) { const d = k.mk(k.cyl(1.15, 1.15, 0.05, 9), k.M(0xb07a45), pathX(z), 0.035, z, world); d.rotation.y = rand() * 3; d.scale.set(1 + rand() * 0.15, 1, 0.9) }
  for (let z = -1; z > -14; z -= 1.3) for (const s of [-1, 1]) k.mk(k.geo('peb', () => new THREE.DodecahedronGeometry(0.14, 0)), k.M(0xb8b0a0), pathX(z) + s * (1.25 + rand() * 0.2), 0.06, z + rand() * 0.4, world)
  for (let z = 0.8; z < 4; z += 0.5) k.mk(k.cyl(0.9, 0.9, 0.05, 9), k.M(0xd9c79a), 0, 0.04, z, world)
  k.mk(k.rb(14.6, 0.06, 5.4, 0.03), k.M(0xd3c29a), 0, 0.03, 3.4, world)
  // River and bridge.
  const water = k.canvasTex(64, 64, g => {
    g.fillStyle = '#4DB4F0'; g.fillRect(0, 0, 64, 64); g.strokeStyle = 'rgba(255,255,255,.55)'; g.lineWidth = 3
    for (let i = 0; i < 4; i++) { g.beginPath(); for (let x = 0; x <= 64; x += 4) { const y = 8 + i * 16 + Math.sin((x / 64) * Math.PI * 2 + i) * 3; if (x) g.lineTo(x, y); else g.moveTo(x, y) } g.stroke() }
  })
  water.wrapS = water.wrapT = THREE.RepeatWrapping
  water.repeat.set(16, 1)
  k.mk(k.rb(46, 0.04, 2.1, 0.02), k.M(0x8c5e35), -1, 0.02, RIVER, world)
  k.mk(k.rb(46, 0.05, 1.6, 0.02), k.MB(0xffffff, { map: water }), -1, 0.045, RIVER, world)
  for (const s of [-1, 1]) k.mk(k.rb(46, 0.03, 0.14, 0.01), k.MB(0xe6f6ff), -1, 0.07, RIVER + s * 0.8, world)
  const br = k.grp(world, pathX(RIVER), 0, RIVER)
  k.mk(k.rb(2.3, 0.2, 2.6, 0.06), k.M(WOOD), 0, 0.17, 0, br)
  for (let i = -1; i <= 1; i++) k.mk(k.rb(2.32, 0.04, 0.06, 0.01), k.M(WOOD_D), 0, 0.27, i * 0.75, br)
  for (const s of [-1, 1]) { k.mk(k.rb(0.1, 0.12, 2.6, 0.04), k.M(WOOD_D), s * 1.1, 0.58, 0, br); for (const zz of [-1.2, 0, 1.2]) k.mk(k.rb(0.14, 0.46, 0.14, 0.04), k.M(WOOD), s * 1.1, 0.42, zz, br) }

  // Walls. The left front section takes the hits.
  const broken: THREE.Object3D[] = []
  const hitSegX = -4.2
  const hitSeg = wallSeg(k, world, hitSegX, 0, 5.3, 0, broken)
  wallSeg(k, world, 4.2, 0, 5.3, 0)
  wallSeg(k, world, -7.4, 2.8, 4.2, Math.PI / 2)
  wallSeg(k, world, 7.4, 2.8, 4.2, -Math.PI / 2)
  const crackTex = k.canvasTex(256, 128, g => {
    g.lineCap = 'round'; g.lineJoin = 'round'
    const branch = (x: number, y: number, a: number, len: number, wd: number) => {
      const pts: [number, number][] = [[x, y]]
      for (let i = 0; i < len; i++) { a += Math.sin(i * 7.3 + x) * 0.6; x += Math.cos(a) * 12; y += Math.sin(a) * 12; pts.push([x, y]) }
      for (const [c, lw, o] of [['rgba(255,250,235,.7)', wd + 2, 2], ['#3E3428', wd, 0]] as const) {
        g.strokeStyle = c; g.lineWidth = lw; g.beginPath()
        pts.forEach(([px, py], i) => (i ? g.lineTo(px + o, py + o) : g.moveTo(px + o, py + o)))
        g.stroke()
      }
      return pts
    }
    const m = branch(128, 8, 1.4, 9, 6); branch(m[3][0], m[3][1], 0.2, 6, 4); branch(m[5][0], m[5][1], 2.6, 6, 4); branch(m[2][0], m[2][1], 2.9, 4, 3)
  })
  const crackMat = k.MB(0xffffff, { map: crackTex, transparent: true, depthWrite: false, polygonOffset: true, polygonOffsetFactor: -2 }, true)
  // Cracks on the outer face (toward the horde), the top and the inner face.
  k.mk(new THREE.PlaneGeometry(2.4, 1.2), crackMat, 1.2, 0.7, -0.57, hitSeg).rotation.y = Math.PI
  k.mk(new THREE.PlaneGeometry(2.0, 1.0), crackMat, 1.2, 1.27, 0, hitSeg).rotation.x = -Math.PI / 2
  k.mk(new THREE.PlaneGeometry(2.4, 1.2), crackMat, 1.2, 0.7, 0.57, hitSeg)
  const rubble = k.grp(world)
  for (let i = 0; i < 6; i++) k.mk(k.geo('rub' + (i % 3), () => new THREE.DodecahedronGeometry(0.1 + (i % 3) * 0.06, 0)), k.M(STONE_D), -3.6 + rand() * 1.6, 0.1, -0.85 - rand() * 0.5, rubble)
  k.mk(k.rb(0.44, 0.36, 0.34, 0.08), k.M(STONE), -2.4, 0.2, -1.05, rubble).rotation.set(0.4, 0.6, 0.9)

  // Gate with banners and torches.
  const G = k.grp(world)
  const torches: THREE.Object3D[] = []
  for (const s of [-1, 1]) {
    k.mk(k.rb(1.1, 0.3, 1.6, 0.1), k.M(STONE_D), s * 1.55, 0.15, 0, G)
    k.mk(k.rb(0.95, 2.0, 1.45, 0.12), k.M(STONE), s * 1.55, 1.0, 0, G)
    for (const zz of [-0.45, 0.45]) k.mk(k.rb(0.36, 0.34, 0.36, 0.07), k.M(STONE), s * 1.55 + (zz > 0 ? 0.22 : -0.22) * s, 2.66, zz * 0.9, G)
    for (const face of [-1, 1]) {
      const bn = k.mk(k.rb(0.62, 1.0, 0.05, 0.03), k.M(0x3e8ef0), s * 1.55, 1.15, face * 0.76, G)
      k.mk(k.rb(0.62, 0.1, 0.06, 0.02), k.M(GOLD), 0, -0.5, 0, bn)
      k.mk(k.rb(0.24, 0.24, 0.06, 0.03), k.M(GOLD), 0, 0.1, face * 0.01, bn).rotation.z = Math.PI / 4
    }
    for (const face of [-1, 1]) {
      k.mk(k.cyl(0.05, 0.04, 0.32, 6), k.M(WOOD_D), s * 1.04, 1.45, face * 0.55, G)
      const fl = k.glow(0xffa43a, 0.9, 0.9)
      fl.position.set(s * 1.04, 1.7, face * 0.58)
      G.add(fl)
      torches.push(fl, k.mk(k.sph(0.08, 8, 6), k.MB(0xffd34a), s * 1.04, 1.66, face * 0.56, G))
    }
  }
  k.mk(k.rb(4.2, 0.55, 1.5, 0.12), k.M(STONE_D), 0, 2.22, 0, G)
  k.mk(k.rb(4.0, 0.14, 1.4, 0.05), k.M(STONE), 0, 2.55, 0, G)
  k.mk(k.rb(2.18, 1.9, 0.32, 0.08), k.M(WOOD), 0, 0.95, -0.05, G)
  for (const x of [-0.55, 0, 0.55]) k.mk(k.rb(0.05, 1.8, 0.05, 0.02), k.M(WOOD_D), x, 0.95, 0.12, G)
  for (const y of [0.45, 1.35]) { k.mk(k.rb(2.22, 0.14, 0.38, 0.04), k.M(0x4a4a58), 0, y, -0.05, G); for (const x of [-0.8, -0.27, 0.27, 0.8]) k.mk(k.sph(0.05, 6, 4), k.M(GOLD), x, y, 0.15, G) }

  // Towers and keep: they shoot during the wave and glow when they win a crown.
  const towers: Tower[] = []
  const tower = (x: number, z: number, r: number, h: number, idx: number) => {
    const g = k.grp(world, x, 0, z)
    k.mk(k.cyl(r + 0.2, r + 0.32, 0.42, 14), k.M(STONE_D), 0, 0.21, 0, g)
    k.mk(k.cyl(r, r + 0.06, h, 14), k.M(STONE), 0, h / 2, 0, g)
    k.mk(k.cyl(r + 0.05, r + 0.05, 0.14, 14), k.M(STONE_D), 0, h * 0.5, 0, g)
    k.mk(k.cyl(r + 0.28, r + 0.1, 0.42, 14), k.M(STONE), 0, h + 0.14, 0, g)
    for (let i = 0; i < 9; i++) { const a = (i / 9) * Math.PI * 2; k.mk(k.rb(0.36, 0.34, 0.3, 0.07), k.M(STONE), Math.sin(a) * (r + 0.14), h + 0.5, Math.cos(a) * (r + 0.14), g).rotation.y = a }
    const roofH = h * 0.85
    k.mk(k.cyl(r + 0.62, r + 0.62, 0.12, 12), k.M(0x2c4f86), 0, h + 0.48, 0, g)
    const roof = k.mk(k.cone(r + 0.66, roofH, 12), k.M(ROOF), 0, h + 0.52 + roofH / 2, 0, g)
    k.mk(k.sph(0.14, 10, 8), k.M(GOLD), 0, h + 0.52 + roofH + 0.02, 0, g)
    k.flag(g, 0, h + 0.52 + roofH - 0.1, 0, 0.7, 0.38)
    for (const a of [0, -Math.PI / 2, -Math.PI / 4]) { const w = k.mk(k.rb(0.3, 0.48, 0.12, 0.06), k.M(0x2e2420), Math.sin(a) * r, h * 0.74, Math.cos(a) * r, g); w.rotation.y = a }
    const glow = k.glow(0xffe27a, 3.4, 0)
    glow.position.set(0, h + 1.2, 0)
    g.add(glow)
    towers.push({ g, top: new THREE.Vector3(x, h + 0.3, z), roof, glow, idx, cd: 0 })
  }
  tower(-7.4, 0.2, 1.2, 3.0, 0)
  tower(7.4, 0.2, 1.2, 3.0, 2)
  const K = k.grp(world, 0, 0, 4.4)
  k.mk(k.rb(3.6, 0.42, 3.6, 0.14), k.M(STONE_D), 0, 0.21, 0, K)
  k.mk(k.rb(3.1, 3.5, 3.1, 0.16), k.M(STONE), 0, 1.75, 0, K)
  k.mk(k.rb(3.16, 0.16, 3.16, 0.06), k.M(STONE_D), 0, 1.6, 0, K)
  k.mk(k.rb(3.5, 0.4, 3.5, 0.12), k.M(STONE), 0, 3.6, 0, K)
  for (let i = 0; i < 4; i++) for (const s of [-1, 0, 1]) { const a = (i * Math.PI) / 2, px = s * 1.05, pz = 1.6; k.mk(k.rb(0.42, 0.34, 0.3, 0.07), k.M(STONE), Math.cos(a) * px + Math.sin(a) * pz, 3.95, -Math.sin(a) * px + Math.cos(a) * pz, K).rotation.y = a }
  const keepRoof = k.mk(k.cone(2.75, 2.9, 4), k.M(ROOF), 0, 3.85 + 1.45, 0, K)
  keepRoof.rotation.y = Math.PI / 4
  k.mk(k.sph(0.18, 10, 8), k.M(GOLD), 0, 6.8, 0, K)
  k.flag(K, 0, 6.7, 0, 1.25, 0.66)
  for (const [x, z, r] of [[0, 1.56, 0], [-1.56, 0, -Math.PI / 2], [0.8, 1.56, 0], [-0.8, 1.56, 0], [-1.56, 0.8, -Math.PI / 2], [-1.56, -0.8, -Math.PI / 2]]) {
    k.mk(k.rb(0.36, 0.58, 0.1, 0.07), k.M(0x2e2420), x, 2.5, z, K).rotation.y = r
    k.mk(k.rb(0.46, 0.1, 0.14, 0.03), k.M(STONE_D), x, 2.17, z, K).rotation.y = r
  }
  k.mk(k.rb(0.9, 1.3, 0.14, 0.1), k.M(WOOD), 0, 0.85, 1.56, K)
  const keepGlow = k.glow(0xffe27a, 5, 0)
  keepGlow.position.set(0, 5.5, 0)
  K.add(keepGlow)
  towers.splice(1, 0, { g: K, top: new THREE.Vector3(0, 4.3, 4.4), roof: keepRoof, glow: keepGlow, idx: 1, cd: 0 })
  k.blob(world, 0, 4.4, 5.2, 5.2, 0.45); k.blob(world, -7.4, 0.2, 3.6, 3.6, 0.5); k.blob(world, 7.4, 0.2, 3.6, 3.6, 0.5)
  for (const [x, z] of [[-5.6, 2.6], [-5.1, 3.2], [5.4, 2.4]]) { k.mk(k.cyl(0.28, 0.28, 0.6, 10), k.M(WOOD), x, 0.3, z, world); k.mk(k.cyl(0.3, 0.3, 0.07, 10), k.M(0x4a4a58), x, 0.45, z, world); k.blob(world, x, z, 0.9) }
  for (const [x, z] of [[5.9, 3.4], [-3.4, 5.6]]) { k.mk(k.rb(0.6, 0.55, 0.6, 0.07), k.M(0xb7824a), x, 0.28, z, world); k.blob(world, x, z, 1.0) }

  // Decor.
  for (const [x, z, s, pn] of [[-10.2, -2, 1.1], [-11.4, -4.5, 0.95, 1], [-10.6, -10.2, 1.2], [-12, -7.4, 1, 1], [-11.5, 1.5, 1.1, 1], [-10.8, 5, 1.2], [10.6, -3.4, 1.15, 1], [11.6, -6.2, 1], [10.4, -10.8, 1.1, 1], [10.8, 4.6, 1.2], [12, 1.4, 1, 1], [-8.8, 7.6, 1.1, 1], [-3, 8.4, 1], [4.6, 8.2, 1.1, 1], [9, 7.6, 1], [-13.5, -12.5, 1.3, 1], [13.5, -12, 1.2, 1]] as number[][])
    tree(k, world, x, z, s, !!pn)
  for (const [x, z, s] of [[-8.6, -6.2, 0.8], [8.8, -8.6, 1], [-6.6, -12.6, 0.7], [9.6, -1.6, 0.6], [3.6, -12.8, 0.8], [-9.2, -13.2, 0.9]]) {
    const r = k.mk(k.geo('rock', () => new THREE.DodecahedronGeometry(0.5, 0)), k.M(0xa9a398), x, 0.22 * s, z, world)
    r.scale.set(s * 1.1, s * 0.75, s); r.rotation.y = rand() * 3
    k.blob(world, x, z, 1.3 * s, 1.1 * s, 0.4)
  }
  for (const [x, z, s = 1] of [[-9.4, -1.2], [9.2, -4.8], [-8.4, -9.8], [6.5, -10.4, 0.8], [-2.4, -13, 0.9], [8.4, 2.8], [-8.9, 3.6]]) {
    k.mk(k.sph(0.38 * s, 10, 8), k.M(0x5da530), x, 0.25 * s, z, world); k.mk(k.sph(0.3 * s, 10, 8), k.M(0x6db83a), x + 0.3 * s, 0.2 * s, z + 0.1, world); k.blob(world, x, z, 1.1 * s, 0.9 * s, 0.35)
  }
  const flowers = [0xff6fa8, 0xffe15a, 0xffffff, 0xb98cff]
  for (let i = 0; i < 40; i++) { const x = (rand() - 0.5) * 22, z = -1.5 - rand() * 11; if (Math.abs(x - pathX(z)) < 1.5 || Math.abs(z - RIVER) < 1.2) continue; k.mk(k.sph(0.06, 6, 4), k.M(flowers[i % 4]), x, 0.06, z, world) }

  // Backlog fog: clouds and the silhouettes of tasks not yet planned.
  const clouds: FortressWorld['clouds'] = []
  for (let i = 0; i < 16; i++) {
    const x = -16 + i * 2.2 + rand(), z = -15.2 - rand() * 3.5, g = k.grp(world, x, 0, z)
    const n = 3 + Math.floor(rand() * 3)
    for (let j = 0; j < n; j++) { const r = Math.round((0.8 + rand()) * 100) / 100; k.mk(k.sph(r, 14, 10), k.M(j % 2 ? 0xffffff : 0xede9fb), (j - n / 2) * 0.9, r * 0.55 + rand() * 0.3, rand() * 0.8, g) }
    clouds.push({ g, x, ph: rand() * 6 })
  }
  const sil = k.MB(0x5a4e8a, { transparent: true, opacity: 0.55 })
  for (const [x, z, s] of [[-6, -14.6, 1], [-1.5, -14.9, 1.4], [3.5, -14.5, 0.9], [7.5, -14.8, 1.2]]) {
    const g = k.grp(world, x, 0, z)
    g.scale.setScalar(s)
    k.mk(k.sph(0.42, 10, 8), sil, 0, 0.5, 0, g); k.mk(k.sph(0.3, 10, 8), sil, 0, 1.05, 0, g)
    k.mk(k.cone(0.1, 0.3, 5), sil, -0.25, 1.25, 0, g).rotation.z = 0.6; k.mk(k.cone(0.1, 0.3, 5), sil, 0.25, 1.25, 0, g).rotation.z = -0.6
  }
  const cshadows: THREE.Mesh[] = []
  for (let i = 0; i < 3; i++) { const m = k.blob(world, -20 + i * 14, -4 + i * 3, 7, 4.5, 0.16); m.position.y = 0.06; cshadows.push(m) }

  const setDamage = (d: number) => {
    crackMat.opacity = Math.max(0, Math.min(1, d * 1.6))
    const heavy = d >= 0.35
    rubble.visible = heavy
    for (const m of broken) {
      if (m.userData.fall) m.visible = !heavy
      if (m.userData.tilt) { m.rotation.z = heavy ? 0.25 : 0; m.position.y = heavy ? 1.38 : 1.43 }
    }
  }
  setDamage(0)
  return { world, hitSeg, hitSegX, towers, water, clouds, cshadows, torches, setDamage }
}
