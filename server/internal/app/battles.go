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

type opponent struct {
	UserID      string       `json:"user_id"`
	Name        string       `json:"display_name"`
	Team        string       `json:"team"`
	TownHall    int          `json:"townhall"`
	Trophies    int          `json:"trophies"`
	ShieldUntil *time.Time   `json:"shield_until"`
	RematchAt   *time.Time   `json:"rematch_at"`
	Available   bool         `json:"available"`
	Reason      string       `json:"reason"`
	Defense     game.Defense `json:"defense"`
}

// candidates lists players in the actor's battle scope with the reason each one can or cannot be attacked now.
func candidates(ctx context.Context, q dbtx, a actor, st game.Settings, only string) ([]opponent, error) {
	myCity, err := loadCity(ctx, q, a.ID)
	if err != nil {
		return nil, err
	}
	myTH := myCity.TownHall()
	rows, err := q.Query(ctx, `SELECT u.id::text,u.display_name,t.name,p.trophies,p.shield_until,
 (SELECT MAX(b.created_at) FROM battles b WHERE b.attacker_id=$1 AND b.defender_id=u.id)
 FROM players p JOIN users u ON u.id=p.user_id JOIN teams t ON t.id=u.team_id
 WHERE u.active AND u.id<>$1 AND ($2 OR u.team_id::text=$3) AND ($4='' OR u.id::text=$4) ORDER BY p.trophies DESC`,
		a.ID, st.BattleScope == "all", a.TeamID, only)
	if err != nil {
		return nil, err
	}
	type raw struct {
		o    opponent
		last *time.Time
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (raw, error) {
		var r raw
		err := row.Scan(&r.o.UserID, &r.o.Name, &r.o.Team, &r.o.Trophies, &r.o.ShieldUntil, &r.last)
		return r, err
	})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := []opponent{}
	for _, r := range list {
		o := r.o
		c, err := loadCity(ctx, q, o.UserID)
		if err != nil {
			return nil, err
		}
		o.TownHall = c.TownHall()
		o.Defense = game.DefenseOf(c)
		if o.ShieldUntil != nil && !o.ShieldUntil.After(now) {
			o.ShieldUntil = nil
		}
		if r.last != nil {
			at := r.last.Add(time.Duration(st.RematchMinutes) * time.Minute)
			if at.After(now) {
				o.RematchAt = &at
			}
		}
		switch {
		case !game.InRange(myTH, o.TownHall):
			o.Reason = fmt.Sprintf("Ратуша %d: подбираются соперники с ратушей %d–%d", o.TownHall, max(1, myTH-1), myTH+1)
		case o.ShieldUntil != nil:
			o.Reason = "Под защитой после недавней атаки"
		case o.RematchAt != nil:
			o.Reason = "Вы недавно атаковали этого игрока"
		default:
			o.Available = true
		}
		out = append(out, o)
	}
	return out, nil
}

func (s *Server) opponents(w http.ResponseWriter, r *http.Request) {
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
	list, err := candidates(ctx, s.db, a, st, "")
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": list, "scope": st.BattleScope, "attack_energy": st.AttackEnergy, "shield_minutes": st.ShieldMinutes, "rematch_minutes": st.RematchMinutes})
}

// opponentCity shows a defender's layout (buildings only; never their tasks or balances).
func (s *Server) opponentCity(w http.ResponseWriter, r *http.Request) {
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
	list, err := candidates(ctx, s.db, a, st, id)
	if err != nil {
		failErr(w, err)
		return
	}
	if len(list) == 0 {
		fail(w, 404, "not_found", "Соперник не найден или недоступен.")
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

func (s *Server) attack(w http.ResponseWriter, r *http.Request) {
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
			return 0, nil, refuse(422, "invalid_opponent", "Выберите соперника.")
		}
		if a.TeamID == "" {
			return 0, nil, refuse(403, "not_a_player", "Игра доступна участникам команд.")
		}
		if err := settle(ctx, tx, a.ID); err != nil {
			return 0, nil, err
		}
		if err := settle(ctx, tx, in.DefenderID); err != nil {
			return 0, nil, err
		}
		// Both players are locked in id order, so two crossing attacks cannot deadlock.
		if err := lockPlayers(ctx, tx, a.ID, in.DefenderID); err != nil {
			return 0, nil, err
		}
		st, err := loadSettings(ctx, tx)
		if err != nil {
			return 0, nil, err
		}
		list, err := candidates(ctx, tx, a, st, in.DefenderID)
		if err != nil {
			return 0, nil, err
		}
		if len(list) == 0 {
			return 0, nil, refuse(404, "not_found", "Соперник не найден или недоступен.")
		}
		opp := list[0]
		if !opp.Available {
			code := "out_of_range"
			if opp.ShieldUntil != nil {
				code = "shielded"
			} else if opp.RematchAt != nil {
				code = "rematch"
			}
			return 0, nil, refuse(409, code, opp.Reason+".")
		}
		b, err := loadBalance(ctx, tx, a.ID)
		if err != nil {
			return 0, nil, err
		}
		if err = debtRefusal(b); err != nil {
			return 0, nil, err
		}
		if b.Energy < st.AttackEnergy {
			return 0, nil, refuse(402, "insufficient_energy", fmt.Sprintf("Для атаки нужно %d энергии, есть %d.", st.AttackEnergy, b.Energy))
		}
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
			return 0, nil, refuse(422, "empty_army", "Выберите войска для атаки.")
		}
		defCity, err := loadCity(ctx, tx, in.DefenderID)
		if err != nil {
			return 0, nil, err
		}
		defense := game.DefenseOf(defCity)
		seed := rand.Int64N(1 << 53) // fits a JavaScript number exactly
		outcome := game.Fight(army, defense, seed, st.RandomSpread)
		reward := game.RewardFor(st, outcome, defense.TownHall)
		myCity, err := loadCity(ctx, tx, a.ID)
		if err != nil {
			return 0, nil, err
		}
		capG, _ := myCity.Caps()
		gold := game.Credit(b.Gold-0, reward.Gold, capG)
		armyJSON, _ := json.Marshal(army)
		lossJSON, _ := json.Marshal(outcome.Losses)
		defJSON, _ := json.Marshal(defense)
		roundsJSON, _ := json.Marshal(outcome.Rounds)
		var battleID string
		if err = tx.QueryRow(ctx, `INSERT INTO battles(attacker_id,defender_id,seed,army,losses,defense,rounds,destruction,stars,gold_reward,xp_reward,attacker_trophies,defender_trophies)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id::text`,
			a.ID, in.DefenderID, seed, armyJSON, lossJSON, defJSON, roundsJSON, outcome.Destruction, outcome.Stars, gold, reward.XP, reward.AttackerTrophies, reward.DefenderTrophies).Scan(&battleID); err != nil {
			return 0, nil, err
		}
		if _, err = applyLedger(ctx, tx, ledgerEntry{UserID: a.ID, Kind: "attack", Energy: -st.AttackEnergy, BattleID: battleID, ActorID: a.ID, Note: "Атака, соперник: " + opp.Name}); err != nil {
			return 0, nil, err
		}
		gf := reward.Gold
		if _, err = applyLedger(ctx, tx, ledgerEntry{UserID: a.ID, Kind: "battle_reward", Gold: gold, XP: reward.XP, GoldFormula: &gf, BattleID: battleID, Note: fmt.Sprintf("Бой (соперник: %s): %d%%, звёзд %d", opp.Name, outcome.Destruction, outcome.Stars)}); err != nil {
			return 0, nil, err
		}
		// Attacking drops your own shield; the defender gets one. The defender loses no resources and no buildings.
		if _, err = tx.Exec(ctx, `UPDATE players SET trophies=GREATEST(0,trophies+$2), shield_until=NULL WHERE user_id=$1`, a.ID, reward.AttackerTrophies); err != nil {
			return 0, nil, err
		}
		if _, err = tx.Exec(ctx, `UPDATE players SET trophies=GREATEST(0,trophies+$2), shield_until=now()+$3*interval '1 minute' WHERE user_id=$1`, in.DefenderID, reward.DefenderTrophies, st.ShieldMinutes); err != nil {
			return 0, nil, err
		}
		for k, n := range outcome.Losses {
			if n > 0 {
				if _, err = tx.Exec(ctx, `UPDATE troops SET count=count-$3 WHERE user_id=$1 AND type=$2`, a.ID, k, n); err != nil {
					return 0, nil, err
				}
			}
		}
		verdict := "поражение"
		if outcome.Stars > 0 {
			verdict = fmt.Sprintf("победа, звёзд: %d", outcome.Stars)
		}
		if err = audit(ctx, tx, a.ID, a.TeamID, "battle.attack", "battle", battleID, fmt.Sprintf("Атака, соперник %s: %d%% разрушений, %s", opp.Name, outcome.Destruction, verdict)); err != nil {
			return 0, nil, err
		}
		if err = auditUser(ctx, tx, in.DefenderID, "battle.defended", "battle", battleID, fmt.Sprintf("На город напал(а) %s: %d%% разрушений. Здания и ресурсы не пострадали, включён щит.", a.Name, outcome.Destruction)); err != nil {
			return 0, nil, err
		}
		out, err := battleDTO(ctx, tx, a, battleID)
		return 201, out, err
	})
}

func battleDTO(ctx context.Context, q dbtx, a actor, id string) (map[string]any, error) {
	var att, def, attName, defName string
	var seed int64
	var army, losses, defense, rounds []byte
	var destruction, stars, at, dt int
	var gold, xp int64
	var created time.Time
	err := q.QueryRow(ctx, `SELECT b.attacker_id::text,b.defender_id::text,ua.display_name,ud.display_name,b.seed,b.army,b.losses,b.defense,b.rounds,
 b.destruction,b.stars,b.gold_reward,b.xp_reward,b.attacker_trophies,b.defender_trophies,b.created_at
 FROM battles b JOIN users ua ON ua.id=b.attacker_id JOIN users ud ON ud.id=b.defender_id WHERE b.id=$1`, id).
		Scan(&att, &def, &attName, &defName, &seed, &army, &losses, &defense, &rounds, &destruction, &stars, &gold, &xp, &at, &dt, &created)
	if err != nil {
		return nil, err
	}
	if !a.admin() && a.ID != att && a.ID != def {
		return nil, refuse(404, "not_found", "Бой не найден.")
	}
	return map[string]any{"id": id, "attacker_id": att, "defender_id": def, "attacker": attName, "defender": defName, "seed": seed,
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
	rows, err := s.db.Query(r.Context(), `SELECT b.id::text,b.attacker_id::text,ua.display_name,ud.display_name,b.destruction,b.stars,b.gold_reward,b.attacker_trophies,b.defender_trophies,b.created_at
 FROM battles b JOIN users ua ON ua.id=b.attacker_id JOIN users ud ON ud.id=b.defender_id
 WHERE b.attacker_id=$1 OR b.defender_id=$1 ORDER BY b.created_at DESC LIMIT 50`, a.ID)
	if err != nil {
		failErr(w, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		var id, att, attName, defName string
		var destruction, stars, at, dt int
		var gold int64
		var created time.Time
		err := row.Scan(&id, &att, &attName, &defName, &destruction, &stars, &gold, &at, &dt, &created)
		return map[string]any{"id": id, "role": map[bool]string{true: "attack", false: "defense"}[att == a.ID], "attacker": attName, "defender": defName,
			"destruction": destruction, "stars": stars, "gold": gold, "attacker_trophies": at, "defender_trophies": dt, "created_at": created}, err
	})
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, map[string]any{"items": out})
}
