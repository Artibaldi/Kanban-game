// Package app is the HTTP API: work (tasks, ratings) and game (city, army, battles).
// Every rule is enforced here on the server; the web client only displays what the server decided.
package app

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	db      *pgxpool.Pool
	origins map[string]bool
	static  string
	// now is replaceable in tests.
	now func() time.Time
}

type Config struct {
	Origins   []string
	StaticDir string
}

func New(pool *pgxpool.Pool, cfg Config) *Server {
	s := &Server{db: pool, origins: map[string]bool{}, static: cfg.StaticDir, now: time.Now}
	for _, o := range cfg.Origins {
		s.origins[strings.TrimSpace(o)] = true
	}
	return s
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.guardMutation)
		r.Get("/csrf", s.csrf)
		r.Get("/demo-users", s.demoUsers)
		r.Post("/sessions", s.login)
		r.Post("/sessions/logout", s.logout)
		r.Get("/me", s.me)
		r.Patch("/me/settings", s.updateSettings)
		r.Post("/me/game-rules-ack", s.ackGameRules)
		r.Get("/game/config", s.gameConfig)

		r.Get("/teams", s.listTeams)
		r.Get("/projects", s.listProjects)
		r.Get("/users", s.listTeamUsers)
		r.Get("/tasks", s.listTasks)
		r.Post("/tasks", s.createTask)
		r.Get("/tasks/{id}", s.getTask)
		r.Patch("/tasks/{id}", s.patchTask)
		r.Post("/tasks/{id}/transition", s.transitionTask)
		r.Post("/tasks/{id}/estimate", s.approveEstimate)
		r.Post("/tasks/{id}/comments", s.addComment)
		r.Get("/review", s.reviewQueue)

		r.Get("/player", s.player)
		r.Get("/ledger", s.ledger)
		r.Get("/events", s.events)
		r.Get("/progress/me", s.myProgress)
		r.Get("/progress/team", s.teamProgress)
		r.Get("/burndown", s.burndown)
		r.Get("/fortress", s.fortress)
		r.Get("/waves", s.waves)

		r.Get("/city", s.city)
		r.Post("/city/buildings", s.placeBuilding)
		r.Post("/city/buildings/{id}/upgrade", s.upgradeBuilding)
		r.Get("/army", s.army)
		r.Post("/army/train", s.train)
		r.Get("/duels", s.duels)
		r.Get("/duels/{id}/city", s.duelCity)
		r.Post("/battles/friendly", s.friendlyBattle)
		r.Get("/battles/{id}/city", s.battleCity)
		r.Get("/challenges", s.listChallenges)
		r.Post("/challenges", s.createChallenge)
		r.Post("/challenges/{id}/accept", s.answerChallenge("accept"))
		r.Post("/challenges/{id}/decline", s.answerChallenge("decline"))
		r.Post("/challenges/{id}/cancel", s.answerChallenge("cancel"))
		r.Get("/challenges/{id}/city", s.challengeCity)
		r.Post("/challenges/{id}/attack", s.attackChallenge)
		r.Get("/battles", s.listBattles)
		r.Get("/battles/{id}", s.getBattle)

		r.Get("/admin/users", s.adminUsers)
		r.Patch("/admin/users/{id}", s.adminPatchUser)
		r.Get("/admin/settings", s.adminSettings)
		r.Put("/admin/settings", s.adminPutSettings)
		r.Get("/admin/teams", s.adminTeams)
		r.Patch("/admin/teams/{id}", s.adminPatchTeam)
	})
	if s.static != "" {
		r.Get("/*", s.spa)
	}
	return r
}

// spa serves the built web client; unknown paths get index.html so client routes survive a reload.
func (s *Server) spa(w http.ResponseWriter, r *http.Request) {
	clean := filepath.Clean("/" + r.URL.Path)
	path := filepath.Join(s.static, clean)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		http.ServeFile(w, r, path)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.static, "index.html"))
}

// ---- responses and errors ----

// apiError is a refusal with a stable code for the client and a Russian message for people.
type apiError struct {
	Status  int
	Code    string
	Message string
}

func (e *apiError) Error() string { return e.Message }

func refuse(status int, code, message string) error {
	return &apiError{Status: status, Code: code, Message: message}
}

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func fail(w http.ResponseWriter, status int, code, message string) {
	respond(w, status, map[string]any{"code": code, "message": message})
}

// failErr maps an error to a response: refusals keep their status; database trouble is 503; the rest is 500.
func failErr(w http.ResponseWriter, err error) {
	var ae *apiError
	if errors.As(err, &ae) {
		fail(w, ae.Status, ae.Code, ae.Message)
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "not_found", "Запись не найдена или недоступна.")
		return
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "40P01" {
		fail(w, 409, "busy", "Сервер занят. Повторите действие.")
		return
	}
	log.Printf("internal error: %v", err)
	fail(w, 503, "unavailable", "Сервис временно недоступен. Попробуйте ещё раз.")
}

// readBody reads the JSON body once; the raw bytes also serve as the idempotency digest input.
func readBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return []byte("{}"), nil
	}
	if ct := r.Header.Get("Content-Type"); r.ContentLength != 0 && !strings.HasPrefix(ct, "application/json") {
		return nil, refuse(415, "content_type", "Нужен JSON.")
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, refuse(400, "invalid_body", "Не удалось прочитать запрос.")
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		raw = []byte("{}")
	}
	return raw, nil
}

func decodeInto(raw []byte, out any) error {
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return refuse(422, "invalid_json", "Некорректные данные запроса.")
	}
	return nil
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	out := make([]byte, n)
	for i, v := range b {
		out[i] = alphabet[int(v)%len(alphabet)]
	}
	return string(out)
}

// ---- CSRF and origin (pattern from Devops-pilot guardMutation) ----

const (
	sessionCookieName = "kg_session"
	csrfCookieName    = "kg_csrf"
)

func (s *Server) csrf(w http.ResponseWriter, r *http.Request) {
	token := randomToken(43)
	http.SetCookie(w, &http.Cookie{Name: csrfCookieName, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 12 * 3600})
	respond(w, 200, map[string]string{"csrf_token": token})
}

func (s *Server) guardMutation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if !s.origins[r.Header.Get("Origin")] {
			fail(w, 403, "invalid_origin", "Запрос с этого адреса не разрешён.")
			return
		}
		cookie, err := r.Cookie(csrfCookieName)
		value := r.Header.Get("X-CSRF-Token")
		if err != nil || len(value) < 32 || subtle.ConstantTimeCompare([]byte(value), []byte(cookie.Value)) != 1 {
			fail(w, 403, "csrf_required", "Обновите страницу и повторите действие.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---- transactions ----

// inTx runs fn in one transaction and commits only when it returns nil.
func (s *Server) inTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func audit(ctx context.Context, tx pgx.Tx, actorID, teamID, action, targetType, targetID, summary string) error {
	_, err := tx.Exec(ctx, `INSERT INTO audit_events(actor_id,team_id,action,target_type,target_id,summary) VALUES($1,$2,$3,$4,$5,$6)`,
		nullable(actorID), nullable(teamID), action, targetType, nullable(targetID), summary)
	return err
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func validUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i, c := range v {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
				return false
			}
		}
	}
	return true
}

func idParam(r *http.Request) (string, error) {
	id := chi.URLParam(r, "id")
	if !validUUID(id) {
		return "", refuse(404, "not_found", "Запись не найдена.")
	}
	return id, nil
}

func plural(n int64, one, few, many string) string {
	a := n
	if a < 0 {
		a = -a
	}
	switch {
	case a%10 == 1 && a%100 != 11:
		return fmt.Sprintf("%d %s", n, one)
	case a%10 >= 2 && a%10 <= 4 && (a%100 < 10 || a%100 >= 20):
		return fmt.Sprintf("%d %s", n, few)
	default:
		return fmt.Sprintf("%d %s", n, many)
	}
}
