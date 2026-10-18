package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"kanban-game/server/internal/game"
)

func (s *Server) gameConfig(w http.ResponseWriter, r *http.Request) {
	st, err := loadSettings(r.Context(), s.db)
	if err != nil {
		failErr(w, err)
		return
	}
	defs := []game.BuildingDef{}
	for _, k := range game.BuildingOrder {
		defs = append(defs, game.Buildings[k])
	}
	troops := []game.TroopDef{}
	for _, k := range game.TroopOrder {
		troops = append(troops, game.Troops[k])
	}
	respond(w, 200, map[string]any{
		"grid": game.MaxGrid, "builders": game.MaxBuilders, "base_gold_cap": game.BaseGoldCap, "base_energy_cap": game.BaseEnergyCap,
		"buildings": defs, "troops": troops, "settings": st, "points_scale": []int{1, 2, 3, 5, 8},
	})
}

// armyOf returns ready troops and the housing of troops still in training.
func armyOf(ctx context.Context, q dbtx, userID string) (map[string]int, int, error) {
	troops := map[string]int{}
	for _, k := range game.TroopOrder {
		troops[k] = 0
	}
	rows, err := q.Query(ctx, `SELECT type,count FROM troops WHERE user_id=$1`, userID)
	if err != nil {
		return nil, 0, err
	}
	for rows.Next() {
		var k string
		var n int
		if err = rows.Scan(&k, &n); err != nil {
			rows.Close()
			return nil, 0, err
		}
		troops[k] = n
	}
	rows.Close()
	queued := map[string]int{}
	rows, err = q.Query(ctx, `SELECT type,SUM(count)::int FROM training_queue WHERE user_id=$1 GROUP BY type`, userID)
	if err != nil {
		return nil, 0, err
	}
	for rows.Next() {
		var k string
		var n int
		if err = rows.Scan(&k, &n); err != nil {
			rows.Close()
			return nil, 0, err
		}
		queued[k] = n
	}
	rows.Close()
	return troops, game.Housing(queued), rows.Err()
}

type cityBuilding struct {
	game.Building
	Name      string          `json:"name"`
	Spec      game.LevelSpec  `json:"spec"`
	Next      *game.LevelSpec `json:"next"`
	StartedAt *time.Time      `json:"started_at"`
	MaxLevel  int             `json:"max_level"`
}

func cityDTO(ctx context.Context, q dbtx, userID string) (map[string]any, error) {
	c, err := loadCity(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	st, err := loadSettings(ctx, q)
	if err != nil {
		return nil, err
	}
	out := []cityBuilding{}
	for _, b := range c.Buildings {
		def := game.Buildings[b.Type]
		spec, _ := def.Spec(b.Level)
		cb := cityBuilding{Building: b, Name: def.Name, Spec: spec, MaxLevel: def.MaxLevel()}
		if n, ok := def.Spec(b.Level + 1); ok {
			cb.Next = &n
		}
		if b.Busy() && b.TargetLevel != nil {
			if target, ok := def.Spec(*b.TargetLevel); ok {
				started := b.BusyUntil.Add(-game.BuildDuration(st, target))
				cb.StartedAt = &started
			}
		}
		out = append(out, cb)
	}
	th := c.TownHall()
	limits := map[string]map[string]int{}
	for k, def := range game.Buildings {
		limits[k] = map[string]int{"built": c.Count(k), "allowed": def.Allowed(th)}
	}
	start, size := game.Land(th)
	return map[string]any{"buildings": out, "limits": limits, "townhall": th, "grid": game.MaxGrid, "land": map[string]int{"start": start, "size": size}}, nil
}

// rivalCityDTO is a city as another player sees it before a battle: the traps are hidden.
func rivalCityDTO(ctx context.Context, q dbtx, userID string) (map[string]any, error) {
	out, err := cityDTO(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	shown := []cityBuilding{}
	for _, b := range out["buildings"].([]cityBuilding) {
		if b.Type != "trap" {
			shown = append(shown, b)
		}
	}
	out["buildings"] = shown
	delete(out["limits"].(map[string]map[string]int), "trap")
	return out, nil
}

func (s *Server) city(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	if err := s.settleNow(r.Context(), a.ID); err != nil {
		failErr(w, err)
		return
	}
	c, err := cityDTO(r.Context(), s.db, a.ID)
	if err != nil {
		failErr(w, err)
		return
	}
	p, err := playerState(r.Context(), s.db, a.ID)
	if err != nil {
		failErr(w, err)
		return
	}
	c["player"] = p
	c["server_time"] = time.Now()
	respond(w, 200, c)
}

// gameTx starts a game action: settle timers and lock the player row so balance checks and spending are atomic.
func gameTx(ctx context.Context, tx pgx.Tx, a actor) (balance, game.City, game.Settings, error) {
	if a.TeamID == "" {
		return balance{}, game.City{}, game.Settings{}, refuse(403, "not_a_player", "Игра доступна участникам команд.")
	}
	if err := settle(ctx, tx, a.ID); err != nil {
		return balance{}, game.City{}, game.Settings{}, err
	}
	if err := lockPlayers(ctx, tx, a.ID); err != nil {
		return balance{}, game.City{}, game.Settings{}, err
	}
	b, err := loadBalance(ctx, tx, a.ID)
	if err != nil {
		return b, game.City{}, game.Settings{}, err
	}
	c, err := loadCity(ctx, tx, a.ID)
	if err != nil {
		return b, c, game.Settings{}, err
	}
	st, err := loadSettings(ctx, tx)
	return b, c, st, err
}

func asAPI(r *game.Refusal) error {
	status := 409
	if r.Code == "insufficient_gold" || r.Code == "insufficient_energy" {
		status = 402
	}
	return refuse(status, r.Code, r.Message)
}

func debtRefusal(b balance) error {
	if b.Gold < 0 || b.Energy < 0 {
		return refuse(402, "debt", "Баланс отрицательный после отмены награды. Тратить ресурсы можно после погашения долга следующими наградами.")
	}
	return nil
}

func (s *Server) placeBuilding(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	s.command(w, r, a, func(ctx context.Context, tx pgx.Tx, raw []byte) (int, any, error) {
		var in struct {
			Type string `json:"type"`
			X    int    `json:"x"`
			Y    int    `json:"y"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		b, c, st, err := gameTx(ctx, tx, a)
		if err != nil {
			return 0, nil, err
		}
		if err = debtRefusal(b); err != nil {
			return 0, nil, err
		}
		spec, rf := game.CheckPlace(c, in.Type, in.X, in.Y, b.Gold)
		if rf != nil {
			return 0, nil, asAPI(rf)
		}
		var id string
		if game.Buildings[in.Type].Decor {
			// Decor is ready at once and needs no builder.
			err = tx.QueryRow(ctx, `INSERT INTO buildings(user_id,type,level,x,y) VALUES($1,$2,1,$3,$4) RETURNING id::text`, a.ID, in.Type, in.X, in.Y).Scan(&id)
		} else {
			dur := game.BuildDuration(st, spec)
			err = tx.QueryRow(ctx, `INSERT INTO buildings(user_id,type,level,target_level,busy_until,x,y) VALUES($1,$2,0,1,now()+$3*interval '1 millisecond',$4,$5) RETURNING id::text`,
				a.ID, in.Type, dur.Milliseconds(), in.X, in.Y).Scan(&id)
		}
		if err != nil {
			return 0, nil, err
		}
		name := game.Buildings[in.Type].Name
		if _, err = applyLedger(ctx, tx, ledgerEntry{UserID: a.ID, Kind: "build", Gold: -spec.Cost, BuildingID: id, ActorID: a.ID, Note: name + ": постройка"}); err != nil {
			return 0, nil, err
		}
		summary := fmt.Sprintf("%s: строится (%d золота)", name, spec.Cost)
		if game.Buildings[in.Type].Decor {
			summary = fmt.Sprintf("%s: поставлено (%d золота)", name, spec.Cost)
		}
		if err = audit(ctx, tx, a.ID, a.TeamID, "building.started", "building", id, summary); err != nil {
			return 0, nil, err
		}
		out, err := cityDTO(ctx, tx, a.ID)
		if err != nil {
			return 0, nil, err
		}
		out["player"], err = playerState(ctx, tx, a.ID)
		return 201, out, err
	})
}

func (s *Server) upgradeBuilding(w http.ResponseWriter, r *http.Request) {
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
		b, c, st, err := gameTx(ctx, tx, a)
		if err != nil {
			return 0, nil, err
		}
		if err = debtRefusal(b); err != nil {
			return 0, nil, err
		}
		var target *game.Building
		for i := range c.Buildings {
			if c.Buildings[i].ID == id {
				target = &c.Buildings[i]
			}
		}
		if target == nil {
			return 0, nil, refuse(404, "not_found", "Здание не найдено.")
		}
		level, _, _ := game.Level(st, b.XP)
		spec, rf := game.CheckUpgrade(c, *target, b.Gold, level)
		if rf != nil {
			return 0, nil, asAPI(rf)
		}
		dur := game.BuildDuration(st, spec)
		if _, err = tx.Exec(ctx, `UPDATE buildings SET target_level=$2, busy_until=now()+$3*interval '1 millisecond' WHERE id=$1`, id, spec.Level, dur.Milliseconds()); err != nil {
			return 0, nil, err
		}
		name := game.Buildings[target.Type].Name
		if _, err = applyLedger(ctx, tx, ledgerEntry{UserID: a.ID, Kind: "upgrade", Gold: -spec.Cost, BuildingID: id, ActorID: a.ID, Note: fmt.Sprintf("%s: улучшение до уровня %d", name, spec.Level)}); err != nil {
			return 0, nil, err
		}
		if err = audit(ctx, tx, a.ID, a.TeamID, "building.upgrade", "building", id, fmt.Sprintf("%s: улучшение до уровня %d", name, spec.Level)); err != nil {
			return 0, nil, err
		}
		out, err := cityDTO(ctx, tx, a.ID)
		if err != nil {
			return 0, nil, err
		}
		out["player"], err = playerState(ctx, tx, a.ID)
		return 200, out, err
	})
}

func armyDTO(ctx context.Context, q dbtx, userID string) (map[string]any, error) {
	troops, _, err := armyOf(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT id::text,type,count,started_at,finish_at FROM training_queue WHERE user_id=$1 ORDER BY finish_at`, userID)
	if err != nil {
		return nil, err
	}
	queue, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		var id, kind string
		var count int
		var started, finish time.Time
		err := row.Scan(&id, &kind, &count, &started, &finish)
		return map[string]any{"id": id, "type": kind, "count": count, "started_at": started, "finish_at": finish}, err
	})
	if err != nil {
		return nil, err
	}
	p, err := playerState(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	levels, research, err := troopLevels(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	c, err := loadCity(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	var res any
	if research != nil {
		res = research
	}
	// Why a troop type cannot be trained yet, for the cards.
	locked := map[string]string{}
	for _, k := range game.TroopOrder {
		if rf := game.CanTrain(c, k); rf != nil {
			locked[k] = rf.Message
		}
	}
	return map[string]any{"troops": troops, "queue": queue, "player": p, "server_time": time.Now(), "locked": locked,
		"levels": levels, "research": res, "max_troop_level": game.MaxTroopLevel(c), "research_costs": game.Research}, nil
}

// troopResearch is the workshop job in progress.
type troopResearch struct {
	Type        string    `json:"type"`
	TargetLevel int       `json:"target_level"`
	StartedAt   time.Time `json:"started_at"`
	FinishAt    time.Time `json:"finish_at"`
}

// troopLevels: every troop type's level (1 without a row) and the research in progress, if any.
func troopLevels(ctx context.Context, q dbtx, userID string) (map[string]int, *troopResearch, error) {
	levels := map[string]int{}
	for _, k := range game.TroopOrder {
		levels[k] = 1
	}
	rows, err := q.Query(ctx, `SELECT type, level, target_level, started_at, busy_until FROM troop_levels WHERE user_id=$1`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var research *troopResearch
	for rows.Next() {
		var kind string
		var level int
		var target *int
		var started, until *time.Time
		if err = rows.Scan(&kind, &level, &target, &started, &until); err != nil {
			return nil, nil, err
		}
		levels[kind] = level
		if target != nil && started != nil && until != nil {
			research = &troopResearch{Type: kind, TargetLevel: *target, StartedAt: *started, FinishAt: *until}
		}
	}
	return levels, research, rows.Err()
}

// research: the workshop raises one troop type a level — gold now, the level when the timer runs out.
func (s *Server) research(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	s.command(w, r, a, func(ctx context.Context, tx pgx.Tx, raw []byte) (int, any, error) {
		var in struct {
			Type string `json:"type"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		b, c, st, err := gameTx(ctx, tx, a)
		if err != nil {
			return 0, nil, err
		}
		if err = debtRefusal(b); err != nil {
			return 0, nil, err
		}
		levels, research, err := troopLevels(ctx, tx, a.ID)
		if err != nil {
			return 0, nil, err
		}
		spec, rf := game.CheckResearch(c, in.Type, levels[in.Type], research != nil, b.Gold)
		if rf != nil {
			return 0, nil, asAPI(rf)
		}
		dur := game.BuildDuration(st, game.LevelSpec{Seconds: spec.Seconds})
		if _, err = tx.Exec(ctx, `INSERT INTO troop_levels(user_id,type,level,target_level,started_at,busy_until) VALUES($1,$2,$3,$4,now(),now()+$5*interval '1 millisecond')
 ON CONFLICT (user_id,type) DO UPDATE SET target_level=EXCLUDED.target_level, started_at=EXCLUDED.started_at, busy_until=EXCLUDED.busy_until`,
			a.ID, in.Type, levels[in.Type], spec.Level, dur.Milliseconds()); err != nil {
			if uniqueViolation(err) {
				return 0, nil, refuse(409, "workshop_busy", "Мастерская уже работает. Дождитесь окончания улучшения.")
			}
			return 0, nil, err
		}
		name := game.Troops[in.Type].Name
		if _, err = applyLedger(ctx, tx, ledgerEntry{UserID: a.ID, Kind: "research", Gold: -spec.Cost, ActorID: a.ID, Note: fmt.Sprintf("Мастерская: %s, уровень %d", name, spec.Level)}); err != nil {
			return 0, nil, err
		}
		if err = audit(ctx, tx, a.ID, a.TeamID, "army.research", "troops", in.Type, fmt.Sprintf("Мастерская: %s → уровень %d", name, spec.Level)); err != nil {
			return 0, nil, err
		}
		out, err := armyDTO(ctx, tx, a.ID)
		return 201, out, err
	})
}

func (s *Server) army(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	if err := s.settleNow(r.Context(), a.ID); err != nil {
		failErr(w, err)
		return
	}
	out, err := armyDTO(r.Context(), s.db, a.ID)
	if err != nil {
		failErr(w, err)
		return
	}
	respond(w, 200, out)
}

// train queues troops: energy is paid now, units arrive one batch after another; housing counts the queue too.
func (s *Server) train(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	s.command(w, r, a, func(ctx context.Context, tx pgx.Tx, raw []byte) (int, any, error) {
		var in struct {
			Type  string `json:"type"`
			Count int    `json:"count"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		def, ok := game.Troops[in.Type]
		if !ok || in.Count < 1 || in.Count > 50 {
			return 0, nil, refuse(422, "invalid_fields", "Выберите тип войск и количество от 1 до 50.")
		}
		b, c, st, err := gameTx(ctx, tx, a)
		if err != nil {
			return 0, nil, err
		}
		if err = debtRefusal(b); err != nil {
			return 0, nil, err
		}
		if rf := game.CanTrain(c, in.Type); rf != nil {
			return 0, nil, asAPI(rf)
		}
		capacity := c.ArmyCapacity()
		if capacity == 0 {
			return 0, nil, refuse(409, "no_barracks", "Постройте казарму, чтобы готовить войска.")
		}
		troops, queued, err := armyOf(ctx, tx, a.ID)
		if err != nil {
			return 0, nil, err
		}
		if game.Housing(troops)+queued+def.Housing*in.Count > capacity {
			return 0, nil, refuse(409, "army_full", fmt.Sprintf("Не хватает места в казармах: занято %d из %d.", game.Housing(troops)+queued, capacity))
		}
		cost := def.Cost * int64(in.Count)
		if b.Energy < cost {
			return 0, nil, refuse(402, "insufficient_energy", fmt.Sprintf("Недостаточно энергии: нужно %d, есть %d.", cost, b.Energy))
		}
		dur := game.TrainDuration(st, def, in.Count)
		if _, err = tx.Exec(ctx, `INSERT INTO training_queue(user_id,type,count,started_at,finish_at)
 SELECT $1,$2,$3,s,s+$4*interval '1 millisecond' FROM (SELECT GREATEST(now(), COALESCE((SELECT MAX(finish_at) FROM training_queue WHERE user_id=$1), now())) s) x`,
			a.ID, in.Type, in.Count, dur.Milliseconds()); err != nil {
			return 0, nil, err
		}
		if _, err = applyLedger(ctx, tx, ledgerEntry{UserID: a.ID, Kind: "train", Energy: -cost, ActorID: a.ID, Note: fmt.Sprintf("Обучение: %s × %d", def.Name, in.Count)}); err != nil {
			return 0, nil, err
		}
		if err = audit(ctx, tx, a.ID, a.TeamID, "army.training", "troops", in.Type, fmt.Sprintf("Начато обучение: %s × %d", def.Name, in.Count)); err != nil {
			return 0, nil, err
		}
		out, err := armyDTO(ctx, tx, a.ID)
		return 201, out, err
	})
}

// moveBuilding: «Переставить» — free and instant, onto a free cell of the player's land.
func (s *Server) moveBuilding(w http.ResponseWriter, r *http.Request) {
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
			X int `json:"x"`
			Y int `json:"y"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		_, c, _, err := gameTx(ctx, tx, a)
		if err != nil {
			return 0, nil, err
		}
		b, ok := c.Find(id)
		if !ok {
			return 0, nil, refuse(404, "not_found", "Здание не найдено.")
		}
		if rf := game.CheckMove(c, b, in.X, in.Y); rf != nil {
			return 0, nil, asAPI(rf)
		}
		if _, err = tx.Exec(ctx, `UPDATE buildings SET x=$2, y=$3 WHERE id=$1`, id, in.X, in.Y); err != nil {
			return 0, nil, err
		}
		if err = audit(ctx, tx, a.ID, a.TeamID, "building.moved", "building", id, fmt.Sprintf("%s: переставлено", game.Buildings[b.Type].Name)); err != nil {
			return 0, nil, err
		}
		out, err := cityDTO(ctx, tx, a.ID)
		if err != nil {
			return 0, nil, err
		}
		out["player"], err = playerState(ctx, tx, a.ID)
		return 200, out, err
	})
}

// removeDecor: decor can be taken away; its gold is not returned. Real buildings stay.
func (s *Server) removeDecor(w http.ResponseWriter, r *http.Request) {
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
		_, c, _, err := gameTx(ctx, tx, a)
		if err != nil {
			return 0, nil, err
		}
		b, ok := c.Find(id)
		if !ok {
			return 0, nil, refuse(404, "not_found", "Здание не найдено.")
		}
		if !game.Buildings[b.Type].Decor {
			return 0, nil, refuse(409, "not_decor", "Убрать можно только украшения: здания города остаются.")
		}
		if _, err = tx.Exec(ctx, `DELETE FROM buildings WHERE id=$1`, id); err != nil {
			return 0, nil, err
		}
		if err = audit(ctx, tx, a.ID, a.TeamID, "decor.removed", "building", id, fmt.Sprintf("%s: убрано", game.Buildings[b.Type].Name)); err != nil {
			return 0, nil, err
		}
		out, err := cityDTO(ctx, tx, a.ID)
		if err != nil {
			return 0, nil, err
		}
		out["player"], err = playerState(ctx, tx, a.ID)
		return 200, out, err
	})
}

// marketTrade: one exchange at the market. Nothing is earned here — the market only turns one resource into the
// other at the work-reward rate, minus its fee.
func (s *Server) marketTrade(w http.ResponseWriter, r *http.Request) {
	a, ok := s.authed(w, r)
	if !ok {
		return
	}
	s.command(w, r, a, func(ctx context.Context, tx pgx.Tx, raw []byte) (int, any, error) {
		var in struct {
			Give   string `json:"give"`
			Amount int64  `json:"amount"`
		}
		if err := decodeInto(raw, &in); err != nil {
			return 0, nil, err
		}
		b, c, st, err := gameTx(ctx, tx, a)
		if err != nil {
			return 0, nil, err
		}
		if err = debtRefusal(b); err != nil {
			return 0, nil, err
		}
		dGold, dEnergy, rf := game.Exchange(st, c, in.Give, in.Amount)
		if rf == nil {
			rf = game.TradeFits(c, b.Gold, b.Energy, dGold, dEnergy)
		}
		if rf != nil {
			return 0, nil, asAPI(rf)
		}
		note := fmt.Sprintf("Рынок: %d золота → %d энергии", -dGold, dEnergy)
		if in.Give == "energy" {
			note = fmt.Sprintf("Рынок: %d энергии → %d золота", -dEnergy, dGold)
		}
		if _, err = applyLedger(ctx, tx, ledgerEntry{UserID: a.ID, Kind: "market", Gold: dGold, Energy: dEnergy, ActorID: a.ID, Note: note}); err != nil {
			return 0, nil, err
		}
		out, err := playerState(ctx, tx, a.ID)
		return 200, out, err
	})
}
