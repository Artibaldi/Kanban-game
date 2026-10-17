package app

import (
	"context"
	"errors"
	"fmt"
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

// tallyWave judges a wave: its frozen plan, the plan's points closed inside the wave (a task counts when its last
// work-points event in the window is an approval, so approve → reopen → approve counts once and a reopen of an
// earlier wave's approval costs nothing here), and the enemies overdue at the wave's end.
func tallyWave(ctx context.Context, q dbtx, w waveRow) (game.WaveTally, error) {
	var t game.WaveTally
	err := q.QueryRow(ctx, `SELECT COALESCE(SUM(wp.points),0),
 COALESCE(SUM(wp.points) FILTER (WHERE COALESCE((SELECT e.points>0 FROM work_points_events e
   WHERE e.task_id=wp.task_id AND e.created_at>=$2 AND e.created_at<$3 ORDER BY e.id DESC LIMIT 1), false)),0)
 FROM wave_plan wp WHERE wp.wave_id=$1`, w.ID, w.StartsAt, w.EndsAt).Scan(&t.Planned, &t.Closed)
	if err != nil {
		return t, err
	}
	err = q.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE t.priority='P1') FROM tasks t JOIN projects p ON p.id=t.project_id
 WHERE p.team_id=$1 AND `+enemyFilter+` AND t.due_date < ($2::timestamptz AT TIME ZONE 'UTC')::date`, w.TeamID, w.EndsAt).Scan(&t.Overdue, &t.OverdueP1)
	return t, err
}

// settleWave closes a finished wave once: crowns, the team chest for every member (whatever their game mode),
// the fortress repair and the feed event. A second call, or a concurrent one, finds it settled and does nothing.
func settleWave(ctx context.Context, tx pgx.Tx, w waveRow, now time.Time) error {
	var settled *time.Time
	if err := tx.QueryRow(ctx, `SELECT settled_at FROM waves WHERE id=$1 FOR UPDATE`, w.ID).Scan(&settled); err != nil || settled != nil {
		return err
	}
	st, err := loadSettings(ctx, tx)
	if err != nil {
		return err
	}
	tally, err := tallyWave(ctx, tx, w)
	if err != nil {
		return err
	}
	crowns := game.Crowns(st, tally)
	summary := game.WaveSummary(tally, crowns)
	bonus := 0.0
	if tally.Planned > 0 {
		members, err := teamPlayers(ctx, tx, w.TeamID)
		if err != nil {
			return err
		}
		if bonus, err = teamDefenseBonus(ctx, tx, st, members); err != nil {
			return err
		}
		if gold, energy := game.WaveChest(st, crowns, bonus); gold > 0 || energy > 0 {
			if err = payWaveChest(ctx, tx, w, members, gold, energy, summary, now); err != nil {
				return err
			}
		}
		if err = repairFortress(ctx, tx, st, w, crowns); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE waves SET planned_points=$2, closed_points=$3, overdue_count=$4, overdue_p1_count=$5, crowns=$6,
 defense_bonus=$7, summary=$8, settled_at=$9 WHERE id=$1`,
		w.ID, tally.Planned, tally.Closed, tally.Overdue, tally.OverdueP1, crowns, bonus, summary, now); err != nil {
		return err
	}
	return audit(ctx, tx, "", w.TeamID, "wave.settled", "wave", w.ID, "Волна завершена. "+summary)
}

// teamPlayers are the active members of a team that have a city, leads included.
func teamPlayers(ctx context.Context, q dbtx, teamID string) ([]string, error) {
	rows, err := q.Query(ctx, `SELECT u.id::text FROM users u JOIN players p ON p.user_id=u.id WHERE u.active AND u.team_id=$1 ORDER BY u.id`, teamID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// teamDefenseBonus: the members' average defense score turned into a reward bonus. It never changes the crowns.
func teamDefenseBonus(ctx context.Context, q dbtx, st game.Settings, members []string) (float64, error) {
	if len(members) == 0 {
		return 0, nil
	}
	var total int64
	for _, id := range members {
		city, err := loadCity(ctx, q, id)
		if err != nil {
			return 0, err
		}
		army, _, err := armyOf(ctx, q, id)
		if err != nil {
			return 0, err
		}
		total += game.DefenseScore(game.DefenseOf(city), army)
	}
	return game.DefenseBonus(st, total/int64(len(members))), nil
}

func payWaveChest(ctx context.Context, tx pgx.Tx, w waveRow, members []string, gold, energy int64, summary string, now time.Time) error {
	if err := lockPlayers(ctx, tx, members...); err != nil {
		return err
	}
	for _, id := range members {
		bal, err := loadBalance(ctx, tx, id)
		if err != nil {
			return err
		}
		city, err := loadCity(ctx, tx, id)
		if err != nil {
			return err
		}
		capG, capE := city.Caps()
		g, e := game.Credit(bal.Gold, gold, capG), game.Credit(bal.Energy, energy, capE)
		if _, err = applyLedger(ctx, tx, ledgerEntry{UserID: id, Kind: "wave_reward", Gold: g, Energy: e, GoldFormula: &gold, EnergyFormula: &energy,
			WaveID: w.ID, Note: "Общая награда волны. " + summary, At: &now}); err != nil {
			return err
		}
	}
	return nil
}

// repairFortress restores the fortress by the crowns and grows its (cosmetic) level with the waves won.
func repairFortress(ctx context.Context, tx pgx.Tx, st game.Settings, w waveRow, crowns int) error {
	hp, err := lockFortress(ctx, tx, w.TeamID)
	if err != nil {
		return err
	}
	next := game.RepairHP(st, hp, crowns)
	if _, err = tx.Exec(ctx, `INSERT INTO fortress_events(team_id,kind,wave_id,delta) VALUES($1,'wave_repair',$2,$3)`, w.TeamID, w.ID, next-hp); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE fortresses SET hp=$2, updated_at=now(),
 level=LEAST(5, 1 + (SELECT count(*) FROM waves WHERE team_id=$1 AND crowns>=2) + CASE WHEN $3>=2 THEN 1 ELSE 0 END) WHERE team_id=$1`, w.TeamID, next, crowns)
	return err
}

// lockFortress returns the team's fortress strength, creating the fortress on first use, and locks it.
func lockFortress(ctx context.Context, tx pgx.Tx, teamID string) (int, error) {
	if _, err := tx.Exec(ctx, `INSERT INTO fortresses(team_id) VALUES($1) ON CONFLICT DO NOTHING`, teamID); err != nil {
		return 0, err
	}
	var hp int
	err := tx.QueryRow(ctx, `SELECT hp FROM fortresses WHERE team_id=$1 FOR UPDATE`, teamID).Scan(&hp)
	return hp, err
}

// applyOverdueDamage: on a team's working day every overdue enemy hits the fortress once; returns the new damage.
func applyOverdueDamage(ctx context.Context, tx pgx.Tx, teamID string, day time.Time) (int, error) {
	var workDays []int16
	if err := tx.QueryRow(ctx, `SELECT work_days FROM teams WHERE id=$1`, teamID).Scan(&workDays); err != nil {
		return 0, err
	}
	days := make([]int, len(workDays))
	for i, d := range workDays {
		days[i] = int(d)
	}
	if !game.IsWorkDay(day, days) {
		return 0, nil
	}
	st, err := loadSettings(ctx, tx)
	if err != nil {
		return 0, err
	}
	hp, err := lockFortress(ctx, tx, teamID)
	if err != nil {
		return 0, err
	}
	date := day.UTC().Format("2006-01-02")
	rows, err := tx.Query(ctx, `SELECT t.id::text, t.points FROM tasks t JOIN projects p ON p.id=t.project_id
 WHERE p.team_id=$1 AND `+enemyFilter+` AND t.due_date < $2::date`, teamID, date)
	if err != nil {
		return 0, err
	}
	type overdue struct {
		id     string
		points int
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (overdue, error) {
		var o overdue
		err := row.Scan(&o.id, &o.points)
		return o, err
	})
	if err != nil {
		return 0, err
	}
	damage := 0
	for _, o := range list {
		d := game.OverdueDamage(st, o.points)
		tag, err := tx.Exec(ctx, `INSERT INTO fortress_events(team_id,kind,task_id,day,delta) VALUES($1,'overdue_damage',$2,$3::date,$4)
 ON CONFLICT (task_id,day) DO NOTHING`, teamID, o.id, date, -d)
		if err != nil {
			return 0, err
		}
		if tag.RowsAffected() == 1 {
			damage += d
		}
	}
	if damage > 0 {
		if _, err = tx.Exec(ctx, `UPDATE fortresses SET hp=$2, updated_at=now() WHERE team_id=$1`, teamID, game.DamageHP(st, hp, damage)); err != nil {
			return 0, err
		}
	}
	return damage, nil
}

// coopTick is the co-op clock: for every team it opens the current wave, settles finished ones and applies
// today's overdue damage. Every step is idempotent, so it may run late, twice or concurrently.
func (s *Server) coopTick(ctx context.Context) error {
	now := s.now()
	rows, err := s.db.Query(ctx, `SELECT id::text FROM teams ORDER BY name`)
	if err != nil {
		return err
	}
	teams, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	var errs []error
	for _, team := range teams {
		err := s.inTx(ctx, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, `SELECT id::text, starts_at, ends_at FROM waves WHERE team_id=$1 AND settled_at IS NULL AND ends_at<=$2 ORDER BY ends_at`, team, now)
			if err != nil {
				return err
			}
			due, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (waveRow, error) {
				w := waveRow{TeamID: team}
				err := row.Scan(&w.ID, &w.StartsAt, &w.EndsAt)
				return w, err
			})
			if err != nil {
				return err
			}
			for _, w := range due {
				if err = settleWave(ctx, tx, w, now); err != nil {
					return fmt.Errorf("settle wave %s: %w", w.ID, err)
				}
			}
			if _, err = ensureWave(ctx, tx, team, now); err != nil {
				return err
			}
			_, err = applyOverdueDamage(ctx, tx, team, now)
			return err
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
