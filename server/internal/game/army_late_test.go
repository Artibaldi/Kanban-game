package game

import "testing"

func TestRidersGoForTheTowers(t *testing.T) {
	d := Defense{WallHP: 5000, Walls: 10, TowerHP: 600, Towers: 1, TownHallHP: 900, TownHall: 4}
	o := Fight(map[string]int{"rider": 6}, lv1(), d, 1, 0)
	if o.Rounds[0].TowerHit == 0 || o.Rounds[0].WallDamage != 0 {
		t.Fatalf("riders ride around the walls to the towers: %+v", o.Rounds[0])
	}
}

func TestCrossbowHitsHeavyUnits(t *testing.T) {
	d := Defense{TowerHP: 650, CrossbowDPS: 100, Crossbows: 1, TownHallHP: 900000, TownHall: 4}
	o := Fight(map[string]int{"guard": 10, "ram": 5}, lv1(), d, 1, 0)
	r := o.Rounds[0]
	if r.CrossbowDamage != 300 || r.TowerDamage != 0 {
		t.Fatalf("three bolts a round: %+v", r)
	}
	// By hit points alone a guard would fall first; the bolts go for the heavy units.
	if r.Killed["ram"] != 1 || r.Killed["guard"] != 0 {
		t.Fatalf("crossbow kills: %v", r.Killed)
	}
}

func TestStableAndLateTownHall(t *testing.T) {
	th4 := city(Building{Type: "townhall", Level: 4, X: 7, Y: 7})
	if r := CanTrain(th4, "rider"); r == nil || r.Code != "no_stable" {
		t.Fatalf("riders need a stable: %v", r)
	}
	if r := CanTrain(city(Building{Type: "townhall", Level: 4, X: 7, Y: 7}, Building{Type: "stable", Level: 1, X: 3, Y: 3}), "rider"); r != nil {
		t.Fatalf("stable: %v", r)
	}
	th := Buildings["townhall"]
	if th.MaxLevel() != 7 || th.Levels[5].RequiresLevel <= th.Levels[4].RequiresLevel || th.Levels[6].HP <= th.Levels[5].HP {
		t.Fatalf("town hall 6 and 7: %+v", th.Levels[5:])
	}
	if Buildings["stable"].Allowed(3) != 0 || Buildings["stable"].Allowed(4) != 1 {
		t.Fatal("the stable opens with town hall 4")
	}
	cb := Buildings["crossbow"]
	if cb.Allowed(3) != 0 || cb.Allowed(4) != 1 || cb.Allowed(7) <= cb.Allowed(5) || cb.Levels[len(cb.Levels)-1].RequiresTH != 6 {
		t.Fatalf("crossbow towers by town hall: %v", cb.MaxCount)
	}
	for _, k := range []string{"wall", "tower", "barracks", "storage"} {
		if Buildings[k].Allowed(7) < Buildings[k].Allowed(5) {
			t.Fatalf("%s: town hall 7 allows no less than 5", k)
		}
	}
	if Buildings["wall"].Allowed(7) <= Buildings["wall"].Allowed(5) {
		t.Fatal("more walls for the late town hall")
	}
	d := DefenseOf(city(Building{Type: "townhall", Level: 6, X: 7, Y: 7}, Building{Type: "crossbow", Level: 1, X: 3, Y: 3}))
	if d.Crossbows != 1 || d.CrossbowDPS != cb.Levels[0].DPS || d.TowerHP != cb.Levels[0].HP || d.TownHallHP != th.Levels[5].HP {
		t.Fatalf("crossbow in the defence: %+v", d)
	}
}
