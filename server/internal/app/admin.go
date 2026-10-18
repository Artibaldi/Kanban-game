package app

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"slices"

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
		var oldTeam *string
		var wasActive bool
		if err := tx.QueryRow(ctx, `SELECT team_id::text, active FROM users WHERE id=$1`, id).Scan(&oldTeam, &wasActive); err != nil {
			return 0, nil, err
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET role=$2, team_id=$3, active=$4 WHERE id=$1`, id, role, team, active); err != nil {
			return 0, nil, err
		}
		// Leaving the team or being switched off closes the user's challenges: nobody fights a former colleague.
		if (wasActive && !active) || (oldTeam == nil) != (team == nil) || (oldTeam != nil && team != nil && *oldTeam != *team) {
			if err := closeChallengesOf(ctx, tx, id); err != nil {
				return 0, nil, err
			}
			// Troops lent by or to them go home.
			if err := returnReinforcements(ctx, tx, id, false); err != nil {
				return 0, nil, err
			}
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
	if _, err = tx.Exec(ctx, `INSERT INTO buildings(user_id,type,level,x,y) VALUES($1,'townhall',1,$2,$2)`, userID, game.MaxGrid/2-1); err != nil {
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

type teamSettingsDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	WorkDays  []int  `json:"work_days"`
	WaveWeeks int    `json:"wave_weeks"`
}

func loadTeamSettings(ctx context.Context, q dbtx, id string) ([]teamSettingsDTO, error) {
	rows, err := q.Query(ctx, `SELECT id::text, name, work_days, wave_weeks FROM teams WHERE ($1='' OR id::text=$1) ORDER BY name`, id)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (teamSettingsDTO, error) {
		var t teamSettingsDTO
		var days []int16
		err := row.Scan(&t.ID, &t.Name, &days, &t.WaveWeeks)
		t.WorkDays = make([]int, len(days))
		for i, d := range days {
			t.WorkDays[i] = int(d)
		}
		return t, err
	})
}

// adminTeams lists every team's working days and wave length.
func (s *Server) adminTeams(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.adminOnly(w, r); !ok {
		return
	}
	list, err := loadTeamSettings(r.Context(), s.db, "")
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": list})
}

// adminPatchTeam changes a team's working days (no fortress damage on other days) and wave length.
// A new length applies from the next wave: the open wave keeps its end.
func (s *Server) adminPatchTeam(w http.ResponseWriter, r *http.Request) {
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
			WorkDays  *[]int `json:"work_days"`
			WaveWeeks *int   `json:"wave_weeks"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		if in.WorkDays != nil {
			seen := map[int]bool{}
			for _, d := range *in.WorkDays {
				if d < 1 || d > 7 || seen[d] {
					return 0, nil, refuse(422, "invalid_work_days", "Рабочие дни — дни недели от 1 (пн) до 7 (вс), без повторов.")
				}
				seen[d] = true
			}
			if len(seen) == 0 {
				return 0, nil, refuse(422, "invalid_work_days", "Выберите хотя бы один рабочий день.")
			}
			days := slices.Sorted(maps.Keys(seen))
			if _, err := tx.Exec(ctx, `UPDATE teams SET work_days=$2 WHERE id=$1`, id, days); err != nil {
				return 0, nil, err
			}
		}
		if in.WaveWeeks != nil {
			if *in.WaveWeeks < 1 || *in.WaveWeeks > 4 {
				return 0, nil, refuse(422, "invalid_wave_weeks", "Длина волны — от 1 до 4 недель.")
			}
			if _, err := tx.Exec(ctx, `UPDATE teams SET wave_weeks=$2 WHERE id=$1`, id, *in.WaveWeeks); err != nil {
				return 0, nil, err
			}
		}
		list, err := loadTeamSettings(ctx, tx, id)
		if err != nil {
			return 0, nil, err
		}
		if len(list) == 0 {
			return 0, nil, refuse(404, "not_found", "Команда не найдена.")
		}
		if err := audit(ctx, tx, a.ID, id, "admin.team", "team", id, "Изменены рабочие дни и длина волны команды «"+list[0].Name+"»"); err != nil {
			return 0, nil, err
		}
		return 200, list[0], nil
	})
}
