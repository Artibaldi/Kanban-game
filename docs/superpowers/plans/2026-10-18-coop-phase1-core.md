# Кооператив против бэклога — Этап 1 (ядро) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Заменить набеги на города коллег командной обороной против бэклога: крепость команды, орда из задач, урон от просрочек, недельные волны с коронами и общей наградой, «Прогресс» вместо рейтинга, дружеские бои, правило «не для оценки».

**Architecture:** Чистые правила (короны, урон, окна волн, бонус обороны) — в пакете `server/internal/game`, без базы. Состояние — в новой миграции `0002_coop.sql`. План волны фиксируется в `wave_plan` теми же транзакциями, что двигают задачи. Закрытие волн и урон считает фоновый тик внутри сервера, идемпотентно. Клиент получает страницы «Крепость», «Прогресс», «Дружеские бои» на существующих компонентах; новая визуальная система — этап 2, не здесь.

**Tech Stack:** Go 1.26, chi, pgx v5, PostgreSQL 18; Vue 3.5, Pinia, vue-router, TypeScript 5.8, Vite 7; тесты — `go test`, `node --test`.

**Spec:** `docs/superpowers/specs/2026-10-17-coop-backlog-defense-design.md` (разделы 4.1–4.4, 4.6 — общий сундук, 4.7 — «Прогресс», 4.8, 4.11, 8, 9, 10, 11 — фаза 1). Сундуки-церемонии, боссы, дорога трофеев, новички, «Спасибо», анимированный бой волны и новая визуальная система — этапы 2–3, в этом плане их нет.

## Global Constraints

- Новых зависимостей нет ни на сервере, ни в клиенте.
- Любое изменение ресурсов — строка в `ledger` через `applyLedger`; балансы не меняются в обход.
- Изменяющие запросы идут через `s.command(...)` (Idempotency-Key, транзакция).
- `work_points_events` пишет только подтверждение и возврат задачи. Игра туда не пишет.
- Режим «Работа» (`users.game_mode=false`) скрывает в клиенте все игровые экраны и ресурсы, но **не** отменяет начисления: награда волны пишется в `ledger` всем участникам команды.
- Все даты волн и урона — UTC. Неделя начинается в понедельник 00:00 UTC.
- Новый серверный код берёт время из `s.now()`, а не `time.Now()`.
- Тексты интерфейса и ошибок — по-русски, в стиле существующих `refuse(...)`. Эмодзи не используются.
- Задача — враг, если `state IN ('ready','in_progress','review')` и сложность согласована (`points_approved_at IS NOT NULL`). Задачи в `backlog` — «туман», не враги, урона не наносят.
- Числа по умолчанию (спецификация, раздел 4): урон 2 % за балл в сутки; минимум прочности 10 %; ремонт по коронам 0/10/25/100; пороги корон 0.5 и 0.8; общий сундук золото 0/60/140/250, энергия 0/15/35/60; бонус обороны до 0.20; дружеский бой — 5 опыта, не больше 3 боёв в день.
- Фраза правила дословно: «Игровые данные не используются для оценки работы».

## Review Focus

1. Первое действие новой недели — подтверждение задачи. Задача должна попасть в план волны **до** перехода, иначе она пропадёт из «Закрыто» (Task 3, `TestWaveSnapshotBeforeFirstApproval`).
2. Задачу вернули в `backlog` посреди волны — она остаётся в плане, три короны «отодвиганием» не получить (Task 3, `TestWavePlanKeepsTaskMovedToBacklog`).
3. Тик закрытия волны сработал дважды или параллельно — награда не удваивается (Task 4, `TestSettleWaveTwiceNoDoubleReward`).
4. Задачу, подтверждённую в прошлой волне, вернули и снова подтвердили в текущей — она считается в «Закрыто» текущей волны ровно один раз (Task 4, `TestReopenedFromPreviousWaveCountsOnce`).
5. Администратор без команды и человек в режиме «Работа»: `/fortress` для первого — понятный отказ `not_a_player`, второму награда волны всё равно начисляется (Task 4 и Task 5).

---

## File Structure

Сервер:
- Create `server/internal/game/coop.go` — чистые правила кооператива. Test: `server/internal/game/coop_test.go`.
- Modify `server/internal/game/economy.go` — поля настроек кооператива, удаление настроек набегов.
- Modify `server/internal/game/battle.go` — удалить `RewardFor`, `BattleReward`, `InRange`.
- Create `server/db/migrations/0002_coop.sql`.
- Create `server/internal/app/waves.go` — окно волны, снимок плана, подсчёт волны, закрытие, урон, тик.
- Create `server/internal/app/fortress.go` — `GET /fortress`, `GET /waves`.
- Create `server/internal/app/progress.go` — `GET /progress/me`, `GET /progress/team`.
- Modify `server/internal/app/battles.go` — только дружеские бои.
- Modify `server/internal/app/tasks.go`, `rewards.go`, `store.go`, `player.go`, `auth.go`, `admin.go`, `server.go`, `seed.go`.
- Create `server/internal/app/coop_e2e_test.go`. Modify `server/internal/app/e2e_test.go`.

Клиент:
- Modify `web/src/types.ts`, `web/src/stores/session.ts`, `web/src/app/router.ts`, `web/src/app/App.vue`.
- Create `web/src/lib/fortress.ts` + `web/tests/fortress.test.ts`.
- Create `web/src/pages/FortressPage.vue`, `web/src/components/FortressScene.vue`.
- Create `web/src/pages/ProgressPage.vue` (заменяет `RatingsPage.vue`, тот удаляется).
- Create `web/src/pages/DuelsPage.vue` (заменяет `OpponentsPage.vue`, тот удаляется).
- Create `web/src/components/GameRulesDialog.vue`.
- Modify `web/src/pages/BattlePage.vue`, `SettingsPage.vue`, `AdminPage.vue`, `HomePage.vue`.

---

### Task 1: Правила кооператива в пакете `game`

**Files:**
- Create: `server/internal/game/coop.go`
- Modify: `server/internal/game/economy.go`
- Test: `server/internal/game/coop_test.go`

**Interfaces:**
- Produces (все в `package game`):
  - Поля `Settings` (JSON-имена в скобках): `OverdueDamagePerPoint float64` (`overdue_damage_per_point`, 2), `FortressFloor int` (`fortress_floor`, 10), `RepairByCrowns [4]int` (`repair_by_crowns`, `{0,10,25,100}`), `CrownThresholds [2]float64` (`crown_thresholds`, `{0.5,0.8}`), `WaveChestGold [4]int64` (`wave_chest_gold`, `{0,60,140,250}`), `WaveChestEnergy [4]int64` (`wave_chest_energy`, `{0,15,35,60}`), `MaxDefenseBonus float64` (`max_defense_bonus`, 0.20), `DefenseScoreFull int64` (`defense_score_full`, 4000), `FriendlyXP int64` (`friendly_xp`, 5), `FriendlyPerDay int` (`friendly_per_day`, 3).
  - Поля набегов (`AttackEnergy`, `ShieldMinutes`, `RematchMinutes`, `BattleScope`, `BattleGold*`, `BattleXPPerStar`) и функции `RewardFor`, `InRange`, `BattleReward` здесь **не** трогаются — их использует `app/battles.go`; удаляет их Task 6.
  - `func IsEnemy(state string, pointsApproved bool) bool`
  - `func EnemyKind(points int) string` — `"goblin"` (1–2), `"knight"` (3), `"giant"` (5), `"dragon"` (8).
  - `func WaveWindow(t time.Time, weeks int) (start, end time.Time)` — UTC, окна по `weeks` недель, отсчёт от понедельника 1970-01-05 00:00 UTC; `weeks` вне 1..4 трактуется как 1.
  - `func IsWorkDay(day time.Time, workDays []int) bool` — ISO-день недели 1 (пн) … 7 (вс).
  - `type WaveTally struct { Planned, Closed, Overdue, OverdueP1 int }` (JSON: `planned`, `closed`, `overdue`, `overdue_p1`).
  - `func Crowns(s Settings, t WaveTally) int` — 0, если `Planned == 0`.
  - `func OverdueDamage(s Settings, points int) int` — `round(points × OverdueDamagePerPoint)`.
  - `func DamageHP(s Settings, hp, damage int) int` — `max(FortressFloor, hp-damage)`, но не выше `hp`.
  - `func RepairHP(s Settings, hp, crowns int) int` — 3 короны → 100; иначе `min(100, hp+RepairByCrowns[crowns])`.
  - `func DefenseScore(d Defense, army map[string]int) int64` — `WallHP + TowerHP + Σ Troops[k].HP × n`.
  - `func DefenseBonus(s Settings, avgScore int64) float64` — `MaxDefenseBonus × min(1, avgScore / DefenseScoreFull)`.
  - `func WaveChest(s Settings, crowns int, bonus float64) (gold, energy int64)` — `round(base × (1+bonus))`.
  - `func CrownsWord(n int) string` — «корона» / «короны» / «корон» по правилам русского склонения.
  - `func WaveSummary(t WaveTally, crowns int) string`.

- [ ] **Step 1: Write the failing tests** in `coop_test.go`

```go
func TestCrowns(t *testing.T) {
	s := DefaultSettings()
	cases := []struct{ t WaveTally; want int }{
		{WaveTally{Planned: 0}, 0},
		{WaveTally{Planned: 30, Closed: 14}, 0},
		{WaveTally{Planned: 30, Closed: 15}, 1},
		{WaveTally{Planned: 30, Closed: 24, Overdue: 1}, 2},              // пример из спецификации
		{WaveTally{Planned: 30, Closed: 24, Overdue: 1, OverdueP1: 1}, 1},
		{WaveTally{Planned: 30, Closed: 30, Overdue: 1}, 2},
		{WaveTally{Planned: 30, Closed: 30}, 3},
	}
	for _, c := range cases {
		if got := Crowns(s, c.t); got != c.want {
			t.Errorf("%+v: got %d want %d", c.t, got, c.want)
		}
	}
}

func TestFortressHP(t *testing.T) {
	s := DefaultSettings()
	if d := OverdueDamage(s, 5); d != 10 { t.Fatalf("5 points → %d", d) }
	if hp := DamageHP(s, 15, 10); hp != 10 { t.Fatalf("floor: %d", hp) }
	if hp := DamageHP(s, 8, 4); hp != 8 { t.Fatalf("below floor never drops further: %d", hp) }
	if hp := RepairHP(s, 40, 2); hp != 65 { t.Fatalf("2 crowns: %d", hp) }
	if hp := RepairHP(s, 95, 1); hp != 100 { t.Fatalf("cap: %d", hp) }
	if hp := RepairHP(s, 12, 3); hp != 100 { t.Fatalf("3 crowns: %d", hp) }
}

func TestWaveWindow(t *testing.T) {
	wed := time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC)
	s, e := WaveWindow(wed, 1)
	if !s.Equal(time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)) || !e.Equal(time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("week: %v %v", s, e)
	}
	s2, e2 := WaveWindow(wed, 2)
	if e2.Sub(s2) != 14*24*time.Hour || s2.Weekday() != time.Monday || wed.Before(s2) || !wed.Before(e2) {
		t.Fatalf("sprint: %v %v", s2, e2)
	}
	if s3, _ := WaveWindow(time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC), 1); !s3.Equal(e) {
		t.Fatalf("boundary belongs to the next wave: %v", s3)
	}
}

func TestEnemiesAndWords(t *testing.T) {
	if IsEnemy("backlog", true) || IsEnemy("ready", false) || IsEnemy("done", true) || !IsEnemy("review", true) {
		t.Fatal("IsEnemy")
	}
	if EnemyKind(2) != "goblin" || EnemyKind(3) != "knight" || EnemyKind(5) != "giant" || EnemyKind(8) != "dragon" {
		t.Fatal("EnemyKind")
	}
	if !IsWorkDay(time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC), []int{1, 2, 3, 4, 5}) ||
		IsWorkDay(time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC), []int{1, 2, 3, 4, 5}) {
		t.Fatal("IsWorkDay: friday yes, sunday no")
	}
	if CrownsWord(1) != "корона" || CrownsWord(2) != "короны" || CrownsWord(0) != "корон" || CrownsWord(11) != "корон" {
		t.Fatal("CrownsWord")
	}
	got := WaveSummary(WaveTally{Planned: 30, Closed: 24, Overdue: 1}, 2)
	if got != "Закрыто 24 из 30 баллов. Просрочено: 1. Итог: 2 короны." {
		t.Fatalf("summary: %q", got)
	}
}

func TestWaveChestAndBonus(t *testing.T) {
	s := DefaultSettings()
	if b := DefenseBonus(s, 2000); b != 0.10 { t.Fatalf("half score: %v", b) }
	if b := DefenseBonus(s, 9000); b != 0.20 { t.Fatalf("capped: %v", b) }
	if g, e := WaveChest(s, 2, 0.10); g != 154 || e != 39 { t.Fatalf("2 crowns +10%%: %d %d", g, e) }
	if g, e := WaveChest(s, 0, 0.2); g != 0 || e != 0 { t.Fatalf("0 crowns: %d %d", g, e) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./internal/game/ -run 'Crowns|Fortress|WaveWindow|Enemies|WaveChest' -v`
Expected: FAIL — `undefined: Crowns` и т. п.

- [ ] **Step 3: Implement `coop.go` and the `Settings` changes**

`Crowns`: 3 — `Closed ≥ Planned && Overdue == 0`; 2 — `Closed ≥ ⌈0.8·Planned⌉`-эквивалент (`float64(Closed) ≥ th[1]·float64(Planned)`) и `OverdueP1 == 0`; 1 — `float64(Closed) ≥ th[0]·float64(Planned)`. `Normalize` подставляет значения по умолчанию для нулевых/некорректных полей кооператива (`RepairByCrowns` и `WaveChest*` — если все элементы нулевые; пороги — если не `0 < a < b ≤ 1`). `WaveSummary` при `Planned == 0` возвращает «Спокойная неделя: врагов не было.»

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd server && go test ./internal/game/ -v`
Expected: PASS, все тесты пакета.

- [ ] **Step 5: Commit**

```bash
git add server/internal/game
git commit -m "game: правила кооператива — короны, урон, окна волн, бонус обороны"
```

---

### Task 2: Миграция `0002_coop.sql`

**Files:**
- Create: `server/db/migrations/0002_coop.sql`
- Modify: `server/internal/app/store.go:70-95` (`ledgerEntry.WaveID`, вставка `wave_id`)
- Test: `server/internal/app/coop_e2e_test.go` (новый файл)
- Modify: `server/internal/app/e2e_test.go:27-75` — `testServer` возвращает ещё и `*Server`

**Interfaces:**
- Produces:
  - `teams.work_days smallint[] NOT NULL DEFAULT '{1,2,3,4,5}'`, `teams.wave_weeks smallint NOT NULL DEFAULT 1 CHECK (wave_weeks BETWEEN 1 AND 4)`.
  - `users.accept_friendly boolean NOT NULL DEFAULT false`, `users.game_rules_ack_at timestamptz`.
  - `fortresses(team_id uuid PK REFERENCES teams, hp integer NOT NULL DEFAULT 100 CHECK (hp BETWEEN 0 AND 100), level integer NOT NULL DEFAULT 1, updated_at timestamptz NOT NULL DEFAULT now())`.
  - `waves(id uuid PK DEFAULT gen_random_uuid(), team_id uuid NOT NULL REFERENCES teams, starts_at timestamptz NOT NULL, ends_at timestamptz NOT NULL, planned_points int, closed_points int, overdue_count int, overdue_p1_count int, crowns int CHECK (crowns BETWEEN 0 AND 3), defense_bonus numeric(4,3), summary text, settled_at timestamptz, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE (team_id, starts_at))` + индекс `(settled_at, ends_at) WHERE settled_at IS NULL`.
  - `wave_plan(wave_id uuid REFERENCES waves, task_id uuid REFERENCES tasks, points int NOT NULL, joined_at timestamptz NOT NULL, PRIMARY KEY (wave_id, task_id))`.
  - `fortress_events(id bigserial PK, team_id uuid NOT NULL REFERENCES teams, kind text NOT NULL CHECK (kind IN ('overdue_damage','wave_repair')), task_id uuid REFERENCES tasks, day date, wave_id uuid REFERENCES waves, delta int NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE (task_id, day))` + `CREATE UNIQUE INDEX fortress_one_repair ON fortress_events(wave_id) WHERE kind='wave_repair'`.
  - `ledger.wave_id uuid REFERENCES waves`; ограничение `ledger_kind_check` пересоздаётся с добавлением `'wave_reward'`; `CREATE UNIQUE INDEX ledger_one_wave_reward ON ledger(user_id, wave_id) WHERE kind='wave_reward'`.
  - `battles.kind text NOT NULL DEFAULT 'raid' CHECK (kind IN ('raid','friendly'))`.
  - `UPDATE players SET trophies=0, shield_until=NULL;` и `INSERT INTO fortresses(team_id) SELECT id FROM teams ON CONFLICT DO NOTHING;`
  - `ledgerEntry.WaveID string`, пишется через `nullable(e.WaveID)`.
  - `func testServer(t *testing.T) (*httptest.Server, *pgxpool.Pool, *Server)`.

- [ ] **Step 1: Write the failing test** in `coop_e2e_test.go`

```go
func TestCoopSchema(t *testing.T) {
	_, pool, _ := testServer(t)
	ctx := context.Background()
	var hp, weeks int
	if err := pool.QueryRow(ctx, `SELECT f.hp, t.wave_weeks FROM fortresses f JOIN teams t ON t.id=f.team_id WHERE t.name='Платформа'`).Scan(&hp, &weeks); err != nil {
		t.Fatal(err)
	}
	if weeks != 1 { t.Fatalf("default wave length: %d", weeks) }
	var trophies int
	_ = pool.QueryRow(ctx, `SELECT COALESCE(MAX(trophies),0) FROM players`).Scan(&trophies)
	if trophies != 0 { t.Fatalf("raid trophies must be reset: %d", trophies) }
	if _, err := pool.Exec(ctx, `INSERT INTO ledger(user_id,kind,note) SELECT id,'wave_reward','t' FROM users WHERE login='alexey'`); err != nil {
		t.Fatalf("wave_reward kind must be allowed: %v", err)
	}
}
```

Обновить вызов в `TestEndToEnd`: `ts, pool, _ := testServer(t)`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && DATABASE_URL=postgres://kanban:kanban-local-only@127.0.0.1:15433/kanban?sslmode=disable go test ./internal/app/ -run TestCoopSchema -v`
Expected: FAIL — `relation "fortresses" does not exist`.

- [ ] **Step 3: Write `0002_coop.sql`; add `WaveID` to `ledgerEntry` and the `wave_id` column to the `INSERT` in `applyLedger`; make `testServer` return `s`**

Сид выполняется после миграций, поэтому в `seed.go`: заменить `INSERT INTO players(user_id,trophies) VALUES($1,$2)` на `INSERT INTO players(user_id) VALUES($1)`, удалить `seedTrophies`, после создания команд добавить `INSERT INTO fortresses(team_id) … ON CONFLICT DO NOTHING`.

- [ ] **Step 4: Run tests**

Run: `cd server && DATABASE_URL=... go test ./... `
Expected: `TestCoopSchema` PASS; `TestEndToEnd` PASS (набеги ещё живы).

- [ ] **Step 5: Commit**

```bash
git add server/db/migrations/0002_coop.sql server/internal/app/store.go server/internal/app/e2e_test.go server/internal/app/coop_e2e_test.go server/internal/app/seed.go
git commit -m "db: крепости, волны, план волны, события крепости"
```

---

### Task 3: Волна и её зафиксированный план

**Files:**
- Create: `server/internal/app/waves.go`
- Modify: `server/internal/app/tasks.go` (`transitionTask` ~711, `approveEstimate` ~620, `patchTask` ~423)
- Test: `server/internal/app/coop_e2e_test.go`

**Interfaces:**
- Consumes: `game.WaveWindow`, `game.IsEnemy` (Task 1); таблицы `waves`, `wave_plan` (Task 2).
- Produces:
  - `type waveRow struct { ID, TeamID string; StartsAt, EndsAt time.Time }`
  - `func ensureWave(ctx context.Context, tx pgx.Tx, teamID string, now time.Time) (waveRow, error)` — создаёт волну текущего окна (`teams.wave_weeks`) через `INSERT … ON CONFLICT (team_id, starts_at) DO NOTHING RETURNING`; **только создавший** вставляет снимок: все задачи-враги команды с текущими `points`, `joined_at = now`. Если строка уже была — читает её.
  - `func syncWavePlan(ctx context.Context, tx pgx.Tx, taskID string, now time.Time) error` — если задача сейчас враг: `INSERT INTO wave_plan … ON CONFLICT (wave_id, task_id) DO UPDATE SET points = EXCLUDED.points`. Если не враг — ничего (задача остаётся в плане).
  - Вызовы: в начале транзакции `transitionTask`, `approveEstimate`, `patchTask` — `ensureWave(ctx, tx, t.TeamID, s.now())` (снимок до изменения); в конце — `syncWavePlan(ctx, tx, id, s.now())`.

- [ ] **Step 1: Write the failing tests**

```go
// shiftCurrentWave moves the team's current wave one period into the past, as if a new week had started.
func shiftCurrentWave(t *testing.T, pool *pgxpool.Pool, team string) {
	_, err := pool.Exec(context.Background(), `UPDATE waves SET starts_at=starts_at-interval '7 days', ends_at=ends_at-interval '7 days'
 WHERE team_id=(SELECT id FROM teams WHERE name=$1) AND settled_at IS NULL`, team)
	if err != nil { t.Fatal(err) }
}

func planOf(t *testing.T, pool *pgxpool.Pool, team string) map[string]int {
	rows, _ := pool.Query(context.Background(), `SELECT tk.number, wp.points FROM wave_plan wp JOIN waves w ON w.id=wp.wave_id JOIN tasks tk ON tk.id=wp.task_id
 WHERE w.team_id=(SELECT id FROM teams WHERE name=$1) AND w.starts_at<=now() AND w.ends_at>now()`, team)
	out := map[string]int{}
	for rows.Next() { var n string; var p int; _ = rows.Scan(&n, &p); out[n] = p }
	return out
}

func TestWaveSnapshotBeforeFirstApproval(t *testing.T) {
	ts, pool, _ := testServer(t)
	irina := login(t, ts, "irina")
	review := findTask(t, irina, "Импорт платежей из банка") // PLT-10, в review у Марии
	shiftCurrentWave(t, pool, "Платформа")
	irina.must(200, "POST", "/tasks/"+review["id"].(string)+"/transition", map[string]any{"to": "done"}, "")
	if p := planOf(t, pool, "Платформа"); p["PLT-10"] != 5 {
		t.Fatalf("approved task must be in the new wave plan: %v", p)
	}
}

func TestWavePlanKeepsTaskMovedToBacklog(t *testing.T) {
	ts, pool, _ := testServer(t)
	irina := login(t, ts, "irina")
	ready := findTask(t, irina, "Валидация ИНН при регистрации") // PLT-12, ready, 2 балла
	irina.must(200, "POST", "/tasks/"+ready["id"].(string)+"/transition", map[string]any{"to": "backlog"}, "")
	if p := planOf(t, pool, "Платформа"); p["PLT-12"] != 2 {
		t.Fatalf("task moved to backlog stays in the plan: %v", p)
	}
}

func TestBacklogTaskJoinsPlanWhenReady(t *testing.T) {
	ts, pool, _ := testServer(t)
	irina := login(t, ts, "irina")
	bl := findTask(t, irina, "Мультивалютные счета") // PLT-14, backlog, 8, оценка согласована
	if _, ok := planOf(t, pool, "Платформа")["PLT-14"]; ok { t.Fatal("backlog is fog, not plan") }
	irina.must(200, "POST", "/tasks/"+bl["id"].(string)+"/transition", map[string]any{"to": "ready"}, "")
	if p := planOf(t, pool, "Платформа"); p["PLT-14"] != 8 { t.Fatalf("became an enemy: %v", p) }
}
```

Названия задач и состояния взяты из демо-данных (`seed.go`, скриншот доски). Если у PLT-14 оценка не согласована в сиде, в тесте сначала `POST /tasks/{id}/estimate {"points":8}` от `irina`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && DATABASE_URL=... go test ./internal/app/ -run 'Wave|Backlog' -v`
Expected: FAIL — пустой план.

- [ ] **Step 3: Implement `ensureWave`, `syncWavePlan` and the call sites**

- [ ] **Step 4: Run tests**

Run: `cd server && DATABASE_URL=... go test ./internal/app/ -v`
Expected: PASS, включая `TestEndToEnd`.

- [ ] **Step 5: Commit**

```bash
git add server/internal/app/waves.go server/internal/app/tasks.go server/internal/app/coop_e2e_test.go
git commit -m "Волна: фиксированный план из задач-врагов"
```

---

### Task 4: Подсчёт и закрытие волны, урон от просрочек, тик

**Files:**
- Modify: `server/internal/app/waves.go`, `server/internal/app/store.go` (`RunTimers`)
- Test: `server/internal/app/coop_e2e_test.go`

**Interfaces:**
- Consumes: `ensureWave` (Task 3); `game.Crowns`, `game.WaveSummary`, `game.DamageHP`, `game.RepairHP`, `game.OverdueDamage`, `game.DefenseScore`, `game.DefenseBonus`, `game.WaveChest`, `game.IsWorkDay` (Task 1); `loadCity`, `armyOf`, `loadBalance`, `lockPlayers`, `applyLedger`, `game.Credit`, `audit`.
- Produces:
  - `func tallyWave(ctx context.Context, q dbtx, w waveRow) (game.WaveTally, error)`:
    - `Planned` — `SUM(points)` из `wave_plan`.
    - `Closed` — сумма `wp.points` задач плана, у которых **последнее** событие `work_points_events` внутри `[starts_at, ends_at)` положительное:
      ```sql
      SELECT COALESCE(SUM(wp.points),0) FROM wave_plan wp
      WHERE wp.wave_id=$1 AND COALESCE((SELECT w.points>0 FROM work_points_events w
        WHERE w.task_id=wp.task_id AND w.created_at>=$2 AND w.created_at<$3 ORDER BY w.id DESC LIMIT 1), false)
      ```
    - `Overdue`, `OverdueP1` — задачи-враги команды с `due_date < ends_at::date` (на момент подсчёта).
  - `func settleWave(ctx context.Context, tx pgx.Tx, w waveRow, now time.Time) error` — блокирует строку волны (`FOR UPDATE`), выходит, если `settled_at` уже стоит; считает тally и короны; при `Planned > 0`: бонус обороны = `DefenseBonus(avg DefenseScore по участникам)`, каждому участнику (`users.active AND team_id=…`, любая роль) — `applyLedger` с `Kind: "wave_reward"`, `WaveID`, золото/энергия через `game.Credit` с ёмкостью хранилищ, заметка `"Волна: " + summary`; ремонт крепости `RepairHP` + строка `fortress_events(kind='wave_repair', wave_id, delta)`. Всегда: `waves` ← tally, crowns, bonus, summary, `settled_at=now`; `audit(..., "wave.settled", "wave", id, summary)`.
  - `func applyOverdueDamage(ctx context.Context, tx pgx.Tx, teamID string, day time.Time) (int, error)` — если не рабочий день команды, 0. Иначе вставляет `fortress_events(kind='overdue_damage', task_id, day, delta=-OverdueDamage(points))` для задач-врагов с `due_date < day` с `ON CONFLICT (task_id, day) DO NOTHING RETURNING delta`; новую сумму урона применяет `DamageHP`; возвращает нанесённый урон.
  - Уровень крепости (только внешний вид): после закрытия волны `fortresses.level = min(5, 1 + число закрытых волн команды с crowns ≥ 2)`.
  - `func (s *Server) coopTick(ctx context.Context) error` — по каждой команде в отдельной транзакции: `ensureWave(now)`, `settleWave` для всех её волн с `ends_at <= now AND settled_at IS NULL`, `applyOverdueDamage(today UTC)`.
  - `RunTimers`: второй тикер раз в 30 секунд вызывает `coopTick`, плюс один вызов сразу при старте (догоняет пропущенное закрытие).

- [ ] **Step 1: Write the failing tests**

```go
// endCurrentWave puts the team's open wave entirely in the past so the next tick settles it.
func endCurrentWave(t *testing.T, pool *pgxpool.Pool, team string) {
	_, err := pool.Exec(context.Background(), `UPDATE waves SET starts_at=now()-interval '7 days', ends_at=now()-interval '1 second'
 WHERE team_id=(SELECT id FROM teams WHERE name=$1) AND settled_at IS NULL`, team)
	if err != nil { t.Fatal(err) }
}

// keepOnlyInPlan narrows the open wave's plan to one task, so the crowns are predictable (closing it = 100 %).
func keepOnlyInPlan(t *testing.T, pool *pgxpool.Pool, team, number string) {
	_, err := pool.Exec(context.Background(), `DELETE FROM wave_plan wp USING waves w, tasks tk
 WHERE wp.wave_id=w.id AND tk.id=wp.task_id AND w.settled_at IS NULL AND tk.number<>$2 AND w.team_id=(SELECT id FROM teams WHERE name=$1)`, team, number)
	if err != nil { t.Fatal(err) }
}

func TestSettleWaveRewardsWholeTeam(t *testing.T) {
	ts, pool, s := testServer(t)
	ctx := context.Background()
	irina, maria := login(t, ts, "irina"), login(t, ts, "maria")
	maria.must(200, "PATCH", "/me/settings", map[string]any{"game_mode": false}, "") // режим «Работа»
	review := findTask(t, irina, "Импорт платежей из банка")
	irina.must(200, "POST", "/tasks/"+review["id"].(string)+"/transition", map[string]any{"to": "done"}, "")
	keepOnlyInPlan(t, pool, "Платформа", "PLT-10")
	endCurrentWave(t, pool, "Платформа")
	if err := s.coopTick(ctx); err != nil { t.Fatal(err) }
	var crowns *int
	var summary string
	_ = pool.QueryRow(ctx, `SELECT crowns, summary FROM waves WHERE settled_at IS NOT NULL AND team_id=(SELECT id FROM teams WHERE name='Платформа') ORDER BY ends_at DESC LIMIT 1`).Scan(&crowns, &summary)
	if crowns == nil || *crowns < 1 || !strings.HasPrefix(summary, "Закрыто 5 из 5 баллов.") { t.Fatalf("settled: %v %q", crowns, summary) }
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM ledger l JOIN users u ON u.id=l.user_id WHERE l.kind='wave_reward' AND u.login IN ('alexey','maria','dmitry','irina')`).Scan(&n)
	if n != 4 { t.Fatalf("every member incl. lead and work-mode maria gets the chest: %d", n) }
}

func TestSettleWaveTwiceNoDoubleReward(t *testing.T) {
	ts, pool, s := testServer(t)
	ctx := context.Background()
	irina := login(t, ts, "irina")
	irina.must(200, "POST", "/tasks/"+findTask(t, irina, "Импорт платежей из банка")["id"].(string)+"/transition", map[string]any{"to": "done"}, "")
	keepOnlyInPlan(t, pool, "Платформа", "PLT-10")
	endCurrentWave(t, pool, "Платформа")
	errs := make(chan error, 2)
	go func() { errs <- s.coopTick(ctx) }()
	go func() { errs <- s.coopTick(ctx) }()
	if e1, e2 := <-errs, <-errs; e1 != nil || e2 != nil { t.Fatal(e1, e2) }
	var n, users int
	_ = pool.QueryRow(ctx, `SELECT count(*), count(DISTINCT user_id) FROM ledger WHERE kind='wave_reward'`).Scan(&n, &users)
	if n == 0 || n != users { t.Fatalf("one wave reward per user: %d rows for %d users", n, users) }
}

func TestReopenedFromPreviousWaveCountsOnce(t *testing.T) {
	ts, pool, s := testServer(t)
	ctx := context.Background()
	irina, maria := login(t, ts, "irina"), login(t, ts, "maria")
	id := findTask(t, irina, "Импорт платежей из банка")["id"].(string)
	irina.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "done"}, "")
	endCurrentWave(t, pool, "Платформа")
	if err := s.coopTick(ctx); err != nil { t.Fatal(err) }
	irina.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "in_progress", "comment": "Нет теста"}, "")
	maria.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "review"}, "")
	irina.must(200, "POST", "/tasks/"+id+"/transition", map[string]any{"to": "done"}, "")
	var w waveRow
	_ = pool.QueryRow(ctx, `SELECT id::text, team_id::text, starts_at, ends_at FROM waves WHERE settled_at IS NULL AND team_id=(SELECT id FROM teams WHERE name='Платформа')`).Scan(&w.ID, &w.TeamID, &w.StartsAt, &w.EndsAt)
	tally, err := tallyWave(ctx, pool, w)
	if err != nil { t.Fatal(err) }
	if tally.Closed < 5 || tally.Closed > tally.Planned { t.Fatalf("re-approved task counts once: %+v", tally) }
	var p int
	_ = pool.QueryRow(ctx, `SELECT points FROM wave_plan WHERE wave_id=$1 AND task_id=$2`, w.ID, id).Scan(&p)
	if p != 5 { t.Fatalf("re-entered the plan with its points: %d", p) }
}

func TestOverdueDamageOncePerDayAndFloor(t *testing.T) {
	_, pool, _ := testServer(t)
	ctx := context.Background()
	var team string
	_ = pool.QueryRow(ctx, `SELECT id::text FROM teams WHERE name='Платформа'`).Scan(&team)
	_, _ = pool.Exec(ctx, `UPDATE fortresses SET hp=100 WHERE team_id=$1`, team)
	friday := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	_, _ = pool.Exec(ctx, `UPDATE tasks SET due_date=$1 WHERE number='PLT-10'`, friday.AddDate(0, 0, -3)) // 5 баллов, review
	run := func(day time.Time) int {
		tx, _ := pool.Begin(ctx)
		defer tx.Rollback(ctx)
		d, err := applyOverdueDamage(ctx, tx, team, day)
		if err != nil { t.Fatal(err) }
		_ = tx.Commit(ctx)
		return d
	}
	first := run(friday)
	if first < 10 { t.Fatalf("PLT-10 alone deals 10: %d", first) }
	if again := run(friday); again != 0 { t.Fatalf("same day twice: %d", again) }
	if sunday := run(friday.AddDate(0, 0, 2)); sunday != 0 { t.Fatalf("no damage on a day off: %d", sunday) }
	_, _ = pool.Exec(ctx, `UPDATE fortresses SET hp=12 WHERE team_id=$1`, team)
	run(friday.AddDate(0, 0, 3))
	var hp int
	_ = pool.QueryRow(ctx, `SELECT hp FROM fortresses WHERE team_id=$1`, team).Scan(&hp)
	if hp != 10 { t.Fatalf("floor 10: %d", hp) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && DATABASE_URL=... go test ./internal/app/ -run 'Settle|Reopened|Overdue' -v`
Expected: FAIL — `undefined: s.coopTick`.

- [ ] **Step 3: Implement `tallyWave`, `settleWave`, `applyOverdueDamage`, `coopTick`, the ticker**

- [ ] **Step 4: Run tests**

Run: `cd server && DATABASE_URL=... go test ./... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/internal/app/waves.go server/internal/app/store.go server/internal/app/coop_e2e_test.go
git commit -m "Волна: подсчёт корон, общая награда, урон от просрочек, фоновый тик"
```

---

### Task 5: API крепости и истории волн

**Files:**
- Create: `server/internal/app/fortress.go`
- Modify: `server/internal/app/server.go` (маршруты)
- Test: `server/internal/app/coop_e2e_test.go`

**Interfaces:**
- Consumes: `ensureWave`, `tallyWave` (Tasks 3–4), `game.EnemyKind`.
- Produces:
  - `GET /api/v1/fortress` → 200:
    ```json
    { "fortress": {"hp": 85, "level": 1, "siege": false},
      "wave": {"id": "...", "starts_at": "...", "ends_at": "...", "days_left": 3, "planned": 30, "closed": 12, "overdue": 1, "overdue_p1": 0, "crowns_now": 0},
      "enemies": [{"task_id": "...", "number": "PLT-10", "title": "...", "points": 5, "priority": "P1", "due_date": "2026-10-15",
                   "kind": "giant", "row": 1, "at_gates": false, "overdue": true}],
      "fog": 3,
      "last_wave": {"id": "...", "ends_at": "...", "crowns": 2, "summary": "..."} }
    ```
    `row` = номер приоритета (P1 → 1). `at_gates` = не просрочена и до срока ≤ 2 дней. `siege` = `hp <= FortressFloor`. Имён исполнителей в ответе нет. `last_wave` — `null`, если закрытых волн нет. Без команды — `refuse(403, "not_a_player", …)` как в `loadBalance`. Обработчик сначала вызывает `ensureWave` в транзакции.
  - `GET /api/v1/waves` → `{"items": [{"id","starts_at","ends_at","planned","closed","overdue","crowns","summary","settled_at"}]}` — 12 последних закрытых волн команды.

- [ ] **Step 1: Write the failing test**

```go
func TestFortressAPI(t *testing.T) {
	ts, _, _ := testServer(t)
	alexey := login(t, ts, "alexey")
	f := alexey.must(200, "GET", "/fortress", nil, "").Body
	if num(f["fortress"].(map[string]any)["hp"]) <= 0 { t.Fatalf("hp: %v", f) }
	enemies := f["enemies"].([]any)
	if len(enemies) == 0 { t.Fatal("seeded board has enemies") }
	for _, e := range enemies {
		m := e.(map[string]any)
		if _, leaked := m["assignee_name"]; leaked { t.Fatal("no names next to enemies") }
		if m["kind"] == "" { t.Fatalf("kind: %v", m) }
	}
	if num(f["wave"].(map[string]any)["planned"]) == 0 { t.Fatal("plan snapshot exists") }
	login(t, ts, "admin").refused("not_a_player", "GET", "/fortress", nil)
	alexey.must(200, "GET", "/waves", nil, "")
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./internal/app/ -run TestFortressAPI -v` → FAIL 404.

- [ ] **Step 3: Implement `fortress(w, r)` and `waves(w, r)` handlers; register `r.Get("/fortress", s.fortress)` and `r.Get("/waves", s.waves)`**

- [ ] **Step 4: Run** — `go test ./... -count=1` → PASS.

- [ ] **Step 5: Commit**

```bash
git add server/internal/app/fortress.go server/internal/app/server.go server/internal/app/coop_e2e_test.go
git commit -m "API: крепость команды и история волн"
```

---

### Task 6: Дружеские бои вместо набегов

**Files:**
- Modify: `server/internal/app/battles.go` (переписать `candidates`/`opponents`/`opponentCity`/`attack`), `server/internal/app/server.go`, `server/internal/app/auth.go` (`updateSettings`, `me`), `server/internal/app/player.go` (удалить `battleRating`, убрать `shield_until` из `playerState`), `server/internal/game/battle.go` (удалить `BattleReward`, `RewardFor`, `InRange`), `server/internal/game/economy.go` (удалить поля набегов `AttackEnergy`, `ShieldMinutes`, `RematchMinutes`, `BattleScope`, `BattleGoldBase`, `BattleGoldPerPercent`, `BattleXPPerStar` и их нормализацию), `server/internal/game/game_test.go` (удалить тесты удалённых функций, если есть), `server/internal/app/seed.go`
- Test: `server/internal/app/e2e_test.go:277-305` (раздел 5), `server/internal/app/coop_e2e_test.go`

**Interfaces:**
- Produces:
  - `GET /api/v1/duels` → `{"items": [{"user_id","display_name","townhall","defense","available","reason"}], "today": 1, "per_day": 3}` — только своя команда, `accept_friendly = true`, не сам игрок. `available=false` с причиной «Лимит дружеских боёв на сегодня исчерпан», если сегодня (UTC) уже `FriendlyPerDay` боёв.
  - `GET /api/v1/duels/{id}/city` — то же, что старый `opponentCity`, но только для доступного соперника из списка.
  - `POST /api/v1/battles/friendly` `{defender_id, army}` → 201 `battleDTO`. Отказы: `invalid_opponent` (422), `not_found` (404 — не в команде или не принимает бои), `friendly_limit` (429), `invalid_army`/`not_enough_troops`/`empty_army` как раньше. Энергия не списывается, войска не теряются, трофеи не меняются, щит не ставится; `battles.kind='friendly'`, `gold_reward=0`, `attacker_trophies=0`, `defender_trophies=0`; `applyLedger(Kind:"battle_reward", XP: FriendlyXP)`. Аудит защитнику: «Дружеский бой: на ваш город вышел(а) {имя}. Ничего не потеряно.»
  - `PATCH /me/settings` принимает `accept_friendly *bool`; `GET /me` возвращает `accept_friendly`.
  - Удаляются маршруты `/opponents`, `/opponents/{id}/city`, `POST /battles`, `/ratings/battle`.
  - `battleDTO` и `listBattles` возвращают `kind`.
  - Сид: `accept_friendly=true` для `alexey`, `maria`, `anna`.

- [ ] **Step 1: Rewrite section 5 of `TestEndToEnd` (failing)**

```go
	// 5. Army and a friendly duel: no energy, no losses, no trophies; limit per day.
	// (обучение войск — как было)
	var maria string
	for _, o := range items(alexey.must(200, "GET", "/duels", nil, "")) {
		if o["display_name"] == "Мария Иванова" { maria = o["user_id"].(string) }
	}
	before := alexey.must(200, "GET", "/player", nil, "").Body
	armyBefore := alexey.must(200, "GET", "/army", nil, "").Body["troops"].(map[string]any)
	duel := alexey.must(201, "POST", "/battles/friendly", map[string]any{"defender_id": maria, "army": map[string]int{"guard": 8, "raider": 6, "ram": 3}}, "")
	if duel.Body["kind"] != "friendly" || len(duel.Body["rounds"].([]any)) == 0 { t.Fatalf("duel: %s", duel.Raw) }
	after := alexey.must(200, "GET", "/player", nil, "").Body
	if num(after["energy"]) != num(before["energy"]) || num(after["trophies"]) != num(before["trophies"]) { t.Fatalf("free: %v → %v", before, after) }
	if got := alexey.must(200, "GET", "/army", nil, "").Body["troops"].(map[string]any); num(got["guard"]) != num(armyBefore["guard"]) { t.Fatal("no losses") }
	for i := 0; i < 2; i++ { alexey.must(201, "POST", "/battles/friendly", map[string]any{"defender_id": maria, "army": map[string]int{"guard": 1}}, "") }
	alexey.refused("friendly_limit", "POST", "/battles/friendly", map[string]any{"defender_id": maria, "army": map[string]int{"guard": 1}})
	dmitryID := login(t, ts, "dmitry").must(200, "GET", "/me", nil, "").Body["id"].(string)
	alexey.refused("not_found", "POST", "/battles/friendly", map[string]any{"defender_id": dmitryID, "army": map[string]int{"guard": 1}}) // не принимает бои
```

Проверка «бои не меняют рабочий рейтинг» сохраняется, но через `/progress/me` (появится в Task 7; до него — прямой SQL по `work_points_events`).

- [ ] **Step 2: Run** — `go test ./internal/app/ -run TestEndToEnd -v` → FAIL 404 на `/duels`.

- [ ] **Step 3: Implement the friendly-duel handlers, settings flag, route changes; delete raid code and `RewardFor`/`InRange`/`BattleReward`**

- [ ] **Step 4: Run** — `cd server && go vet ./... && DATABASE_URL=... go test ./... -count=1` → PASS.

- [ ] **Step 5: Commit**

```bash
git add server
git commit -m "Дружеские бои вместо набегов: без энергии, потерь и трофеев"
```

---

### Task 7: «Прогресс» вместо рейтинга и правило «не для оценки» (сервер)

**Files:**
- Create: `server/internal/app/progress.go`
- Modify: `server/internal/app/player.go` (удалить `workRating`), `server/internal/app/server.go`, `server/internal/app/auth.go`
- Test: `server/internal/app/e2e_test.go` (`weekPoints` → `/progress/me`), `server/internal/app/coop_e2e_test.go`

**Interfaces:**
- Produces:
  - `GET /api/v1/progress/me` → `{"weeks": ["2026-08-24", …8], "weekly": [5,0,…], "this_week": 7, "best_week": 9, "total": 47}` — только свои `work_points_events` (по `user_id`), без чужих данных.
  - `GET /api/v1/progress/team` → `{"weeks": […8], "team_weekly": […8], "overdue_now": 1, "waves": [{"ends_at","crowns","summary"} … последние 8]}` — без разбивки по людям для любой роли. `?team=` для администратора — как в `teamScope`.
  - `/burndown` остаётся.
  - Удаляется `GET /ratings/work`.
  - `GET /me` возвращает `game_rules_ack: bool`; `POST /me/game-rules-ack` ставит `users.game_rules_ack_at=now()` (идемпотентно), 200 `{"game_rules_ack": true}`.
  - Новый хелпер теста: `func weekPoints(t *testing.T, c *client) float64` — читает `this_week` из `/progress/me` самого клиента.

- [ ] **Step 1: Write the failing tests** — в `TestEndToEnd` заменить вызовы `weekPoints(t, irina, "алексей")` на `weekPoints(t, alexey)` и т. п.; добавить:

```go
func TestProgressHasNoPerPersonTable(t *testing.T) {
	ts, _, _ := testServer(t)
	irina := login(t, ts, "irina")
	team := irina.must(200, "GET", "/progress/team", nil, "")
	if _, ok := team.Body["items"]; ok || strings.Contains(team.Raw, "display_name") { t.Fatalf("no per-person rows: %s", team.Raw) }
	login(t, ts, "alexey").must(200, "GET", "/progress/me", nil, "")
	irina.must(404, "GET", "/ratings/work", nil, "")
	if irina.must(200, "GET", "/me", nil, "").Body["game_rules_ack"] != false { t.Fatal("not acknowledged yet") }
	irina.must(200, "POST", "/me/game-rules-ack", map[string]any{}, "")
	if irina.must(200, "GET", "/me", nil, "").Body["game_rules_ack"] != true { t.Fatal("acknowledged") }
}
```

Если неизвестный путь `/api/v1/...` сейчас отдаёт не 404, а SPA, проверить отсутствие маршрута через код ответа, который реально отдаёт сервер, и зафиксировать его в тесте.

- [ ] **Step 2: Run** → FAIL 404 на `/progress/team`.

- [ ] **Step 3: Implement handlers, routes, `me` field, ack endpoint**

- [ ] **Step 4: Run** — `go test ./... -count=1` → PASS.

- [ ] **Step 5: Commit**

```bash
git add server
git commit -m "Прогресс вместо рейтинга, подтверждение правила игры"
```

---

### Task 8: Настройки администратора и демо-данные

**Files:**
- Modify: `server/internal/app/admin.go`, `server/internal/app/server.go`, `server/internal/app/seed.go`
- Test: `server/internal/app/coop_e2e_test.go`

**Interfaces:**
- Produces:
  - `GET /api/v1/admin/teams` → `{"items": [{"id","name","work_days":[1,2,3,4,5],"wave_weeks":1}]}`.
  - `PATCH /api/v1/admin/teams/{id}` `{work_days?: int[], wave_weeks?: int}` → 200 строка команды. Отказы: `invalid_work_days` (422 — пусто, повторы или вне 1..7), `invalid_wave_weeks` (422 — вне 1..4). Аудит `admin.team`.
  - Новые поля `Settings` (Task 1) уже идут через существующие `GET/PUT /admin/settings`.
  - Сид: крепость «Платформы» `hp=85`; три закрытые волны «Платформы» в прошлых неделях с короной 3, 1, 2 и `summary` из `game.WaveSummary`; задача PLT-10 (P1) со сроком в прошлом, чтобы был просроченный враг; `ensureWave` для каждой команды после вставки задач.

- [ ] **Step 1: Write the failing test**

```go
func TestAdminTeamsAndSeed(t *testing.T) {
	ts, _, _ := testServer(t)
	admin := login(t, ts, "admin")
	teams := items(admin.must(200, "GET", "/admin/teams", nil, ""))
	id := teams[0]["id"].(string)
	admin.refused("invalid_wave_weeks", "PATCH", "/admin/teams/"+id, map[string]any{"wave_weeks": 5})
	admin.refused("invalid_work_days", "PATCH", "/admin/teams/"+id, map[string]any{"work_days": []int{}})
	got := admin.must(200, "PATCH", "/admin/teams/"+id, map[string]any{"wave_weeks": 2, "work_days": []int{1, 2, 3, 4}}, "").Body
	if num(got["wave_weeks"]) != 2 { t.Fatalf("patched: %v", got) }
	login(t, ts, "alexey").refused("forbidden", "PATCH", "/admin/teams/"+id, map[string]any{"wave_weeks": 1})
	if len(items(login(t, ts, "alexey").must(200, "GET", "/waves", nil, ""))) < 3 { t.Fatal("seeded past waves") }
}
```

- [ ] **Step 2: Run** → FAIL 404.

- [ ] **Step 3: Implement admin team endpoints and seed additions**

- [ ] **Step 4: Run** — `cd server && go vet ./... && DATABASE_URL=... go test ./... -count=1` → PASS.

- [ ] **Step 5: Commit**

```bash
git add server
git commit -m "Админка: рабочие дни и длина волны команды; демо-данные крепости и волн"
```

---

### Task 9: Клиент — типы, API и правила крепости

**Files:**
- Modify: `web/src/types.ts`, `web/src/stores/session.ts`
- Create: `web/src/lib/fortress.ts`
- Test: `web/tests/fortress.test.ts`

**Interfaces:**
- Consumes: ответы API из Tasks 5–8.
- Produces:
  - Типы (только добавления, чтобы существующие страницы собирались): `Fortress`, `WaveNow`, `Enemy` (`task_id`, `number`, `title`, `points`, `priority`, `due_date: string | null`, `kind: 'goblin'|'knight'|'giant'|'dragon'`, `row: 1|2|3|4`, `at_gates`, `overdue`), `WaveSummary`, `FortressState` (весь ответ `/fortress`), `MyProgress`, `TeamProgress`, `Duelist`; в `EconomySettings` — новые поля Task 1; `Battle` получает `kind: 'raid'|'friendly'`; `Me` получает `accept_friendly`, `game_rules_ack`. Удаление `Opponent`, `shield_until` и полей набегов — в Task 11 вместе со страницами, которые их используют.
  - `session` store: `setAcceptFriendly(v: boolean): Promise<void>`, `ackGameRules(): Promise<void>`.
  - `lib/fortress.ts`:
    - `enemyName(kind: Enemy['kind']): string` — «Гоблин», «Рыцарь», «Великан», «Дракон».
    - `crownsText(n: number): string` — «0 корон», «1 корона», «2 короны», «5 корон».
    - `daysLeftText(n: number): string` — «последний день», «1 день», «2 дня», «5 дней».
    - `enemyRows(enemies: Enemy[]): Enemy[][]` — 4 ряда по `row`; внутри ряда сначала `overdue`, затем `at_gates`, затем по `due_date` (без срока — в конце).
    - `waveProgress(w: WaveNow): number` — `closed / planned` в диапазоне 0..1, 0 при пустом плане.

- [ ] **Step 1: Write the failing tests** in `web/tests/fortress.test.ts`

```ts
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { crownsText, daysLeftText, enemyRows, waveProgress } from '../src/lib/fortress.ts'
import type { Enemy, WaveNow } from '../src/types.ts'

const e = (o: Partial<Enemy>): Enemy => ({ task_id: 'x', number: 'T-1', title: 't', points: 3, priority: 'P2', due_date: null, kind: 'knight', row: 2, at_gates: false, overdue: false, ...o })

test('crowns and days in Russian', () => {
  assert.equal(crownsText(1), '1 корона')
  assert.equal(crownsText(3), '3 короны')
  assert.equal(crownsText(0), '0 корон')
  assert.equal(daysLeftText(0), 'последний день')
  assert.equal(daysLeftText(2), '2 дня')
  assert.equal(daysLeftText(5), '5 дней')
})

test('enemy rows: overdue first, then at the gates, then by due date', () => {
  const rows = enemyRows([e({ number: 'a', row: 1, due_date: '2026-10-20' }), e({ number: 'b', row: 1, overdue: true }), e({ number: 'c', row: 1, at_gates: true }), e({ number: 'd', row: 4 })])
  assert.equal(rows.length, 4)
  assert.deepEqual(rows[0].map(x => x.number), ['b', 'c', 'a'])
  assert.deepEqual(rows[3].map(x => x.number), ['d'])
})

test('wave progress', () => {
  assert.equal(waveProgress({ planned: 30, closed: 24 } as WaveNow), 0.8)
  assert.equal(waveProgress({ planned: 0, closed: 0 } as WaveNow), 0)
})
```

- [ ] **Step 2: Run** — `cd web && npm test` → FAIL (модуль не найден).

- [ ] **Step 3: Implement types, store actions, `lib/fortress.ts`**

- [ ] **Step 4: Run** — `cd web && npm test && npm run build` → тесты PASS, сборка без ошибок.

- [ ] **Step 5: Commit**

```bash
git add web/src/types.ts web/src/stores/session.ts web/src/lib/fortress.ts web/tests/fortress.test.ts
git commit -m "Клиент: типы и правила крепости"
```

---

### Task 10: Клиент — страница «Крепость» и навигация

**Files:**
- Create: `web/src/pages/FortressPage.vue`, `web/src/components/FortressScene.vue`
- Modify: `web/src/app/router.ts`, `web/src/app/App.vue`, `web/src/pages/HomePage.vue`

**Interfaces:**
- Consumes: `FortressState`, `lib/fortress.ts` (Task 9); `GET /fortress`, `GET /waves`.
- Produces:
  - Маршрут `/fortress` (`FortressPage`, `meta: { title: 'Крепость', game: true }`). Маршруты `/fortress/duels` и `/progress` добавляет Task 11.
  - Навигация (`App.vue`, массив `all`): пункт «Бои» (`/opponents`) заменяется на «Крепость» (`/fortress`, game). Мобильная — первые 5 после фильтра без `/army`, `/admin`, `/review`.
  - `FortressScene.vue` props: `{ fortress: Fortress; rows: Enemy[][]; fog: number }`, emits `select: [enemy: Enemy]`. SVG в стиле существующей `CityScene.vue`: крепость внизу, 4 ряда врагов над ней (иконка по `kind`, размер растёт с баллами), просроченные — у стены, туман сверху с числом `fog`. Каждый враг — `role="button"`, `aria-label="{Враг}: {номер} {название}, {баллы} балл., {P}"`, фокус с клавиатуры, Enter = select.
  - `FortressPage.vue`: заголовок «Крепость команды»; строка показателей — прочность `hp %` (и «Осада», если `siege`), «Волна: {закрыто} из {план} баллов» + полоса `waveProgress`, «До конца волны: {daysLeftText}», «Прошлая волна: {crownsText}»; сцена; карточка выбранного врага с кнопкой «Открыть на доске» (`/board?task={id}` — если доска не умеет открывать задачу по параметру, добавить это в `BoardPage.vue`); список истории волн (`summary`); ссылка «Дружеские бои». Пустые состояния: врагов нет — «Тишина на границе. Возьмите задачи из бэклога в план.»; `not_a_player` — «Крепость есть у команд. Попросите администратора добавить вас в команду.»
  - `HomePage.vue`: карточка «Город» получает строку «Крепость: {hp}% · волна {закрыто}/{план}» со ссылкой на `/fortress`.

- [ ] **Step 1: Implement pages, scene, routes, nav**

- [ ] **Step 2: Verify build** — `cd web && npm test && npm run build`. Expected: тесты PASS, сборка без ошибок типов.

- [ ] **Step 3: Verify in the browser** — `scripts/dev.sh`, войти как `alexey`: «Крепость» в меню, враги в 4 рядах, просроченный PLT-10 у стены, нажатие на врага открывает карточку и ведёт на доску; переключение в «Работа» прячет «Крепость» и уводит со страницы; ширина 360 px без горизонтальной прокрутки; навигация по врагам клавиатурой.

- [ ] **Step 4: Commit**

```bash
git add web
git commit -m "Клиент: крепость команды, орда, история волн, новая навигация"
```

---

### Task 11: Клиент — «Прогресс», дружеские бои, правило игры, настройки, админка

**Files:**
- Create: `web/src/pages/ProgressPage.vue`, `web/src/pages/DuelsPage.vue`, `web/src/components/GameRulesDialog.vue`
- Delete: `web/src/pages/RatingsPage.vue`, `web/src/pages/OpponentsPage.vue`
- Modify: `web/src/pages/BattlePage.vue`, `web/src/pages/SettingsPage.vue`, `web/src/pages/AdminPage.vue`, `web/src/app/App.vue`, `web/src/app/router.ts`, `web/src/types.ts`

**Interfaces:**
- Consumes: Tasks 6–9.
- Produces:
  - Маршруты: `/fortress/duels` (`DuelsPage`, `meta: { title: 'Дружеские бои', game: true }`), `/progress` (`ProgressPage`, `meta: { title: 'Прогресс' }`); редиректы `/opponents → /fortress/duels`, `/ratings → /progress`. Пункт навигации «Рейтинги» → «Прогресс».
  - `types.ts`: удалить `Opponent`, `Player.shield_until` и поля набегов в `EconomySettings`.
  - `ProgressPage.vue`: блок «Мой прогресс» (`WeeklyBars` по `/progress/me`, «Эта неделя», «Лучшая неделя», «Всего»), блок «Команда» (`WeeklyBars` по `team_weekly`, «Просрочено сейчас», список последних волн с коронами), существующий `BurndownChart`. Пояснение: «Здесь только ваш личный прогресс и общий прогресс команды. Сравнения между коллегами нет.» Никаких таблиц по людям.
  - `DuelsPage.vue`: на основе удалённого `OpponentsPage.vue` — список коллег, принимающих бои, превью города, выбор отряда, кнопка «Дружеский бой», счётчик «Сегодня: {today} из {per_day}». Текст: «Дружеский бой ничего не стоит: войска не гибнут, трофеи не меняются. Награда — {friendly_xp} опыта.» Пустое состояние: «Пока никто из команды не принимает дружеские бои.» Отказ `friendly_limit` показывается тостом с текстом сервера.
  - `BattlePage.vue`: заголовок «Дружеский бой: {атакующий} и {защитник}»; убрать трофеи, золото и потери из итога; ссылка «Назад» → `/fortress/duels`.
  - `SettingsPage.vue`: переключатель «Принимать дружеские бои» (`session.setAcceptFriendly`); под переключателем «Работа / Игра» постоянный текст: «Игровые данные не используются для оценки работы. Режим «Работа» убирает игру, но командные награды продолжают копиться.»
  - `GameRulesDialog.vue`: на `Modal.vue`; показывается в `App.vue`, когда `session.gameOn && session.me && !session.me.game_rules_ack`. Заголовок «Как устроена игра». Текст: «Команда вместе защищает крепость от своего бэклога. Задачи, взятые в план, — враги; подтверждённая задача — побеждённый враг. Раз в неделю волна подводит итог, и вся команда получает общую награду. Игровые данные не используются для оценки работы.» Кнопка «Понятно» → `session.ackGameRules()`. Escape и закрытие крестиком тоже подтверждают.
  - `AdminPage.vue`: в «Экономике» поля новых коэффициентов (Task 1), удалить поля набегов; новый раздел «Команды»: для каждой команды чекбоксы рабочих дней (Пн…Вс) и выбор длины волны 1–4 недели, сохранение через `PATCH /admin/teams/{id}`.

- [ ] **Step 1: Implement the pages, dialog and admin section**

- [ ] **Step 2: Verify build** — `cd web && npm test && npm run build` → PASS.

- [ ] **Step 3: Verify in the browser** — как `alexey`: при первом входе окно правила, после «Понятно» не появляется снова (и после перезагрузки); «Прогресс» без таблицы по людям; дружеский бой против Марии — энергия не меняется, после трёх боёв отказ; как `irina` («Прогресс» — без таблицы и для руководителя); как `admin`: сохранение рабочих дней и длины волны, неверное значение показывает ошибку сервера.

- [ ] **Step 4: Commit**

```bash
git add web
git commit -m "Клиент: прогресс, дружеские бои, правило игры, настройки команд"
```

---

### Task 12: Полная проверка и README

**Files:**
- Modify: `README.md` (разделы «Идея», «Где что находится», «Правила» → «Бои» заменить на «Крепость и волны» и «Дружеские бои»; убрать боевой рейтинг; добавить фразу правила; обновить «Как пройти весь цикл»)

Владелец продукта правил README в рабочей копии (незакоммиченные изменения). Перед правкой прочитать текущий файл и вносить изменения поверх его версии, не откатывая чужие правки.

- [ ] **Step 1: Run the full suite** — `scripts/test.sh`. Expected: `go vet` без замечаний, все Go-тесты и e2e PASS, `npm test` PASS, сборка без ошибок.

- [ ] **Step 2: Update README**; таблица маршрутов: `/fortress`, `/fortress/duels`, `/progress` вместо `/opponents`, `/ratings`. Скриншоты не пересъёмка — это этап 2, старые `opponents.png`/`ratings.png` убрать из таблицы.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "README: кооператив против бэклога"
```
