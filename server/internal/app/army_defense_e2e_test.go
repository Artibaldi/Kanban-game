package app

import (
	"context"
	"testing"
)

func TestWorkshopArchersAndHiddenTraps(t *testing.T) {
	ts, pool, _ := testServer(t)
	alexey, maria := login(t, ts, "alexey"), login(t, ts, "maria")
	ctx := context.Background()

	// Archers need an archery range; the army says so before anyone tries.
	if l := alexey.must(200, "GET", "/army", nil, "").Body["locked"].(map[string]any); l["archer"] == nil || l["guard"] != nil {
		t.Fatalf("locked: %v", l)
	}
	alexey.refused("no_archery", "POST", "/army/train", map[string]any{"type": "archer", "count": 2})
	alexey.refused("no_workshop", "POST", "/army/research", map[string]any{"type": "guard"})
	if _, err := pool.Exec(ctx, `INSERT INTO buildings(user_id,type,level,x,y) SELECT id,'archery',1,1,1 FROM users WHERE login='alexey'
	 UNION ALL SELECT id,'workshop',1,2,1 FROM users WHERE login='alexey'`); err != nil {
		t.Fatal(err)
	}
	alexey.must(201, "POST", "/army/train", map[string]any{"type": "archer", "count": 2}, "")

	// Research: gold now, the level when the timer runs out; one at a time.
	army := alexey.must(201, "POST", "/army/research", map[string]any{"type": "guard"}, "").Body
	res := army["research"].(map[string]any)
	if res["type"] != "guard" || num(res["target_level"]) != 2 || num(army["max_troop_level"]) != 2 {
		t.Fatalf("research: %v max %v", res, army["max_troop_level"])
	}
	alexey.refused("workshop_busy", "POST", "/army/research", map[string]any{"type": "raider"})
	if _, err := pool.Exec(ctx, `UPDATE troop_levels SET busy_until=now()-interval '1 second' WHERE user_id=(SELECT id FROM users WHERE login='alexey')`); err != nil {
		t.Fatal(err)
	}
	army = alexey.must(200, "GET", "/army", nil, "").Body
	if num(army["levels"].(map[string]any)["guard"]) != 2 || army["research"] != nil {
		t.Fatalf("guards reach level 2: %v %v", army["levels"], army["research"])
	}
	alexey.refused("workshop_level", "POST", "/army/research", map[string]any{"type": "guard"})

	// Maria's traps are invisible to a rival before the battle.
	if _, err := pool.Exec(ctx, `INSERT INTO buildings(user_id,type,level,x,y) SELECT id,'trap',1,3,3 FROM users WHERE login='maria'`); err != nil {
		t.Fatal(err)
	}
	maria.must(200, "PATCH", "/me/settings", map[string]any{"accept_friendly": true}, "")
	var mariaID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM users WHERE login='maria'`).Scan(&mariaID); err != nil {
		t.Fatal(err)
	}
	preview := alexey.must(200, "GET", "/duels/"+mariaID+"/city", nil, "").Body
	for _, b := range preview["buildings"].([]any) {
		if b.(map[string]any)["type"] == "trap" {
			t.Fatal("the trap shows in a rival's preview")
		}
	}
	if d := preview["opponent"].(map[string]any)["defense"].(map[string]any); d["traps"] != nil || d["trap_dmg"] != nil {
		t.Fatalf("the defence gives the trap away: %v", d)
	}
	// Her own city shows it.
	own := maria.must(200, "GET", "/city", nil, "").Body
	found := false
	for _, b := range own["buildings"].([]any) {
		found = found || b.(map[string]any)["type"] == "trap"
	}
	if !found {
		t.Fatal("the owner sees her trap")
	}
}
