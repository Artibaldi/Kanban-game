package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"kanban-game/server/internal/game"
)

// Challenges to friendly duels. The challenge is the consent: a colleague who does not accept duels in general
// can be fought once they accept a challenge — and only by the one who sent it. An open challenge is the
// author's own city, posted for the whole team: each colleague may attack it once. Declining is quiet.

type challenge struct {
	ID        string    `json:"id"`
	FromID    string    `json:"from_id"`
	FromName  string    `json:"from_name"`
	ToID      *string   `json:"to_id"`
	ToName    *string   `json:"to_name"`
	Open      bool      `json:"open"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Attacked  bool      `json:"attacked"`
	TeamID    string    `json:"-"`
}

const challengeSelect = `SELECT c.id::text, c.from_id::text, uf.display_name, c.to_id::text, ut.display_name, c.to_id IS NULL, c.status,
 c.created_at, c.expires_at, c.team_id::text,
 EXISTS (SELECT 1 FROM battles b WHERE b.challenge_id=c.id AND b.attacker_id::text=$1)
 FROM friendly_challenges c JOIN users uf ON uf.id=c.from_id LEFT JOIN users ut ON ut.id=c.to_id `

func scanChallenges(rows pgx.Rows) ([]challenge, error) {
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (challenge, error) {
		var c challenge
		err := row.Scan(&c.ID, &c.FromID, &c.FromName, &c.ToID, &c.ToName, &c.Open, &c.Status, &c.CreatedAt, &c.ExpiresAt, &c.TeamID, &c.Attacked)
		return c, err
	})
}

// expireChallenges closes the team's live challenges whose day has passed.
func expireChallenges(ctx context.Context, q dbtx, teamID string, now time.Time) error {
	_, err := q.Exec(ctx, `UPDATE friendly_challenges SET status='closed' WHERE team_id=$1 AND status IN ('pending','accepted','open') AND expires_at <= $2`, teamID, now)
	return err
}

func loadChallenge(ctx context.Context, tx pgx.Tx, a actor, id string) (challenge, error) {
	if !validUUID(id) {
		return challenge{}, refuse(404, "not_found", "Вызов не найден.")
	}
	rows, err := tx.Query(ctx, challengeSelect+`WHERE c.id=$2 AND c.team_id::text=$3 FOR UPDATE OF c`, a.ID, id, a.TeamID)
	if err != nil {
		return challenge{}, err
	}
	list, err := scanChallenges(rows)
	if err != nil {
		return challenge{}, err
	}
	if len(list) == 0 {
		return challenge{}, refuse(404, "not_found", "Вызов не найден.")
	}
	return list[0], nil
}

func (c challenge) live(now time.Time) bool {
	return c.ExpiresAt.After(now) && (c.Status == "pending" || c.Status == "accepted" || c.Status == "open")
}

// defenderFor checks that the actor may fight under the challenge now and returns whose city it is.
func (c challenge) defenderFor(a actor, now time.Time) (string, string, error) {
	if c.Open {
		if c.FromID == a.ID {
			return "", "", refuse(422, "invalid_opponent", "Это ваш собственный вызов: на него отвечают коллеги.")
		}
		if !c.live(now) {
			return "", "", refuse(409, "challenge_closed", "Вызов уже закрыт.")
		}
		if c.Attacked {
			return "", "", refuse(409, "already_attacked", "Вы уже ответили на этот вызов.")
		}
		return c.FromID, c.FromName, nil
	}
	if c.FromID != a.ID {
		return "", "", refuse(403, "forbidden", "Напасть по вызову может только тот, кто его бросил.")
	}
	if c.Status == "pending" && c.live(now) {
		return "", "", refuse(409, "challenge_not_accepted", "Коллега ещё не принял вызов.")
	}
	if c.Status != "accepted" || !c.live(now) {
		return "", "", refuse(409, "challenge_closed", "Вызов уже закрыт.")
	}
	return *c.ToID, *c.ToName, nil
}

// listChallenges: everything the Arena shows — incoming, outgoing, the team's open challenges, the team,
// the daily limit, and duels against my city in the last day.
func (s *Server) listChallenges(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	if a.TeamID == "" {
		failErr(w, refuse(403, "not_a_player", "Игра доступна участникам команд."))
		return
	}
	ctx, now := r.Context(), s.now()
	if err := expireChallenges(ctx, s.db, a.TeamID, now); err != nil {
		failErr(w, err)
		return
	}
	query := func(where string, args ...any) ([]challenge, error) {
		rows, err := s.db.Query(ctx, challengeSelect+where, append([]any{a.ID}, args...)...)
		if err != nil {
			return nil, err
		}
		list, err := scanChallenges(rows)
		if list == nil {
			list = []challenge{}
		}
		return list, err
	}
	incoming, err := query(`WHERE c.to_id::text=$1 AND c.status IN ('pending','accepted') ORDER BY c.created_at DESC`)
	if err != nil {
		failErr(w, err)
		return
	}
	outgoing, err := query(`WHERE c.from_id::text=$1 AND c.created_at > $2 ORDER BY c.created_at DESC LIMIT 10`, now.Add(-7*24*time.Hour))
	if err != nil {
		failErr(w, err)
		return
	}
	open, err := query(`WHERE c.team_id::text=$2 AND c.to_id IS NULL AND c.status='open' AND c.from_id::text<>$1 ORDER BY c.created_at DESC`, a.TeamID)
	if err != nil {
		failErr(w, err)
		return
	}
	st, err := loadSettings(ctx, s.db)
	if err != nil {
		failErr(w, err)
		return
	}
	today, limit, err := friendlyLimit(ctx, s.db, a.ID, st, now)
	if err != nil {
		failErr(w, err)
		return
	}
	team, err := teamForArena(ctx, s.db, a)
	if err != nil {
		failErr(w, err)
		return
	}
	rows, err := s.db.Query(ctx, `SELECT b.id::text, u.id::text, u.display_name, b.stars, b.destruction, b.created_at FROM battles b JOIN users u ON u.id=b.attacker_id
 WHERE b.defender_id::text=$1 AND b.kind='friendly' AND b.created_at > $2 ORDER BY b.created_at DESC LIMIT 10`, a.ID, now.Add(-24*time.Hour))
	if err != nil {
		failErr(w, err)
		return
	}
	against, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		var id, uid, name string
		var stars, dest int
		var at time.Time
		err := row.Scan(&id, &uid, &name, &stars, &dest, &at)
		return map[string]any{"id": id, "attacker_id": uid, "attacker": name, "stars": stars, "destruction": dest, "created_at": at}, err
	})
	if err != nil {
		failErr(w, err)
		return
	}
	if against == nil {
		against = []map[string]any{}
	}
	respond(w, 200, map[string]any{"incoming": incoming, "outgoing": outgoing, "open": open, "team": team,
		"today": today, "per_day": limit, "friendly_xp": st.FriendlyXP, "against_me": against})
}

// teamForArena: every active colleague with a city — with or without duels switched on.
func teamForArena(ctx context.Context, q dbtx, a actor) ([]map[string]any, error) {
	rows, err := q.Query(ctx, `SELECT u.id::text, u.display_name, u.accept_friendly FROM users u JOIN players p ON p.user_id=u.id
 WHERE u.active AND u.id::text<>$1 AND u.team_id::text=$2 ORDER BY u.display_name`, a.ID, a.TeamID)
	if err != nil {
		return nil, err
	}
	type row struct {
		id, name string
		accepts  bool
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		var x row
		err := r.Scan(&x.id, &x.name, &x.accepts)
		return x, err
	})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(list))
	for _, x := range list {
		c, err := loadCity(ctx, q, x.id)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"user_id": x.id, "display_name": x.name, "townhall": c.TownHall(), "defense": game.DefenseOf(c), "accepts_friendly": x.accepts})
	}
	return out, nil
}

func (s *Server) createChallenge(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	s.command(w, r, a, func(ctx context.Context, tx pgx.Tx, raw []byte) (int, any, error) {
		var in struct {
			ToID *string `json:"to_id"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		if a.TeamID == "" {
			return 0, nil, refuse(403, "not_a_player", "Игра доступна участникам команд.")
		}
		now := s.now()
		if err := expireChallenges(ctx, tx, a.TeamID, now); err != nil {
			return 0, nil, err
		}
		var to any
		summary := "Открытый вызов команде: город открыт для дружеского боя на сутки"
		if in.ToID != nil && *in.ToID != "" {
			if !validUUID(*in.ToID) || *in.ToID == a.ID {
				return 0, nil, refuse(422, "invalid_opponent", "Выберите коллегу для вызова.")
			}
			var name string
			err := tx.QueryRow(ctx, `SELECT u.display_name FROM users u JOIN players p ON p.user_id=u.id WHERE u.id=$1 AND u.active AND u.team_id::text=$2`, *in.ToID, a.TeamID).Scan(&name)
			if errors.Is(err, pgx.ErrNoRows) {
				return 0, nil, refuse(404, "not_found", "Коллега не найден в вашей команде.")
			}
			if err != nil {
				return 0, nil, err
			}
			var n int
			if err = tx.QueryRow(ctx, `SELECT count(*) FROM friendly_challenges WHERE from_id=$1 AND to_id=$2 AND status IN ('pending','accepted')`, a.ID, *in.ToID).Scan(&n); err != nil {
				return 0, nil, err
			}
			if n > 0 {
				return 0, nil, refuse(409, "challenge_exists", "Вызов этому коллеге уже ждёт ответа.")
			}
			to = *in.ToID
			summary = fmt.Sprintf("Вызов на дружеский бой: %s", name)
		} else {
			var n int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM friendly_challenges WHERE from_id=$1 AND to_id IS NULL AND status='open'`, a.ID).Scan(&n); err != nil {
				return 0, nil, err
			}
			if n > 0 {
				return 0, nil, refuse(409, "challenge_exists", "Ваш открытый вызов уже висит в крепости.")
			}
		}
		status := "pending"
		if to == nil {
			status = "open"
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO friendly_challenges(team_id,from_id,to_id,status,expires_at) VALUES($1,$2,$3,$4,$5) RETURNING id::text`,
			a.TeamID, a.ID, to, status, now.Add(game.ChallengeTTL)).Scan(&id); err != nil {
			return 0, nil, err
		}
		if err := audit(ctx, tx, a.ID, a.TeamID, "challenge.sent", "challenge", id, summary); err != nil {
			return 0, nil, err
		}
		c, err := loadChallenge(ctx, tx, a, id)
		return 201, c, err
	})
}

// answerChallenge: accept or decline (the addressee), cancel (the author).
func (s *Server) answerChallenge(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, ok := s.authed(w, r)
		if !ok {
			return
		}
		id, err := idParam(r)
		if err != nil {
			failErr(w, err)
			return
		}
		s.command(w, r, a, func(ctx context.Context, tx pgx.Tx, _ []byte) (int, any, error) {
			now := s.now()
			c, err := loadChallenge(ctx, tx, a, id)
			if err != nil {
				return 0, nil, err
			}
			next := map[string]string{"accept": "accepted", "decline": "declined", "cancel": "cancelled"}[action]
			if action == "cancel" {
				if c.FromID != a.ID {
					return 0, nil, refuse(403, "forbidden", "Отменить вызов может только его автор.")
				}
				if !c.live(now) || c.Status == "declined" {
					return 0, nil, refuse(409, "challenge_closed", "Вызов уже закрыт.")
				}
			} else {
				if c.ToID == nil || *c.ToID != a.ID {
					return 0, nil, refuse(403, "forbidden", "Ответить на вызов может только тот, кому он брошен.")
				}
				if c.Status != "pending" || !c.live(now) {
					return 0, nil, refuse(409, "challenge_closed", "Вызов уже закрыт.")
				}
			}
			if _, err = tx.Exec(ctx, `UPDATE friendly_challenges SET status=$2 WHERE id=$1`, id, next); err != nil {
				return 0, nil, err
			}
			if action == "accept" {
				if err = auditUser(ctx, tx, c.FromID, "challenge.accepted", "challenge", id, fmt.Sprintf("%s принял(а) ваш вызов: можно выходить в бой.", a.Name)); err != nil {
					return 0, nil, err
				}
			}
			c.Status = next
			return 200, c, nil
		})
	}
}

// challengeCity: the opponent's city for a challenge the actor may answer now.
func (s *Server) challengeCity(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		failErr(w, err)
		return
	}
	var out map[string]any
	err = s.inTx(r.Context(), func(tx pgx.Tx) error {
		c, err := loadChallenge(r.Context(), tx, a, id)
		if err != nil {
			return err
		}
		def, name, err := c.defenderFor(a, s.now())
		if err != nil {
			return err
		}
		if out, err = cityDTO(r.Context(), tx, def); err != nil {
			return err
		}
		c2, err := loadCity(r.Context(), tx, def)
		if err != nil {
			return err
		}
		out["defense"] = game.DefenseOf(c2)
		out["opponent_id"] = def
		out["challenge"] = c
		out["opponent_name"] = name
		return nil
	})
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, out)
}

// attackChallenge: the friendly duel a challenge allows; a direct challenge is used up by it.
func (s *Server) attackChallenge(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		failErr(w, err)
		return
	}
	s.command(w, r, a, func(ctx context.Context, tx pgx.Tx, raw []byte) (int, any, error) {
		var in struct {
			Army map[string]int `json:"army"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		c, err := loadChallenge(ctx, tx, a, id)
		if err != nil {
			return 0, nil, err
		}
		def, name, err := c.defenderFor(a, s.now())
		if err != nil {
			return 0, nil, err
		}
		battleID, err := s.fightFriendly(ctx, tx, a, def, name, in.Army, &c.ID)
		if err != nil {
			return 0, nil, err
		}
		if !c.Open {
			if _, err = tx.Exec(ctx, `UPDATE friendly_challenges SET status='done' WHERE id=$1`, c.ID); err != nil {
				return 0, nil, err
			}
		}
		out, err := battleDTO(ctx, tx, a, battleID)
		return 201, out, err
	})
}
