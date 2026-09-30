package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// actor is the signed-in user as the server sees them for one request.
type actor struct {
	ID       string
	Name     string
	Login    string
	Role     string // user | lead | admin
	TeamID   string // empty for an administrator without a team
	TeamName string
	GameMode bool
}

func (a actor) lead() bool  { return a.Role == "lead" || a.Role == "admin" }
func (a actor) admin() bool { return a.Role == "admin" }

// leadOf: a team lead manages their own team; an administrator manages every team.
func (a actor) leadOf(teamID string) bool {
	return a.admin() || a.Role == "lead" && a.TeamID == teamID
}

// sees: members see their own team; an administrator sees all teams.
func (a actor) sees(teamID string) bool { return a.admin() || a.TeamID == teamID }

func digest(token string) []byte { sum := sha256.Sum256([]byte(token)); return sum[:] }

// current resolves the session cookie. DEMO: sessions are created by picking a user, without a password.
func (s *Server) current(r *http.Request) (actor, error) {
	var a actor
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return a, refuse(401, "login_required", "Войдите, чтобы продолжить.")
	}
	var team, teamName *string
	err = s.db.QueryRow(r.Context(), `SELECT u.id::text,u.display_name,u.login,u.role,u.team_id::text,t.name,u.game_mode
 FROM sessions s JOIN users u ON u.id=s.user_id LEFT JOIN teams t ON t.id=u.team_id
 WHERE s.token_digest=$1 AND s.revoked_at IS NULL AND s.expires_at>now() AND u.active`, digest(c.Value)).
		Scan(&a.ID, &a.Name, &a.Login, &a.Role, &team, &teamName, &a.GameMode)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, refuse(401, "login_required", "Сессия закончилась. Войдите снова.")
	}
	if err != nil {
		return a, err
	}
	if team != nil {
		a.TeamID = *team
	}
	if teamName != nil {
		a.TeamName = *teamName
	}
	return a, nil
}

// authed wraps a handler that needs a signed-in user.
func (s *Server) authed(w http.ResponseWriter, r *http.Request) (actor, bool) {
	a, err := s.current(r)
	if err != nil {
		failErr(w, err)
		return a, false
	}
	return a, true
}

func (s *Server) demoUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT u.login,u.display_name,u.role,COALESCE(t.name,''),u.title FROM users u LEFT JOIN teams t ON t.id=u.team_id
 WHERE u.active ORDER BY t.name NULLS FIRST, CASE u.role WHEN 'admin' THEN 0 WHEN 'lead' THEN 1 ELSE 2 END, u.display_name`)
	if err != nil {
		failErr(w, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		var login, name, role, team, title string
		err := row.Scan(&login, &name, &role, &team, &title)
		return map[string]any{"login": login, "display_name": name, "role": role, "team": team, "title": title}, err
	})
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": out, "notice": "Демонстрационный вход без пароля. Только для локального прототипа."})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	raw, err := readBody(r)
	if err != nil {
		failErr(w, err)
		return
	}
	var in struct {
		Login string `json:"login"`
	}
	if err = decodeInto(raw, &in); err != nil {
		failErr(w, err)
		return
	}
	var id string
	err = s.db.QueryRow(r.Context(), `SELECT id::text FROM users WHERE login=$1 AND active`, in.Login).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 401, "invalid_login", "Такого демо-пользователя нет.")
		return
	}
	if err != nil {
		failErr(w, err)
		return
	}
	token := randomToken(43)
	if _, err = s.db.Exec(r.Context(), `INSERT INTO sessions(token_digest,user_id,expires_at) VALUES($1,$2,now()+interval '12 hours')`, digest(token), id); err != nil {
		failErr(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 12 * 3600})
	// A new CSRF token for the new identity.
	csrf := randomToken(43)
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: csrf, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 12 * 3600})
	respond(w, 200, map[string]any{"ok": true, "csrf_token": csrf})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		if _, err = s.db.Exec(r.Context(), `UPDATE sessions SET revoked_at=now() WHERE token_digest=$1`, digest(c.Value)); err != nil {
			failErr(w, err)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, MaxAge: -1, Expires: time.Unix(0, 0)})
	respond(w, 200, map[string]any{"ok": true})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	respond(w, 200, meDTO(a))
}

func meDTO(a actor) map[string]any {
	return map[string]any{
		"id": a.ID, "display_name": a.Name, "login": a.Login, "role": a.Role,
		"team_id": nullable(a.TeamID), "team_name": a.TeamName, "game_mode": a.GameMode,
		"plays": a.TeamID != "", "demo_auth": true,
	}
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	raw, err := readBody(r)
	if err != nil {
		failErr(w, err)
		return
	}
	var in struct {
		GameMode *bool `json:"game_mode"`
	}
	if err = decodeInto(raw, &in); err != nil {
		failErr(w, err)
		return
	}
	if in.GameMode != nil {
		if _, err = s.db.Exec(r.Context(), `UPDATE users SET game_mode=$2 WHERE id=$1`, a.ID, *in.GameMode); err != nil {
			failErr(w, err)
			return
		}
		a.GameMode = *in.GameMode
	}
	respond(w, 200, meDTO(a))
}

// lockPlayers locks game rows in a fixed order (by id) so two concurrent battles cannot deadlock.
func lockPlayers(ctx context.Context, tx pgx.Tx, ids ...string) error {
	_, err := tx.Exec(ctx, `SELECT user_id FROM players WHERE user_id = ANY($1::uuid[]) ORDER BY user_id FOR UPDATE`, ids)
	return err
}
