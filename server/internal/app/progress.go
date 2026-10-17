package app

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// Progress replaces the per-person rating: everyone sees their own weeks against their own past,
// and the team sees its total, its overdue enemies and its waves. Nobody is ranked against a colleague.

// progressWeeks are the Mondays of the last 8 weeks, oldest first, in UTC like the waves.
func progressWeeks(now time.Time) (first time.Time, weeks []string, index map[string]int) {
	first = startOfWeek(now.UTC()).AddDate(0, 0, -7*7)
	weeks = make([]string, 8)
	index = map[string]int{}
	for i := range weeks {
		weeks[i] = first.AddDate(0, 0, 7*i).Format("2006-01-02")
		index[weeks[i]] = i
	}
	return first, weeks, index
}

// weeklyPoints sums work points by week for one user ("" = the whole team).
func weeklyPoints(ctx context.Context, q dbtx, teamID, userID string, now time.Time) ([]string, []int64, error) {
	first, weeks, index := progressWeeks(now)
	rows, err := q.Query(ctx, `SELECT to_char(date_trunc('week', created_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD'), SUM(points) FROM work_points_events
 WHERE team_id=$1 AND ($2='' OR user_id::text=$2) AND created_at>=$3 GROUP BY 1`, teamID, userID, first)
	if err != nil {
		return nil, nil, err
	}
	series := make([]int64, 8)
	var wk string
	var pts int64
	_, err = pgx.ForEachRow(rows, []any{&wk, &pts}, func() error {
		if i, ok := index[wk]; ok {
			series[i] += pts
		}
		return nil
	})
	return weeks, series, err
}

func (s *Server) myProgress(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	if a.TeamID == "" {
		failErr(w, noTeam())
		return
	}
	ctx := r.Context()
	weeks, series, err := weeklyPoints(ctx, s.db, a.TeamID, a.ID, s.now())
	if err != nil {
		failErr(w, err)
		return
	}
	var total int64
	if err = s.db.QueryRow(ctx, `SELECT COALESCE(SUM(points),0) FROM work_points_events WHERE user_id=$1`, a.ID).Scan(&total); err != nil {
		failErr(w, err)
		return
	}
	best := int64(0)
	for _, v := range series {
		best = max(best, v)
	}
	respond(w, 200, map[string]any{"weeks": weeks, "weekly": series, "this_week": series[len(series)-1], "best_week": best, "total": total,
		"explanation": "Баллы сложности ваших задач, подтверждённых руководителем. Сравнения с коллегами нет."})
}

func (s *Server) teamProgress(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	team, err := teamScope(a, r)
	if err != nil {
		failErr(w, err)
		return
	}
	ctx := r.Context()
	weeks, series, err := weeklyPoints(ctx, s.db, team, "", s.now())
	if err != nil {
		failErr(w, err)
		return
	}
	var overdue int
	if err = s.db.QueryRow(ctx, `SELECT count(*) FROM tasks t JOIN projects p ON p.id=t.project_id
 WHERE p.team_id=$1 AND `+enemyFilter+` AND t.due_date < $2::date`, team, s.now().UTC().Format("2006-01-02")).Scan(&overdue); err != nil {
		failErr(w, err)
		return
	}
	waves, err := settledWaves(ctx, s.db, team, 8)
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"weeks": weeks, "team_weekly": series, "overdue_now": overdue, "waves": waves})
}

// ackGameRules records that the user has read how the game works, including that it is never used to judge work.
func (s *Server) ackGameRules(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	s.command(w, r, a, func(ctx context.Context, tx pgx.Tx, _ []byte) (int, any, error) {
		_, err := tx.Exec(ctx, `UPDATE users SET game_rules_ack_at=COALESCE(game_rules_ack_at, now()) WHERE id=$1`, a.ID)
		return 200, map[string]any{"game_rules_ack": true}, err
	})
}
