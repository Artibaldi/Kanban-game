package game

import (
	"testing"
	"time"
)

func TestLandGrowsWithTheTownHall(t *testing.T) {
	if MaxGrid != 16 {
		t.Fatalf("grid: %d", MaxGrid)
	}
	for th, want := range map[int][2]int{0: {2, 12}, 1: {2, 12}, 2: {2, 12}, 3: {1, 14}, 4: {0, 16}, 5: {0, 16}} {
		if start, size := Land(th); start != want[0] || size != want[1] {
			t.Errorf("town hall %d: %d,%d want %v", th, start, size, want)
		}
	}
	c := city(Building{Type: "townhall", Level: 1, X: 7, Y: 7})
	if _, r := CheckPlace(c, "wall", 1, 7, 1000); r == nil || r.Code != "invalid_cell" {
		t.Fatalf("outside the land of town hall 1: %v", r)
	}
	if _, r := CheckPlace(c, "wall", 2, 13, 1000); r != nil {
		t.Fatalf("the edge of the land: %v", r)
	}
	big := city(Building{Type: "townhall", Level: 4, X: 7, Y: 7})
	if _, r := CheckPlace(big, "wall", 0, 15, 5000); r != nil {
		t.Fatalf("town hall 4 owns the whole island: %v", r)
	}
}

func TestDecorIsLooksOnly(t *testing.T) {
	for _, k := range []string{"road", "tree", "flowers", "lantern", "pond"} {
		d, ok := Buildings[k]
		if !ok || !d.Decor || len(d.Levels) != 1 || d.Levels[0].HP != 0 || d.Levels[0].Seconds != 0 || d.Levels[0].Cost <= 0 {
			t.Fatalf("%s: %+v", k, d)
		}
	}
	lvl, until := 2, time.Now()
	busy := city(Building{Type: "townhall", Level: 1, X: 7, Y: 7},
		Building{Type: "wall", X: 3, Y: 3, TargetLevel: &lvl, BusyUntil: &until}, Building{Type: "wall", X: 4, Y: 3, TargetLevel: &lvl, BusyUntil: &until})
	if _, r := CheckPlace(busy, "road", 5, 5, 100); r != nil {
		t.Fatalf("decor needs no builder: %v", r)
	}
	plain := DefenseOf(city(Building{Type: "townhall", Level: 1, X: 7, Y: 7}))
	decorated := DefenseOf(city(Building{Type: "townhall", Level: 1, X: 7, Y: 7}, Building{Type: "pond", Level: 1, X: 3, Y: 3}, Building{Type: "road", Level: 1, X: 4, Y: 3}))
	if plain != decorated {
		t.Fatalf("decor never changes the defence: %+v vs %+v", plain, decorated)
	}
}

func TestMoveRules(t *testing.T) {
	th := Building{ID: "th", Type: "townhall", Level: 1, X: 7, Y: 7}
	tw := Building{ID: "tw", Type: "tower", Level: 1, X: 5, Y: 5}
	c := city(th, tw)
	if r := CheckMove(c, tw, 7, 7); r == nil || r.Code != "cell_occupied" {
		t.Fatalf("occupied: %v", r)
	}
	if r := CheckMove(c, tw, 14, 5); r == nil || r.Code != "invalid_cell" {
		t.Fatalf("outside the land: %v", r)
	}
	if r := CheckMove(c, th, 3, 3); r != nil {
		t.Fatalf("even the town hall moves: %v", r)
	}
	if r := CheckMove(c, tw, 5, 5); r == nil || r.Code != "same_cell" {
		t.Fatalf("same cell: %v", r)
	}
}
