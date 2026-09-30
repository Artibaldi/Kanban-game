package app

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"

	"kanban-game/server/internal/game"
)

func (s *Server) adminOnly(w http.ResponseWriter, r *http.Request) (actor, bool) {
	a, ok := s.authed(w, r)
	if !ok {
		return a, false
	}
	if !a.admin() {
		fail(w, 403, "forbidden", "Раздел доступен администратору.")
		return a, false
	}
	return a, true
}

func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.adminOnly(w, r); !ok {
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT u.id::text,u.login,u.display_name,u.role,u.team_id::text,COALESCE(t.name,''),u.title,u.active
 FROM users u LEFT JOIN teams t ON t.id=u.team_id ORDER BY t.name NULLS FIRST,u.display_name`)
	if err != nil {
		failErr(w, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		var id, login, name, role, teamName, title string
		var team *string
		var active bool
		err := row.Scan(&id, &login, &name, &role, &team, &teamName, &title, &active)
		return map[string]any{"id": id, "login": login, "display_name": name, "role": role, "team_id": team, "team_name": teamName, "title": title, "active": active}, err
	})
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": out})
}

// adminPatchUser changes role, team or activity. A user who joins a team gets a player and a town hall.
func (s *Server) adminPatchUser(w http.ResponseWriter, r *http.Request) {
	a, ok := s.adminOnly(w, r)
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
			Role   *string  `json:"role"`
			TeamID **string `json:"team_id"`
			Active *bool    `json:"active"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		var role string
		var team *string
		var active bool
		if err := tx.QueryRow(ctx, `SELECT role,team_id::text,active FROM users WHERE id=$1 FOR UPDATE`, id).Scan(&role, &team, &active); err != nil {
			return 0, nil, err
		}
		if in.Role != nil {
			if *in.Role != "user" && *in.Role != "lead" && *in.Role != "admin" {
				return 0, nil, refuse(422, "invalid_fields", "Роль: user, lead или admin.")
			}
			role = *in.Role
		}
		if in.TeamID != nil {
			team = *in.TeamID
			if team != nil && (*team == "" || !validUUID(*team)) {
				team = nil
			}
		}
		if in.Active != nil {
			active = *in.Active
		}
		if role != "admin" && team == nil {
			return 0, nil, refuse(422, "team_required", "Пользователю и руководителю нужна команда.")
		}
		if id == a.ID && (role != "admin" || !active) {
			return 0, nil, refuse(409, "self_lockout", "Нельзя снять собственный доступ администратора.")
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET role=$2, team_id=$3, active=$4 WHERE id=$1`, id, role, team, active); err != nil {
			return 0, nil, err
		}
		if team != nil {
			if err := ensurePlayer(ctx, tx, id); err != nil {
				return 0, nil, err
			}
		}
		if err := audit(ctx, tx, a.ID, "", "admin.user_changed", "user", id, "Изменены роль/команда/активность"); err != nil {
			return 0, nil, err
		}
		return 200, map[string]any{"ok": true}, nil
	})
}

// ensurePlayer creates the game state of a team member once: start balance and a level-1 town hall in the centre.
func ensurePlayer(ctx context.Context, tx pgx.Tx, userID string) error {
	tag, err := tx.Exec(ctx, `INSERT INTO players(user_id) VALUES($1) ON CONFLICT DO NOTHING`, userID)
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	st, err := loadSettings(ctx, tx)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO buildings(user_id,type,level,x,y) VALUES($1,'townhall',1,$2,$2)`, userID, game.GridSize/2-1); err != nil {
		return err
	}
	_, err = applyLedger(ctx, tx, ledgerEntry{UserID: userID, Kind: "seed", Gold: st.StartGold, Energy: st.StartEnergy, Note: "Стартовый запас города"})
	return err
}

func (s *Server) adminSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.adminOnly(w, r); !ok {
		return
	}
	st, err := loadSettings(r.Context(), s.db)
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"settings": st, "defaults": game.DefaultSettings()})
}

func (s *Server) adminPutSettings(w http.ResponseWriter, r *http.Request) {
	a, ok := s.adminOnly(w, r)
	if !ok {
		return
	}
	s.command(w, r, a, func(ctx context.Context, tx pgx.Tx, raw []byte) (int, any, error) {
		in, err := loadSettings(ctx, tx)
		if err != nil {
			return 0, nil, err
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		in = in.Normalize()
		body, _ := json.Marshal(in)
		if _, err := tx.Exec(ctx, `INSERT INTO settings(key,value) VALUES('economy',$1) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value, updated_at=now()`, body); err != nil {
			return 0, nil, err
		}
		if err := audit(ctx, tx, a.ID, "", "admin.settings", "settings", "economy", "Изменены коэффициенты экономики"); err != nil {
			return 0, nil, err
		}
		return 200, map[string]any{"settings": in}, nil
	})
}
