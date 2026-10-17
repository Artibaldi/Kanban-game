// Package game holds the pure rules of the service: rewards, levels, buildings, troops and battles.
// Nothing here touches the database, so every formula is unit-tested on its own.
package game

import (
	"math"
	"time"
)

// Settings are the coefficients an administrator may change. Stored as JSON under settings.key='economy'.
type Settings struct {
	GoldPerPoint         float64            `json:"gold_per_point"`
	EnergyPerPoint       float64            `json:"energy_per_point"`
	XPPerPoint           float64            `json:"xp_per_point"`
	PriorityMult         map[string]float64 `json:"priority_mult"`
	OnTimeBonus          float64            `json:"on_time_bonus"`
	LevelXPBase          float64            `json:"level_xp_base"`
	LevelXPExp           float64            `json:"level_xp_exp"`
	AttackEnergy         int64              `json:"attack_energy"`
	ShieldMinutes        int                `json:"shield_minutes"`
	RematchMinutes       int                `json:"rematch_minutes"`
	RandomSpread         float64            `json:"random_spread"`
	BattleGoldBase       float64            `json:"battle_gold_base"`
	BattleGoldPerPercent float64            `json:"battle_gold_per_percent"`
	BattleXPPerStar      float64            `json:"battle_xp_per_star"`
	// TimeScale multiplies every build and training time; 1 = short demo timers.
	TimeScale float64 `json:"time_scale"`
	// BattleScope: "team" — opponents only from the attacker's team; "all" — from every team.
	BattleScope string `json:"battle_scope"`
	// StartGold and StartEnergy are granted once when a player joins.
	StartGold   int64 `json:"start_gold"`
	StartEnergy int64 `json:"start_energy"`

	// Co-op: fortress damage in percent per overdue point a day, its floor, repair and the team chest by crowns.
	OverdueDamagePerPoint float64    `json:"overdue_damage_per_point"`
	FortressFloor         int        `json:"fortress_floor"`
	RepairByCrowns        [4]int     `json:"repair_by_crowns"`
	CrownThresholds       [2]float64 `json:"crown_thresholds"`
	WaveChestGold         [4]int64   `json:"wave_chest_gold"`
	WaveChestEnergy       [4]int64   `json:"wave_chest_energy"`
	// MaxDefenseBonus is reached when the members' average defense score is DefenseScoreFull.
	MaxDefenseBonus  float64 `json:"max_defense_bonus"`
	DefenseScoreFull int64   `json:"defense_score_full"`
	FriendlyXP       int64   `json:"friendly_xp"`
	FriendlyPerDay   int     `json:"friendly_per_day"`
}

func DefaultSettings() Settings {
	return Settings{
		GoldPerPoint:         40,
		EnergyPerPoint:       10,
		XPPerPoint:           25,
		PriorityMult:         map[string]float64{"P1": 1.2, "P2": 1.1, "P3": 1.0, "P4": 0.9},
		OnTimeBonus:          0.10,
		LevelXPBase:          100,
		LevelXPExp:           1.5,
		AttackEnergy:         20,
		ShieldMinutes:        10,
		RematchMinutes:       30,
		RandomSpread:         0.10,
		BattleGoldBase:       50,
		BattleGoldPerPercent: 3,
		BattleXPPerStar:      15,
		TimeScale:            1,
		BattleScope:          "team",
		StartGold:            300,
		StartEnergy:          60,

		OverdueDamagePerPoint: 2,
		FortressFloor:         10,
		RepairByCrowns:        [4]int{0, 10, 25, 100},
		CrownThresholds:       [2]float64{0.5, 0.8},
		WaveChestGold:         [4]int64{0, 60, 140, 250},
		WaveChestEnergy:       [4]int64{0, 15, 35, 60},
		MaxDefenseBonus:       0.20,
		DefenseScoreFull:      4000,
		FriendlyXP:            5,
		FriendlyPerDay:        3,
	}
}

// Normalize fills missing or invalid values from the defaults, so a partial admin edit cannot break the game.
func (s Settings) Normalize() Settings {
	d := DefaultSettings()
	pos := func(v *float64, def float64) {
		if !(*v > 0) || math.IsInf(*v, 0) {
			*v = def
		}
	}
	pos(&s.GoldPerPoint, d.GoldPerPoint)
	pos(&s.EnergyPerPoint, d.EnergyPerPoint)
	pos(&s.XPPerPoint, d.XPPerPoint)
	pos(&s.LevelXPBase, d.LevelXPBase)
	pos(&s.LevelXPExp, d.LevelXPExp)
	pos(&s.BattleGoldBase, d.BattleGoldBase)
	pos(&s.BattleGoldPerPercent, d.BattleGoldPerPercent)
	pos(&s.BattleXPPerStar, d.BattleXPPerStar)
	pos(&s.TimeScale, d.TimeScale)
	if s.OnTimeBonus < 0 || s.OnTimeBonus > 1 {
		s.OnTimeBonus = d.OnTimeBonus
	}
	if s.RandomSpread < 0 || s.RandomSpread > 0.3 {
		s.RandomSpread = d.RandomSpread
	}
	if s.AttackEnergy < 0 {
		s.AttackEnergy = d.AttackEnergy
	}
	if s.ShieldMinutes < 0 {
		s.ShieldMinutes = d.ShieldMinutes
	}
	if s.RematchMinutes < 0 {
		s.RematchMinutes = d.RematchMinutes
	}
	if s.BattleScope != "team" && s.BattleScope != "all" {
		s.BattleScope = d.BattleScope
	}
	if s.StartGold < 0 {
		s.StartGold = d.StartGold
	}
	if s.StartEnergy < 0 {
		s.StartEnergy = d.StartEnergy
	}
	if !(s.OverdueDamagePerPoint > 0) || s.OverdueDamagePerPoint > 100 {
		s.OverdueDamagePerPoint = d.OverdueDamagePerPoint
	}
	if s.FortressFloor <= 0 || s.FortressFloor > 100 {
		s.FortressFloor = d.FortressFloor
	}
	if s.RepairByCrowns == [4]int{} {
		s.RepairByCrowns = d.RepairByCrowns
	}
	if th := s.CrownThresholds; !(th[0] > 0 && th[0] < th[1] && th[1] <= 1) {
		s.CrownThresholds = d.CrownThresholds
	}
	if s.WaveChestGold == [4]int64{} {
		s.WaveChestGold = d.WaveChestGold
	}
	if s.WaveChestEnergy == [4]int64{} {
		s.WaveChestEnergy = d.WaveChestEnergy
	}
	if s.MaxDefenseBonus < 0 || s.MaxDefenseBonus > 1 {
		s.MaxDefenseBonus = d.MaxDefenseBonus
	}
	if s.DefenseScoreFull <= 0 {
		s.DefenseScoreFull = d.DefenseScoreFull
	}
	if s.FriendlyXP <= 0 {
		s.FriendlyXP = d.FriendlyXP
	}
	if s.FriendlyPerDay <= 0 {
		s.FriendlyPerDay = d.FriendlyPerDay
	}
	if s.PriorityMult == nil {
		s.PriorityMult = map[string]float64{}
	}
	for k, v := range d.PriorityMult {
		if m, ok := s.PriorityMult[k]; !ok || !(m > 0) {
			s.PriorityMult[k] = v
		}
	}
	return s
}

// ValidPoints is the agreed complexity scale.
func ValidPoints(p int) bool { return p == 1 || p == 2 || p == 3 || p == 5 || p == 8 }

// Reward is what a confirmed task is worth, before the storage cap.
type Reward struct {
	Gold   int64 `json:"gold"`
	Energy int64 `json:"energy"`
	XP     int64 `json:"xp"`
	OnTime bool  `json:"on_time"`
}

// TaskReward: gold = gold_per_point × points × priority multiplier × (1 + on-time bonus); energy and xp are per point.
// A task is on time when approved on or before its due date; a task without a due date gets no bonus.
func TaskReward(s Settings, points int, priority string, due *time.Time, approvedAt time.Time) Reward {
	mult := s.PriorityMult[priority]
	if mult == 0 {
		mult = 1
	}
	onTime := false
	if due != nil {
		end := time.Date(due.Year(), due.Month(), due.Day(), 23, 59, 59, 0, approvedAt.Location())
		onTime = !approvedAt.After(end)
	}
	gold := s.GoldPerPoint * float64(points) * mult
	if onTime {
		gold *= 1 + s.OnTimeBonus
	}
	return Reward{
		Gold:   int64(math.Round(gold)),
		Energy: int64(math.Round(s.EnergyPerPoint * float64(points))),
		XP:     int64(math.Round(s.XPPerPoint * float64(points))),
		OnTime: onTime,
	}
}

// XPForNext is the experience needed to go from level n to n+1: base × n^exp.
func XPForNext(s Settings, level int) int64 {
	return int64(math.Round(s.LevelXPBase * math.Pow(float64(level), s.LevelXPExp)))
}

// Level derives the player level from total experience; negative experience (after a reversal) stays at level 1.
// Returns the level, experience inside the level and experience needed for the next one.
func Level(s Settings, xp int64) (level int, into int64, need int64) {
	level = 1
	rest := xp
	if rest < 0 {
		rest = 0
	}
	for level < 100 {
		n := XPForNext(s, level)
		if rest < n {
			return level, rest, n
		}
		rest -= n
		level++
	}
	return level, rest, XPForNext(s, level)
}

// Credit applies an income to a balance under a cap. Debt (a negative balance) is always repaid in full;
// only the part above the cap is lost. Returns the amount actually credited.
func Credit(balance, amount, cap int64) int64 {
	if amount <= 0 {
		return amount
	}
	room := cap - balance
	if room <= 0 {
		return 0
	}
	if amount > room {
		return room
	}
	return amount
}
