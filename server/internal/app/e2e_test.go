package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kanban-game/server/db"
)

// The end-to-end test runs the real HTTP API against a throwaway PostgreSQL database created next to DATABASE_URL.
// It is skipped when DATABASE_URL is not set. Run: DATABASE_URL=postgres://kanban:kanban-local-only@127.0.0.1:15433/kanban?sslmode=disable go test ./...

func testServer(t *testing.T) (*httptest.Server, *pgxpool.Pool, *Server) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	name := "kanban_e2e_" + hex.EncodeToString(b)
	if _, err = admin.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(dsn)
	u.Path = "/" + name
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), `DROP DATABASE `+name+` WITH (FORCE)`)
		admin.Close()
	})
	if err = db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// Fast timers for the test: every build and training time × 0.001.
	if _, err = pool.Exec(ctx, `INSERT INTO settings(key,value) VALUES('economy','{"time_scale":0.001}')`); err != nil {
		t.Fatal(err)
	}
	var h http.Handler
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeHTTP(w, r) }))
	s := New(pool, Config{Origins: []string{ts.URL}})
	h = s.Routes()
	if _, err = s.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	tctx, cancel := context.WithCancel(ctx)
	go s.RunTimers(tctx)
	t.Cleanup(func() { cancel(); ts.Close() })
	return ts, pool, s
}

type client struct {
	t    *testing.T
	base string
	http *http.Client
	csrf string
}

type resp struct {
	Status int
	Replay bool
	Body   map[string]any
	Raw    string
}

func login(t *testing.T, ts *httptest.Server, who string) *client {
	jar, _ := cookiejar.New(nil)
	c := &client{t: t, base: ts.URL + "/api/v1", http: &http.Client{Jar: jar}}
	r := c.do("GET", "/csrf", nil, "")
	c.csrf = r.Body["csrf_token"].(string)
	r = c.do("POST", "/sessions", map[string]any{"login": who}, "")
	if r.Status != 200 {
		t.Fatalf("login %s: %d %s", who, r.Status, r.Raw)
	}
	c.csrf = r.Body["csrf_token"].(string)
	return c
}

func key() string { b := make([]byte, 12); _, _ = rand.Read(b); return hex.EncodeToString(b) }

func (c *client) do(method, path string, body any, idem string) resp {
	c.t.Helper()
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, c.base+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != "GET" {
		req.Header.Set("Origin", strings.TrimSuffix(c.base, "/api/v1"))
		req.Header.Set("X-CSRF-Token", c.csrf)
		if idem == "" {
			idem = key()
		}
		if path != "/sessions" {
			req.Header.Set("Idempotency-Key", idem)
		}
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := resp{Status: res.StatusCode, Replay: res.Header.Get("Idempotent-Replay") == "true", Raw: string(raw)}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

func (c *client) must(status int, method, path string, body any, idem string) resp {
	c.t.Helper()
	r := c.do(method, path, body, idem)
	if r.Status != status {
		c.t.Fatalf("%s %s: want %d, got %d %s", method, path, status, r.Status, r.Raw)
	}
	return r
}

func (c *client) refused(code string, method, path string, body any) {
	c.t.Helper()
	r := c.do(method, path, body, "")
	if r.Status < 400 || r.Body["code"] != code {
		c.t.Fatalf("%s %s: want refusal %q, got %d %s", method, path, code, r.Status, r.Raw)
	}
}

func items(r resp) []map[string]any {
	out := []map[string]any{}
	for _, v := range r.Body["items"].([]any) {
		out = append(out, v.(map[string]any))
	}
	return out
}

func findTask(t *testing.T, c *client, title string) map[string]any {
	for _, it := range items(c.must(200, "GET", "/tasks", nil, "")) {
		if it["title"] == title {
			return it
		}
	}
	t.Fatalf("task %q not found", title)
	return nil
}

func weekPoints(t *testing.T, c *client, login string) float64 {
	for _, it := range items(c.must(200, "GET", "/ratings/work?period=week", nil, "")) {
		if strings.HasPrefix(strings.ToLower(it["display_name"].(string)), login) {
			return it["points"].(float64)
		}
	}
	t.Fatalf("no rating row for %s", login)
	return 0
}

func num(v any) float64 { return v.(float64) }

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for i := 0; i < 40; i++ {
		if cond() {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestEndToEnd(t *testing.T) {
	ts, pool, _ := testServer(t)
	ctx := context.Background()
	alexey := login(t, ts, "alexey")
	irina := login(t, ts, "irina")
	anna := login(t, ts, "anna")

	// 1. Take a ready, unassigned task into work: it gets assigned; then send it to review.
	task := findTask(t, alexey, "Валидация ИНН при регистрации")
	id := task["id"].(string)
	r := alexey.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "in_progress"}, "")
	if r.Body["task"].(map[string]any)["assignee_name"] != "Алексей Петров" {
		t.Fatalf("take into work must assign: %s", r.Raw)
	}
	alexey.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "review"}, "")

	// A member cannot confirm; a member of another team cannot even see the task.
	alexey.refused("forbidden", "POST", "/tasks/"+id+"/transition", map[string]any{"to": "done"})
	anna.refused("not_found", "GET", "/tasks/"+id, nil)

	before := alexey.must(200, "GET", "/player", nil, "").Body
	weekBefore := weekPoints(t, irina, "алексей")

	// 2. The lead confirms. The same request repeated is a replay; a new request is refused; nothing is paid twice.
	k := key()
	first := irina.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "done"}, k)
	reward := first.Body["reward"].(map[string]any)
	if num(reward["xp"]) != 50 || num(reward["energy"]) != 20 {
		t.Fatalf("2 points reward: %s", first.Raw)
	}
	again := irina.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "done"}, k)
	if !again.Replay || again.Raw != first.Raw {
		t.Fatalf("repeated key must replay: %v %s", again.Replay, again.Raw)
	}
	// A new request to move a done task into «Выполнено» only reorders the column: no second reward.
	if r := irina.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "done"}, ""); r.Body["reward"] != nil {
		t.Fatalf("second move to done must not pay: %s", r.Raw)
	}
	if r := irina.do("POST", "/tasks/"+id+"/transition", map[string]any{"to": "review"}, k); r.Body["code"] != "idempotency_conflict" {
		t.Fatalf("same key, other body: %s", r.Raw)
	}
	var rewards int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ledger WHERE task_id=$1 AND kind='task_reward'`, id).Scan(&rewards); err != nil || rewards != 1 {
		t.Fatalf("exactly one reward row, got %d %v", rewards, err)
	}
	after := alexey.must(200, "GET", "/player", nil, "").Body
	if num(after["xp"])-num(before["xp"]) != 50 || num(after["gold"])-num(before["gold"]) != num(reward["gold"]) {
		t.Fatalf("balance moved once: before %v after %v reward %v", before, after, reward)
	}
	if got := weekPoints(t, irina, "алексей"); got != weekBefore+2 {
		t.Fatalf("work rating +2: %v → %v", weekBefore, got)
	}

	// 3. Complexity is locked after confirmation; self-confirmation is refused.
	irina.refused("points_locked", "PATCH", "/tasks/"+id, map[string]any{"points": 8})
	own := findTask(t, irina, "Лимиты подключений pgbouncer")
	irina.refused("self_approval", "POST", "/tasks/"+own["id"].(string)+"/transition", map[string]any{"to": "done"})
	// A member cannot start a task whose estimate is not agreed.
	dmitry := login(t, ts, "dmitry")
	unagreed := findTask(t, dmitry, "Шаблоны писем")
	dmitry.refused("estimate_not_agreed", "POST", "/tasks/"+unagreed["id"].(string)+"/transition", map[string]any{"to": "in_progress"})

	// 4. City: build a wall, wait for the server timer, upgrade it; a player without gold is refused.
	city := alexey.must(201, "POST", "/city/buildings", map[string]any{"type": "wall", "x": 0, "y": 0}, "")
	var wallID string
	for _, b := range city.Body["buildings"].([]any) {
		bm := b.(map[string]any)
		if bm["type"] == "wall" && num(bm["x"]) == 0 && num(bm["y"]) == 0 {
			wallID = bm["id"].(string)
		}
	}
	alexey.refused("cell_occupied", "POST", "/city/buildings", map[string]any{"type": "wall", "x": 0, "y": 0})
	waitFor(t, "wall built", func() bool {
		var lvl int
		_ = pool.QueryRow(ctx, `SELECT level FROM buildings WHERE id=$1 AND busy_until IS NULL`, wallID).Scan(&lvl)
		return lvl == 1
	})
	alexey.must(200, "POST", "/city/buildings/"+wallID+"/upgrade", map[string]any{}, "")
	if _, err := pool.Exec(ctx, `UPDATE players SET gold=10 WHERE user_id=(SELECT id FROM users WHERE login='dmitry')`); err != nil {
		t.Fatal(err)
	}
	dmitry.refused("insufficient_gold", "POST", "/city/buildings", map[string]any{"type": "wall", "x": 0, "y": 0})

	// 5. Army and a friendly duel: no energy, no losses, no trophies; a daily limit; only colleagues who accept duels.
	alexey.must(201, "POST", "/army/train", map[string]any{"type": "guard", "count": 2}, "")
	waitFor(t, "training", func() bool {
		a := alexey.must(200, "GET", "/army", nil, "").Body
		return num(a["troops"].(map[string]any)["guard"]) == 12
	})
	var maria string
	for _, o := range items(alexey.must(200, "GET", "/duels", nil, "")) {
		if o["display_name"] == "Мария Иванова" {
			maria = o["user_id"].(string)
		}
	}
	if maria == "" {
		t.Fatal("maria accepts friendly duels in the seed")
	}
	alexey.must(200, "GET", "/duels/"+maria+"/city", nil, "")
	workBefore := weekPoints(t, irina, "алексей")
	var duelWorkBefore int
	_ = pool.QueryRow(ctx, `SELECT COALESCE(SUM(points),0) FROM work_points_events WHERE user_id=(SELECT id FROM users WHERE login='alexey')`).Scan(&duelWorkBefore)
	pBefore := alexey.must(200, "GET", "/player", nil, "").Body
	armyBefore := alexey.must(200, "GET", "/army", nil, "").Body["troops"].(map[string]any)
	duel := alexey.must(201, "POST", "/battles/friendly", map[string]any{"defender_id": maria, "army": map[string]int{"guard": 8, "raider": 6, "ram": 3}}, "")
	if duel.Body["kind"] != "friendly" || len(duel.Body["rounds"].([]any)) == 0 {
		t.Fatalf("duel: %s", duel.Raw)
	}
	pAfter := alexey.must(200, "GET", "/player", nil, "").Body
	if num(pAfter["energy"]) != num(pBefore["energy"]) || num(pAfter["trophies"]) != num(pBefore["trophies"]) || num(pAfter["gold"]) != num(pBefore["gold"]) {
		t.Fatalf("a friendly duel is free: %v → %v", pBefore, pAfter)
	}
	if got := alexey.must(200, "GET", "/army", nil, "").Body["troops"].(map[string]any); num(got["guard"]) != num(armyBefore["guard"]) || num(got["ram"]) != num(armyBefore["ram"]) {
		t.Fatalf("no losses: %v → %v", armyBefore, got)
	}
	for i := 0; i < 2; i++ {
		alexey.must(201, "POST", "/battles/friendly", map[string]any{"defender_id": maria, "army": map[string]int{"guard": 1}}, "")
	}
	alexey.refused("friendly_limit", "POST", "/battles/friendly", map[string]any{"defender_id": maria, "army": map[string]int{"guard": 1}})
	dmitryID := dmitry.must(200, "GET", "/me", nil, "").Body["id"].(string)
	alexey.refused("not_found", "POST", "/battles/friendly", map[string]any{"defender_id": dmitryID, "army": map[string]int{"guard": 1}}) // does not accept duels
	alexey.must(404, "GET", "/opponents", nil, "")
	var duelWorkAfter int
	_ = pool.QueryRow(ctx, `SELECT COALESCE(SUM(points),0) FROM work_points_events WHERE user_id=(SELECT id FROM users WHERE login='alexey')`).Scan(&duelWorkAfter)
	if duelWorkAfter != duelWorkBefore {
		t.Fatalf("duels must not change the work rating: %v → %v", duelWorkBefore, duelWorkAfter)
	}
	var mariaBuildings int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM buildings WHERE user_id=$1`, maria).Scan(&mariaBuildings)
	if mariaBuildings == 0 {
		t.Fatal("defender city must stay intact")
	}

	// 6. Reopen a confirmed task after the reward was spent: reversal row, rating down, debt blocks spending.
	irina.refused("comment_required", "POST", "/tasks/"+id+"/transition", map[string]any{"to": "in_progress"})
	if _, err := pool.Exec(ctx, `UPDATE players SET gold=10 WHERE user_id=(SELECT id FROM users WHERE login='alexey')`); err != nil {
		t.Fatal(err)
	}
	rv := irina.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "in_progress", "comment": "Нет проверки для ИП"}, "")
	if rv.Body["reversal"].(map[string]any)["debt"] != true {
		t.Fatalf("spent reward must leave a debt: %s", rv.Raw)
	}
	var reversals int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM ledger WHERE task_id=$1 AND kind='task_reward_reversal' AND reverses_id IS NOT NULL`, id).Scan(&reversals)
	if reversals != 1 {
		t.Fatalf("one reversal row, got %d", reversals)
	}
	if got := weekPoints(t, irina, "алексей"); got != workBefore-2 {
		t.Fatalf("reopen subtracts points: %v → %v", workBefore, got)
	}
	alexey.refused("debt", "POST", "/city/buildings", map[string]any{"type": "wall", "x": 11, "y": 11})

	// 7. The task can be confirmed again later and pays again exactly once (the old reward stays reversed).
	alexey.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "review"}, "")
	irina.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "done"}, "")
	var live int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM ledger WHERE task_id=$1 AND kind='task_reward' AND NOT reversed`, id).Scan(&live)
	if live != 1 {
		t.Fatalf("one live reward after re-confirmation, got %d", live)
	}
	fmt.Println("end-to-end scenario passed")
}
