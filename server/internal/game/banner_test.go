package game

import "testing"

func TestBannerHoldsReinforcements(t *testing.T) {
	none := city(Building{Type: "townhall", Level: 2, X: 7, Y: 7})
	if none.GarrisonCapacity() != 0 {
		t.Fatal("no banner, no room")
	}
	c := city(Building{Type: "townhall", Level: 2, X: 7, Y: 7}, Building{Type: "banner", Level: 1, X: 3, Y: 3})
	room := Buildings["banner"].Levels[0].Garrison
	if c.GarrisonCapacity() != room || room <= 0 {
		t.Fatalf("banner room: %d", c.GarrisonCapacity())
	}
	if r := CheckDonate(none, 0, "guard", 1, 5); r == nil || r.Code != "no_banner" {
		t.Fatalf("no banner: %v", r)
	}
	if r := CheckDonate(c, 0, "guard", 2, 1); r == nil || r.Code != "not_enough_troops" {
		t.Fatalf("donor has one guard: %v", r)
	}
	if r := CheckDonate(c, room-1, "ram", 1, 5); r == nil || r.Code != "banner_full" {
		t.Fatalf("a ram takes two places: %v", r)
	}
	if r := CheckDonate(c, 0, "guard", room, room); r != nil {
		t.Fatalf("fills the banner: %v", r)
	}
	for _, bad := range []struct {
		kind  string
		count int
	}{{"dragon", 1}, {"guard", 0}, {"guard", -1}} {
		if r := CheckDonate(c, 0, bad.kind, bad.count, 9); r == nil || r.Code != "invalid_input" {
			t.Fatalf("%v: %v", bad, r)
		}
	}
}

func TestGarrisonDefends(t *testing.T) {
	d := Defense{WallHP: 0, TowerHP: 600, TowerDPS: 0, Towers: 1, TownHallHP: 1800, TownHall: 3}
	plain := Fight(map[string]int{"raider": 10}, lv1(), d, 5, 0)
	d.Garrison = map[string]int{"guard": 6}
	held := Fight(map[string]int{"raider": 10}, lv1(), d, 5, 0)
	r := held.Rounds[0]
	if r.GarrisonDamage == 0 || r.GarrisonHit == 0 {
		t.Fatalf("the garrison fights back and takes the first blows: %+v", r)
	}
	if held.Destruction >= plain.Destruction || held.Losses["raider"] <= plain.Losses["raider"] {
		t.Fatalf("reinforcements make the city harder: %d%%/%d lost vs %d%%/%d lost", held.Destruction, held.Losses["raider"], plain.Destruction, plain.Losses["raider"])
	}
	// The garrison is not a building: it never counts towards destruction.
	if d.TowerHP+d.TownHallHP != 2400 {
		t.Fatal("setup")
	}
	if DefenseScore(d, nil) != int64(d.WallHP+d.TowerHP)+int64(6*Troops["guard"].HP) {
		t.Fatalf("lent troops still count for the fortress: %d", DefenseScore(d, nil))
	}
}

func TestWaveTrophies(t *testing.T) {
	cases := []struct {
		crowns []int
		want   []string
	}{
		{nil, nil},
		{[]int{0, 0}, nil},
		{[]int{1}, []string{"flags"}},
		{[]int{0, 3}, []string{"flags", "arch"}},
		{[]int{2, 2, 1, 2}, []string{"flags"}},
		{[]int{2, 3, 2}, []string{"flags", "arch", "fountain"}},
		{[]int{3, 0, 3, 1, 3}, []string{"flags", "arch", "statue"}},
	}
	for _, c := range cases {
		got := Trophies(c.crowns)
		for _, k := range RewardOrder {
			want := false
			for _, w := range c.want {
				want = want || w == k
			}
			if got[k] != want {
				t.Fatalf("%v: %s = %v, want %v", c.crowns, k, got[k], want)
			}
		}
	}
	for _, k := range RewardOrder {
		d := Buildings[k]
		if !d.Decor || d.Reward == "" || d.Levels[0].Cost != 0 {
			t.Fatalf("%s is free decor with a reason: %+v", k, d)
		}
	}
	if r := RewardLocked("arch", []int{2}); r == nil || r.Code != "reward_locked" {
		t.Fatalf("arch locked: %v", r)
	}
	if r := RewardLocked("arch", []int{3}); r != nil {
		t.Fatalf("arch open: %v", r)
	}
	if r := RewardLocked("road", nil); r != nil {
		t.Fatal("plain decor is never locked")
	}
}
