package game

import (
	"testing"
	"time"
)

func TestCrowns(t *testing.T) {
	s := DefaultSettings()
	cases := []struct {
		t    WaveTally
		want int
	}{
		{WaveTally{Planned: 0}, 0},
		{WaveTally{Planned: 30, Closed: 14}, 0},
		{WaveTally{Planned: 30, Closed: 15}, 1},
		{WaveTally{Planned: 30, Closed: 24, Overdue: 1}, 2}, // the example from the spec
		{WaveTally{Planned: 30, Closed: 24, Overdue: 1, OverdueP1: 1}, 1},
		{WaveTally{Planned: 30, Closed: 30, Overdue: 1}, 2},
		{WaveTally{Planned: 30, Closed: 30}, 3},
	}
	for _, c := range cases {
		if got := Crowns(s, c.t); got != c.want {
			t.Errorf("%+v: got %d want %d", c.t, got, c.want)
		}
	}
}

func TestFortressHP(t *testing.T) {
	s := DefaultSettings()
	if d := OverdueDamage(s, 5); d != 10 {
		t.Fatalf("5 points → %d", d)
	}
	if hp := DamageHP(s, 15, 10); hp != 10 {
		t.Fatalf("floor: %d", hp)
	}
	if hp := DamageHP(s, 8, 4); hp != 8 {
		t.Fatalf("below floor never drops further: %d", hp)
	}
	if hp := RepairHP(s, 40, 2); hp != 65 {
		t.Fatalf("2 crowns: %d", hp)
	}
	if hp := RepairHP(s, 95, 1); hp != 100 {
		t.Fatalf("cap: %d", hp)
	}
	if hp := RepairHP(s, 12, 3); hp != 100 {
		t.Fatalf("3 crowns: %d", hp)
	}
}

func TestWaveWindow(t *testing.T) {
	wed := time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC)
	s, e := WaveWindow(wed, 1)
	if !s.Equal(time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)) || !e.Equal(time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("week: %v %v", s, e)
	}
	s2, e2 := WaveWindow(wed, 2)
	if e2.Sub(s2) != 14*24*time.Hour || s2.Weekday() != time.Monday || wed.Before(s2) || !wed.Before(e2) {
		t.Fatalf("sprint: %v %v", s2, e2)
	}
	if s3, _ := WaveWindow(time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC), 1); !s3.Equal(e) {
		t.Fatalf("boundary belongs to the next wave: %v", s3)
	}
	if s4, e4 := WaveWindow(wed, 9); !s4.Equal(s) || !e4.Equal(e) {
		t.Fatalf("invalid length is one week: %v %v", s4, e4)
	}
}

func TestEnemiesAndWords(t *testing.T) {
	if IsEnemy("backlog", true) || IsEnemy("ready", false) || IsEnemy("done", true) || !IsEnemy("review", true) {
		t.Fatal("IsEnemy")
	}
	if EnemyKind(2) != "goblin" || EnemyKind(3) != "knight" || EnemyKind(5) != "giant" || EnemyKind(8) != "dragon" {
		t.Fatal("EnemyKind")
	}
	if !IsWorkDay(time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC), []int{1, 2, 3, 4, 5}) ||
		IsWorkDay(time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC), []int{1, 2, 3, 4, 5}) {
		t.Fatal("IsWorkDay: friday yes, sunday no")
	}
	if CrownsWord(1) != "корона" || CrownsWord(2) != "короны" || CrownsWord(0) != "корон" || CrownsWord(11) != "корон" || CrownsWord(21) != "корона" {
		t.Fatal("CrownsWord")
	}
	got := WaveSummary(WaveTally{Planned: 30, Closed: 24, Overdue: 1}, 2)
	if got != "Закрыто 24 из 30 баллов. Просрочено: 1. Итог: 2 короны." {
		t.Fatalf("summary: %q", got)
	}
	if got := WaveSummary(WaveTally{}, 0); got != "Спокойная неделя: врагов не было." {
		t.Fatalf("calm: %q", got)
	}
}

func TestWaveChestAndBonus(t *testing.T) {
	s := DefaultSettings()
	if b := DefenseBonus(s, 2000); b != 0.10 {
		t.Fatalf("half score: %v", b)
	}
	if b := DefenseBonus(s, 9000); b != 0.20 {
		t.Fatalf("capped: %v", b)
	}
	if g, e := WaveChest(s, 2, 0.10); g != 154 || e != 39 {
		t.Fatalf("2 crowns +10%%: %d %d", g, e)
	}
	if g, e := WaveChest(s, 0, 0.2); g != 0 || e != 0 {
		t.Fatalf("0 crowns: %d %d", g, e)
	}
	if score := DefenseScore(Defense{WallHP: 100, TowerHP: 300}, map[string]int{"guard": 2}); score != 600 {
		t.Fatalf("defense score: %d", score)
	}
}

func TestCoopSettingsNormalize(t *testing.T) {
	s := Settings{}.Normalize()
	d := DefaultSettings()
	if s.RepairByCrowns != d.RepairByCrowns || s.CrownThresholds != d.CrownThresholds || s.WaveChestGold != d.WaveChestGold ||
		s.FortressFloor != 10 || s.OverdueDamagePerPoint != 2 || s.FriendlyPerDay != 3 || s.FriendlyXP != 5 {
		t.Fatalf("defaults: %+v", s)
	}
	bad := DefaultSettings()
	bad.CrownThresholds = [2]float64{0.9, 0.5}
	if bad.Normalize().CrownThresholds != d.CrownThresholds {
		t.Fatal("thresholds must be 0 < a < b <= 1")
	}
}
