package game

import (
	"testing"
	"time"
)

func lv1() map[string]int { return map[string]int{} }

func TestArchersShootOverWalls(t *testing.T) {
	d := Defense{WallHP: 5000, Walls: 10, TownHallHP: 900, OtherHP: 800, TownHall: 2}
	archers := Fight(map[string]int{"archer": 10}, lv1(), d, 1, 0)
	guards := Fight(map[string]int{"guard": 10}, lv1(), d, 1, 0)
	if archers.Rounds[0].BaseDamage == 0 || archers.Rounds[0].WallDamage != 0 {
		t.Fatalf("archers hit buildings behind the walls: %+v", archers.Rounds[0])
	}
	if guards.Rounds[0].BaseDamage != 0 || guards.Rounds[0].WallDamage == 0 {
		t.Fatalf("guards still break walls first: %+v", guards.Rounds[0])
	}
}

func TestMortarPunishesCrowds(t *testing.T) {
	d := Defense{TowerHP: 450, MortarDmg: 6, Mortars: 1, TownHallHP: 900000, TownHall: 3}
	small := Fight(map[string]int{"guard": 3}, lv1(), d, 1, 0)
	crowd := Fight(map[string]int{"guard": 30}, lv1(), d, 1, 0)
	// Splash: every hit lands on up to MortarSplash units, two volleys a round.
	if small.Rounds[0].MortarDamage != 6*3*2 || crowd.Rounds[0].MortarDamage != 6*MortarSplash*2 {
		t.Fatalf("mortar damage: %d and %d", small.Rounds[0].MortarDamage, crowd.Rounds[0].MortarDamage)
	}
	none := Fight(map[string]int{"guard": 30}, lv1(), Defense{TowerHP: 450, TownHallHP: 900000, TownHall: 3}, 1, 0)
	if none.Rounds[0].MortarDamage != 0 {
		t.Fatal("no mortar, no splash")
	}
}

func TestTrapsFireOnce(t *testing.T) {
	d := Defense{TrapDmg: 120, Traps: 2, TownHallHP: 900000, TownHall: 2}
	o := Fight(map[string]int{"raider": 10}, lv1(), d, 1, 0)
	if o.Rounds[0].TrapDamage != 120 || !hasEvent(o.Rounds[0], "trap_sprung") {
		t.Fatalf("round 1: %+v", o.Rounds[0])
	}
	if o.Rounds[0].Killed["raider"] != 2 { // 120 damage on 45-hp raiders: two fall, one is wounded
		t.Fatalf("trap kills: %v", o.Rounds[0].Killed)
	}
	for _, r := range o.Rounds[1:] {
		if r.TrapDamage != 0 || hasEvent(r, "trap_sprung") {
			t.Fatalf("a trap fires only once: round %d", r.N)
		}
	}
}

func TestTroopLevelsStrengthen(t *testing.T) {
	d := Defense{WallHP: 400, TowerHP: 600, TowerDPS: 32, Towers: 2, TownHallHP: 1800, OtherHP: 1500, TownHall: 3}
	base := Fight(map[string]int{"guard": 15}, lv1(), d, 7, 0)
	strong := Fight(map[string]int{"guard": 15}, map[string]int{"guard": 3}, d, 7, 0)
	if strong.Destruction <= base.Destruction || strong.Losses["guard"] > base.Losses["guard"] {
		t.Fatalf("level 3 guards: %d%% / %d lost vs %d%% / %d lost", strong.Destruction, strong.Losses["guard"], base.Destruction, base.Losses["guard"])
	}
	if TroopMult(1) != 1 || TroopMult(3) != 1.2 || TroopMult(0) != 1 {
		t.Fatalf("mult: %v %v", TroopMult(1), TroopMult(3))
	}
}

func TestDefenseOfNewBuildings(t *testing.T) {
	d := DefenseOf(city(Building{Type: "townhall", Level: 3, X: 7, Y: 7}, Building{Type: "mortar", Level: 1, X: 3, Y: 3},
		Building{Type: "trap", Level: 1, X: 4, Y: 3}, Building{Type: "trap", Level: 2, X: 5, Y: 3}, Building{Type: "archery", Level: 1, X: 6, Y: 3}))
	m := Buildings["mortar"].Levels[0]
	if d.Mortars != 1 || d.MortarDmg != m.Splash || d.TowerHP != m.HP {
		t.Fatalf("mortar is a defence: %+v", d)
	}
	if d.Traps != 2 || d.TrapDmg != Buildings["trap"].Levels[0].TrapDmg+Buildings["trap"].Levels[1].TrapDmg {
		t.Fatalf("traps: %+v", d)
	}
	if d.OtherHP != Buildings["archery"].Levels[0].HP {
		t.Fatalf("traps have no hit points, the range does: %+v", d)
	}
	v := d.Visible()
	if v.Traps != 0 || v.TrapDmg != 0 || v.MortarDmg != d.MortarDmg {
		t.Fatalf("a rival never sees the traps: %+v", v)
	}
}

func TestTrainingAndResearchRules(t *testing.T) {
	plain := city(Building{Type: "townhall", Level: 3, X: 7, Y: 7})
	if r := CanTrain(plain, "archer"); r == nil || r.Code != "no_archery" {
		t.Fatalf("archers need a range: %v", r)
	}
	if r := CanTrain(plain, "guard"); r != nil {
		t.Fatalf("guards need nothing: %v", r)
	}
	lvl, until := 1, time.Now().Add(time.Minute)
	building := city(Building{Type: "townhall", Level: 3, X: 7, Y: 7}, Building{Type: "archery", Level: 0, TargetLevel: &lvl, BusyUntil: &until, X: 3, Y: 3})
	if r := CanTrain(building, "archer"); r == nil {
		t.Fatal("a range under construction trains nobody yet")
	}
	ws := city(Building{Type: "townhall", Level: 3, X: 7, Y: 7}, Building{Type: "workshop", Level: 1, X: 3, Y: 3})
	if MaxTroopLevel(plain) != 1 || MaxTroopLevel(ws) != 2 {
		t.Fatalf("max troop level: %d %d", MaxTroopLevel(plain), MaxTroopLevel(ws))
	}
	if _, r := CheckResearch(plain, "guard", 1, false, 5000); r == nil || r.Code != "no_workshop" {
		t.Fatalf("no workshop: %v", r)
	}
	spec, r := CheckResearch(ws, "guard", 1, false, 5000)
	if r != nil || spec.Level != 2 || spec.Cost <= 0 || spec.Seconds <= 0 {
		t.Fatalf("guard → 2: %+v %v", spec, r)
	}
	if _, r := CheckResearch(ws, "guard", 2, false, 5000); r == nil || r.Code != "workshop_level" {
		t.Fatalf("level 3 needs workshop 2: %v", r)
	}
	if _, r := CheckResearch(ws, "guard", 1, true, 5000); r == nil || r.Code != "workshop_busy" {
		t.Fatalf("one research at a time: %v", r)
	}
	if _, r := CheckResearch(ws, "guard", 1, false, 10); r == nil || r.Code != "insufficient_gold" {
		t.Fatalf("gold: %v", r)
	}
	if _, r := CheckResearch(ws, "dragon", 1, false, 5000); r == nil || r.Code != "invalid_troop" {
		t.Fatalf("unknown troop: %v", r)
	}
}

func hasEvent(r Round, e string) bool {
	for _, x := range r.Events {
		if x == e {
			return true
		}
	}
	return false
}
