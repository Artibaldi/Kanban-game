package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	"kanban-game/server/internal/game"
)

// garrisonOf: the troops colleagues lent to a player's banner, by type, and the places they take.
func garrisonOf(ctx context.Context, q dbtx, ownerID string) (map[string]int, int, error) {
	rows, err := q.Query(ctx, `SELECT type, SUM(count)::int FROM reinforcements WHERE owner_id=$1 GROUP BY type`, ownerID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out, held := map[string]int{}, 0
	for rows.Next() {
		var kind string
		var n int
		if err = rows.Scan(&kind, &n); err != nil {
			return nil, 0, err
		}
		out[kind] = n
		held += game.Troops[kind].Housing * n
	}
	return out, held, rows.Err()
}

// defenseOfPlayer: a player's defence as a battle meets it — the city plus the banner's garrison.
func defenseOfPlayer(ctx context.Context, q dbtx, userID string) (game.Defense, error) {
	c, err := loadCity(ctx, q, userID)
	if err != nil {
		return game.Defense{}, err
	}
	d := game.DefenseOf(c)
	if d.Garrison, _, err = garrisonOf(ctx, q, userID); err != nil {
		return d, err
	}
	if len(d.Garrison) == 0 {
		d.Garrison = nil
	}
	return d, nil
}

// teamCrowns: the crowns of the team's settled waves, oldest first.
func teamCrowns(ctx context.Context, q dbtx, teamID string) ([]int, error) {
	rows, err := q.Query(ctx, `SELECT crowns FROM waves WHERE team_id::text=$1 AND settled_at IS NOT NULL AND crowns IS NOT NULL ORDER BY starts_at`, teamID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int])
}

// lockTeamLending: one lending operation at a time per team (send, release, a member moving away), so the set of
// donors and owners cannot change under it and the players below can be locked in a single id-ordered statement.
func lockTeamLending(ctx context.Context, tx pgx.Tx, teamID string) error {
	if teamID == "" {
		return nil
	}
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('lending:' || $1))`, teamID)
	return err
}

// lockLending locks userID together with everyone whose troops it holds (and, unless ownerOnly, everyone holding
// its troops) in one id-ordered statement. Call it under lockTeamLending.
func lockLending(ctx context.Context, tx pgx.Tx, userID string, ownerOnly bool) error {
	rows, err := tx.Query(ctx, `SELECT donor_id::text FROM reinforcements WHERE owner_id=$1
 UNION SELECT owner_id::text FROM reinforcements WHERE NOT $2 AND donor_id=$1`, userID, ownerOnly)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	return lockPlayers(ctx, tx, append(ids, userID)...)
}

// returnReinforcements sends home every troop lent by or to userID (released, or when someone leaves the team).
// Troops come back even over the camp's housing: nothing lent is ever lost.
func returnReinforcements(ctx context.Context, tx pgx.Tx, userID string, ownerOnly bool) error {
	rows, err := tx.Query(ctx, `DELETE FROM reinforcements WHERE owner_id=$1 OR (NOT $2 AND donor_id=$1) RETURNING donor_id::text, type, count`, userID, ownerOnly)
	if err != nil {
		return err
	}
	type back struct {
		donor, kind string
		n           int
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (back, error) {
		var b back
		err := r.Scan(&b.donor, &b.kind, &b.n)
		return b, err
	})
	if err != nil {
		return err
	}
	for _, b := range list {
		if _, err = tx.Exec(ctx, `INSERT INTO troops(user_id,type,count) VALUES($1,$2,$3) ON CONFLICT (user_id,type) DO UPDATE SET count=troops.count+EXCLUDED.count`, b.donor, b.kind, b.n); err != nil {
			return err
		}
	}
	return nil
}

func bannerDTO(ctx context.Context, q dbtx, a actor) (map[string]any, error) {
	c, err := loadCity(ctx, q, a.ID)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT r.donor_id::text, u.display_name, r.type, r.count FROM reinforcements r JOIN users u ON u.id=r.donor_id
 WHERE r.owner_id=$1 ORDER BY u.display_name, r.type`, a.ID)
	if err != nil {
		return nil, err
	}
	lent, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (map[string]any, error) {
		var donor, name, kind string
		var n int
		err := r.Scan(&donor, &name, &kind, &n)
		return map[string]any{"donor_id": donor, "donor_name": name, "type": kind, "count": n}, err
	})
	if err != nil {
		return nil, err
	}
	_, held, err := garrisonOf(ctx, q, a.ID)
	if err != nil {
		return nil, err
	}
	// Teammates' banners, so troops can be sent where there is room. No counts of who gave how much.
	rows, err = q.Query(ctx, `SELECT u.id::text, u.display_name FROM users u JOIN players p ON p.user_id=u.id
 WHERE u.active AND u.id::text<>$1 AND u.team_id::text=$2 ORDER BY u.display_name`, a.ID, a.TeamID)
	if err != nil {
		return nil, err
	}
	type mate struct{ id, name string }
	mates, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (mate, error) {
		var m mate
		err := r.Scan(&m.id, &m.name)
		return m, err
	})
	if err != nil {
		return nil, err
	}
	team := []map[string]any{}
	for _, m := range mates {
		mc, err := loadCity(ctx, q, m.id)
		if err != nil {
			return nil, err
		}
		_, h, err := garrisonOf(ctx, q, m.id)
		if err != nil {
			return nil, err
		}
		team = append(team, map[string]any{"user_id": m.id, "display_name": m.name, "capacity": mc.GarrisonCapacity(), "held": h})
	}
	crowns, err := teamCrowns(ctx, q, a.TeamID)
	if err != nil {
		return nil, err
	}
	troops, _, err := armyOf(ctx, q, a.ID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"capacity": c.GarrisonCapacity(), "held": held, "reinforcements": lent, "team": team,
		"crowns": crowns, "trophies": game.Trophies(crowns), "troops": troops}, nil
}

func (s *Server) banner(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	if a.TeamID == "" {
		failErr(w, refuse(403, "not_a_player", "Игра доступна участникам команд."))
		return
	}
	if err := s.settleNow(r.Context(), a.ID); err != nil {
		failErr(w, err)
		return
	}
	out, err := bannerDTO(r.Context(), s.db, a)
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, out)
}

// donate: send ready troops to a teammate's banner. They leave your camp and come back when released.
func (s *Server) donate(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	s.command(w, r, a, func(ctx context.Context, tx pgx.Tx, raw []byte) (int, any, error) {
		var in struct {
			ToID  string `json:"to_id"`
			Type  string `json:"type"`
			Count int    `json:"count"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		if in.ToID == a.ID {
			return 0, nil, refuse(422, "invalid_target", "Подкрепления отправляют коллегам, не себе.")
		}
		var name string
		err := tx.QueryRow(ctx, `SELECT u.display_name FROM users u JOIN players p ON p.user_id=u.id WHERE u.id::text=$1 AND u.active AND u.team_id::text=$2`, in.ToID, a.TeamID).Scan(&name)
		if err == pgx.ErrNoRows || a.TeamID == "" {
			return 0, nil, refuse(404, "not_found", "Коллега не найден в вашей команде.")
		}
		if err != nil {
			return 0, nil, err
		}
		// Both players, in a fixed order, under the team's lending lock: no other send or release can interleave.
		if err = lockTeamLending(ctx, tx, a.TeamID); err != nil {
			return 0, nil, err
		}
		if err = lockPlayers(ctx, tx, a.ID, in.ToID); err != nil {
			return 0, nil, err
		}
		// Checked again under the users' rows: an administrator moving either of them waits for us, or we see the move.
		var same bool
		if err = tx.QueryRow(ctx, `SELECT count(*)=2 AND count(DISTINCT team_id)=1 AND bool_and(active) FROM (SELECT team_id, active FROM users WHERE id IN ($1,$2) FOR SHARE) u`, a.ID, in.ToID).Scan(&same); err != nil {
			return 0, nil, err
		}
		if !same {
			return 0, nil, refuse(404, "not_found", "Коллега не найден в вашей команде.")
		}
		if _, _, _, err = gameTx(ctx, tx, a); err != nil {
			return 0, nil, err
		}
		owner, err := loadCity(ctx, tx, in.ToID)
		if err != nil {
			return 0, nil, err
		}
		_, held, err := garrisonOf(ctx, tx, in.ToID)
		if err != nil {
			return 0, nil, err
		}
		troops, _, err := armyOf(ctx, tx, a.ID)
		if err != nil {
			return 0, nil, err
		}
		if rf := game.CheckDonate(owner, held, in.Type, in.Count, troops[in.Type]); rf != nil {
			return 0, nil, asAPI(rf)
		}
		if _, err = tx.Exec(ctx, `UPDATE troops SET count=count-$3 WHERE user_id=$1 AND type=$2`, a.ID, in.Type, in.Count); err != nil {
			return 0, nil, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO reinforcements(owner_id,donor_id,type,count) VALUES($1,$2,$3,$4)
 ON CONFLICT (owner_id,donor_id,type) DO UPDATE SET count=reinforcements.count+EXCLUDED.count`, in.ToID, a.ID, in.Type, in.Count); err != nil {
			return 0, nil, err
		}
		if err = audit(ctx, tx, a.ID, a.TeamID, "banner.reinforced", "user", in.ToID, fmt.Sprintf("Подкрепление на знамя → %s", name)); err != nil {
			return 0, nil, err
		}
		out, err := bannerDTO(ctx, tx, a)
		return 201, out, err
	})
}

// release: the banner's owner sends every lent troop home.
func (s *Server) release(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	s.command(w, r, a, func(ctx context.Context, tx pgx.Tx, _ []byte) (int, any, error) {
		// Donors' camps change too: lock the owner and every donor together, in id order, before anything else.
		if err := lockTeamLending(ctx, tx, a.TeamID); err != nil {
			return 0, nil, err
		}
		if err := lockLending(ctx, tx, a.ID, true); err != nil {
			return 0, nil, err
		}
		if _, _, _, err := gameTx(ctx, tx, a); err != nil {
			return 0, nil, err
		}
		var lent int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM reinforcements WHERE owner_id=$1`, a.ID).Scan(&lent); err != nil {
			return 0, nil, err
		}
		if err := returnReinforcements(ctx, tx, a.ID, true); err != nil {
			return 0, nil, err
		}
		if lent > 0 {
			if err := audit(ctx, tx, a.ID, a.TeamID, "banner.released", "user", a.ID, "Подкрепления со знамени вернулись домой"); err != nil {
				return 0, nil, err
			}
		}
		out, err := bannerDTO(ctx, tx, a)
		return 200, out, err
	})
}
