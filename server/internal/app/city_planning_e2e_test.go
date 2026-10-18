package app

import (
	"fmt"
	"testing"
)

type cell struct{ x, y int }

func cityCells(t *testing.T, c *client) (map[cell]map[string]any, map[string]any) {
	t.Helper()
	body := c.must(200, "GET", "/city", nil, "").Body
	out := map[cell]map[string]any{}
	for _, b := range body["buildings"].([]any) {
		m := b.(map[string]any)
		out[cell{int(num(m["x"])), int(num(m["y"]))}] = m
	}
	return out, body
}

func freeCell(t *testing.T, cells map[cell]map[string]any, start, size int) cell {
	t.Helper()
	for y := start; y < start+size; y++ {
		for x := start; x < start+size; x++ {
			if cells[cell{x, y}] == nil {
				return cell{x, y}
			}
		}
	}
	t.Fatal("no free cell")
	return cell{}
}

func TestCityPlanning(t *testing.T) {
	ts, _, _ := testServer(t)
	alexey, dmitry := login(t, ts, "alexey"), login(t, ts, "dmitry")

	// The island is 16×16; the old 12×12 cities sit in its centre and the land grows with the town hall.
	cells, body := cityCells(t, alexey)
	land := body["land"].(map[string]any)
	if num(land["start"]) != 1 || num(land["size"]) != 14 || num(body["grid"]) != 16 {
		t.Fatalf("town hall 3 land: %v grid %v", land, body["grid"])
	}
	if th := cells[cell{7, 7}]; th == nil || th["type"] != "townhall" {
		t.Fatalf("the town hall stands in the centre: %v", th)
	}
	_, dBody := cityCells(t, dmitry)
	if dl := dBody["land"].(map[string]any); num(dl["start"]) != 2 || num(dl["size"]) != 12 {
		t.Fatalf("town hall 1 land: %v", dl)
	}
	dmitry.refused("invalid_cell", "POST", "/city/buildings", map[string]any{"type": "road", "x": 1, "y": 1})

	// Move: free and instant, on your own land, onto a free cell.
	var tower map[string]any
	for _, b := range cells {
		if b["type"] == "tower" {
			tower = b
			break
		}
	}
	to := freeCell(t, cells, 1, 14)
	goldBefore := num(alexey.must(200, "GET", "/player", nil, "").Body["gold"])
	moved := alexey.must(200, "POST", "/city/buildings/"+tower["id"].(string)+"/move", map[string]any{"x": to.x, "y": to.y}, "").Body
	after, _ := cityCells(t, alexey)
	if after[to] == nil || after[to]["id"] != tower["id"] || moved["buildings"] == nil {
		t.Fatalf("moved to %v: %v", to, after[to])
	}
	if num(alexey.must(200, "GET", "/player", nil, "").Body["gold"]) != goldBefore {
		t.Fatal("moving is free")
	}
	alexey.refused("cell_occupied", "POST", "/city/buildings/"+tower["id"].(string)+"/move", map[string]any{"x": 7, "y": 7})
	alexey.refused("invalid_cell", "POST", "/city/buildings/"+tower["id"].(string)+"/move", map[string]any{"x": 0, "y": 0})
	dmitry.refused("not_found", "POST", "/city/buildings/"+tower["id"].(string)+"/move", map[string]any{"x": 2, "y": 2})

	// Decor: built at once without a builder, costs a little gold, can be removed (no refund).
	spot := freeCell(t, after, 1, 14)
	city := alexey.must(201, "POST", "/city/buildings", map[string]any{"type": "road", "x": spot.x, "y": spot.y}, "").Body
	var road map[string]any
	for _, b := range city["buildings"].([]any) {
		if m := b.(map[string]any); m["type"] == "road" {
			road = m
		}
	}
	if road == nil || num(road["level"]) != 1 || road["busy_until"] != nil {
		t.Fatalf("decor is ready at once: %v", road)
	}
	if g := num(alexey.must(200, "GET", "/player", nil, "").Body["gold"]); g != goldBefore-5 {
		t.Fatalf("a road costs 5 gold: %v → %v", goldBefore, g)
	}
	alexey.must(200, "DELETE", "/city/buildings/"+road["id"].(string), nil, "")
	final, _ := cityCells(t, alexey)
	if final[spot] != nil {
		t.Fatal("the road is gone")
	}
	alexey.refused("not_decor", "DELETE", "/city/buildings/"+tower["id"].(string), nil)
	_ = fmt.Sprint()
}
