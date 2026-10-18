package app

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"kanban-game/server/internal/game"
)

// dbtx is what both the pool and a transaction offer; read helpers accept either.
type dbtx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// loadSettings reads the economy coefficients; missing or broken values fall back to the defaults.
func loadSettings(ctx context.Context, q dbtx) (game.Settings, error) {
	var raw []byte
	err := q.QueryRow(ctx, `SELECT value FROM settings WHERE key='economy'`).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return game.DefaultSettings(), nil
	}
	if err != nil {
		return game.Settings{}, err
	}
	// Stored values override the defaults key by key, so a partial edit keeps every other coefficient.
	s := game.DefaultSettings()
	if err = json.Unmarshal(raw, &s); err != nil {
		return game.DefaultSettings(), nil
	}
	return s.Normalize(), nil
}

func loadCity(ctx context.Context, q dbtx, userID string) (game.City, error) {
	rows, err := q.Query(ctx, `SELECT id::text,type,level,target_level,busy_until,x,y FROM buildings WHERE user_id=$1 ORDER BY created_at,id`, userID)
	if err != nil {
		return game.City{}, err
	}
	bs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (game.Building, error) {
		var b game.Building
		err := row.Scan(&b.ID, &b.Type, &b.Level, &b.TargetLevel, &b.BusyUntil, &b.X, &b.Y)
		return b, err
	})
	return game.City{Buildings: bs}, err
}

type balance struct {
	Gold, Energy, XP int64
	Trophies         int
	ShieldUntil      *time.Time
}

func loadBalance(ctx context.Context, q dbtx, userID string) (balance, error) {
	var b balance
	err := q.QueryRow(ctx, `SELECT gold,energy,xp,trophies,shield_until FROM players WHERE user_id=$1`, userID).
		Scan(&b.Gold, &b.Energy, &b.XP, &b.Trophies, &b.ShieldUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return b, refuse(403, "not_a_player", "У этой учётной записи нет города: игра доступна участникам команд.")
	}
	return b, err
}

// ledgerEntry is one resource operation; apply writes the journal row and moves the balance in the same transaction.
type ledgerEntry struct {
	UserID, Kind                 string
	Gold, Energy, XP             int64
	GoldFormula, EnergyFormula   *int64
	TaskID, BattleID, BuildingID string
	WaveID                       string
	ActorID, Note                string
	ReversesID                   *int64
	At                           *time.Time
}

func applyLedger(ctx context.Context, tx pgx.Tx, e ledgerEntry) (int64, error) {
	var id int64
	at := time.Now()
	if e.At != nil {
		at = *e.At
	}
	err := tx.QueryRow(ctx, `INSERT INTO ledger(user_id,kind,gold,energy,xp,gold_formula,energy_formula,task_id,battle_id,building_id,actor_id,note,created_at,reverses_id,wave_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id`,
		e.UserID, e.Kind, e.Gold, e.Energy, e.XP, e.GoldFormula, e.EnergyFormula,
		nullable(e.TaskID), nullable(e.BattleID), nullable(e.BuildingID), nullable(e.ActorID), e.Note, at, e.ReversesID, nullable(e.WaveID)).Scan(&id)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(ctx, `UPDATE players SET gold=gold+$2, energy=energy+$3, xp=xp+$4, updated_at=now() WHERE user_id=$1`, e.UserID, e.Gold, e.Energy, e.XP)
	return id, err
}

// settle completes every construction and training whose timer has run out. userID "" settles everyone
// (the background timer); a handler settles its own user first so a reload always shows the true state.
func settle(ctx context.Context, tx pgx.Tx, userID string) error {
	rows, err := tx.Query(ctx, `UPDATE buildings b SET level=target_level, target_level=NULL, busy_until=NULL
 WHERE b.id IN (SELECT id FROM buildings WHERE busy_until<=now() AND ($1='' OR user_id::text=$1) FOR UPDATE SKIP LOCKED)
 RETURNING b.id::text, b.user_id::text, b.type, b.level`, userID)
	if err != nil {
		return err
	}
	type done struct {
		id, user, kind string
		level          int
	}
	built, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (done, error) {
		var d done
		err := row.Scan(&d.id, &d.user, &d.kind, &d.level)
		return d, err
	})
	if err != nil {
		return err
	}
	for _, d := range built {
		name := game.Buildings[d.kind].Name
		if err = auditUser(ctx, tx, d.user, "building.completed", "building", d.id, name+": готов уровень "+itoa(d.level)); err != nil {
			return err
		}
	}
	rows, err = tx.Query(ctx, `DELETE FROM training_queue q
 WHERE q.id IN (SELECT id FROM training_queue WHERE finish_at<=now() AND ($1='' OR user_id::text=$1) FOR UPDATE SKIP LOCKED)
 RETURNING q.user_id::text, q.type, q.count`, userID)
	if err != nil {
		return err
	}
	type trained struct {
		user, kind string
		count      int
	}
	ready, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (trained, error) {
		var t trained
		err := row.Scan(&t.user, &t.kind, &t.count)
		return t, err
	})
	if err != nil {
		return err
	}
	for _, t := range ready {
		if _, err = tx.Exec(ctx, `INSERT INTO troops(user_id,type,count) VALUES($1,$2,$3) ON CONFLICT (user_id,type) DO UPDATE SET count=troops.count+EXCLUDED.count`, t.user, t.kind, t.count); err != nil {
			return err
		}
		if err = auditUser(ctx, tx, t.user, "army.trained", "troops", t.kind, "Обучено: "+game.Troops[t.kind].Name+" × "+itoa(t.count)); err != nil {
			return err
		}
	}
	// Workshop research whose timer has run out.
	rows, err = tx.Query(ctx, `UPDATE troop_levels t SET level=target_level, target_level=NULL, started_at=NULL, busy_until=NULL
 WHERE (t.user_id, t.type) IN (SELECT user_id, type FROM troop_levels WHERE busy_until<=now() AND ($1='' OR user_id::text=$1) FOR UPDATE SKIP LOCKED)
 RETURNING t.user_id::text, t.type, t.level`, userID)
	if err != nil {
		return err
	}
	raised, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (done, error) {
		var d done
		err := row.Scan(&d.user, &d.kind, &d.level)
		return d, err
	})
	if err != nil {
		return err
	}
	for _, d := range raised {
		if err = auditUser(ctx, tx, d.user, "army.researched", "troops", d.kind, game.Troops[d.kind].Name+": уровень "+itoa(d.level)); err != nil {
			return err
		}
	}
	return nil
}

// auditUser writes an event on behalf of a user, with their team.
func auditUser(ctx context.Context, tx pgx.Tx, userID, action, targetType, targetID, summary string) error {
	_, err := tx.Exec(ctx, `INSERT INTO audit_events(actor_id,team_id,action,target_type,target_id,summary) SELECT id,team_id,$2,$3,$4,$5 FROM users WHERE id=$1`,
		userID, action, targetType, nullable(targetID), summary)
	return err
}

// settleNow runs settle for one user in its own transaction (read handlers).
func (s *Server) settleNow(ctx context.Context, userID string) error {
	return s.inTx(ctx, func(tx pgx.Tx) error { return settle(ctx, tx, userID) })
}

// RunTimers is the server-side clock: it completes due buildings and training every second, and runs the co-op
// tick (waves, fortress damage) at start and every 30 seconds, until ctx ends.
func (s *Server) RunTimers(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	coop := time.NewTicker(30 * time.Second)
	defer coop.Stop()
	s.runCoopTick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-coop.C:
			s.runCoopTick(ctx)
		case <-t.C:
			if err := s.inTx(ctx, func(tx pgx.Tx) error { return settle(ctx, tx, "") }); err != nil && ctx.Err() == nil {
				log.Printf("timers: %v", err)
			}
		}
	}
}

func (s *Server) runCoopTick(ctx context.Context) {
	if err := s.coopTick(ctx); err != nil && ctx.Err() == nil {
		log.Printf("co-op: %v", err)
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
