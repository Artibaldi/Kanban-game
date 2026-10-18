// Team banner rules on the client, mirroring game.CheckDonate, to explain a disabled «Отправить».

export interface BannerMate { user_id?: string; display_name?: string; capacity: number; held: number }
export interface Reinforcement { donor_id: string; donor_name: string; type: string; count: number }
export interface BannerState { capacity: number; held: number; reinforcements: Reinforcement[]; team: BannerMate[]; crowns: number[]; trophies: Record<string, boolean>; troops: Record<string, number> }

/** donateBlock: why `count` troops taking `housing` places each cannot go to the mate's banner; '' if they can. */
export function donateBlock(mate: BannerMate, housing: number, count: number, ready: number): string {
  if (!Number.isInteger(count) || count < 1) return 'Укажите количество — целое число.'
  if (mate.capacity === 0) return 'У коллеги ещё нет знамени команды.'
  if (count > ready) return `Готово только ${ready}.`
  if (mate.held + housing * count > mate.capacity) return `На знамени свободно места: ${mate.capacity - mate.held}.`
  return ''
}
