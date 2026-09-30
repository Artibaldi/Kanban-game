package game

import (
	"reflect"
	"testing"
	"time"
)

func TestTaskRewardExample(t *testing.T) {
	s := DefaultSettings()
	due := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	r := TaskReward(s, 5, "P2", &due, time.Date(2026, 10, 5, 18, 0, 0, 0, time.UTC))
	if r.Gold != 242 || r.Energy != 50 || r.XP != 125 || !r.OnTime {
		t.Fatalf("5 points P2 on time: %+v", r)
	}
	late := TaskReward(s, 5, "P2", &due, time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC))
	if late.Gold != 220 || late.OnTime {
		t.Fatalf("late: %+v", late)
	}
	none := TaskReward(s, 1, "P4", nil, time.Now())
	if none.Gold != 36 || none.Energy != 10 || none.XP != 25 {
		t.Fatalf("1 point P4: %+v", none)
	}
}

func TestLevel(t *testing.T) {
	s := DefaultSettings()
	if l, _, need := Level(s, 0); l != 1 || need != 100 {
		t.Fatalf("level 0 xp: %d %d", l, need)
	}
	// 100 to reach 2, 283 more to reach 3.
	if l, into, _ := Level(s, 150); l != 2 || into != 50 {
		t.Fatalf("150 xp: %d %d", l, into)
	}
	if l, _, _ := Level(s, 383); l != 3 {
		t.Fatalf("383 xp: %d", l)
	}
	if l, into, _ := Level(s, -40); l != 1 || into != 0 {
		t.Fatalf("negative xp: %d %d", l, into)
	}
}

func TestCredit(t *testing.T) {
	cases := []struct{ balance, amount, cap, want int64 }{
		{100, 50, 1000, 50},
		{980, 50, 1000, 20},
		{1200, 50, 1000, 0},
		{-300, 200, 1000, 200}, // debt is repaid in full
		{100, -80, 1000, -80},  // a reversal is never capped
	}
	for _, c := range cases {
		if got := Credit(c.balance, c.amount, c.cap); got != c.want {
			t.Errorf("Credit(%d,%d,%d)=%d want %d", c.balance, c.amount, c.cap, got, c.want)
		}
	}
}

func city(bs ...Building) City { return City{Buildings: bs} }

func TestPlaceRules(t *testing.T) {
	c := city(Building{Type: "townhall", Level: 1, X: 5, Y: 5}, Building{Type: "tower", Level: 1, X: 1, Y: 1})
	if _, r := CheckPlace(c, "barracks", 5, 5, 1000); r == nil || r.Code != "cell_occupied" {
		t.Fatalf("occupied: %v", r)
	}
	if _, r := CheckPlace(c, "barracks", 12, 0, 1000); r == nil || r.Code != "invalid_cell" {
		t.Fatalf("outside: %v", r)
	}
	if _, r := CheckPlace(c, "tower", 2, 2, 1000); r == nil || r.Code != "building_limit" {
		t.Fatalf("limit: %v", r)
	}
	if _, r := CheckPlace(c, "barracks", 2, 2, 100); r == nil || r.Code != "insufficient_gold" {
		t.Fatalf("gold: %v", r)
	}
	if _, r := CheckPlace(c, "barracks", 2, 2, -5); r == nil || r.Code != "insufficient_gold" {
		t.Fatalf("debt blocks spending: %v", r)
	}
	if spec, r := CheckPlace(c, "barracks", 2, 2, 150); r != nil || spec.Cost != 150 {
		t.Fatalf("ok: %v %v", spec, r)
	}
	lvl := 2
	until := time.Now()
	busy := city(Building{Type: "townhall", Level: 1}, Building{Type: "wall", X: 1, Level: 0, TargetLevel: &lvl, BusyUntil: &until}, Building{Type: "wall", X: 2, Level: 0, TargetLevel: &lvl, BusyUntil: &until})
	if _, r := CheckPlace(busy, "wall", 3, 3, 1000); r == nil || r.Code != "builders_busy" {
		t.Fatalf("builders: %v", r)
	}
}

func TestUpgradeRules(t *testing.T) {
	c := city(Building{Type: "townhall", Level: 1}, Building{ID: "b", Type: "barracks", Level: 1, X: 1})
	if _, r := CheckUpgrade(c, c.Buildings[1], 5000, 10); r == nil || r.Code != "townhall_level" {
		t.Fatalf("th requirement: %v", r)
	}
	if _, r := CheckUpgrade(c, c.Buildings[0], 400, 1); r == nil || r.Code != "player_level" {
		t.Fatalf("player level: %v", r)
	}
	if spec, r := CheckUpgrade(c, c.Buildings[0], 400, 2); r != nil || spec.Level != 2 {
		t.Fatalf("th upgrade: %v %v", spec, r)
	}
}

func TestFightDeterministicAndBounded(t *testing.T) {
	d := DefenseOf(city(
		Building{Type: "townhall", Level: 2},
		Building{Type: "tower", Level: 2}, Building{Type: "tower", Level: 1},
		Building{Type: "wall", Level: 2}, Building{Type: "wall", Level: 2}, Building{Type: "wall", Level: 1},
		Building{Type: "barracks", Level: 1}, Building{Type: "storage", Level: 1},
	))
	army := map[string]int{"guard": 8, "raider": 6, "ram": 2}
	a := Fight(army, d, 42, 0.1)
	b := Fight(army, d, 42, 0.1)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same seed must give same battle")
	}
	if a.Destruction < 0 || a.Destruction > 100 || a.Stars < 0 || a.Stars > 3 || len(a.Rounds) > MaxRounds {
		t.Fatalf("bounds: %+v", a)
	}
	for _, k := range TroopOrder {
		if a.Losses[k]+a.Survivors[k] != army[k] {
			t.Fatalf("troops must be conserved: %s %d+%d != %d", k, a.Losses[k], a.Survivors[k], army[k])
		}
	}
	// Spread is bounded: without randomness the result stays close.
	flat := Fight(army, d, 42, 0)
	if diff := flat.Destruction - a.Destruction; diff > 25 || diff < -25 {
		t.Fatalf("random factor too strong: %d vs %d", flat.Destruction, a.Destruction)
	}
}

func TestFightRamsBeatWalls(t *testing.T) {
	d := DefenseOf(city(Building{Type: "townhall", Level: 1}, Building{Type: "wall", Level: 2}, Building{Type: "wall", Level: 2}, Building{Type: "wall", Level: 2}, Building{Type: "tower", Level: 1}))
	// Same housing: 3 rams take the room of 6 raiders.
	withRams := Fight(map[string]int{"guard": 4, "ram": 3}, d, 7, 0)
	noRams := Fight(map[string]int{"guard": 4, "raider": 6}, d, 7, 0)
	if withRams.Destruction <= noRams.Destruction {
		t.Fatalf("rams should break fortified cities better: %d vs %d", withRams.Destruction, noRams.Destruction)
	}
}

func TestRewardForLossAndWin(t *testing.T) {
	s := DefaultSettings()
	lose := RewardFor(s, Outcome{Destruction: 0, Stars: 0}, 2)
	if lose.Gold != 0 || lose.AttackerTrophies != -5 || lose.DefenderTrophies != 5 {
		t.Fatalf("loss: %+v", lose)
	}
	win := RewardFor(s, Outcome{Destruction: 60, Stars: 1}, 1)
	if win.Gold != 230 || win.XP != 20 || win.AttackerTrophies != 8 {
		t.Fatalf("win: %+v", win)
	}
}
