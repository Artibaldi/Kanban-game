export type Role = 'user' | 'lead' | 'admin'
export type TaskState = 'backlog' | 'ready' | 'in_progress' | 'review' | 'done'
export type Priority = 'P1' | 'P2' | 'P3' | 'P4'

export interface Me {
  id: string; display_name: string; login: string; role: Role
  team_id: string | null; team_name: string; game_mode: boolean; plays: boolean; demo_auth: boolean
  accept_friendly: boolean; game_rules_ack: boolean
}

export interface Reward { gold: number; energy: number; xp: number }

export interface Task {
  id: string; number: string; project_id: string; project_key: string; project_name: string; team_id: string
  title: string; description: string; state: TaskState
  assignee_id: string | null; assignee_name: string | null
  priority: Priority; points: number; points_approved: boolean; points_approved_by: string | null
  due_date: string | null; rank: number; approved_by: string | null; approved_at: string | null
  version: number; created_by_id: string; created_by: string; created_at: string; updated_at: string
  reward: Reward | null
}

export interface HistoryItem { id: number; actor: string; kind: string; from: string | null; to: string | null; note: string; created_at: string }
export interface CommentItem { id: string; author: string; body: string; created_at: string }
export interface LedgerItem {
  id: number; kind: string; gold: number; energy: number; xp: number
  gold_formula: number | null; energy_formula: number | null; reversed: boolean; reverses_id: number | null
  user?: string; task_number?: string; note: string; created_at: string
}
export interface TaskDetail { task: Task; comments: CommentItem[]; history: HistoryItem[]; rewards: LedgerItem[] }

export interface RewardResult {
  gold: number; energy: number; xp: number; gold_formula: number; energy_formula: number
  on_time: boolean; points: number; assignee: string; formula: string
}
export interface ReversalResult { gold: number; energy: number; xp: number; points: number; debt: boolean }

export interface Project { id: string; key: string; name: string; team_id: string; team_name: string; open_points: number }
export interface TeamUser { id: string; display_name: string; role: Role; team_id: string; title: string }

export interface Player {
  gold: number; energy: number; xp: number; gold_cap: number; energy_cap: number
  level: number; level_xp: number; level_need: number; trophies: number
  debt: boolean; townhall: number; builders_busy: number; builders: number
  army_capacity: number; army_housing: number; army_queued: number
}

export type DecorType = 'road' | 'tree' | 'flowers' | 'lantern' | 'pond'
export type BuildingType = 'townhall' | 'barracks' | 'storage' | 'wall' | 'tower' | 'tavern' | DecorType

export interface LevelSpec {
  level: number; cost: number; seconds: number; requires_th: number; requires_level?: number
  hp: number; dps?: number; capacity?: number; gold_cap?: number; energy_cap?: number
}
export interface BuildingDef { type: BuildingType; name: string; description: string; max_count: number[]; levels: LevelSpec[]; defense: boolean; decor?: boolean }
export interface TroopDef { type: string; name: string; role: string; hp: number; attack: number; siege_mult: number; housing: number; cost: number; seconds: number }

export interface Settings {
  gold_per_point: number; energy_per_point: number; xp_per_point: number
  priority_mult: Record<string, number>; on_time_bonus: number
  level_xp_base: number; level_xp_exp: number; random_spread: number
  time_scale: number; start_gold: number; start_energy: number
  overdue_damage_per_point: number; fortress_floor: number; repair_by_crowns: number[]; crown_thresholds: number[]
  wave_chest_gold: number[]; wave_chest_energy: number[]; max_defense_bonus: number; defense_score_full: number
  friendly_xp: number; friendly_per_day: number
}

export interface GameConfig {
  grid: number; builders: number; base_gold_cap: number; base_energy_cap: number
  buildings: BuildingDef[]; troops: TroopDef[]; settings: Settings; points_scale: number[]
}

export interface CityBuilding {
  id: string; type: BuildingType; level: number; target_level: number | null; busy_until: string | null
  x: number; y: number; name: string; spec: LevelSpec; next: LevelSpec | null; started_at: string | null; max_level: number
}
export interface City {
  buildings: CityBuilding[]; limits: Record<string, { built: number; allowed: number }>; townhall: number
  player?: Player; server_time?: string
  /** The island is grid×grid; the player's land is the centred square start…start+size. */
  grid?: number; land?: { start: number; size: number }
}

export interface Defense { wall_hp: number; tower_hp: number; tower_dps: number; towers: number; walls: number; townhall_hp: number; other_hp: number; townhall_level: number }
export interface Round {
  n: number; tower_damage: number; killed: Record<string, number>; alive: Record<string, number>
  wall_damage: number; tower_hit: number; base_damage: number
  wall_hp: number; tower_hp: number; townhall_hp: number; other_hp: number; events: string[]
}
export interface Battle {
  id: string; kind: 'raid' | 'friendly'; attacker_id: string; defender_id: string; attacker: string; defender: string; seed: number
  army: Record<string, number>; losses: Record<string, number>; defense: Defense; rounds: Round[]
  destruction: number; stars: number; gold: number; xp: number; attacker_trophies: number; defender_trophies: number; created_at: string
}
export interface FeedItem { id: number; actor: string; action: string; target_type: string; target_id: string | null; summary: string; created_at: string }

export interface Fortress { hp: number; level: number; siege: boolean }
export interface WaveNow {
  id: string; starts_at: string; ends_at: string; days_left: number
  planned: number; closed: number; overdue: number; overdue_p1: number; crowns_now: number
}
export interface Enemy {
  task_id: string; number: string; title: string; points: number; priority: Priority; due_date: string | null
  kind: 'goblin' | 'knight' | 'giant' | 'dragon'; row: number; at_gates: boolean; overdue: boolean
  state: 'ready' | 'in_progress' | 'review'; mine: boolean
}
export interface WaveSummary {
  id: string; starts_at?: string; ends_at: string; planned?: number; closed?: number; overdue?: number
  crowns: number; summary: string; settled_at?: string; defense_bonus?: number
  /** What this viewer was paid for the wave (null: nothing was paid) and the wall repair it brought. */
  chest?: { gold: number; energy: number; repair: number } | null
}
export interface FortressState { fortress: Fortress; wave: WaveNow; enemies: Enemy[]; fog: number; last_wave: WaveSummary | null }
export interface MyProgress { weeks: string[]; weekly: number[]; this_week: number; best_week: number; total: number; explanation: string }
export interface TeamProgress { weeks: string[]; team_weekly: number[]; overdue_now: number; waves: WaveSummary[] }
export interface Duelist { user_id: string; display_name: string; townhall: number; available: boolean; reason: string; defense: Defense }
export interface TeamSettings { id: string; name: string; work_days: number[]; wave_weeks: number }

export type ChallengeStatus = 'pending' | 'accepted' | 'declined' | 'cancelled' | 'done' | 'open' | 'closed'
export interface Challenge {
  id: string; from_id: string; from_name: string; to_id: string | null; to_name: string | null; open: boolean
  status: ChallengeStatus; created_at: string; expires_at: string; attacked: boolean
}
export interface ArenaMember { user_id: string; display_name: string; townhall: number; defense: Defense; accepts_friendly: boolean }
export interface ArenaState {
  incoming: Challenge[]; outgoing: Challenge[]; open: Challenge[]; team: ArenaMember[]
  today: number; per_day: number; friendly_xp: number
  against_me: { id: string; attacker_id: string; attacker: string; stars: number; destruction: number; created_at: string }[]
}
