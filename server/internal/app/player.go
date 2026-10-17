package app

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"kanban-game/server/internal/game"
)

// playerState is the resource bar: balances, level, caps, debt and the city summary.
func playerState(ctx context.Context, q dbtx, userID string) (map[string]any, error) {
	s, err := loadSettings(ctx, q)
	if err != nil {
		return nil, err
	}
	b, err := loadBalance(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	c, err := loadCity(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	capG, capE := c.Caps()
	level, into, need := game.Level(s, b.XP)
	troops, queued, err := armyOf(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	housing := game.Housing(troops)
	return map[string]any{
		"gold": b.Gold, "energy": b.Energy, "xp": b.XP, "gold_cap": capG, "energy_cap": capE,
		"level": level, "level_xp": into, "level_need": need, "trophies": b.Trophies,
		"debt": b.Gold < 0 || b.Energy < 0, "townhall": c.TownHall(), "builders_busy": c.BuildersBusy(), "builders": game.MaxBuilders,
		"army_capacity": c.ArmyCapacity(), "army_housing": housing, "army_queued": queued,
	}, nil
}

func (s *Server) player(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	if err := s.settleNow(r.Context(), a.ID); err != nil {
		failErr(w, err)
		return
	}
	out, err := playerState(r.Context(), s.db, a.ID)
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, out)
}

// ledger is the actor's own operations journal.
func (s *Server) ledger(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT l.id,l.kind,l.gold,l.energy,l.xp,l.gold_formula,l.energy_formula,l.reversed,l.reverses_id,COALESCE(t.number,''),l.note,l.created_at
 FROM ledger l LEFT JOIN tasks t ON t.id=l.task_id WHERE l.user_id=$1 ORDER BY l.id DESC LIMIT 200`, a.ID)
	if err != nil {
		failErr(w, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		m, err := scanLedger(row)
		if err == nil {
			m["task_number"] = m["user"]
			delete(m, "user")
		}
		return m, err
	})
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": out})
}

// events is the feed: the team's work and game events, plus the actor's own.
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	limit := 40
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 && n <= 200 {
		limit = n
	}
	rows, err := s.db.Query(r.Context(), `SELECT e.id,COALESCE(u.display_name,'Система'),e.action,e.target_type,e.target_id,e.summary,e.created_at
 FROM audit_events e LEFT JOIN users u ON u.id=e.actor_id
 WHERE $1 OR e.team_id::text=$2 OR e.actor_id::text=$3 ORDER BY e.id DESC LIMIT $4`, a.admin(), a.TeamID, a.ID, limit)
	if err != nil {
		failErr(w, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		var id int64
		var who, action, tt, summary string
		var target *string
		var at time.Time
		err := row.Scan(&id, &who, &action, &tt, &target, &summary, &at)
		return map[string]any{"id": id, "actor": who, "action": action, "target_type": tt, "target_id": target, "summary": summary, "created_at": at}, err
	})
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": out})
}

// teamScope resolves which team a rating is about: the actor's own, or any team for an administrator.
func teamScope(a actor, r *http.Request) (string, error) {
	team := r.URL.Query().Get("team")
	if team == "" || !a.admin() {
		team = a.TeamID
	}
	if team == "" {
		return "", refuse(422, "team_required", "Выберите команду.")
	}
	if !validUUID(team) {
		return "", refuse(404, "not_found", "Команда не найдена.")
	}
	return team, nil
}

// workRating: sum of complexity points of confirmed tasks (reversals subtract) inside one team, optionally one project.
// Battles never write to work_points_events, so they cannot affect this rating.
func startOfWeek(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	wd := int(d.Weekday())
	if wd == 0 {
		wd = 7
	}
	return d.AddDate(0, 0, -(wd - 1))
}

// burndown: remaining backlog points per day, with scope added/re-estimated/reopened shown apart from completion.
func (s *Server) burndown(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	team, err := teamScope(a, r)
	if err != nil {
		failErr(w, err)
		return
	}
	project := r.URL.Query().Get("project")
	if project != "" && !validUUID(project) {
		project = ""
	}
	days := 30
	if n, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && n >= 7 && n <= 120 {
		days = n
	}
	ctx := r.Context()
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1))
	var base int64
	if err = s.db.QueryRow(ctx, `SELECT COALESCE(SUM(delta),0) FROM scope_events WHERE team_id=$1 AND ($2='' OR project_id::text=$2) AND created_at<$3`, team, project, start).Scan(&base); err != nil {
		failErr(w, err)
		return
	}
	rows, err := s.db.Query(ctx, `SELECT date_trunc('day', created_at), kind, SUM(delta) FROM scope_events
 WHERE team_id=$1 AND ($2='' OR project_id::text=$2) AND created_at>=$3 GROUP BY 1,2`, team, project, start)
	if err != nil {
		failErr(w, err)
		return
	}
	type agg struct {
		day   time.Time
		kind  string
		delta int64
	}
	aggs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (agg, error) {
		var g agg
		err := row.Scan(&g.day, &g.kind, &g.delta)
		return g, err
	})
	if err != nil {
		failErr(w, err)
		return
	}
	type day struct {
		Date        string `json:"date"`
		Added       int64  `json:"added"`
		Reestimated int64  `json:"reestimated"`
		Reopened    int64  `json:"reopened"`
		Completed   int64  `json:"completed"`
		Remaining   int64  `json:"remaining"`
	}
	series := make([]day, days)
	idx := map[string]int{}
	for i := range series {
		series[i].Date = start.AddDate(0, 0, i).Format("2006-01-02")
		idx[series[i].Date] = i
	}
	for _, g := range aggs {
		i, ok := idx[g.day.In(now.Location()).Format("2006-01-02")]
		if !ok {
			continue
		}
		switch g.kind {
		case "added":
			series[i].Added += g.delta
		case "reestimated":
			series[i].Reestimated += g.delta
		case "reopened":
			series[i].Reopened += g.delta
		case "completed":
			series[i].Completed += -g.delta
		}
	}
	remaining := base
	for i := range series {
		remaining += series[i].Added + series[i].Reestimated + series[i].Reopened - series[i].Completed
		series[i].Remaining = remaining
	}
	respond(w, 200, map[string]any{"start_remaining": base, "days": series})
}
