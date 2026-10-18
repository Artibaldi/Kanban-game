package app

import (
	"context"
	"encoding/json"
	"strings"
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
	// Test timers are a thousand times faster: keep the research running while we check it blocks the next one.
	if _, err := pool.Exec(ctx, `UPDATE troop_levels SET busy_until=now()+interval '1 hour' WHERE user_id=(SELECT id FROM users WHERE login='alexey')`); err != nil {
		t.Fatal(err)
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

// Building, upgrading, moving and finishing a trap never reaches the team feed: teammates are the only rivals.
func TestTrapsStayOutOfTheTeamFeed(t *testing.T) {
	ts, _, _ := testServer(t)
	alexey, maria := login(t, ts, "alexey"), login(t, ts, "maria")
	city := alexey.must(201, "POST", "/city/buildings", map[string]any{"type": "trap", "x": 2, "y": 13}, "").Body
	var id string
	for _, b := range city["buildings"].([]any) {
		if m := b.(map[string]any); m["type"] == "trap" {
			id = m["id"].(string)
		}
	}
	waitFor(t, "the trap is built", func() bool {
		for _, b := range alexey.must(200, "GET", "/city", nil, "").Body["buildings"].([]any) {
			if m := b.(map[string]any); m["id"] == id && m["busy_until"] == nil {
				return true
			}
		}
		return false
	})
	alexey.must(200, "POST", "/city/buildings/"+id+"/move", map[string]any{"x": 3, "y": 13}, "")
	seen := func(c *client) int {
		n := 0
		var feed struct{ Items []map[string]any }
		_ = json.Unmarshal([]byte(c.must(200, "GET", "/events?limit=200", nil, "").Raw), &feed)
		for _, e := range feed.Items {
			if strings.Contains(e["summary"].(string), "Ловушка") {
				n++
			}
		}
		return n
	}
	if seen(maria) != 0 {
		t.Fatal("a teammate learns about the trap from the feed")
	}
	if seen(alexey) < 3 {
		t.Fatalf("the owner sees their own trap events: %d", seen(alexey))
	}
}
