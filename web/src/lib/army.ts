// Workshop rules on the client, mirroring the server (game.TroopMult, game.CheckResearch) for the cards.

export interface Research { type: string; target_level: number; started_at: string; finish_at: string }
export interface ResearchCost { level: number; cost: number; seconds: number }
export interface Workshop { levels: Record<string, number>; research: Research | null; max_troop_level: number; research_costs: ResearchCost[]; locked?: Record<string, string> }

/** troopMult: each level above 1 adds 10% hit points and attack. */
export const troopMult = (level: number) => (level <= 1 ? 1 : 1 + 0.1 * (level - 1))

/** nextResearch: the price of the next level, or null at the top. */
export function nextResearch(type: string, w: Workshop): ResearchCost | null {
  const cur = Math.max(1, w.levels[type] || 1)
  return w.research_costs[cur - 1] ?? null
}

/** researchBlock: why the troop cannot be raised right now; '' if it can. */
export function researchBlock(type: string, w: Workshop, gold: number, debt: boolean): string {
  const cur = Math.max(1, w.levels[type] || 1)
  const next = nextResearch(type, w)
  if (!next) return 'Отряд уже максимального уровня.'
  if (w.max_troop_level <= 1) return 'Постройте мастерскую в городе.'
  if (w.locked?.[type]) return w.locked[type]
  if (cur + 1 > w.max_troop_level) return 'Улучшите мастерскую, чтобы открыть следующий уровень.'
  if (w.research) return 'Мастерская уже работает.'
  if (debt) return 'Сначала погасите долг.'
  if (gold < next.cost) return `Нужно ${next.cost} золота.`
  return ''
}
