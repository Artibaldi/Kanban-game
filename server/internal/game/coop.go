package game

import (
	"fmt"
	"math"
	"time"
)

// Co-op rules: the team defends one fortress against its own backlog. Tasks taken into the plan are enemies,
// a confirmed task is a defeated enemy, overdue enemies damage the walls, and a weekly wave sums it up in crowns.

// IsEnemy: a task the team has taken into work with an agreed estimate. Backlog is fog, done is defeated.
func IsEnemy(state string, pointsApproved bool) bool {
	return pointsApproved && (state == "ready" || state == "in_progress" || state == "review")
}

// EnemyKind maps the complexity to the enemy's size.
func EnemyKind(points int) string {
	switch {
	case points <= 2:
		return "goblin"
	case points == 3:
		return "knight"
	case points <= 5:
		return "giant"
	default:
		return "dragon"
	}
}

// waveEpoch is a Monday; waves of any length are counted from it so every period starts on a Monday 00:00 UTC.
var waveEpoch = time.Date(1970, 1, 5, 0, 0, 0, 0, time.UTC)

// WaveWindow returns the [start, end) of the wave containing t; weeks outside 1..4 count as one week.
func WaveWindow(t time.Time, weeks int) (start, end time.Time) {
	if weeks < 1 || weeks > 4 {
		weeks = 1
	}
	period := time.Duration(weeks) * 7 * 24 * time.Hour
	n := t.UTC().Sub(waveEpoch) / period
	start = waveEpoch.Add(n * period)
	return start, start.Add(period)
}

// IsWorkDay: workDays are ISO weekdays, 1 = Monday … 7 = Sunday.
func IsWorkDay(day time.Time, workDays []int) bool {
	wd := int(day.UTC().Weekday())
	if wd == 0 {
		wd = 7
	}
	for _, d := range workDays {
		if d == wd {
			return true
		}
	}
	return false
}

// WaveTally is what a wave is judged by: points planned and closed, overdue enemies at its end.
type WaveTally struct {
	Planned   int `json:"planned"`
	Closed    int `json:"closed"`
	Overdue   int `json:"overdue"`
	OverdueP1 int `json:"overdue_p1"`
}

// Crowns: 1 — half the plan closed; 2 — the second threshold and no overdue P1; 3 — all closed and nothing overdue.
func Crowns(s Settings, t WaveTally) int {
	if t.Planned <= 0 {
		return 0
	}
	share := float64(t.Closed) / float64(t.Planned)
	switch {
	case t.Closed >= t.Planned && t.Overdue == 0:
		return 3
	case share >= s.CrownThresholds[1] && t.OverdueP1 == 0:
		return 2
	case share >= s.CrownThresholds[0]:
		return 1
	}
	return 0
}

// OverdueDamage is the daily damage of one overdue enemy, in percent of the fortress.
func OverdueDamage(s Settings, points int) int {
	return int(math.Round(float64(points) * s.OverdueDamagePerPoint))
}

// DamageHP lowers the fortress, never below the floor and never raising a fortress that is already under it.
func DamageHP(s Settings, hp, damage int) int {
	return min(hp, max(s.FortressFloor, hp-damage))
}

// RepairHP restores the fortress after a wave; three crowns rebuild it completely.
func RepairHP(s Settings, hp, crowns int) int {
	if crowns >= 3 {
		return 100
	}
	if crowns < 0 {
		crowns = 0
	}
	return min(100, hp+s.RepairByCrowns[crowns])
}

// DefenseScore is one member's contribution to the fortress: walls, towers and ready troops.
func DefenseScore(d Defense, army map[string]int) int64 {
	score := int64(d.WallHP + d.TowerHP)
	for k, n := range army {
		score += int64(Troops[k].HP * n)
	}
	return score
}

// DefenseBonus grows with the members' average defense score up to MaxDefenseBonus. It never changes the crowns.
func DefenseBonus(s Settings, avgScore int64) float64 {
	if avgScore <= 0 {
		return 0
	}
	ratio := math.Min(1, float64(avgScore)/float64(s.DefenseScoreFull))
	return math.Round(s.MaxDefenseBonus*ratio*1000) / 1000
}

// WaveChest is the team reward every member gets for the wave's crowns.
func WaveChest(s Settings, crowns int, bonus float64) (gold, energy int64) {
	if crowns < 0 || crowns > 3 {
		return 0, 0
	}
	gold = int64(math.Round(float64(s.WaveChestGold[crowns]) * (1 + bonus)))
	energy = int64(math.Round(float64(s.WaveChestEnergy[crowns]) * (1 + bonus)))
	return gold, energy
}

// CrownsWord agrees «корона» with a number.
func CrownsWord(n int) string {
	n %= 100
	switch {
	case n%10 == 1 && n != 11:
		return "корона"
	case n%10 >= 2 && n%10 <= 4 && (n < 12 || n > 14):
		return "короны"
	}
	return "корон"
}

// WaveSummary is the text twin of the wave's result.
func WaveSummary(t WaveTally, crowns int) string {
	if t.Planned <= 0 {
		return "Спокойная неделя: врагов не было."
	}
	return fmt.Sprintf("Закрыто %d из %d баллов. Просрочено: %d. Итог: %d %s.", t.Closed, t.Planned, t.Overdue, crowns, CrownsWord(crowns))
}
