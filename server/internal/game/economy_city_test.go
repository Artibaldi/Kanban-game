package game

import (
	"testing"
	"time"
)

func TestBuilderHutsAddBuilders(t *testing.T) {
	hut := Buildings["hut"]
	if hut.Allowed(2) != 0 || hut.Allowed(3) != 1 || hut.Allowed(4) != 1 || hut.Allowed(5) != 2 {
		t.Fatalf("huts by town hall: %v", hut.MaxCount)
	}
	lvl, until := 1, time.Now().Add(time.Minute)
	c := city(Building{Type: "townhall", Level: 3, X: 7, Y: 7}, Building{Type: "hut", Level: 1, X: 3, Y: 3},
		Building{Type: "hut", Level: 0, TargetLevel: &lvl, BusyUntil: &until, X: 4, Y: 3})
	if c.Builders() != MaxBuilders+1 {
		t.Fatalf("a finished hut adds a builder, one under construction does not: %d", c.Builders())
	}
	// Three builders: two busy (the new hut and a wall) still leave one free.
	wl := 2
	c.Buildings = append(c.Buildings, Building{Type: "wall", Level: 1, TargetLevel: &wl, BusyUntil: &until, X: 5, Y: 3})
	if _, r := CheckPlace(c, "wall", 6, 6, 1000); r != nil {
		t.Fatalf("the third builder takes the job: %v", r)
	}
	c.Buildings = append(c.Buildings, Building{Type: "storage", Level: 1, TargetLevel: &wl, BusyUntil: &until, X: 5, Y: 4})
	if _, r := CheckPlace(c, "wall", 6, 6, 1000); r == nil || r.Code != "builders_busy" {
		t.Fatalf("all three busy: %v", r)
	}
}

func TestCrystalWellRaisesOnlyEnergy(t *testing.T) {
	base := city(Building{Type: "townhall", Level: 2, X: 7, Y: 7})
	g0, e0 := base.Caps()
	well := city(Building{Type: "townhall", Level: 2, X: 7, Y: 7}, Building{Type: "well", Level: 1, X: 3, Y: 3})
	g1, e1 := well.Caps()
	if g1 != g0 || e1 != e0+Buildings["well"].Levels[0].EnergyCap || Buildings["well"].Levels[0].EnergyCap <= 0 {
		t.Fatalf("well caps: gold %d→%d energy %d→%d", g0, g1, e0, e1)
	}
	if Buildings["well"].Allowed(1) != 0 || Buildings["well"].Allowed(2) != 1 {
		t.Fatal("the well opens with town hall 2")
	}
}

func TestMarketExchange(t *testing.T) {
	s := DefaultSettings() // 40 gold and 10 energy per point: 1 energy is worth 4 gold
	if Buildings["market"].Allowed(1) != 0 || Buildings["market"].Allowed(2) != 1 {
		t.Fatal("the market opens with town hall 2")
	}
	none := city(Building{Type: "townhall", Level: 2, X: 7, Y: 7})
	if _, _, r := Exchange(s, none, "gold", 100); r == nil || r.Code != "no_market" {
		t.Fatalf("no market: %v", r)
	}
	m1 := city(Building{Type: "townhall", Level: 2, X: 7, Y: 7}, Building{Type: "market", Level: 1, X: 3, Y: 3})
	// Level 1 keeps 30%: 100 gold → 25 energy before the fee → 17 energy.
	if dg, de, r := Exchange(s, m1, "gold", 100); r != nil || dg != -100 || de != 17 {
		t.Fatalf("gold → energy: %d %d %v", dg, de, r)
	}
	// 10 energy → 40 gold → 28 gold.
	if dg, de, r := Exchange(s, m1, "energy", 10); r != nil || dg != 28 || de != -10 {
		t.Fatalf("energy → gold: %d %d %v", dg, de, r)
	}
	m3 := city(Building{Type: "townhall", Level: 4, X: 7, Y: 7}, Building{Type: "market", Level: 3, X: 3, Y: 3})
	if _, de, _ := Exchange(s, m3, "gold", 100); de != 22 {
		t.Fatalf("level 3 keeps 10%%: %d", de)
	}
	// A round trip always loses: the market never creates value.
	for _, c := range []City{m1, m3} {
		_, e, _ := Exchange(s, c, "gold", 1000)
		g, _, _ := Exchange(s, c, "energy", e)
		if g >= 1000 {
			t.Fatalf("round trip gained: %d", g)
		}
	}
	if _, _, r := Exchange(s, m1, "gold", 3); r == nil || r.Code != "too_little" {
		t.Fatalf("too little to get anything: %v", r)
	}
	for _, bad := range []struct {
		give   string
		amount int64
	}{{"gold", 0}, {"gold", -5}, {"xp", 10}} {
		if _, _, r := Exchange(s, m1, bad.give, bad.amount); r == nil || r.Code != "invalid_input" {
			t.Fatalf("%v: %v", bad, r)
		}
	}
	// While the market is being upgraded it keeps trading at its current level.
	lvl, until := 2, time.Now().Add(time.Minute)
	busy := city(Building{Type: "townhall", Level: 3, X: 7, Y: 7}, Building{Type: "market", Level: 1, TargetLevel: &lvl, BusyUntil: &until, X: 3, Y: 3})
	if _, de, r := Exchange(s, busy, "gold", 100); r != nil || de != 17 {
		t.Fatalf("upgrading market: %d %v", de, r)
	}
}

func TestEconomyBuildingsDefendLikeOthers(t *testing.T) {
	d := DefenseOf(city(Building{Type: "townhall", Level: 3, X: 7, Y: 7}, Building{Type: "market", Level: 1, X: 3, Y: 3},
		Building{Type: "well", Level: 1, X: 4, Y: 3}, Building{Type: "hut", Level: 1, X: 5, Y: 3}))
	if d.OtherHP != Buildings["market"].Levels[0].HP+Buildings["well"].Levels[0].HP+Buildings["hut"].Levels[0].HP {
		t.Fatalf("other hp: %d", d.OtherHP)
	}
}

func TestTradeFits(t *testing.T) {
	c := city(Building{Type: "townhall", Level: 2, X: 7, Y: 7}, Building{Type: "market", Level: 1, X: 3, Y: 3})
	_, capE := c.Caps()
	if r := TradeFits(c, 50, 0, -100, 17); r == nil || r.Code != "insufficient_gold" {
		t.Fatalf("not enough gold: %v", r)
	}
	if r := TradeFits(c, 500, 5, 28, -10); r == nil || r.Code != "insufficient_energy" {
		t.Fatalf("not enough energy: %v", r)
	}
	if r := TradeFits(c, 500, capE-10, -100, 17); r == nil || r.Code != "storage_full" {
		t.Fatalf("the energy store would overflow: %v", r)
	}
	if r := TradeFits(c, 500, capE-17, -100, 17); r != nil {
		t.Fatalf("fills the store exactly: %v", r)
	}
}

func TestExchangeNeverOverflows(t *testing.T) {
	s := DefaultSettings()
	s.GoldPerPoint = 1e19 // a mistaken admin edit: one energy would be worth more gold than int64 holds
	m := city(Building{Type: "townhall", Level: 2, X: 7, Y: 7}, Building{Type: "market", Level: 1, X: 3, Y: 3})
	if dg, _, r := Exchange(s, m, "energy", 10); r == nil || r.Code != "storage_full" {
		t.Fatalf("an absurd payout is refused, not wrapped: %d %v", dg, r)
	}
	if r := TradeFits(m, 1<<62, 10, 1<<62, -1); r == nil || r.Code != "storage_full" {
		t.Fatalf("no wrap-around past the cap: %v", r)
	}
}
