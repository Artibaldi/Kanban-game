// A duel replay drawn on top of the defender's 3D city. Each server round becomes a beat on screen:
// the squad advances on the part of the city that round hit, towers shoot back, fallen figures vanish,
// and parts brought to zero lie in ruins. Visual only — the server's rounds are the outcome.
import * as THREE from 'three'
import type { BuildingType } from '../types'
import type { Part } from '../lib/battleView'
import { eIn, eOut, rand } from './kit'
import type { CityStage } from './cityStage'
import { animateTroop, troopModel } from './troops'

// Decor is not part of a defence: it is never a target and never in ruins.
const partOf: Partial<Record<BuildingType, Part>> = { wall: 'wall', tower: 'tower', townhall: 'townhall', barracks: 'other', storage: 'other', tavern: 'other', hut: 'other', market: 'other', well: 'other', archery: 'other', workshop: 'other', mortar: 'tower', stable: 'other', crossbow: 'tower', banner: 'other' }

interface Fig { type: string; g: THREE.Group; ph: number; alive: boolean; target: string | null; from: THREE.Vector3; to: THREE.Vector3; t: number; dur: number }
interface Ruin { id: string; mats: [THREE.Mesh, THREE.Material][]; rot: number; sy: number }

export class BattleLayer {
  private figs: Fig[] = []
  private ruins = new Map<string, Ruin>()
  /** Projectiles in flight: tower bolts, arrows and mortar shells; `arc` is the height of the curve. */
  private shots: { m: THREE.Mesh; from: THREE.Vector3; to: THREE.Vector3; t: number; arc: number; boom?: boolean }[] = []
  private smokeT = 0
  private puffs: { m: THREE.Mesh; life: number }[] = []
  /** The garrison: colleagues' troops from the defender's banner, standing guard around the town hall. */
  private defenders: { type: string; g: THREE.Group; ph: number; alive: boolean }[] = []
  /** Trap spikes still on the field; a replay or a skip must not leave them behind. */
  private spikes: THREE.Group[] = []
  private target: Part | null = null
  /** The city's buildings, read once per round instead of every frame. */
  private entries: ReturnType<CityStage['entries']> | null = null

  constructor(private city: CityStage) {
    city.onFrame = (dt, t) => this.frame(dt, t)
  }

  /** start: the squad lines up at the front edge of the island; the garrison (shown figures per type) at the town hall. */
  start(squad: Record<string, number>, shown: Record<string, number>, garrison: Record<string, number> = {}) {
    this.clear()
    const k = this.city.toolkit, edge = this.city.size / 2 + 1.4
    const hall = this.city.entries().find(e => e.type === 'townhall')
    const ring = Object.entries(garrison).flatMap(([type, n]) => Array.from({ length: n }, () => type))
    ring.forEach((type, j) => {
      const a = (j / Math.max(1, ring.length)) * Math.PI * 2, c = hall?.holder.position ?? new THREE.Vector3()
      const g = troopModel(k, type)
      g.scale.setScalar(type === 'ram' ? 0.7 : 0.85)
      k.blob(g, 0, 0, 0.7, 0.7, 0.4)
      g.position.set(c.x + Math.cos(a) * 1.7, 0.21, c.z + Math.sin(a) * 1.7)
      g.rotation.y = Math.atan2(Math.cos(a), Math.sin(a)) // facing outward
      this.city.group.add(g)
      this.defenders.push({ type, g, ph: rand() * 6, alive: true })
    })
    let i = 0
    const total = Object.values(shown).reduce((a, b) => a + b, 0)
    for (const [type, n] of Object.entries(shown)) {
      if (!squad[type]) continue
      for (let j = 0; j < n; j++, i++) {
        const g = troopModel(k, type)
        g.scale.setScalar(type === 'ram' ? 0.8 : 0.95)
        k.blob(g, 0, 0, type === 'ram' ? 1.4 : 0.8, type === 'ram' ? 1.6 : 0.8, 0.45)
        const gap = Math.min(1.0, (this.city.size + 2) / Math.max(1, total))
        const x = (i - (total - 1) / 2) * gap
        g.position.set(x, 0.21, edge + (j % 2) * 0.7)
        g.rotation.y = Math.PI
        this.city.group.add(g)
        this.figs.push({ type, g, ph: rand() * 6, alive: true, target: null, from: g.position.clone(), to: g.position.clone(), t: 1, dur: 1 })
      }
    }
  }

  /**
   * round: one beat. `target` is the part this round hit, `ruined` what lies in ruins after it,
   * `alive` how many figures of each type still stand; `fx` says whether mortars fired and traps sprang this round.
   */
  round(target: Part | null, ruined: Set<Part>, alive: Record<string, number>, dur: number, fx: { mortar: boolean; trap: boolean; crossbow?: boolean; garrison?: number } = { mortar: false, trap: false }) {
    this.target = target
    this.entries = null
    const k = this.city.toolkit
    // Fallen figures: the ones beyond the alive count, front of the squad first.
    for (const type of new Set(this.figs.map(f => f.type))) {
      const standing = this.figs.filter(f => f.type === type && f.alive)
      // The front of the squad falls first.
      for (const f of standing.slice(0, Math.max(0, standing.length - (alive[type] ?? 0)))) {
        f.alive = false
        const p = f.g.position.clone(), g = f.g
        k.tween(0.2, q => { g.rotation.x = (-Math.PI / 2) * eIn(q) }, undefined, 'duel')
        k.wait(0.25, () => { k.poof(this.city.group, p.setY(0.6), 0.7, 3); this.city.group.remove(g) }, 'duel')
      }
    }
    // The garrison wears down: as many defenders stand as its share of hit points left.
    if (fx.garrison !== undefined) {
      const up = this.defenders.filter(d => d.alive)
      const keep = fx.garrison > 0 ? Math.max(1, Math.round(fx.garrison * this.defenders.length)) : 0
      for (const d of up.slice(0, Math.max(0, up.length - keep))) {
        d.alive = false
        const g = d.g, p = g.position.clone()
        k.tween(0.2, q => { g.rotation.x = (-Math.PI / 2) * eIn(q) }, undefined, 'duel')
        k.wait(0.25, () => { k.poof(this.city.group, p.setY(0.6), 0.6, 3); this.city.group.remove(g) }, 'duel')
      }
    }
    // Ruins.
    for (const e of this.city.entries()) { const p = partOf[e.type]; if (p && ruined.has(p) && !this.ruins.has(e.id)) this.ruin(e) }
    // Advance: each standing figure heads for the nearest standing building of the target part.
    const targets = this.city.entries().filter(e => target && partOf[e.type] === target && !this.ruins.has(e.id))
    // Archers shoot over the walls: while the others break them, they aim at what stands behind.
    const behind = this.city.entries().filter(e => { const p = partOf[e.type]; return p && p !== 'wall' && !this.ruins.has(e.id) })
    for (const f of this.figs) {
      if (!f.alive) continue
      // Riders ride around the walls straight for the towers (or what is left behind the walls).
      const towersUp = behind.filter(e => partOf[e.type] === 'tower')
      const pool = f.type === 'rider' && target === 'wall' ? (towersUp.length ? towersUp : behind) : f.type === 'archer' && target === 'wall' ? behind : targets
      let best: (typeof targets)[number] | null = null, bd = Infinity
      for (const e of pool) { const d = e.holder.position.distanceTo(f.g.position); if (d < bd) { bd = d; best = e } }
      f.target = best?.id ?? null
      f.from = f.g.position.clone()
      if (best) {
        const dir = f.g.position.clone().sub(best.holder.position).setY(0)
        const stand = best.holder.position.clone().add(dir.setLength(f.type === 'archer' ? 3.2 : best.type === 'wall' ? 0.95 : 1.35)).setY(0.21)
        stand.x += (rand() - 0.5) * 0.5; stand.z += (rand() - 0.5) * 0.5
        // Archers keep their distance but stay on the island.
        const edge = this.city.size / 2 + 0.6
        stand.x = Math.max(-edge, Math.min(edge, stand.x)); stand.z = Math.max(-edge, Math.min(edge, stand.z))
        f.to = stand
      } else f.to = f.g.position.clone()
      f.t = 0
      f.dur = dur * 0.6
    }
    // Standing towers shoot back at the squad; crossbow towers aim at the heavy figures first.
    const heavy = this.figs.filter(f => f.alive && (f.type === 'ram' || f.type === 'rider'))
    if (fx.crossbow) for (const cb of this.city.entries().filter(e => e.type === 'crossbow' && !this.ruins.has(e.id))) for (let n = 0; n < 2; n++) {
      const pick = heavy.length ? heavy : this.figs.filter(f => f.alive)
      if (!pick.length) break
      const f = pick[Math.floor(rand() * pick.length)]
      const from = cb.holder.position.clone().setY(cb.height + 0.2)
      const m = k.mk(k.rb(0.06, 0.06, 0.6, 0.01), k.MB(0x5a3418), from.x, from.y, from.z, this.city.group)
      this.shots.push({ m, from, to: f.g.position.clone().setY(0.6), t: -n * 0.4 - rand() * 0.2, arc: 0.4 })
    }
    const towers = this.city.entries().filter(e => e.type === 'tower' && !this.ruins.has(e.id))
    const living = this.figs.filter(f => f.alive)
    for (const t of towers) for (let n = 0; n < 2 && living.length; n++) {
      const f = living[Math.floor(rand() * living.length)]
      const from = t.holder.position.clone().setY(t.height + 0.3)
      const m = k.mk(k.sph(0.13, 8, 6), k.MB(0xffe14a), from.x, from.y, from.z, this.city.group)
      this.shots.push({ m, from, to: f.g.position.clone().setY(0.6), t: -n * 0.35 - rand() * 0.2, arc: 1.6 })
    }
    // Arrows: each archer looses one at its target once it has taken its place.
    for (const f of living) {
      if (f.type !== 'archer' || !f.target) continue
      const e = this.city.entries().find(x => x.id === f.target)
      if (!e) continue
      const from = f.to.clone().setY(0.8), m = k.mk(k.rb(0.04, 0.04, 0.4, 0.01), k.MB(0x8a5a2b), from.x, from.y, from.z, this.city.group)
      this.shots.push({ m, from, to: e.holder.position.clone().setY(Math.min(1.4, e.height * 0.6)), t: -(dur * 0.6) / 0.55 - rand() * 0.3, arc: 0.7 })
    }
    // Mortar shells: high, slow and heavy; they burst among the squad.
    if (fx.mortar) for (const mo of this.city.entries().filter(e => e.type === 'mortar' && !this.ruins.has(e.id))) for (let n = 0; n < 2 && living.length; n++) {
      const f = living[Math.floor(rand() * living.length)]
      const from = mo.holder.position.clone().setY(1.0)
      const m = k.mk(k.sph(0.2, 10, 8), k.MB(0x2e2e36), from.x, from.y, from.z, this.city.group)
      this.shots.push({ m, from, to: f.g.position.clone().setY(0.3), t: -0.2 - n * 0.5, arc: 3.2, boom: true })
    }
    // Traps spring under the front of the squad: spikes shoot up and sink back.
    if (fx.trap) for (const f of living.slice(0, 3)) {
      const p = f.g.position.clone(), spikes = k.grp(this.city.group, p.x, 0.05, p.z)
      for (let i = 0; i < 5; i++) k.mk(k.cone(0.07, 0.45, 5), k.M(0xc9d2de), (rand() - 0.5) * 0.6, 0.2, (rand() - 0.5) * 0.6, spikes)
      spikes.scale.y = 0.01
      this.spikes.push(spikes)
      k.tween(0.18, q => { spikes.scale.y = 0.01 + eOut(q) }, () => {
        k.sparks(this.city.group, p.clone().setY(0.4), 6, 0xffffff, 2)
        k.wait(0.5, () => k.tween(0.3, q => { spikes.scale.y = 1 - q }, () => this.city.group.remove(spikes), 'duel'), 'duel')
      }, 'duel')
    }
  }

  private ruin(e: ReturnType<CityStage['entries']>[number]) {
    const k = this.city.toolkit, dark = k.M(0x6b625a), darker = k.M(0x4f4842)
    const mats: Ruin['mats'] = []
    e.model.traverse(o => {
      const m = o as THREE.Mesh
      if (!m.isMesh || o.userData.hull || !(m.material instanceof THREE.MeshToonMaterial)) return
      mats.push([m, m.material])
      m.material = o.position.y > 0.6 ? darker : dark
    })
    const r: Ruin = { id: e.id, mats, rot: e.model.rotation.z, sy: e.model.scale.y }
    this.ruins.set(e.id, r)
    const model = e.model, tilt = (rand() - 0.5) * 0.3
    k.tween(0.35, q => { model.scale.y = r.sy * (1 - 0.4 * eOut(q)); model.rotation.z = r.rot + tilt * q }, undefined, 'duel')
    k.poof(this.city.group, e.holder.position.clone().setY(0.8), 1.1, 6)
    k.debris(this.city.group, e.holder.position.clone().setY(1.0), 6)
  }

  private frame(dt: number, t: number) {
    const k = this.city.toolkit, entries = this.entries ??= this.city.entries()
    for (const f of this.figs) {
      if (!f.alive) continue
      f.t += dt
      const q = Math.min(1, f.t / f.dur)
      if (q < 1) {
        f.g.position.lerpVectors(f.from, f.to, eOut(q))
        const d = f.to.clone().sub(f.from)
        if (d.lengthSq() > 0.01) f.g.rotation.y = Math.atan2(d.x, d.z)
      }
      const attacking = q >= 1 && !!f.target
      if (attacking) {
        const e = entries.find(x => x.id === f.target)
        if (e) { const d = e.holder.position.clone().sub(f.g.position); f.g.rotation.y = Math.atan2(d.x, d.z) }
      }
      animateTroop(f.g, f.type, t, f.ph, q < 1 ? 'walk' : attacking ? 'attack' : 'idle')
    }
    for (let i = this.shots.length - 1; i >= 0; i--) {
      const s = this.shots[i]
      s.t += dt / 0.55
      if (s.t < 0) { s.m.visible = false; continue }
      s.m.visible = true
      const q = Math.min(1, s.t)
      const prev = s.m.position.clone()
      s.m.position.lerpVectors(s.from, s.to, q)
      s.m.position.y += Math.sin(q * Math.PI) * s.arc
      if (s.arc < 1) s.m.lookAt(s.m.position.clone().add(s.m.position.clone().sub(prev)))
      if (q >= 1) {
        if (s.boom) { k.poof(this.city.group, s.m.position.clone().setY(0.4), 1.0, 6); k.debris(this.city.group, s.m.position.clone().setY(0.4), 4) }
        else k.sparks(this.city.group, s.m.position, 4, 0xffe27a, 2)
        this.city.group.remove(s.m)
        this.shots.splice(i, 1)
      }
    }
    for (const d of this.defenders) if (d.alive) animateTroop(d.g, d.type, t, d.ph, this.target ? 'attack' : 'idle')
    // Smoke rises from the ruins.
    this.smokeT += dt
    if (this.smokeT > 0.6 && this.ruins.size) {
      this.smokeT = 0
      for (const e of entries) {
        if (!this.ruins.has(e.id) || rand() > 0.5) continue
        const p = e.holder.position
        const m = k.mk(k.sph(0.22, 8, 6), k.MB(0x5a5550, { transparent: true, opacity: 0.8 }, true), p.x + (rand() - 0.5) * 0.6, p.y + e.height * 0.6, p.z + (rand() - 0.5) * 0.6, this.city.group)
        this.puffs.push({ m, life: 0 })
      }
    }
    for (let i = this.puffs.length - 1; i >= 0; i--) {
      const q = this.puffs[i]
      q.life += dt
      q.m.position.y += dt * 0.7
      q.m.scale.setScalar(1 + q.life * 1.2)
      ;(q.m.material as THREE.MeshBasicMaterial).opacity = Math.max(0, 0.8 - q.life / 2)
      if (q.life > 2) { this.city.group.remove(q.m); k.release(q.m.material as THREE.Material); this.puffs.splice(i, 1) }
    }
    // The building under attack trembles.
    if (this.target) for (const e of entries) {
      if (partOf[e.type] !== this.target || this.ruins.has(e.id)) continue
      const hit = this.figs.some(f => f.alive && f.target === e.id && f.t >= f.dur)
      e.model.position.x = hit ? Math.sin(t * 60) * 0.03 : 0
    }
  }

  /** cheer: survivors hop when the battle is over. */
  cheer() {
    const k = this.city.toolkit
    for (const e of this.city.entries()) e.model.position.x = 0
    for (const f of this.figs) if (f.alive) { const g = f.g, y = g.position.y; k.tween(0.5, q => { g.position.y = y + Math.sin(q * Math.PI * 2) * 0.35 * (1 - q) }) }
    this.target = null
  }

  /** clear: removes the squad and restores every ruined building (for a replay). */
  clear() {
    // Unfinished falls and ruin tweens must not write into the restored city.
    this.city.toolkit.cancel('duel')
    for (const f of this.figs) this.city.group.remove(f.g)
    for (const s of this.shots) this.city.group.remove(s.m)
    for (const p of this.puffs) { this.city.group.remove(p.m); this.city.toolkit.release(p.m.material as THREE.Material) }
    for (const s of this.spikes) this.city.group.remove(s)
    for (const d of this.defenders) this.city.group.remove(d.g)
    for (const e of this.city.entries()) {
      const r = this.ruins.get(e.id)
      if (!r) continue
      for (const [m, mat] of r.mats) m.material = mat
      e.model.scale.y = r.sy
      e.model.rotation.z = r.rot
      e.model.position.x = 0
    }
    this.figs = []; this.shots = []; this.puffs = []; this.spikes = []; this.defenders = []; this.ruins.clear(); this.target = null
  }
}
