package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"kanban-game/server/internal/game"
)

var stateNames = map[string]string{
	"backlog": "Бэклог", "ready": "К выполнению", "in_progress": "В работе", "review": "На проверке", "done": "Выполнено",
}

type rewardDTO struct {
	Gold   int64 `json:"gold"`
	Energy int64 `json:"energy"`
	XP     int64 `json:"xp"`
}

type task struct {
	ID               string     `json:"id"`
	Number           string     `json:"number"`
	ProjectID        string     `json:"project_id"`
	ProjectKey       string     `json:"project_key"`
	ProjectName      string     `json:"project_name"`
	TeamID           string     `json:"team_id"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	State            string     `json:"state"`
	AssigneeID       *string    `json:"assignee_id"`
	AssigneeName     *string    `json:"assignee_name"`
	Priority         string     `json:"priority"`
	Points           int        `json:"points"`
	PointsApproved   bool       `json:"points_approved"`
	PointsApprovedBy *string    `json:"points_approved_by"`
	DueDate          *string    `json:"due_date"`
	Rank             int64      `json:"rank"`
	ApprovedBy       *string    `json:"approved_by"`
	ApprovedAt       *time.Time `json:"approved_at"`
	Version          int64      `json:"version"`
	CreatedByID      string     `json:"created_by_id"`
	CreatedBy        string     `json:"created_by"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Reward           *rewardDTO `json:"reward"`
}

const taskSelect = `SELECT t.id::text,t.number,t.project_id::text,p.key,p.name,p.team_id::text,t.title,t.description,t.state,
 t.assignee_id::text,ua.display_name,t.priority,t.points,t.points_approved_at IS NOT NULL,upa.display_name,
 to_char(t.due_date,'YYYY-MM-DD'),t.rank,uab.display_name,t.approved_at,t.version,t.created_by::text,uc.display_name,t.created_at,t.updated_at,
 lr.gold,lr.energy,lr.xp
 FROM tasks t JOIN projects p ON p.id=t.project_id
 LEFT JOIN users ua ON ua.id=t.assignee_id
 LEFT JOIN users upa ON upa.id=t.points_approved_by
 LEFT JOIN users uab ON uab.id=t.approved_by
 JOIN users uc ON uc.id=t.created_by
 LEFT JOIN ledger lr ON lr.task_id=t.id AND lr.kind='task_reward' AND NOT lr.reversed`

func scanTask(row pgx.Row) (task, error) {
	var t task
	var g, e, x *int64
	err := row.Scan(&t.ID, &t.Number, &t.ProjectID, &t.ProjectKey, &t.ProjectName, &t.TeamID, &t.Title, &t.Description, &t.State,
		&t.AssigneeID, &t.AssigneeName, &t.Priority, &t.Points, &t.PointsApproved, &t.PointsApprovedBy,
		&t.DueDate, &t.Rank, &t.ApprovedBy, &t.ApprovedAt, &t.Version, &t.CreatedByID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
		&g, &e, &x)
	if g != nil {
		t.Reward = &rewardDTO{Gold: *g, Energy: *e, XP: *x}
	}
	return t, err
}

// loadTask reads one task the actor may see; forUpdate locks the task row for the rest of the transaction.
func loadTask(ctx context.Context, q dbtx, a actor, id string, forUpdate bool) (task, error) {
	sql := taskSelect + ` WHERE t.id=$1`
	if forUpdate {
		sql += ` FOR UPDATE OF t`
	}
	t, err := scanTask(q.QueryRow(ctx, sql, id))
	if err != nil {
		return t, err
	}
	if !a.sees(t.TeamID) {
		return t, refuse(404, "not_found", "Задача не найдена или недоступна.")
	}
	return t, nil
}

func history(ctx context.Context, tx pgx.Tx, taskID, actorID, kind string, from, to *string, note string, at time.Time) error {
	_, err := tx.Exec(ctx, `INSERT INTO task_history(task_id,actor_id,kind,from_value,to_value,note,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		taskID, nullable(actorID), kind, from, to, note, at)
	return err
}

func scope(ctx context.Context, tx pgx.Tx, t task, kind string, delta int, at time.Time) error {
	if delta == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `INSERT INTO scope_events(team_id,project_id,task_id,kind,delta,created_at) VALUES($1,$2,$3,$4,$5,$6)`,
		t.TeamID, t.ProjectID, t.ID, kind, delta, at)
	return err
}

func sp(v string) *string { return &v }

// ---- reference lists ----

func (s *Server) listTeams(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT id::text,name FROM teams WHERE $1 OR id::text=$2 ORDER BY name`, a.admin(), a.TeamID)
	if err != nil {
		failErr(w, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		var id, name string
		err := row.Scan(&id, &name)
		return map[string]any{"id": id, "name": name}, err
	})
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": out})
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT p.id::text,p.key,p.name,p.team_id::text,t.name,
 COALESCE((SELECT SUM(points) FROM tasks WHERE project_id=p.id AND state<>'done'),0)
 FROM projects p JOIN teams t ON t.id=p.team_id WHERE $1 OR p.team_id::text=$2 ORDER BY t.name,p.key`, a.admin(), a.TeamID)
	if err != nil {
		failErr(w, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		var id, key, name, team, teamName string
		var open int64
		err := row.Scan(&id, &key, &name, &team, &teamName, &open)
		return map[string]any{"id": id, "key": key, "name": name, "team_id": team, "team_name": teamName, "open_points": open}, err
	})
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": out})
}

func (s *Server) listTeamUsers(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT u.id::text,u.display_name,u.role,u.team_id::text,u.title FROM users u
 WHERE u.active AND u.team_id IS NOT NULL AND ($1 OR u.team_id::text=$2) ORDER BY u.display_name`, a.admin(), a.TeamID)
	if err != nil {
		failErr(w, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		var id, name, role, team, title string
		err := row.Scan(&id, &name, &role, &team, &title)
		return map[string]any{"id": id, "display_name": name, "role": role, "team_id": team, "title": title}, err
	})
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": out})
}

// ---- tasks ----

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	project := q.Get("project")
	if project != "" && !validUUID(project) {
		project = "00000000-0000-0000-0000-000000000000"
	}
	assignee := q.Get("assignee")
	if assignee == "me" {
		assignee = a.ID
	}
	if assignee != "" && !validUUID(assignee) {
		assignee = "00000000-0000-0000-0000-000000000000"
	}
	rows, err := s.db.Query(r.Context(), taskSelect+`
 WHERE ($1 OR p.team_id::text=$2) AND ($3='' OR t.project_id::text=$3) AND ($4='' OR t.assignee_id::text=$4)
 ORDER BY t.rank, t.created_at`, a.admin(), a.TeamID, project, assignee)
	if err != nil {
		failErr(w, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (task, error) { return scanTask(row) })
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": out})
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		failErr(w, err)
		return
	}
	out, err := taskDetail(r.Context(), s.db, a, id)
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, out)
}

// taskDetail is the card: the task, its comments, its history and its reward journal.
func taskDetail(ctx context.Context, q dbtx, a actor, id string) (map[string]any, error) {
	t, err := loadTask(ctx, q, a, id, false)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT c.id::text,u.display_name,c.body,c.created_at FROM task_comments c JOIN users u ON u.id=c.author_id WHERE c.task_id=$1 ORDER BY c.created_at`, id)
	if err != nil {
		return nil, err
	}
	comments, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		var cid, author, body string
		var at time.Time
		err := row.Scan(&cid, &author, &body, &at)
		return map[string]any{"id": cid, "author": author, "body": body, "created_at": at}, err
	})
	if err != nil {
		return nil, err
	}
	rows, err = q.Query(ctx, `SELECT h.id,COALESCE(u.display_name,'Система'),h.kind,h.from_value,h.to_value,h.note,h.created_at FROM task_history h LEFT JOIN users u ON u.id=h.actor_id WHERE h.task_id=$1 ORDER BY h.id DESC`, id)
	if err != nil {
		return nil, err
	}
	hist, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		var hid int64
		var actor, kind, note string
		var from, to *string
		var at time.Time
		err := row.Scan(&hid, &actor, &kind, &from, &to, &note, &at)
		return map[string]any{"id": hid, "actor": actor, "kind": kind, "from": from, "to": to, "note": note, "created_at": at}, err
	})
	if err != nil {
		return nil, err
	}
	rows, err = q.Query(ctx, `SELECT l.id,l.kind,l.gold,l.energy,l.xp,l.gold_formula,l.energy_formula,l.reversed,l.reverses_id,u.display_name,l.note,l.created_at
 FROM ledger l JOIN users u ON u.id=l.user_id WHERE l.task_id=$1 ORDER BY l.id`, id)
	if err != nil {
		return nil, err
	}
	journal, err := pgx.CollectRows(rows, scanLedger)
	if err != nil {
		return nil, err
	}
	return map[string]any{"task": t, "comments": comments, "history": hist, "rewards": journal}, nil
}

func scanLedger(row pgx.CollectableRow) (map[string]any, error) {
	var id int64
	var kind, who, note string
	var gold, energy, xp int64
	var gf, ef, rev *int64
	var reversed bool
	var at time.Time
	err := row.Scan(&id, &kind, &gold, &energy, &xp, &gf, &ef, &reversed, &rev, &who, &note, &at)
	return map[string]any{"id": id, "kind": kind, "gold": gold, "energy": energy, "xp": xp, "gold_formula": gf, "energy_formula": ef,
		"reversed": reversed, "reverses_id": rev, "user": who, "note": note, "created_at": at}, err
}

type taskInput struct {
	ProjectID   string  `json:"project_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Priority    string  `json:"priority"`
	Points      int     `json:"points"`
	DueDate     *string `json:"due_date"`
	AssigneeID  *string `json:"assignee_id"`
	State       string  `json:"state"`
}

func validPriority(p string) bool { return p == "P1" || p == "P2" || p == "P3" || p == "P4" }

func parseDue(v *string) (*time.Time, error) {
	if v == nil || *v == "" {
		return nil, nil
	}
	d, err := time.Parse("2006-01-02", *v)
	if err != nil {
		return nil, refuse(422, "invalid_fields", "Срок должен быть датой в формате ГГГГ-ММ-ДД.")
	}
	return &d, nil
}

// checkAssignee: the assignee must be an active member of the task's team.
func checkAssignee(ctx context.Context, tx pgx.Tx, teamID string, id *string) error {
	if id == nil || *id == "" {
		return nil
	}
	if !validUUID(*id) {
		return refuse(422, "invalid_assignee", "Исполнитель должен быть участником команды.")
	}
	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM users WHERE id=$1 AND team_id::text=$2 AND active`, *id, teamID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return refuse(422, "invalid_assignee", "Исполнитель должен быть участником команды.")
	}
	return nil
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	s.command(w, r, a, func(ctx context.Context, tx pgx.Tx, raw []byte) (int, any, error) {
		var in taskInput
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		in.Title = strings.TrimSpace(in.Title)
		if in.Priority == "" {
			in.Priority = "P3"
		}
		if in.State == "" {
			in.State = "backlog"
		}
		if in.Title == "" || len([]rune(in.Title)) > 200 || !validPriority(in.Priority) || !validUUID(in.ProjectID) {
			return 0, nil, refuse(422, "invalid_fields", "Проверьте название, проект и приоритет.")
		}
		if !game.ValidPoints(in.Points) {
			return 0, nil, refuse(422, "invalid_points", "Сложность — одно из значений 1, 2, 3, 5, 8.")
		}
		if in.State != "backlog" && in.State != "ready" {
			return 0, nil, refuse(422, "invalid_state", "Новая задача создаётся в бэклоге или в колонке «К выполнению».")
		}
		due, err := parseDue(in.DueDate)
		if err != nil {
			return 0, nil, err
		}
		var teamID, key string
		var n int
		err = tx.QueryRow(ctx, `UPDATE projects SET next_number=next_number+1 WHERE id=$1 RETURNING team_id::text,key,next_number-1`, in.ProjectID).Scan(&teamID, &key, &n)
		if errors.Is(err, pgx.ErrNoRows) || err == nil && !a.sees(teamID) {
			return 0, nil, refuse(404, "not_found", "Проект не найден или недоступен.")
		}
		if err != nil {
			return 0, nil, err
		}
		if err = checkAssignee(ctx, tx, teamID, in.AssigneeID); err != nil {
			return 0, nil, err
		}
		if in.AssigneeID != nil && *in.AssigneeID != "" && *in.AssigneeID != a.ID && !a.leadOf(teamID) {
			return 0, nil, refuse(403, "forbidden", "Назначать задачи другим может только руководитель.")
		}
		// A lead's estimate is agreed at creation; a member's estimate waits for the lead.
		var approvedBy any
		if a.leadOf(teamID) {
			approvedBy = a.ID
		}
		var id string
		err = tx.QueryRow(ctx, `INSERT INTO tasks(number,project_id,title,description,state,assignee_id,priority,points,points_approved_by,points_approved_at,due_date,rank,created_by)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::uuid,CASE WHEN $9::uuid IS NULL THEN NULL ELSE now() END,$10,
  COALESCE((SELECT MAX(rank) FROM tasks WHERE project_id=$2 AND state=$5),0)+1024,$11) RETURNING id::text`,
			key+"-"+itoa(n), in.ProjectID, in.Title, strings.TrimSpace(in.Description), in.State, emptyNil(in.AssigneeID), in.Priority, in.Points, approvedBy, due, a.ID).Scan(&id)
		if err != nil {
			return 0, nil, err
		}
		t, err := loadTask(ctx, tx, a, id, false)
		if err != nil {
			return 0, nil, err
		}
		now := time.Now()
		note := "Сложность " + itoa(in.Points) + " — ждёт согласования руководителем"
		if approvedBy != nil {
			note = "Сложность " + itoa(in.Points) + " согласована при создании"
		}
		if err = history(ctx, tx, id, a.ID, "created", nil, sp(in.State), note, now); err != nil {
			return 0, nil, err
		}
		if err = scope(ctx, tx, t, "added", in.Points, now); err != nil {
			return 0, nil, err
		}
		if err = audit(ctx, tx, a.ID, teamID, "task.created", "task", id, t.Number+" «"+t.Title+"»"); err != nil {
			return 0, nil, err
		}
		return 201, t, nil
	})
}

func emptyNil(v *string) any {
	if v == nil || *v == "" {
		return nil
	}
	return *v
}

// patchTask edits fields. Complexity rules: a member may change points only until the lead agrees them;
// after that only a lead may re-estimate (logged); a completed task's points are locked until it is reopened.
func (s *Server) patchTask(w http.ResponseWriter, r *http.Request) {
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
			Title       *string  `json:"title"`
			Description *string  `json:"description"`
			Priority    *string  `json:"priority"`
			Points      *int     `json:"points"`
			DueDate     **string `json:"due_date"`
			AssigneeID  **string `json:"assignee_id"`
			Version     *int64   `json:"version"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		t, err := loadTask(ctx, tx, a, id, true)
		if err != nil {
			return 0, nil, err
		}
		if in.Version != nil && *in.Version != t.Version {
			return 0, nil, refuse(409, "stale_task", "Задача изменилась. Обновите карточку.")
		}
		lead := a.leadOf(t.TeamID)
		mine := t.AssigneeID != nil && *t.AssigneeID == a.ID || t.CreatedByID == a.ID
		now := time.Now()
		if _, err = ensureWave(ctx, tx, t.TeamID, s.now()); err != nil {
			return 0, nil, err
		}
		changed := false
		if in.Title != nil || in.Description != nil || in.Priority != nil || in.DueDate != nil {
			if !lead && !mine {
				return 0, nil, refuse(403, "forbidden", "Изменять задачу могут исполнитель, автор или руководитель.")
			}
			if t.State == "done" && (in.Priority != nil || in.DueDate != nil) {
				return 0, nil, refuse(409, "task_locked", "Задача подтверждена: приоритет и срок влияют на награду и не меняются.")
			}
		}
		if in.Title != nil {
			v := strings.TrimSpace(*in.Title)
			if v == "" || len([]rune(v)) > 200 {
				return 0, nil, refuse(422, "invalid_fields", "Название обязательно (до 200 символов).")
			}
			if v != t.Title {
				if _, err = tx.Exec(ctx, `UPDATE tasks SET title=$2 WHERE id=$1`, id, v); err != nil {
					return 0, nil, err
				}
				if err = history(ctx, tx, id, a.ID, "title", sp(t.Title), sp(v), "", now); err != nil {
					return 0, nil, err
				}
				changed = true
			}
		}
		if in.Description != nil && strings.TrimSpace(*in.Description) != t.Description {
			if _, err = tx.Exec(ctx, `UPDATE tasks SET description=$2 WHERE id=$1`, id, strings.TrimSpace(*in.Description)); err != nil {
				return 0, nil, err
			}
			if err = history(ctx, tx, id, a.ID, "description", nil, nil, "Описание изменено", now); err != nil {
				return 0, nil, err
			}
			changed = true
		}
		if in.Priority != nil && *in.Priority != t.Priority {
			if !validPriority(*in.Priority) {
				return 0, nil, refuse(422, "invalid_fields", "Приоритет — P1…P4.")
			}
			if _, err = tx.Exec(ctx, `UPDATE tasks SET priority=$2 WHERE id=$1`, id, *in.Priority); err != nil {
				return 0, nil, err
			}
			if err = history(ctx, tx, id, a.ID, "priority", sp(t.Priority), in.Priority, "", now); err != nil {
				return 0, nil, err
			}
			changed = true
		}
		if in.DueDate != nil {
			due, err := parseDue(*in.DueDate)
			if err != nil {
				return 0, nil, err
			}
			var newDue *string
			if due != nil {
				newDue = sp(due.Format("2006-01-02"))
			}
			if !eqStr(newDue, t.DueDate) {
				if _, err = tx.Exec(ctx, `UPDATE tasks SET due_date=$2 WHERE id=$1`, id, due); err != nil {
					return 0, nil, err
				}
				if err = history(ctx, tx, id, a.ID, "due_date", t.DueDate, newDue, "", now); err != nil {
					return 0, nil, err
				}
				changed = true
			}
		}
		if in.AssigneeID != nil {
			next := *in.AssigneeID
			if next != nil && *next == "" {
				next = nil
			}
			if !eqStr(next, t.AssigneeID) {
				if t.State == "done" {
					return 0, nil, refuse(409, "task_locked", "Исполнитель подтверждённой задачи не меняется: награда уже начислена ему.")
				}
				// A member may take an unassigned task or give up their own; a lead assigns anyone in the team.
				selfTake := next != nil && *next == a.ID && t.AssigneeID == nil
				selfDrop := next == nil && t.AssigneeID != nil && *t.AssigneeID == a.ID
				if !lead && !selfTake && !selfDrop {
					return 0, nil, refuse(403, "forbidden", "Назначать задачи другим может только руководитель.")
				}
				if err = checkAssignee(ctx, tx, t.TeamID, next); err != nil {
					return 0, nil, err
				}
				if _, err = tx.Exec(ctx, `UPDATE tasks SET assignee_id=$2 WHERE id=$1`, id, next); err != nil {
					return 0, nil, err
				}
				var fromName, toName *string
				fromName = t.AssigneeName
				if next != nil {
					var n string
					if err = tx.QueryRow(ctx, `SELECT display_name FROM users WHERE id=$1`, *next).Scan(&n); err != nil {
						return 0, nil, err
					}
					toName = &n
				}
				if err = history(ctx, tx, id, a.ID, "assignee", fromName, toName, "", now); err != nil {
					return 0, nil, err
				}
				changed = true
			}
		}
		if in.Points != nil && *in.Points != t.Points {
			if err = reestimate(ctx, tx, a, t, *in.Points, now); err != nil {
				return 0, nil, err
			}
			changed = true
		}
		if changed {
			if _, err = tx.Exec(ctx, `UPDATE tasks SET version=version+1, updated_at=now() WHERE id=$1`, id); err != nil {
				return 0, nil, err
			}
		}
		if err = syncWavePlan(ctx, tx, id, s.now()); err != nil {
			return 0, nil, err
		}
		out, err := taskDetail(ctx, tx, a, id)
		return 200, out, err
	})
}

func eqStr(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// reestimate changes the points of an open task under the complexity rules and records the scope change.
func reestimate(ctx context.Context, tx pgx.Tx, a actor, t task, points int, now time.Time) error {
	if !game.ValidPoints(points) {
		return refuse(422, "invalid_points", "Сложность — одно из значений 1, 2, 3, 5, 8.")
	}
	if t.State == "done" {
		return refuse(409, "points_locked", "Задача подтверждена: сложность зафиксирована. Чтобы изменить её, руководитель должен вернуть задачу в работу.")
	}
	lead := a.leadOf(t.TeamID)
	if t.PointsApproved && !lead {
		return refuse(403, "points_agreed", "Сложность уже согласована. Изменить её может только руководитель.")
	}
	if !lead && !(t.CreatedByID == a.ID || t.AssigneeID != nil && *t.AssigneeID == a.ID) {
		return refuse(403, "forbidden", "Оценку предлагает автор или исполнитель задачи.")
	}
	// A lead's change is itself the agreement; a member's proposal stays unagreed.
	if lead {
		_, err := tx.Exec(ctx, `UPDATE tasks SET points=$2, points_approved_by=$3, points_approved_at=now() WHERE id=$1`, t.ID, points, a.ID)
		if err != nil {
			return err
		}
	} else if _, err := tx.Exec(ctx, `UPDATE tasks SET points=$2 WHERE id=$1`, t.ID, points); err != nil {
		return err
	}
	note := "Предложена новая оценка"
	if lead {
		note = "Переоценка руководителем"
		if t.State == "in_progress" || t.State == "review" {
			note += " во время работы"
		}
	}
	if err := history(ctx, tx, t.ID, a.ID, "points", sp(itoa(t.Points)), sp(itoa(points)), note, now); err != nil {
		return err
	}
	if err := scope(ctx, tx, t, "reestimated", points-t.Points, now); err != nil {
		return err
	}
	return audit(ctx, tx, a.ID, t.TeamID, "task.reestimated", "task", t.ID, t.Number+": сложность "+itoa(t.Points)+" → "+itoa(points))
}

// approveEstimate: the lead agrees the complexity (optionally changing it) before work starts.
func (s *Server) approveEstimate(w http.ResponseWriter, r *http.Request) {
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
			Points int `json:"points"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		t, err := loadTask(ctx, tx, a, id, true)
		if err != nil {
			return 0, nil, err
		}
		if !a.leadOf(t.TeamID) {
			return 0, nil, refuse(403, "forbidden", "Согласовать сложность может только руководитель команды.")
		}
		now := time.Now()
		if _, err = ensureWave(ctx, tx, t.TeamID, s.now()); err != nil {
			return 0, nil, err
		}
		if in.Points == 0 {
			in.Points = t.Points
		}
		if in.Points != t.Points {
			if err = reestimate(ctx, tx, a, t, in.Points, now); err != nil {
				return 0, nil, err
			}
		} else {
			if t.State == "done" {
				return 0, nil, refuse(409, "points_locked", "Задача уже подтверждена.")
			}
			if t.PointsApproved {
				return 0, nil, refuse(409, "already_agreed", "Сложность уже согласована.")
			}
			if _, err = tx.Exec(ctx, `UPDATE tasks SET points_approved_by=$2, points_approved_at=now() WHERE id=$1`, id, a.ID); err != nil {
				return 0, nil, err
			}
			if err = history(ctx, tx, id, a.ID, "points_agreed", nil, sp(itoa(t.Points)), "Сложность согласована", now); err != nil {
				return 0, nil, err
			}
			if err = audit(ctx, tx, a.ID, t.TeamID, "task.estimate_agreed", "task", id, t.Number+": сложность "+itoa(t.Points)+" согласована"); err != nil {
				return 0, nil, err
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE tasks SET version=version+1, updated_at=now() WHERE id=$1`, id); err != nil {
			return 0, nil, err
		}
		if err = syncWavePlan(ctx, tx, id, s.now()); err != nil {
			return 0, nil, err
		}
		out, err := taskDetail(ctx, tx, a, id)
		return 200, out, err
	})
}

func (s *Server) addComment(w http.ResponseWriter, r *http.Request) {
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
			Body string `json:"body"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		in.Body = strings.TrimSpace(in.Body)
		if in.Body == "" || len([]rune(in.Body)) > 4000 {
			return 0, nil, refuse(422, "invalid_fields", "Комментарий пуст или слишком длинный.")
		}
		if _, err := loadTask(ctx, tx, a, id, false); err != nil {
			return 0, nil, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO task_comments(task_id,author_id,body) VALUES($1,$2,$3)`, id, a.ID, in.Body); err != nil {
			return 0, nil, err
		}
		out, err := taskDetail(ctx, tx, a, id)
		return 201, out, err
	})
}

// transitionTask moves a card between columns (or within one). Moving to «Выполнено» is the lead's confirmation
// and pays the reward; moving out of «Выполнено» reverses it.
func (s *Server) transitionTask(w http.ResponseWriter, r *http.Request) {
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
			To      string `json:"to"`
			Index   *int   `json:"index"`
			Version *int64 `json:"version"`
			Comment string `json:"comment"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		if _, ok := stateNames[in.To]; !ok {
			return 0, nil, refuse(422, "invalid_state", "Неизвестная колонка.")
		}
		t, err := loadTask(ctx, tx, a, id, true)
		if err != nil {
			return 0, nil, err
		}
		if in.Version != nil && *in.Version != t.Version {
			return 0, nil, refuse(409, "stale_task", "Задача изменилась. Обновите доску.")
		}
		out := map[string]any{}
		now := time.Now()
		if _, err = ensureWave(ctx, tx, t.TeamID, s.now()); err != nil {
			return 0, nil, err
		}
		from := t.State
		lead := a.leadOf(t.TeamID)
		comment := strings.TrimSpace(in.Comment)
		switch {
		case from == in.To:
			// Reorder only.
		case in.To == "done":
			if from != "review" {
				return 0, nil, refuse(409, "invalid_transition", "Подтвердить можно только задачу на проверке.")
			}
			rw, err := approveTask(ctx, tx, a, t, now)
			if err != nil {
				return 0, nil, err
			}
			out["reward"] = rw
		case from == "done":
			if !lead {
				return 0, nil, refuse(403, "forbidden", "Вернуть подтверждённую задачу может только руководитель.")
			}
			rv, err := reopenTask(ctx, tx, a, t, in.To, comment, now)
			if err != nil {
				return 0, nil, err
			}
			out["reversal"] = rv
		default:
			assignedToMe := t.AssigneeID != nil && *t.AssigneeID == a.ID
			if !lead && !assignedToMe && t.AssigneeID != nil {
				return 0, nil, refuse(403, "forbidden", "Двигать чужую задачу может только руководитель.")
			}
			if in.To == "in_progress" || in.To == "review" {
				if !t.PointsApproved {
					return 0, nil, refuse(409, "estimate_not_agreed", "Сначала руководитель должен согласовать сложность задачи.")
				}
				if t.AssigneeID == nil {
					if a.TeamID != t.TeamID {
						return 0, nil, refuse(409, "assignee_required", "Сначала назначьте исполнителя.")
					}
					// Taking an unassigned card into work assigns it to whoever took it.
					if _, err = tx.Exec(ctx, `UPDATE tasks SET assignee_id=$2 WHERE id=$1`, id, a.ID); err != nil {
						return 0, nil, err
					}
					if err = history(ctx, tx, id, a.ID, "assignee", nil, sp(a.Name), "Взял(а) задачу", now); err != nil {
						return 0, nil, err
					}
				}
			}
			if from == "review" && in.To == "in_progress" && lead && comment != "" {
				if _, err = tx.Exec(ctx, `INSERT INTO task_comments(task_id,author_id,body) VALUES($1,$2,$3)`, id, a.ID, "Возвращено на доработку: "+comment); err != nil {
					return 0, nil, err
				}
			}
			if _, err = tx.Exec(ctx, `UPDATE tasks SET state=$2 WHERE id=$1`, id, in.To); err != nil {
				return 0, nil, err
			}
			if err = history(ctx, tx, id, a.ID, "state", sp(from), sp(in.To), comment, now); err != nil {
				return 0, nil, err
			}
			if err = audit(ctx, tx, a.ID, t.TeamID, "task.moved", "task", id, t.Number+": "+stateNames[from]+" → "+stateNames[in.To]); err != nil {
				return 0, nil, err
			}
		}
		if err = placeInColumn(ctx, tx, t.ProjectID, id, in.To, in.Index); err != nil {
			return 0, nil, err
		}
		if _, err = tx.Exec(ctx, `UPDATE tasks SET version=version+1, updated_at=now() WHERE id=$1`, id); err != nil {
			return 0, nil, err
		}
		if err = syncWavePlan(ctx, tx, id, s.now()); err != nil {
			return 0, nil, err
		}
		nt, err := loadTask(ctx, tx, a, id, false)
		if err != nil {
			return 0, nil, err
		}
		out["task"] = nt
		return 200, out, nil
	})
}

// placeInColumn puts the task at index within its column (end when nil) and renumbers the column with gaps of 1024.
func placeInColumn(ctx context.Context, tx pgx.Tx, projectID, id, state string, index *int) error {
	rows, err := tx.Query(ctx, `SELECT id::text FROM tasks WHERE project_id=$1 AND state=$2 AND id<>$3 ORDER BY rank,created_at FOR UPDATE`, projectID, state, id)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	at := len(ids)
	if index != nil && *index >= 0 && *index < at {
		at = *index
	}
	ids = append(ids[:at], append([]string{id}, ids[at:]...)...)
	_, err = tx.Exec(ctx, `UPDATE tasks t SET rank=v.ord*1024 FROM unnest($1::uuid[]) WITH ORDINALITY AS v(id,ord) WHERE t.id=v.id AND t.rank<>v.ord*1024`, ids)
	return err
}

// reviewQueue: the lead's panel — tasks on review and estimates waiting for agreement.
func (s *Server) reviewQueue(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	if !a.lead() {
		fail(w, 403, "forbidden", "Панель доступна руководителям.")
		return
	}
	rows, err := s.db.Query(r.Context(), taskSelect+` WHERE ($1 OR p.team_id::text=$2) AND (t.state='review' OR (t.points_approved_at IS NULL AND t.state<>'done'))
 ORDER BY t.state DESC, t.updated_at`, a.admin(), a.TeamID)
	if err != nil {
		failErr(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (task, error) { return scanTask(row) })
	if err != nil {
		failErr(w, err)
		return
	}
	review, estimates := []task{}, []task{}
	for _, t := range items {
		if t.State == "review" {
			review = append(review, t)
		}
		if !t.PointsApproved {
			estimates = append(estimates, t)
		}
	}
	respond(w, 200, map[string]any{"review": review, "estimates": estimates, "me": a.ID})
}
