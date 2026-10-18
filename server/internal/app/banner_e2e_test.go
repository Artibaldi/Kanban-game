package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func readyTroops(t *testing.T, c *client) map[string]any {
	t.Helper()
	return c.must(200, "GET", "/army", nil, "").Body["troops"].(map[string]any)
}

func TestBannerReinforcements(t *testing.T) {
	ts, pool, _ := testServer(t)
	admin, alexey, maria := login(t, ts, "admin"), login(t, ts, "alexey"), login(t, ts, "maria")
	mariaID, alexeyID := meID(t, maria), meID(t, alexey)

	alexey.refused("no_banner", "POST", "/banner/donate", map[string]any{"to_id": mariaID, "type": "guard", "count": 1})
	if _, err := pool.Exec(context.Background(), `INSERT INTO buildings(user_id,type,level,x,y) SELECT id,'banner',1,2,2 FROM users WHERE login='maria'`); err != nil {
		t.Fatal(err)
	}
	var mate map[string]any
	for _, m := range alexey.must(200, "GET", "/banner", nil, "").Body["team"].([]any) {
		if m.(map[string]any)["user_id"] == mariaID {
			mate = m.(map[string]any)
		}
	}
	if mate == nil || num(mate["capacity"]) != 6 || num(mate["held"]) != 0 {
		t.Fatalf("maria's banner in the team list: %v", mate)
	}

	guards := num(readyTroops(t, alexey)["guard"])
	alexey.must(201, "POST", "/banner/donate", map[string]any{"to_id": mariaID, "type": "guard", "count": 4}, "")
	if num(readyTroops(t, alexey)["guard"]) != guards-4 {
		t.Fatal("donated guards leave alexey's camp")
	}
	mine := maria.must(200, "GET", "/banner", nil, "").Body
	held := mine["reinforcements"].([]any)
	if num(mine["held"]) != 4 || len(held) != 1 || held[0].(map[string]any)["donor_id"] != alexeyID || num(held[0].(map[string]any)["count"]) != 4 {
		t.Fatalf("maria's banner: %v", mine)
	}
	// The team feed says who reinforced whom, never how many: no tallies of who gave what.
	var feed struct{ Items []map[string]any }
	_ = json.Unmarshal([]byte(maria.must(200, "GET", "/events?limit=50", nil, "").Raw), &feed)
	for _, e := range feed.Items {
		if sum := e["summary"].(string); strings.Contains(sum, "знамя") && strings.ContainsAny(sum, "×0123456789") {
			t.Fatalf("the feed counts troops: %q", sum)
		}
	}
	alexey.refused("banner_full", "POST", "/banner/donate", map[string]any{"to_id": mariaID, "type": "ram", "count": 2})
	alexey.refused("invalid_target", "POST", "/banner/donate", map[string]any{"to_id": alexeyID, "type": "guard", "count": 1})

	// The garrison defends Maria in a friendly duel; nobody dies, the guards stay on her banner.
	maria.must(200, "PATCH", "/me/settings", map[string]any{"accept_friendly": true}, "")
	duel := alexey.must(201, "POST", "/battles/friendly", map[string]any{"defender_id": mariaID, "army": map[string]int{"raider": 4}}, "").Body
	if g := duel["defense"].(map[string]any)["garrison"].(map[string]any); num(g["guard"]) != 4 {
		t.Fatalf("the garrison is in the battle: %v", duel["defense"])
	}
	if num(maria.must(200, "GET", "/banner", nil, "").Body["held"]) != 4 {
		t.Fatal("friendly duels cost the garrison nothing")
	}

	// Release: the troops go home.
	maria.must(200, "POST", "/banner/release", map[string]any{}, "")
	if num(readyTroops(t, alexey)["guard"]) != guards {
		t.Fatal("released guards are back in alexey's camp")
	}

	// Leaving the team sends lent troops home too.
	alexey.must(201, "POST", "/banner/donate", map[string]any{"to_id": mariaID, "type": "guard", "count": 2}, "")
	var mobile string
	for _, tm := range items(admin.must(200, "GET", "/admin/teams", nil, "")) {
		if tm["name"] == "Мобайл" {
			mobile = tm["id"].(string)
		}
	}
	admin.must(200, "PATCH", "/admin/users/"+alexeyID, map[string]any{"team_id": mobile}, "")
	if num(maria.must(200, "GET", "/banner", nil, "").Body["held"]) != 0 || num(readyTroops(t, alexey)["guard"]) != guards {
		t.Fatal("a colleague who moved teams gets the troops back")
	}
}

func TestRewardDecorNeedsTheTeamsCrowns(t *testing.T) {
	ts, pool, _ := testServer(t)
	alexey := login(t, ts, "alexey")
	// The seed's waves may already hold three crowns: start from a team that only ever won one.
	if _, err := pool.Exec(context.Background(), `UPDATE waves SET crowns=1 WHERE team_id=(SELECT team_id FROM users WHERE login='alexey') AND crowns IS NOT NULL`); err != nil {
		t.Fatal(err)
	}
	alexey.refused("reward_locked", "POST", "/city/buildings", map[string]any{"type": "arch", "x": 2, "y": 13})
	city := alexey.must(200, "GET", "/city", nil, "").Body
	if r := city["rewards"].(map[string]any); r["arch"] != false || r["flags"] != true {
		t.Fatalf("the arch is still locked: %v", r)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE waves SET crowns=3, settled_at=now() WHERE id=(SELECT id FROM waves WHERE team_id=(SELECT team_id FROM users WHERE login='alexey') ORDER BY starts_at LIMIT 1)`); err != nil {
		t.Fatal(err)
	}
	gold := num(alexey.must(200, "GET", "/player", nil, "").Body["gold"])
	alexey.must(201, "POST", "/city/buildings", map[string]any{"type": "arch", "x": 2, "y": 13}, "")
	if num(alexey.must(200, "GET", "/player", nil, "").Body["gold"]) != gold {
		t.Fatal("reward decor is free")
	}
}

// Sending and releasing at the same time must queue up, never deadlock (a deadlock surfaces as 409 «busy»).
func TestDonateAndReleaseRace(t *testing.T) {
	ts, pool, _ := testServer(t)
	alexey, maria, dmitry := login(t, ts, "alexey"), login(t, ts, "maria"), login(t, ts, "dmitry")
	mariaID := meID(t, maria)
	if _, err := pool.Exec(context.Background(), `INSERT INTO buildings(user_id,type,level,x,y) SELECT id,'banner',3,2,2 FROM users WHERE login='maria';
	 UPDATE troops SET count=count+200 WHERE type='guard' AND user_id IN (SELECT id FROM users WHERE login IN ('alexey','dmitry'))`); err != nil {
		t.Fatal(err)
	}
	errs := make(chan string, 200)
	done := make(chan bool)
	for _, c := range []*client{alexey, dmitry} {
		go func(c *client) {
			for i := 0; i < 25; i++ {
				if r := c.do("POST", "/banner/donate", map[string]any{"to_id": mariaID, "type": "guard", "count": 1}, ""); r.Status >= 500 || r.Body["code"] == "busy" {
					errs <- r.Raw
				}
			}
			done <- true
		}(c)
	}
	go func() {
		for i := 0; i < 25; i++ {
			if r := maria.do("POST", "/banner/release", map[string]any{}, ""); r.Status >= 500 || r.Body["code"] == "busy" {
				errs <- r.Raw
			}
		}
		done <- true
	}()
	for i := 0; i < 3; i++ {
		<-done
	}
	close(errs)
	for e := range errs {
		t.Fatalf("server error under concurrency: %s", e)
	}
}
