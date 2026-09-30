package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"kanban-game/server/internal/game"
)

type rewardResult struct {
	LedgerID      int64  `json:"ledger_id"`
	Gold          int64  `json:"gold"`
	Energy        int64  `json:"energy"`
	XP            int64  `json:"xp"`
	GoldFormula   int64  `json:"gold_formula"`
	EnergyFormula int64  `json:"energy_formula"`
	OnTime        bool   `json:"on_time"`
	Points        int    `json:"points"`
	Assignee      string `json:"assignee"`
	Formula       string `json:"formula"`
}

// approveTask confirms a task on review: it pays the assignee once, adds the points to the work rating and
// burns the scope, all in the caller's transaction. at is the confirmation time (seeding uses past dates).
func approveTask(ctx context.Context, tx pgx.Tx, a actor, t task, at time.Time) (rewardResult, error) {
	var out rewardResult
	if !a.leadOf(t.TeamID) {
		return out, refuse(403, "forbidden", "Подтверждать выполнение может только руководитель команды.")
	}
	if t.AssigneeID == nil {
		return out, refuse(409, "assignee_required", "У задачи нет исполнителя.")
	}
	if *t.AssigneeID == a.ID {
		return out, refuse(403, "self_approval", "Нельзя подтвердить собственную задачу. Попросите другого руководителя.")
	}
	if !t.PointsApproved {
		return out, refuse(409, "estimate_not_agreed", "Сложность задачи не согласована.")
	}
	settings, err := loadSettings(ctx, tx)
	if err != nil {
		return out, err
	}
	var due *time.Time
	if t.DueDate != nil {
		d, _ := time.Parse("2006-01-02", *t.DueDate)
		due = &d
	}
	rw := game.TaskReward(settings, t.Points, t.Priority, due, at)
	assignee := *t.AssigneeID
	if err = lockPlayers(ctx, tx, assignee); err != nil {
		return out, err
	}
	bal, err := loadBalance(ctx, tx, assignee)
	if err != nil {
		return out, err
	}
	city, err := loadCity(ctx, tx, assignee)
	if err != nil {
		return out, err
	}
	capG, capE := city.Caps()
	gold := game.Credit(bal.Gold, rw.Gold, capG)
	energy := game.Credit(bal.Energy, rw.Energy, capE)
	note := fmt.Sprintf("%s: %d балл. × %s", t.Number, t.Points, t.Priority)
	if rw.OnTime {
		note += " + бонус за срок"
	}
	if gold < rw.Gold || energy < rw.Energy {
		note += "; часть награды не поместилась в хранилище"
	}
	id, err := applyLedger(ctx, tx, ledgerEntry{UserID: assignee, Kind: "task_reward", Gold: gold, Energy: energy, XP: rw.XP,
		GoldFormula: &rw.Gold, EnergyFormula: &rw.Energy, TaskID: t.ID, ActorID: a.ID, Note: note, At: &at})
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" {
		// The unique live-reward index is the last guard against a double payment.
		return out, refuse(409, "already_rewarded", "Награда за эту задачу уже начислена.")
	}
	if err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO work_points_events(user_id,team_id,project_id,task_id,points,ledger_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		assignee, t.TeamID, t.ProjectID, t.ID, t.Points, id, at); err != nil {
		return out, err
	}
	if err = scope(ctx, tx, t, "completed", -t.Points, at); err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `UPDATE tasks SET state='done', approved_by=$2, approved_at=$3 WHERE id=$1`, t.ID, a.ID, at); err != nil {
		return out, err
	}
	msg := fmt.Sprintf("Награда: +%d золота, +%d энергии, +%d опыта", gold, energy, rw.XP)
	if err = history(ctx, tx, t.ID, a.ID, "approved", sp(t.State), sp("done"), msg, at); err != nil {
		return out, err
	}
	name := ""
	if t.AssigneeName != nil {
		name = *t.AssigneeName
	}
	if err = audit(ctx, tx, a.ID, t.TeamID, "task.approved", "task", t.ID, fmt.Sprintf("%s подтверждена: %s получает %d балл. и %+d золота", t.Number, name, t.Points, gold)); err != nil {
		return out, err
	}
	return rewardResult{LedgerID: id, Gold: gold, Energy: energy, XP: rw.XP, GoldFormula: rw.Gold, EnergyFormula: rw.Energy, OnTime: rw.OnTime,
		Points: t.Points, Assignee: name,
		Formula: fmt.Sprintf("%v × %d × %v%s", settings.GoldPerPoint, t.Points, settings.PriorityMult[t.Priority], map[bool]string{true: fmt.Sprintf(" × %v", 1+settings.OnTimeBonus), false: ""}[rw.OnTime])}, nil
}

type reversalResult struct {
	Gold   int64 `json:"gold"`
	Energy int64 `json:"energy"`
	XP     int64 `json:"xp"`
	Points int   `json:"points"`
	Debt   bool  `json:"debt"`
}

// reopenTask returns a confirmed task to work. The reward is cancelled by a reversal row for exactly what was credited;
// if it was already spent the balance goes negative (debt): spending is blocked and the next income repays it first.
func reopenTask(ctx context.Context, tx pgx.Tx, a actor, t task, to, comment string, at time.Time) (reversalResult, error) {
	var out reversalResult
	if comment == "" {
		return out, refuse(422, "comment_required", "Укажите причину возврата: исполнитель увидит её в карточке.")
	}
	var lid int64
	var user string
	var gold, energy, xp int64
	err := tx.QueryRow(ctx, `SELECT id,user_id::text,gold,energy,xp FROM ledger WHERE task_id=$1 AND kind='task_reward' AND NOT reversed FOR UPDATE`, t.ID).
		Scan(&lid, &user, &gold, &energy, &xp)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if err == nil {
		if err = lockPlayers(ctx, tx, user); err != nil {
			return out, err
		}
		if _, err = tx.Exec(ctx, `UPDATE ledger SET reversed=true WHERE id=$1`, lid); err != nil {
			return out, err
		}
		rid, err := applyLedger(ctx, tx, ledgerEntry{UserID: user, Kind: "task_reward_reversal", Gold: -gold, Energy: -energy, XP: -xp,
			TaskID: t.ID, ActorID: a.ID, Note: t.Number + ": задача возвращена — " + comment, ReversesID: &lid, At: &at})
		if err != nil {
			return out, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO work_points_events(user_id,team_id,project_id,task_id,points,ledger_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`,
			user, t.TeamID, t.ProjectID, t.ID, -t.Points, rid, at); err != nil {
			return out, err
		}
		bal, err := loadBalance(ctx, tx, user)
		if err != nil {
			return out, err
		}
		out = reversalResult{Gold: gold, Energy: energy, XP: xp, Points: t.Points, Debt: bal.Gold < 0 || bal.Energy < 0}
	}
	if err = scope(ctx, tx, t, "reopened", t.Points, at); err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `UPDATE tasks SET state=$2, approved_by=NULL, approved_at=NULL WHERE id=$1`, t.ID, to); err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO task_comments(task_id,author_id,body,created_at) VALUES($1,$2,$3,$4)`, t.ID, a.ID, "Возвращено из «Выполнено»: "+comment, at); err != nil {
		return out, err
	}
	note := fmt.Sprintf("Награда отменена: −%d золота, −%d энергии, −%d опыта, −%d балл. рейтинга", out.Gold, out.Energy, out.XP, out.Points)
	if out.Debt {
		note += ". Ресурсы уже были потрачены: баланс ушёл в минус и будет погашен следующими наградами"
	}
	if err = history(ctx, tx, t.ID, a.ID, "reopened", sp("done"), sp(to), note, at); err != nil {
		return out, err
	}
	return out, audit(ctx, tx, a.ID, t.TeamID, "task.reopened", "task", t.ID, t.Number+" возвращена в «"+stateNames[to]+"»: награда отменена")
}
