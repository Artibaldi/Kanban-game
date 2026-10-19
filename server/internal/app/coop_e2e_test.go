package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCoopSchema(t *testing.T) {
	_, pool, _ := testServer(t)
	ctx := context.Background()
	var hp, weeks int
	if err := pool.QueryRow(ctx, `SELECT f.hp, t.wave_weeks FROM fortresses f JOIN teams t ON t.id=f.team_id WHERE t.name='Платформа'`).Scan(&hp, &weeks); err != nil {
		t.Fatal(err)
	}
	if weeks != 1 {
		t.Fatalf("default wave length: %d", weeks)
	}
	var trophies int
	_ = pool.QueryRow(ctx, `SELECT COALESCE(MAX(trophies),0) FROM players`).Scan(&trophies)
	if trophies != 0 {
		t.Fatalf("raid trophies must be reset: %d", trophies)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ledger(user_id,kind,note) SELECT id,'wave_reward','t' FROM users WHERE login='alexey'`); err != nil {
		t.Fatalf("wave_reward kind must be allowed: %v", err)
	}
}

// shiftCurrentWave moves the team's open wave one period into the past, as if a new week had started.
func shiftCurrentWave(t *testing.T, pool *pgxpool.Pool, team string) {
	t.Helper()
	tag, err := pool.Exec(context.Background(), `UPDATE waves SET starts_at=starts_at-interval '70 days', ends_at=ends_at-interval '70 days'
 WHERE team_id=(SELECT id FROM teams WHERE name=$1) AND settled_at IS NULL`, team)
	if err != nil || tag.RowsAffected() == 0 {
		t.Fatalf("shift wave: %v, rows %d", err, tag.RowsAffected())
	}
}

// planOf is the plan of the team's wave that contains now: task number → points.
func planOf(t *testing.T, pool *pgxpool.Pool, team string) map[string]int {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT tk.number, wp.points FROM wave_plan wp JOIN waves w ON w.id=wp.wave_id JOIN tasks tk ON tk.id=wp.task_id
 WHERE w.team_id=(SELECT id FROM teams WHERE name=$1) AND w.starts_at<=now() AND w.ends_at>now()`, team)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var n string
		var p int
		if err := rows.Scan(&n, &p); err != nil {
			t.Fatal(err)
		}
		out[n] = p
	}
	return out
}

func TestWaveSnapshotBeforeFirstApproval(t *testing.T) {
	ts, pool, _ := testServer(t)
	irina := login(t, ts, "irina")
	review := findTask(t, irina, "Импорт платежей из банка") // PLT-10, review, Maria
	if len(planOf(t, pool, "Платформа")) == 0 {
		// The seed creates the current wave; without it there is nothing to shift.
		t.Fatal("current wave must exist after seeding")
	}
	shiftCurrentWave(t, pool, "Платформа")
	irina.must(200, "POST", "/tasks/"+review["id"].(string)+"/transition", map[string]any{"to": "done"}, "")
	if p := planOf(t, pool, "Платформа"); p["PLT-10"] != 5 {
		t.Fatalf("approved task must be in the new wave plan: %v", p)
	}
}

func TestWavePlanKeepsTaskMovedToBacklog(t *testing.T) {
	ts, pool, _ := testServer(t)
	irina := login(t, ts, "irina")
	ready := findTask(t, irina, "Валидация ИНН при регистрации") // PLT-12, ready, 2
	irina.must(200, "POST", "/tasks/"+ready["id"].(string)+"/transition", map[string]any{"to": "backlog"}, "")
	if p := planOf(t, pool, "Платформа"); p["PLT-12"] != 2 {
		t.Fatalf("task moved to backlog stays in the plan: %v", p)
	}
}

func TestBacklogTaskJoinsPlanWhenReady(t *testing.T) {
	ts, pool, _ := testServer(t)
	irina := login(t, ts, "irina")
	bl := findTask(t, irina, "Мультивалютные счета") // PLT-14, backlog, 8, agreed
	if _, ok := planOf(t, pool, "Платформа")["PLT-14"]; ok {
		t.Fatal("backlog is fog, not plan")
	}
	irina.must(200, "POST", "/tasks/"+bl["id"].(string)+"/transition", map[string]any{"to": "ready"}, "")
	if p := planOf(t, pool, "Платформа"); p["PLT-14"] != 8 {
		t.Fatalf("became an enemy: %v", p)
	}
}

// endCurrentWave puts the team's open wave entirely in the past so the next tick settles it.
func endCurrentWave(t *testing.T, pool *pgxpool.Pool, team string) {
	t.Helper()
	// The end comes from the Go clock, like every task and reward timestamp: the database clock in Docker can lag
	// the host by a few milliseconds, which would put a just-approved task after the wave's end.
	_, err := pool.Exec(context.Background(), `UPDATE waves SET starts_at=$2::timestamptz-interval '7 days', ends_at=$2
 WHERE team_id=(SELECT id FROM teams WHERE name=$1) AND settled_at IS NULL`, team, time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

// keepOnlyInPlan narrows the open wave's plan to one task, so the crowns are predictable (closing it = 100 %).
func keepOnlyInPlan(t *testing.T, pool *pgxpool.Pool, team, number string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `DELETE FROM wave_plan wp USING waves w, tasks tk
 WHERE wp.wave_id=w.id AND tk.id=wp.task_id AND w.settled_at IS NULL AND tk.number<>$2 AND w.team_id=(SELECT id FROM teams WHERE name=$1)`, team, number)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSettleWaveRewardsWholeTeam(t *testing.T) {
	ts, pool, s := testServer(t)
	ctx := context.Background()
	irina, maria := login(t, ts, "irina"), login(t, ts, "maria")
	maria.must(200, "PATCH", "/me/settings", map[string]any{"game_mode": false}, "") // work mode
	review := findTask(t, irina, "Импорт платежей из банка")
	irina.must(200, "POST", "/tasks/"+review["id"].(string)+"/transition", map[string]any{"to": "done"}, "")
	keepOnlyInPlan(t, pool, "Платформа", "PLT-10")
	endCurrentWave(t, pool, "Платформа")
	if err := s.coopTick(ctx); err != nil {
		t.Fatal(err)
	}
	var crowns *int
	var summary string
	_ = pool.QueryRow(ctx, `SELECT crowns, summary FROM waves WHERE settled_at IS NOT NULL AND team_id=(SELECT id FROM teams WHERE name='Платформа') ORDER BY ends_at DESC LIMIT 1`).Scan(&crowns, &summary)
	if crowns == nil || *crowns < 1 || !strings.HasPrefix(summary, "Закрыто 5 из 5 баллов.") {
		t.Fatalf("settled: %v %q", crowns, summary)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM ledger l JOIN users u ON u.id=l.user_id WHERE l.kind='wave_reward' AND u.login IN ('alexey','maria','dmitry','irina')`).Scan(&n)
	if n != 4 {
		t.Fatalf("every member incl. lead and work-mode maria gets the chest: %d", n)
	}
	// The chest ceremony shows exactly what this member was paid, read back from the ledger.
	var paid int64
	_ = pool.QueryRow(ctx, `SELECT l.gold FROM ledger l JOIN users u ON u.id=l.user_id WHERE l.kind='wave_reward' AND u.login='irina' ORDER BY l.id DESC LIMIT 1`).Scan(&paid)
	w0 := items(irina.must(200, "GET", "/waves", nil, ""))[0]
	chest, _ := w0["chest"].(map[string]any)
	if chest == nil || int64(num(chest["gold"])) != paid || num(chest["energy"]) <= 0 {
		t.Fatalf("chest from the ledger: paid %d, got %v", paid, w0)
	}
	// Seeded past waves were never paid: no chest to show.
	ws := items(irina.must(200, "GET", "/waves", nil, ""))
	if c := ws[len(ws)-1]["chest"]; c != nil {
		t.Fatalf("an unpaid wave has no chest: %v", ws[len(ws)-1])
	}
	var next int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM waves WHERE settled_at IS NULL AND starts_at<=now() AND ends_at>now() AND team_id=(SELECT id FROM teams WHERE name='Платформа')`).Scan(&next)
	if next != 1 {
		t.Fatalf("the tick opens the current wave: %d", next)
	}
}

func TestSettleWaveTwiceNoDoubleReward(t *testing.T) {
	ts, pool, s := testServer(t)
	ctx := context.Background()
	irina := login(t, ts, "irina")
	irina.must(200, "POST", "/tasks/"+findTask(t, irina, "Импорт платежей из банка")["id"].(string)+"/transition", map[string]any{"to": "done"}, "")
	keepOnlyInPlan(t, pool, "Платформа", "PLT-10")
	endCurrentWave(t, pool, "Платформа")
	errs := make(chan error, 2)
	go func() { errs <- s.coopTick(ctx) }()
	go func() { errs <- s.coopTick(ctx) }()
	if e1, e2 := <-errs, <-errs; e1 != nil || e2 != nil {
		t.Fatal(e1, e2)
	}
	var n, users int
	_ = pool.QueryRow(ctx, `SELECT count(*), count(DISTINCT user_id) FROM ledger WHERE kind='wave_reward'`).Scan(&n, &users)
	if n == 0 || n != users {
		t.Fatalf("one wave reward per user: %d rows for %d users", n, users)
	}
}

func TestReopenedFromPreviousWaveCountsOnce(t *testing.T) {
	ts, pool, s := testServer(t)
	ctx := context.Background()
	irina, maria := login(t, ts, "irina"), login(t, ts, "maria")
	id := findTask(t, irina, "Импорт платежей из банка")["id"].(string)
	irina.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "done"}, "")
	endCurrentWave(t, pool, "Платформа")
	if err := s.coopTick(ctx); err != nil {
		t.Fatal(err)
	}
	irina.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "in_progress", "comment": "Нет теста"}, "")
	maria.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "review"}, "")
	irina.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "done"}, "")
	var w waveRow
	if err := pool.QueryRow(ctx, `SELECT id::text, team_id::text, starts_at, ends_at FROM waves WHERE settled_at IS NULL AND team_id=(SELECT id FROM teams WHERE name='Платформа')`).
		Scan(&w.ID, &w.TeamID, &w.StartsAt, &w.EndsAt); err != nil {
		t.Fatal(err)
	}
	tally, err := tallyWave(ctx, pool, w)
	if err != nil {
		t.Fatal(err)
	}
	if tally.Closed < 5 || tally.Closed > tally.Planned {
		t.Fatalf("re-approved task counts once: %+v", tally)
	}
	var p int
	_ = pool.QueryRow(ctx, `SELECT points FROM wave_plan WHERE wave_id=$1 AND task_id=$2`, w.ID, id).Scan(&p)
	if p != 5 {
		t.Fatalf("re-entered the plan with its points: %d", p)
	}
}

func TestOverdueDamageOncePerDayAndFloor(t *testing.T) {
	_, pool, _ := testServer(t)
	ctx := context.Background()
	var team string
	_ = pool.QueryRow(ctx, `SELECT id::text FROM teams WHERE name='Платформа'`).Scan(&team)
	_, _ = pool.Exec(ctx, `UPDATE fortresses SET hp=100 WHERE team_id=$1`, team)
	friday := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	_, _ = pool.Exec(ctx, `UPDATE tasks SET due_date=$1 WHERE number='PLT-10'`, friday.AddDate(0, 0, -3)) // 5 points, review
	run := func(day time.Time) int {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		d, err := applyOverdueDamage(ctx, tx, team, day)
		if err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		return d
	}
	if first := run(friday); first < 10 {
		t.Fatalf("PLT-10 alone deals 10: %d", first)
	}
	if again := run(friday); again != 0 {
		t.Fatalf("same day twice: %d", again)
	}
	if sunday := run(friday.AddDate(0, 0, 2)); sunday != 0 {
		t.Fatalf("no damage on a day off: %d", sunday)
	}
	_, _ = pool.Exec(ctx, `UPDATE fortresses SET hp=12 WHERE team_id=$1`, team)
	run(friday.AddDate(0, 0, 3))
	var hp int
	_ = pool.QueryRow(ctx, `SELECT hp FROM fortresses WHERE team_id=$1`, team).Scan(&hp)
	if hp != 10 {
		t.Fatalf("floor 10: %d", hp)
	}
}

func TestFortressAPI(t *testing.T) {
	ts, _, _ := testServer(t)
	alexey := login(t, ts, "alexey")
	f := alexey.must(200, "GET", "/fortress", nil, "").Body
	if num(f["fortress"].(map[string]any)["hp"]) <= 0 {
		t.Fatalf("hp: %v", f)
	}
	enemies := f["enemies"].([]any)
	if len(enemies) == 0 {
		t.Fatal("seeded board has enemies")
	}
	overdue := false
	for _, e := range enemies {
		m := e.(map[string]any)
		if _, leaked := m["assignee_name"]; leaked {
			t.Fatal("no names next to enemies")
		}
		if m["kind"] == "" || num(m["row"]) < 1 || num(m["row"]) > 4 {
			t.Fatalf("enemy: %v", m)
		}
		if m["number"] == "PLT-10" {
			overdue = m["overdue"] == true
		}
	}
	if !overdue {
		t.Fatal("PLT-10 is overdue in the seed")
	}
	// The panel needs the stage and whether the task is the viewer's own (to offer «Подтвердить»), never whose it is.
	mine := 0
	for _, e := range enemies {
		m := e.(map[string]any)
		st, _ := m["state"].(string)
		if st != "ready" && st != "in_progress" && st != "review" {
			t.Fatalf("enemy state: %v", m)
		}
		if v, ok := m["mine"].(bool); !ok {
			t.Fatalf("mine flag: %v", m)
		} else if v {
			mine++
		}
	}
	irina := login(t, ts, "irina").must(200, "GET", "/fortress", nil, "").Body["enemies"].([]any)
	irinaMine := 0
	for _, e := range irina {
		if e.(map[string]any)["mine"] == true {
			irinaMine++
		}
	}
	if mine == 0 || mine == len(enemies) || irinaMine == mine {
		t.Fatalf("mine is per viewer: alexey %d of %d, irina %d", mine, len(enemies), irinaMine)
	}
	if num(f["wave"].(map[string]any)["planned"]) == 0 || num(f["fog"]) == 0 {
		t.Fatalf("plan snapshot and fog exist: %v", f["wave"])
	}
	login(t, ts, "admin").refused("not_a_player", "GET", "/fortress", nil)
	alexey.must(200, "GET", "/waves", nil, "")
}

func TestProgressHasNoPerPersonTable(t *testing.T) {
	ts, _, _ := testServer(t)
	irina := login(t, ts, "irina")
	team := irina.must(200, "GET", "/progress/team", nil, "")
	if _, ok := team.Body["items"]; ok || strings.Contains(team.Raw, "display_name") || strings.Contains(team.Raw, "user_id") {
		t.Fatalf("no per-person rows: %s", team.Raw)
	}
	if len(team.Body["team_weekly"].([]any)) != 8 {
		t.Fatalf("8 weeks: %s", team.Raw)
	}
	me := login(t, ts, "alexey").must(200, "GET", "/progress/me", nil, "").Body
	if len(me["weekly"].([]any)) != 8 || num(me["total"]) <= 0 {
		t.Fatalf("my progress: %v", me)
	}
	irina.must(404, "GET", "/ratings/work", nil, "")
	if irina.must(200, "GET", "/me", nil, "").Body["game_rules_ack"] != false {
		t.Fatal("not acknowledged yet")
	}
	irina.must(200, "POST", "/me/game-rules-ack", map[string]any{}, "")
	if irina.must(200, "GET", "/me", nil, "").Body["game_rules_ack"] != true {
		t.Fatal("acknowledged")
	}
}

func TestAdminTeamsAndSeed(t *testing.T) {
	ts, pool, _ := testServer(t)
	admin := login(t, ts, "admin")
	teams := items(admin.must(200, "GET", "/admin/teams", nil, ""))
	if len(teams) != 2 {
		t.Fatalf("two demo teams: %v", teams)
	}
	id := teams[0]["id"].(string) // Мобайл
	anna := login(t, ts, "anna")
	waveBefore := anna.must(200, "GET", "/fortress", nil, "").Body["wave"].(map[string]any)["id"]
	admin.refused("invalid_wave_weeks", "PATCH", "/admin/teams/"+id, map[string]any{"wave_weeks": 5})
	admin.refused("invalid_work_days", "PATCH", "/admin/teams/"+id, map[string]any{"work_days": []int{}})
	admin.refused("invalid_work_days", "PATCH", "/admin/teams/"+id, map[string]any{"work_days": []int{1, 1}})
	admin.refused("invalid_work_days", "PATCH", "/admin/teams/"+id, map[string]any{"work_days": []int{8}})
	got := admin.must(200, "PATCH", "/admin/teams/"+id, map[string]any{"wave_weeks": 2, "work_days": []int{1, 2, 3, 4}}, "").Body
	if num(got["wave_weeks"]) != 2 || len(got["work_days"].([]any)) != 4 {
		t.Fatalf("patched: %v", got)
	}
	if after := anna.must(200, "GET", "/fortress", nil, "").Body["wave"].(map[string]any)["id"]; after != waveBefore {
		t.Fatalf("a new wave length applies from the next wave, not mid-wave: %v → %v", waveBefore, after)
	}
	alexey := login(t, ts, "alexey")
	alexey.refused("forbidden", "PATCH", "/admin/teams/"+id, map[string]any{"wave_weeks": 1})
	ws := items(alexey.must(200, "GET", "/waves", nil, ""))
	if len(ws) < 3 {
		t.Fatalf("seeded past waves: %d", len(ws))
	}
	// The seed sets 85; on a work day the clock has already taken today's overdue damage off it.
	var hp, damage int
	_ = pool.QueryRow(context.Background(), `SELECT f.hp, COALESCE((SELECT -SUM(e.delta) FROM fortress_events e WHERE e.team_id=f.team_id AND e.kind='overdue_damage'), 0)
 FROM fortresses f JOIN teams t ON t.id=f.team_id WHERE t.name='Платформа'`).Scan(&hp, &damage)
	if hp != 85-damage {
		t.Fatalf("seeded fortress: %d, overdue damage %d", hp, damage)
	}
}

func TestLeadCreatedReadyTaskJoinsPlan(t *testing.T) {
	ts, pool, _ := testServer(t)
	irina := login(t, ts, "irina")
	var project string
	for _, p := range items(irina.must(200, "GET", "/projects", nil, "")) {
		if p["key"] == "PLT" {
			project = p["id"].(string)
		}
	}
	created := irina.must(201, "POST", "/tasks", map[string]any{"project_id": project, "title": "Новый враг", "points": 8, "priority": "P2", "state": "ready"}, "").Body
	if p := planOf(t, pool, "Платформа"); p[created["number"].(string)] != 8 {
		t.Fatalf("an enemy created by a lead joins the plan at once: %v", p)
	}
}
