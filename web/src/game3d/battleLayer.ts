// A duel replay drawn on top of the defender's 3D city. Each server round becomes a beat on screen:
// the squad advances on the part of the city that round hit, towers shoot back, fallen figures vanish,
// and parts brought to zero lie in ruins. Visual only — the server's rounds are the outcome.
import * as THREE from 'three'
import type { BuildingType } from '../types'
import type { Part } from '../lib/battleView'
import { eIn, eOut, rand } from './kit'
import type { CityStage } from './cityStage'
import { animateTroop, troopModel } from './troops'

const partOf: Record<BuildingType, Part> = { wall: 'wall', tower: 'tower', townhall: 'townhall', barracks: 'other', storage: 'other' }

interface Fig { type: string; g: THREE.Group; ph: number; alive: boolean; target: string | null; from: THREE.Vector3; to: THREE.Vector3; t: number; dur: number }
interface Ruin { id: string; mats: [THREE.Mesh, THREE.Material][]; rot: number; sy: number }

export class BattleLayer {
  private figs: Fig[] = []
  private ruins = new Map<string, Ruin>()
  private shots: { m: THREE.Mesh; from: THREE.Vector3; to: THREE.Vector3; t: number }[] = []
  private smokeT = 0
  private puffs: { m: THREE.Mesh; life: number }[] = []
  private target: Part | null = null

  constructor(private city: CityStage) {
    city.onFrame = (dt, t) => this.frame(dt, t)
  }

  /** start: the squad lines up at the front edge of the island. */
  start(squad: Record<string, number>, shown: Record<string, number>) {
    this.clear()
    const k = this.city.toolkit, edge = this.city.size / 2 + 1.4
    let i = 0
    const total = Object.values(shown).reduce((a, b) => a + b, 0)
    for (const [type, n] of Object.entries(shown)) {
      if (!squad[type]) continue
      for (let j = 0; j < n; j++, i++) {
        const g = troopModel(k, type)
        g.scale.setScalar(type === 'ram' ? 0.8 : 0.95)
        k.blob(g, 0, 0, type === 'ram' ? 1.4 : 0.8, type === 'ram' ? 1.6 : 0.8, 0.45)
        const x = (i - (total - 1) / 2) * 1.0
        g.position.set(x, 0.21, edge + (j % 2) * 0.7)
        g.rotation.y = Math.PI
        this.city.group.add(g)
        this.figs.push({ type, g, ph: rand() * 6, alive: true, target: null, from: g.position.clone(), to: g.position.clone(), t: 1, dur: 1 })
      }
    }
  }

  /**
   * round: one beat. `target` is the part this round hit, `ruined` what lies in ruins after it,
   * `alive` how many figures of each type still stand.
   */
  round(target: Part | null, ruined: Set<Part>, alive: Record<string, number>, dur: number) {
    this.target = target
    const k = this.city.toolkit
    // Fallen figures: the ones beyond the alive count, front of the squad first.
    for (const type of new Set(this.figs.map(f => f.type))) {
      const standing = this.figs.filter(f => f.type === type && f.alive)
      for (const f of standing.slice(alive[type] ?? 0)) {
        f.alive = false
        const p = f.g.position.clone(), g = f.g
        k.tween(0.2, q => { g.rotation.x = (-Math.PI / 2) * eIn(q) })
        k.wait(0.25, () => { k.poof(this.city.group, p.setY(0.6), 0.7, 3); this.city.group.remove(g) })
      }
    }
    // Ruins.
    for (const e of this.city.entries()) if (ruined.has(partOf[e.type]) && !this.ruins.has(e.id)) this.ruin(e)
    // Advance: each standing figure heads for the nearest standing building of the target part.
    const targets = this.city.entries().filter(e => target && partOf[e.type] === target && !this.ruins.has(e.id))
    for (const f of this.figs) {
      if (!f.alive) continue
      let best: (typeof targets)[number] | null = null, bd = Infinity
      for (const e of targets) { const d = e.holder.position.distanceTo(f.g.position); if (d < bd) { bd = d; best = e } }
      f.target = best?.id ?? null
      f.from = f.g.position.clone()
      if (best) {
        const dir = f.g.position.clone().sub(best.holder.position).setY(0)
        const stand = best.holder.position.clone().add(dir.setLength(best.type === 'wall' ? 0.95 : 1.35)).setY(0.21)
        stand.x += (rand() - 0.5) * 0.5; stand.z += (rand() - 0.5) * 0.5
        f.to = stand
      } else f.to = f.g.position.clone()
      f.t = 0
      f.dur = dur * 0.6
    }
    // Standing towers shoot back at the squad.
    const towers = this.city.entries().filter(e => e.type === 'tower' && !this.ruins.has(e.id))
    const living = this.figs.filter(f => f.alive)
    for (const t of towers) for (let n = 0; n < 2 && living.length; n++) {
      const f = living[Math.floor(rand() * living.length)]
      const from = t.holder.position.clone().setY(t.height + 0.3)
      const m = k.mk(k.sph(0.13, 8, 6), k.MB(0xffe14a), from.x, from.y, from.z, this.city.group)
      this.shots.push({ m, from, to: f.g.position.clone().setY(0.6), t: -n * 0.35 - rand() * 0.2 })
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
    k.tween(0.35, q => { model.scale.y = r.sy * (1 - 0.4 * eOut(q)); model.rotation.z = r.rot + tilt * q })
    k.poof(this.city.group, e.holder.position.clone().setY(0.8), 1.1, 6)
    k.debris(this.city.group, e.holder.position.clone().setY(1.0), 6)
  }

  private frame(dt: number, t: number) {
    const k = this.city.toolkit
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
        const e = this.city.entries().find(x => x.id === f.target)
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
      s.m.position.lerpVectors(s.from, s.to, q)
      s.m.position.y += Math.sin(q * Math.PI) * 1.6
      if (q >= 1) { k.sparks(this.city.group, s.m.position, 4, 0xffe27a, 2); this.city.group.remove(s.m); this.shots.splice(i, 1) }
    }
    // Smoke rises from the ruins.
    this.smokeT += dt
    if (this.smokeT > 0.6 && this.ruins.size) {
      this.smokeT = 0
      for (const e of this.city.entries()) {
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
    if (this.target) for (const e of this.city.entries()) {
      if (partOf[e.type] !== this.target || this.ruins.has(e.id)) continue
      const hit = this.figs.some(f => f.alive && f.target === e.id && f.t >= f.dur)
      e.model.position.x = hit ? Math.sin(t * 60) * 0.03 : 0
    }
  }

  /** cheer: survivors hop when the battle is over. */
  cheer() {
    const k = this.city.toolkit
    for (const f of this.figs) if (f.alive) { const g = f.g, y = g.position.y; k.tween(0.5, q => { g.position.y = y + Math.sin(q * Math.PI * 2) * 0.35 * (1 - q) }) }
    this.target = null
  }

  /** clear: removes the squad and restores every ruined building (for a replay). */
  clear() {
    for (const f of this.figs) this.city.group.remove(f.g)
    for (const s of this.shots) this.city.group.remove(s.m)
    for (const p of this.puffs) { this.city.group.remove(p.m); this.city.toolkit.release(p.m.material as THREE.Material) }
    for (const e of this.city.entries()) {
      const r = this.ruins.get(e.id)
      if (!r) continue
      for (const [m, mat] of r.mats) m.material = mat
      e.model.scale.y = r.sy
      e.model.rotation.z = r.rot
      e.model.position.x = 0
    }
    this.figs = []; this.shots = []; this.puffs = []; this.ruins.clear(); this.target = null
  }
}
