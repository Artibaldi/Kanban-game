// Where troops stand in the army camp: one block per type, front to back — infantry, then riders, then rams.

export const CAMP_ORDER = ['guard', 'raider', 'archer', 'rider', 'ram']

interface Block { per: number; gap: number; depth: number; z: number; dx: number; cap: number }
// The palisade narrows towards the gate, so the front blocks are centred; the rams keep clear of the campfire.
const blocks: Record<string, Block> = {
  guard: { per: 7, gap: 1.0, depth: 1.0, z: 4.7, dx: 0, cap: 12 },
  raider: { per: 7, gap: 1.0, depth: 1.0, z: 2.7, dx: 0, cap: 12 },
  archer: { per: 7, gap: 1.0, depth: 1.0, z: 0.7, dx: 0, cap: 12 },
  rider: { per: 6, gap: 1.35, depth: 1.4, z: -1.5, dx: -0.6, cap: 6 },
  ram: { per: 5, gap: 1.95, depth: 1.6, z: -3.2, dx: -0.6, cap: 5 },
}

/** campCap: how many figures of a type are drawn; the rest is the "×N" tag. */
export const campCap = (type: string) => (blocks[type] ?? blocks.guard).cap

/** campSpot: the i-th figure of a type, in ranks of `per`. */
export function campSpot(type: string, i: number) {
  const b = blocks[type] ?? blocks.guard
  const col = i % b.per, line = Math.floor(i / b.per)
  return { x: (col - (b.per - 1) / 2) * b.gap + b.dx, z: b.z - line * b.depth }
}
