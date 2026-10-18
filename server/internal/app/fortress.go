package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"kanban-game/server/internal/game"
)

type enemyDTO struct {
	TaskID   string  `json:"task_id"`
	Number   string  `json:"number"`
	Title    string  `json:"title"`
	Points   int     `json:"points"`
	Priority string  `json:"priority"`
	DueDate  *string `json:"due_date"`
	Kind     string  `json:"kind"`
	Row      int     `json:"row"`
	AtGates  bool    `json:"at_gates"`
	Overdue  bool    `json:"overdue"`
	State    string  `json:"state"`
	// Mine: the viewer is the assignee. It lets the panel offer «Подтвердить» to a lead without naming anyone.
	Mine bool `json:"mine"`
}

func noTeam() error {
	return refuse(403, "not_a_player", "Крепость есть у команд: попросите администратора добавить вас в команду.")
}

// fortress is the team's game screen: fortress strength, the current wave, the enemies (tasks taken into the
// plan) and the fog (backlog). Enemies carry no assignee: nobody is named next to an overdue task.
func (s *Server) fortress(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	if a.TeamID == "" {
		failErr(w, noTeam())
		return
	}
	ctx := r.Context()
	var out map[string]any
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		var err error
		out, err = fortressState(ctx, tx, a.TeamID, a.ID, s.now())
		return err
	})
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, out)
}

func fortressState(ctx context.Context, tx pgx.Tx, teamID, viewerID string, now time.Time) (map[string]any, error) {
	st, err := loadSettings(ctx, tx)
	if err != nil {
		return nil, err
	}
	wave, err := ensureWave(ctx, tx, teamID, now)
	if err != nil {
		return nil, err
	}
	hp, err := lockFortress(ctx, tx, teamID)
	if err != nil {
		return nil, err
	}
	var level int
	if err = tx.QueryRow(ctx, `SELECT level FROM fortresses WHERE team_id=$1`, teamID).Scan(&level); err != nil {
		return nil, err
	}
	tally, err := tallyWave(ctx, tx, wave)
	if err != nil {
		return nil, err
	}
	today := now.UTC().Format("2006-01-02")
	rows, err := tx.Query(ctx, `SELECT t.id::text, t.number, t.title, t.points, t.priority, to_char(t.due_date,'YYYY-MM-DD'),
 COALESCE(t.due_date < $2::date, false), COALESCE(t.due_date >= $2::date AND t.due_date <= $2::date + 2, false),
 t.state, COALESCE(t.assignee_id::text = $3, false)
 FROM tasks t JOIN projects p ON p.id=t.project_id WHERE p.team_id=$1 AND `+enemyFilter+`
 ORDER BY t.priority, t.due_date NULLS LAST, t.rank`, teamID, today, viewerID)
	if err != nil {
		return nil, err
	}
	enemies, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (enemyDTO, error) {
		var e enemyDTO
		err := row.Scan(&e.TaskID, &e.Number, &e.Title, &e.Points, &e.Priority, &e.DueDate, &e.Overdue, &e.AtGates, &e.State, &e.Mine)
		e.Kind = game.EnemyKind(e.Points)
		e.Row = int(e.Priority[1] - '0')
		return e, err
	})
	if err != nil {
		return nil, err
	}
	var fog int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM tasks t JOIN projects p ON p.id=t.project_id WHERE p.team_id=$1 AND t.state='backlog'`, teamID).Scan(&fog); err != nil {
		return nil, err
	}
	var last any
	var lastID, summary string
	var ends time.Time
	var crowns int
	err = tx.QueryRow(ctx, `SELECT id::text, ends_at, crowns, summary FROM waves WHERE team_id=$1 AND settled_at IS NOT NULL ORDER BY ends_at DESC LIMIT 1`, teamID).
		Scan(&lastID, &ends, &crowns, &summary)
	switch {
	case err == nil:
		last = map[string]any{"id": lastID, "ends_at": ends, "crowns": crowns, "summary": summary}
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, err
	}
	return map[string]any{
		"fortress": map[string]any{"hp": hp, "level": level, "siege": hp <= st.FortressFloor},
		"wave": map[string]any{"id": wave.ID, "starts_at": wave.StartsAt, "ends_at": wave.EndsAt,
			"days_left": int(wave.EndsAt.Sub(now).Hours() / 24), "planned": tally.Planned, "closed": tally.Closed,
			"overdue": tally.Overdue, "overdue_p1": tally.OverdueP1, "crowns_now": game.Crowns(st, tally)},
		"enemies":   enemies,
		"fog":       fog,
		"last_wave": last,
	}, nil
}

// waves lists the team's last settled waves, newest first.
func (s *Server) waves(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	if a.TeamID == "" {
		failErr(w, noTeam())
		return
	}
	items, err := settledWaves(r.Context(), s.db, a.TeamID, 12)
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": items})
}

func settledWaves(ctx context.Context, q dbtx, teamID string, limit int) ([]map[string]any, error) {
	rows, err := q.Query(ctx, `SELECT id::text, starts_at, ends_at, planned_points, closed_points, overdue_count, crowns, summary, settled_at,
 COALESCE(defense_bonus, 0)::float8
 FROM waves WHERE team_id=$1 AND settled_at IS NOT NULL ORDER BY ends_at DESC LIMIT $2`, teamID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		var id, summary string
		var starts, ends, settled time.Time
		var planned, closed, overdue, crowns int
		var bonus float64
		err := row.Scan(&id, &starts, &ends, &planned, &closed, &overdue, &crowns, &summary, &settled, &bonus)
		return map[string]any{"id": id, "starts_at": starts, "ends_at": ends, "planned": planned, "closed": closed,
			"overdue": overdue, "crowns": crowns, "summary": summary, "settled_at": settled, "defense_bonus": bonus}, err
	})
}
