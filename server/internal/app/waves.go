package app

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"kanban-game/server/internal/game"
)

// waveRow is one period of a team's defense against its backlog.
type waveRow struct {
	ID, TeamID       string
	StartsAt, EndsAt time.Time
}

// enemyFilter selects the tasks that are enemies now (see game.IsEnemy); t is tasks, p is projects.
const enemyFilter = `t.state IN ('ready','in_progress','review') AND t.points_approved_at IS NOT NULL`

// ensureWave returns the team's wave containing now, creating it if needed. Only the transaction that creates
// the wave takes the snapshot of the enemies at that moment, so the plan is never rebuilt from a later state.
func ensureWave(ctx context.Context, tx pgx.Tx, teamID string, now time.Time) (waveRow, error) {
	w := waveRow{TeamID: teamID}
	var weeks int
	if err := tx.QueryRow(ctx, `SELECT wave_weeks FROM teams WHERE id=$1`, teamID).Scan(&weeks); err != nil {
		return w, err
	}
	w.StartsAt, w.EndsAt = game.WaveWindow(now, weeks)
	err := tx.QueryRow(ctx, `INSERT INTO waves(team_id,starts_at,ends_at) VALUES($1,$2,$3) ON CONFLICT (team_id,starts_at) DO NOTHING RETURNING id::text`,
		teamID, w.StartsAt, w.EndsAt).Scan(&w.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `SELECT id::text, ends_at FROM waves WHERE team_id=$1 AND starts_at=$2`, teamID, w.StartsAt).Scan(&w.ID, &w.EndsAt)
		return w, err
	}
	if err != nil {
		return w, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO wave_plan(wave_id,task_id,points,joined_at)
 SELECT $1, t.id, t.points, $3 FROM tasks t JOIN projects p ON p.id=t.project_id WHERE p.team_id=$2 AND `+enemyFilter+`
 ON CONFLICT DO NOTHING`, w.ID, teamID, now)
	return w, err
}

// syncWavePlan puts a task that is an enemy now into the current plan, or updates its agreed points there.
// A task that stopped being an enemy stays in the plan as it was.
func syncWavePlan(ctx context.Context, tx pgx.Tx, taskID string, now time.Time) error {
	var teamID, state string
	var points int
	var approved bool
	err := tx.QueryRow(ctx, `SELECT p.team_id::text, t.state, t.points, t.points_approved_at IS NOT NULL FROM tasks t JOIN projects p ON p.id=t.project_id WHERE t.id=$1`, taskID).
		Scan(&teamID, &state, &points, &approved)
	if err != nil || !game.IsEnemy(state, approved) {
		return err
	}
	w, err := ensureWave(ctx, tx, teamID, now)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO wave_plan(wave_id,task_id,points,joined_at) VALUES($1,$2,$3,$4)
 ON CONFLICT (wave_id,task_id) DO UPDATE SET points=EXCLUDED.points`, w.ID, taskID, points, now)
	return err
}
