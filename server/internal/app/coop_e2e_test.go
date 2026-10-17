package app

import (
	"context"
	"testing"

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
	tag, err := pool.Exec(context.Background(), `UPDATE waves SET starts_at=starts_at-interval '7 days', ends_at=ends_at-interval '7 days'
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
