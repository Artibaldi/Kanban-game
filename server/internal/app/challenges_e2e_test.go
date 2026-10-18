package app

import (
	"context"
	"testing"
)

func meID(t *testing.T, c *client) string {
	t.Helper()
	return c.must(200, "GET", "/me", nil, "").Body["id"].(string)
}

func challengeIn(list []any, id string) map[string]any {
	for _, x := range list {
		if m := x.(map[string]any); m["id"] == id {
			return m
		}
	}
	return nil
}

// A challenge is the consent: a colleague who does not accept duels in general can still be fought
// once they accept a challenge, and only by the one who challenged them.
func TestDirectChallenge(t *testing.T) {
	ts, _, _ := testServer(t)
	alexey, dmitry, maria, sergey := login(t, ts, "alexey"), login(t, ts, "dmitry"), login(t, ts, "maria"), login(t, ts, "sergey")
	dmitryID, alexeyID := meID(t, dmitry), meID(t, alexey)

	ch := alexey.must(201, "POST", "/challenges", map[string]any{"to_id": dmitryID}, "").Body
	id := ch["id"].(string)
	if ch["status"] != "pending" || ch["open"] != false {
		t.Fatalf("new challenge: %v", ch)
	}
	alexey.refused("challenge_exists", "POST", "/challenges", map[string]any{"to_id": dmitryID})
	alexey.refused("invalid_opponent", "POST", "/challenges", map[string]any{"to_id": alexeyID})
	alexey.refused("not_found", "POST", "/challenges", map[string]any{"to_id": meID(t, sergey)}) // another team

	// Before it is accepted the city is closed and there is nothing to attack.
	alexey.refused("challenge_not_accepted", "GET", "/challenges/"+id+"/city", nil)
	alexey.refused("challenge_not_accepted", "POST", "/challenges/"+id+"/attack", map[string]any{"army": map[string]int{"guard": 1}})
	alexey.refused("forbidden", "POST", "/challenges/"+id+"/accept", nil)
	maria.refused("forbidden", "POST", "/challenges/"+id+"/accept", nil)

	in := dmitry.must(200, "GET", "/challenges", nil, "").Body
	got := challengeIn(in["incoming"].([]any), id)
	if got == nil || got["from_name"] != "Алексей Петров" {
		t.Fatalf("incoming: %v", in["incoming"])
	}
	if dmitry.must(200, "POST", "/challenges/"+id+"/accept", nil, "").Body["status"] != "accepted" {
		t.Fatal("accepted")
	}
	city := alexey.must(200, "GET", "/challenges/"+id+"/city", nil, "").Body
	if len(city["buildings"].([]any)) == 0 || city["opponent_id"] != dmitryID || city["defense"] == nil {
		t.Fatalf("challenge city: %v", city)
	}
	b := alexey.must(201, "POST", "/challenges/"+id+"/attack", map[string]any{"army": map[string]int{"guard": 1}}, "").Body
	if b["kind"] != "friendly" || b["defender_id"] != dmitryID || b["challenge_id"] != id {
		t.Fatalf("battle: %v", b)
	}
	alexey.refused("challenge_closed", "POST", "/challenges/"+id+"/attack", map[string]any{"army": map[string]int{"guard": 1}})
	// Both sides of a battle can see the defender's city for the replay; nobody else can.
	if len(alexey.must(200, "GET", "/battles/"+b["id"].(string)+"/city", nil, "").Body["buildings"].([]any)) == 0 {
		t.Fatal("battle city for the attacker")
	}
	dmitry.must(200, "GET", "/battles/"+b["id"].(string)+"/city", nil, "")
	maria.refused("not_found", "GET", "/battles/"+b["id"].(string)+"/city", nil)

	// The defender sees what happened to the city, with the way back to answer.
	mine := dmitry.must(200, "GET", "/challenges", nil, "").Body
	if am := mine["against_me"].([]any); len(am) != 1 || am[0].(map[string]any)["id"] != b["id"] || am[0].(map[string]any)["attacker_id"] != alexeyID {
		t.Fatalf("against_me: %v", mine["against_me"])
	}
	if challengeIn(mine["incoming"].([]any), id) != nil {
		t.Fatal("a used challenge leaves the incoming list")
	}
}

func TestDeclineCancelAndExpiry(t *testing.T) {
	ts, pool, _ := testServer(t)
	alexey, maria, dmitry := login(t, ts, "alexey"), login(t, ts, "maria"), login(t, ts, "dmitry")
	alexeyID, dmitryID := meID(t, alexey), meID(t, dmitry)

	id := maria.must(201, "POST", "/challenges", map[string]any{"to_id": alexeyID}, "").Body["id"].(string)
	if alexey.must(200, "POST", "/challenges/"+id+"/decline", nil, "").Body["status"] != "declined" {
		t.Fatal("declined")
	}
	maria.refused("challenge_closed", "POST", "/challenges/"+id+"/attack", map[string]any{"army": map[string]int{"guard": 1}})
	// «Не сейчас» is respected: the same colleague cannot be challenged again right away.
	maria.refused("challenge_cooldown", "POST", "/challenges", map[string]any{"to_id": alexeyID})
	again := maria.must(201, "POST", "/challenges", map[string]any{"to_id": dmitryID}, "").Body["id"].(string)
	maria.must(200, "POST", "/challenges/"+again+"/cancel", nil, "")
	dmitry.refused("challenge_closed", "POST", "/challenges/"+again+"/accept", nil)
	if _, err := pool.Exec(context.Background(), `UPDATE friendly_challenges SET created_at=now()-interval '13 hours' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	maria.must(201, "POST", "/challenges", map[string]any{"to_id": alexeyID}, "")

	old := alexey.must(201, "POST", "/challenges", map[string]any{"to_id": dmitryID}, "").Body["id"].(string)
	if _, err := pool.Exec(context.Background(), `UPDATE friendly_challenges SET expires_at=now()-interval '1 minute' WHERE id=$1`, old); err != nil {
		t.Fatal(err)
	}
	dmitry.refused("challenge_closed", "POST", "/challenges/"+old+"/accept", nil)
	if challengeIn(dmitry.must(200, "GET", "/challenges", nil, "").Body["incoming"].([]any), old) != nil {
		t.Fatal("an expired challenge is gone")
	}
	alexey.must(201, "POST", "/challenges", map[string]any{"to_id": dmitryID}, "")
}

// An open challenge: posting your city is the consent; every colleague may attack it once.
func TestOpenChallengeAndTavernLimit(t *testing.T) {
	ts, pool, _ := testServer(t)
	alexey, maria, dmitry := login(t, ts, "alexey"), login(t, ts, "maria"), login(t, ts, "dmitry")
	ch := dmitry.must(201, "POST", "/challenges", map[string]any{}, "").Body
	id := ch["id"].(string)
	if ch["status"] != "open" || ch["open"] != true {
		t.Fatalf("open: %v", ch)
	}
	dmitry.refused("challenge_exists", "POST", "/challenges", map[string]any{})
	list := alexey.must(200, "GET", "/challenges", nil, "").Body
	o := challengeIn(list["open"].([]any), id)
	if o == nil || o["attacked"] != false {
		t.Fatalf("open list: %v", list["open"])
	}
	alexey.must(200, "GET", "/challenges/"+id+"/city", nil, "")
	alexey.must(201, "POST", "/challenges/"+id+"/attack", map[string]any{"army": map[string]int{"guard": 1}}, "")
	alexey.refused("already_attacked", "POST", "/challenges/"+id+"/attack", map[string]any{"army": map[string]int{"guard": 1}})
	maria.must(201, "POST", "/challenges/"+id+"/attack", map[string]any{"army": map[string]int{"guard": 1}}, "")
	dmitry.refused("invalid_opponent", "POST", "/challenges/"+id+"/attack", map[string]any{"army": map[string]int{"guard": 1}})
	if challengeIn(alexey.must(200, "GET", "/challenges", nil, "").Body["open"].([]any), id)["attacked"] != true {
		t.Fatal("attacked flag")
	}
	// The team list shows every colleague, with or without duels switched on.
	if n := len(alexey.must(200, "GET", "/challenges", nil, "").Body["team"].([]any)); n != 0 {
		t.Fatalf("the badge poll does not load the team: %d", n)
	}
	if n := len(alexey.must(200, "GET", "/challenges?team=1", nil, "").Body["team"].([]any)); n < 2 {
		t.Fatalf("team: %d", n)
	}

	// The daily limit grows with the tavern: 3 → 4 → 5.
	if num(alexey.must(200, "GET", "/challenges", nil, "").Body["per_day"]) != 3 {
		t.Fatal("no tavern: 3")
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO buildings(user_id,type,level,x,y) SELECT id,'tavern',2,0,11 FROM users WHERE login='alexey'`); err != nil {
		t.Fatal(err)
	}
	if num(alexey.must(200, "GET", "/challenges", nil, "").Body["per_day"]) != 5 {
		t.Fatal("tavern 2: 5")
	}
}

// Challenges follow the team as it is now: a colleague moved away or switched off can no longer be fought,
// and a battle's city is visible only while the defender is in your team.
func TestChallengesFollowTheTeam(t *testing.T) {
	ts, _, _ := testServer(t)
	admin, alexey, dmitry, maria := login(t, ts, "admin"), login(t, ts, "alexey"), login(t, ts, "dmitry"), login(t, ts, "maria")
	dmitryID := meID(t, dmitry)
	direct := alexey.must(201, "POST", "/challenges", map[string]any{"to_id": dmitryID}, "").Body["id"].(string)
	dmitry.must(200, "POST", "/challenges/"+direct+"/accept", nil, "")
	open := dmitry.must(201, "POST", "/challenges", map[string]any{}, "").Body["id"].(string)
	b := maria.must(201, "POST", "/challenges/"+open+"/attack", map[string]any{"army": map[string]int{"guard": 1}}, "").Body
	maria.must(200, "GET", "/battles/"+b["id"].(string)+"/city", nil, "")

	var mobile string
	for _, tm := range items(admin.must(200, "GET", "/admin/teams", nil, "")) {
		if tm["name"] == "Мобайл" {
			mobile = tm["id"].(string)
		}
	}
	admin.must(200, "PATCH", "/admin/users/"+dmitryID, map[string]any{"team_id": mobile}, "")

	alexey.refused("challenge_closed", "POST", "/challenges/"+direct+"/attack", map[string]any{"army": map[string]int{"guard": 1}})
	alexey.refused("challenge_closed", "GET", "/challenges/"+direct+"/city", nil)
	if c := challengeIn(alexey.must(200, "GET", "/challenges", nil, "").Body["outgoing"].([]any), direct); c != nil && (c["status"] == "accepted" || c["status"] == "pending") {
		t.Fatalf("a moved colleague's challenge is closed: %v", c)
	}
	maria.refused("not_found", "GET", "/battles/"+b["id"].(string)+"/city", nil)
	if len(dmitry.must(200, "GET", "/challenges", nil, "").Body["incoming"].([]any)) != 0 {
		t.Fatal("old team's challenges do not follow into the new team")
	}
}
