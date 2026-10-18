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
	if len(city["buildings"].([]any)) == 0 {
		t.Fatalf("challenge city: %v", city)
	}
	b := alexey.must(201, "POST", "/challenges/"+id+"/attack", map[string]any{"army": map[string]int{"guard": 1}}, "").Body
	if b["kind"] != "friendly" || b["defender_id"] != dmitryID || b["challenge_id"] != id {
		t.Fatalf("battle: %v", b)
	}
	alexey.refused("challenge_closed", "POST", "/challenges/"+id+"/attack", map[string]any{"army": map[string]int{"guard": 1}})

	// The defender sees what happened to the city, with the way back to answer.
	mine := dmitry.must(200, "GET", "/challenges", nil, "").Body
	if len(mine["against_me"].([]any)) != 1 || mine["against_me"].([]any)[0].(map[string]any)["id"] != b["id"] {
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
	// A declined challenge does not block a new one later.
	again := maria.must(201, "POST", "/challenges", map[string]any{"to_id": alexeyID}, "").Body["id"].(string)
	maria.must(200, "POST", "/challenges/"+again+"/cancel", nil, "")
	alexey.refused("challenge_closed", "POST", "/challenges/"+again+"/accept", nil)

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
	if n := len(alexey.must(200, "GET", "/challenges", nil, "").Body["team"].([]any)); n < 2 {
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
