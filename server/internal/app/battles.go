package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"kanban-game/server/internal/game"
)

// duelist is a teammate who accepts friendly duels.
type duelist struct {
	UserID    string       `json:"user_id"`
	Name      string       `json:"display_name"`
	TownHall  int          `json:"townhall"`
	Available bool         `json:"available"`
	Reason    string       `json:"reason"`
	Defense   game.Defense `json:"defense"`
}

// friendlyToday counts the duels the player started since midnight UTC.
func friendlyToday(ctx context.Context, q dbtx, userID string, now time.Time) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT count(*) FROM battles WHERE attacker_id=$1 AND kind='friendly' AND created_at >= $2`,
		userID, now.UTC().Truncate(24*time.Hour)).Scan(&n)
	return n, err
}

// duelists lists the actor's teammates who accept friendly duels; only, when set, narrows it to one user.
func duelists(ctx context.Context, q dbtx, a actor, st game.Settings, only string, now time.Time) ([]duelist, int, error) {
	today, err := friendlyToday(ctx, q, a.ID, now)
	if err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx, `SELECT u.id::text, u.display_name FROM users u JOIN players p ON p.user_id=u.id
 WHERE u.active AND u.accept_friendly AND u.id<>$1 AND u.team_id=$2 AND ($3='' OR u.id::text=$3) ORDER BY u.display_name`, a.ID, a.TeamID, only)
	if err != nil {
		return nil, 0, err
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (duelist, error) {
		var d duelist
		err := row.Scan(&d.UserID, &d.Name)
		return d, err
	})
	if err != nil {
		return nil, 0, err
	}
	for i := range list {
		c, err := loadCity(ctx, q, list[i].UserID)
		if err != nil {
			return nil, 0, err
		}
		list[i].TownHall = c.TownHall()
		list[i].Defense = game.DefenseOf(c)
		if today >= st.FriendlyPerDay {
			list[i].Reason = "Лимит дружеских боёв на сегодня исчерпан"
		} else {
			list[i].Available = true
		}
	}
	return list, today, nil
}

func (s *Server) duels(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	if a.TeamID == "" {
		fail(w, 403, "not_a_player", "Игра доступна участникам команд.")
		return
	}
	ctx := r.Context()
	if err := s.settleNow(ctx, ""); err != nil {
		failErr(w, err)
		return
	}
	st, err := loadSettings(ctx, s.db)
	if err != nil {
		failErr(w, err)
		return
	}
	list, today, err := duelists(ctx, s.db, a, st, "", s.now())
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": list, "today": today, "per_day": st.FriendlyPerDay, "friendly_xp": st.FriendlyXP})
}

// duelCity shows a duelist's layout (buildings only; never their tasks or balances).
func (s *Server) duelCity(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		failErr(w, err)
		return
	}
	ctx := r.Context()
	st, err := loadSettings(ctx, s.db)
	if err != nil {
		failErr(w, err)
		return
	}
	list, _, err := duelists(ctx, s.db, a, st, id, s.now())
	if err != nil {
		failErr(w, err)
		return
	}
	if len(list) == 0 {
		fail(w, 404, "not_found", "Коллега не найден или не принимает дружеские бои.")
		return
	}
	c, err := cityDTO(ctx, s.db, id)
	if err != nil {
		failErr(w, err)
		return
	}
	c["opponent"] = list[0]
	respond(w, 200, c)
}

// friendlyBattle: a duel with a teammate who accepts them. It costs nothing, kills nobody, moves no trophies;
// the attacker gets a little experience. A few per day.
func (s *Server) friendlyBattle(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	s.command(w, r, a, func(ctx context.Context, tx pgx.Tx, raw []byte) (int, any, error) {
		var in struct {
			DefenderID string         `json:"defender_id"`
			Army       map[string]int `json:"army"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		if !validUUID(in.DefenderID) || in.DefenderID == a.ID {
			return 0, nil, refuse(422, "invalid_opponent", "Выберите коллегу для дружеского боя.")
		}
		if a.TeamID == "" {
			return 0, nil, refuse(403, "not_a_player", "Игра доступна участникам команд.")
		}
		if err := settle(ctx, tx, a.ID); err != nil {
			return 0, nil, err
		}
		if err := lockPlayers(ctx, tx, a.ID); err != nil {
			return 0, nil, err
		}
		st, err := loadSettings(ctx, tx)
		if err != nil {
			return 0, nil, err
		}
		list, today, err := duelists(ctx, tx, a, st, in.DefenderID, s.now())
		if err != nil {
			return 0, nil, err
		}
		if len(list) == 0 {
			return 0, nil, refuse(404, "not_found", "Коллега не найден или не принимает дружеские бои.")
		}
		if today >= st.FriendlyPerDay {
			return 0, nil, refuse(429, "friendly_limit", fmt.Sprintf("Сегодня уже было %d дружеских боя — это лимит. Завтра можно снова.", today))
		}
		opp := list[0]
		troops, _, err := armyOf(ctx, tx, a.ID)
		if err != nil {
			return 0, nil, err
		}
		army := map[string]int{}
		total := 0
		for k, n := range in.Army {
			if _, ok := game.Troops[k]; !ok || n < 0 {
				return 0, nil, refuse(422, "invalid_army", "Неизвестный тип войск.")
			}
			if n > troops[k] {
				return 0, nil, refuse(409, "not_enough_troops", fmt.Sprintf("Готово только %d: %s.", troops[k], game.Troops[k].Name))
			}
			if n > 0 {
				army[k] = n
				total += n
			}
		}
		if total == 0 {
			return 0, nil, refuse(422, "empty_army", "Выберите войска для боя.")
		}
		defCity, err := loadCity(ctx, tx, in.DefenderID)
		if err != nil {
			return 0, nil, err
		}
		defense := game.DefenseOf(defCity)
		seed := rand.Int64N(1 << 53) // fits a JavaScript number exactly
		outcome := game.Fight(army, defense, seed, st.RandomSpread)
		armyJSON, _ := json.Marshal(army)
		lossJSON, _ := json.Marshal(outcome.Losses)
		defJSON, _ := json.Marshal(defense)
		roundsJSON, _ := json.Marshal(outcome.Rounds)
		var battleID string
		if err = tx.QueryRow(ctx, `INSERT INTO battles(kind,attacker_id,defender_id,seed,army,losses,defense,rounds,destruction,stars,gold_reward,xp_reward,attacker_trophies,defender_trophies)
 VALUES('friendly',$1,$2,$3,$4,$5,$6,$7,$8,$9,0,$10,0,0) RETURNING id::text`,
			a.ID, in.DefenderID, seed, armyJSON, lossJSON, defJSON, roundsJSON, outcome.Destruction, outcome.Stars, st.FriendlyXP).Scan(&battleID); err != nil {
			return 0, nil, err
		}
		if _, err = applyLedger(ctx, tx, ledgerEntry{UserID: a.ID, Kind: "battle_reward", XP: st.FriendlyXP, BattleID: battleID,
			Note: "Дружеский бой с " + opp.Name}); err != nil {
			return 0, nil, err
		}
		if err = audit(ctx, tx, a.ID, a.TeamID, "battle.friendly", "battle", battleID, fmt.Sprintf("Дружеский бой с %s: %d%% разрушений", opp.Name, outcome.Destruction)); err != nil {
			return 0, nil, err
		}
		if err = auditUser(ctx, tx, in.DefenderID, "battle.friendly_defended", "battle", battleID, fmt.Sprintf("Дружеский бой: на ваш город вышел(а) %s. Ничего не потеряно.", a.Name)); err != nil {
			return 0, nil, err
		}
		out, err := battleDTO(ctx, tx, a, battleID)
		return 201, out, err
	})
}

func battleDTO(ctx context.Context, q dbtx, a actor, id string) (map[string]any, error) {
	var kind, att, def, attName, defName string
	var seed int64
	var army, losses, defense, rounds []byte
	var destruction, stars, at, dt int
	var gold, xp int64
	var created time.Time
	err := q.QueryRow(ctx, `SELECT b.kind,b.attacker_id::text,b.defender_id::text,ua.display_name,ud.display_name,b.seed,b.army,b.losses,b.defense,b.rounds,
 b.destruction,b.stars,b.gold_reward,b.xp_reward,b.attacker_trophies,b.defender_trophies,b.created_at
 FROM battles b JOIN users ua ON ua.id=b.attacker_id JOIN users ud ON ud.id=b.defender_id WHERE b.id=$1`, id).
		Scan(&kind, &att, &def, &attName, &defName, &seed, &army, &losses, &defense, &rounds, &destruction, &stars, &gold, &xp, &at, &dt, &created)
	if err != nil {
		return nil, err
	}
	if !a.admin() && a.ID != att && a.ID != def {
		return nil, refuse(404, "not_found", "Бой не найден.")
	}
	return map[string]any{"id": id, "kind": kind, "attacker_id": att, "defender_id": def, "attacker": attName, "defender": defName, "seed": seed,
		"army": json.RawMessage(army), "losses": json.RawMessage(losses), "defense": json.RawMessage(defense), "rounds": json.RawMessage(rounds),
		"destruction": destruction, "stars": stars, "gold": gold, "xp": xp, "attacker_trophies": at, "defender_trophies": dt, "created_at": created}, nil
}

func (s *Server) getBattle(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	id, err := idParam(r)
	if err != nil {
		failErr(w, err)
		return
	}
	out, err := battleDTO(r.Context(), s.db, a, id)
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, out)
}

func (s *Server) listBattles(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT b.id::text,b.kind,b.attacker_id::text,ua.display_name,ud.display_name,b.destruction,b.stars,b.gold_reward,b.attacker_trophies,b.defender_trophies,b.created_at
 FROM battles b JOIN users ua ON ua.id=b.attacker_id JOIN users ud ON ud.id=b.defender_id
 WHERE b.attacker_id=$1 OR b.defender_id=$1 ORDER BY b.created_at DESC LIMIT 50`, a.ID)
	if err != nil {
		failErr(w, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		var id, kind, att, attName, defName string
		var destruction, stars, at, dt int
		var gold int64
		var created time.Time
		err := row.Scan(&id, &kind, &att, &attName, &defName, &destruction, &stars, &gold, &at, &dt, &created)
		return map[string]any{"id": id, "kind": kind, "role": map[bool]string{true: "attack", false: "defense"}[att == a.ID], "attacker": attName, "defender": defName,
			"destruction": destruction, "stars": stars, "gold": gold, "attacker_trophies": at, "defender_trophies": dt, "created_at": created}, err
	})
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": out})
}
