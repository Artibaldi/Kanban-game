// Client mirror of the server's building rules, used only to explain why a button is disabled.
import type { BuildingDef, CityBuilding, City, Player } from '../types'

export function placeBlock(def: BuildingDef, city: City, player: Player, builders: number): string {
  const spec = def.levels[0]
  if (player.debt) return 'Сначала погасите долг следующими наградами.'
  if (city.townhall < spec.requires_th) return `Нужна ратуша ${spec.requires_th} уровня.`
  const lim = city.limits[def.type]
  if (lim && lim.built >= lim.allowed) return `Лимит для ратуши ${city.townhall}: ${lim.allowed}. Улучшите ратушу.`
  // Decor is placed at once and needs no builder.
  if (!def.decor && player.builders_busy >= builders) return 'Все строители заняты.'
  if (player.gold < spec.cost) return `Не хватает ${spec.cost - player.gold} золота.`
  return ''
}

export function upgradeBlock(b: CityBuilding, city: City, player: Player, builders: number): string {
  if (b.busy_until) return 'Здание уже строится.'
  if (!b.next) return 'Максимальный уровень.'
  if (player.debt) return 'Сначала погасите долг следующими наградами.'
  if (b.type !== 'townhall' && b.next.requires_th > city.townhall) return `Нужна ратуша ${b.next.requires_th} уровня.`
  if (b.next.requires_level && player.level < b.next.requires_level) return `Нужен уровень игрока ${b.next.requires_level} — опыт дают подтверждённые задачи.`
  if (player.builders_busy >= builders) return 'Все строители заняты.'
  if (player.gold < b.next.cost) return `Не хватает ${b.next.cost - player.gold} золота.`
  return ''
}

export function specLines(type: string, s: { hp: number; dps?: number; capacity?: number; gold_cap?: number; energy_cap?: number }) {
  const out = [`Прочность ${s.hp}`]
  if (s.dps) out.push(`Урон ${s.dps} за залп`)
  if (s.capacity) out.push(`Мест в армии ${s.capacity}`)
  if (s.gold_cap) out.push(`+${s.gold_cap} к запасу золота`)
  if (s.energy_cap) out.push(`+${s.energy_cap} к запасу энергии`)
  if (type === 'townhall') out.push('Открывает здания и уровни')
  return out
}
