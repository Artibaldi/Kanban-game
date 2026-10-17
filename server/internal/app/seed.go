package app

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

// Demo data for the local prototype: two teams, eight people, tasks in every column, six weeks of confirmed work
// (paid through the same approveTask as the live service) and cities from town hall 1 to 3.

type seedUser struct {
	login, name, role, team, title string
}

var seedUsers = []seedUser{
	{"admin", "Администратор", "admin", "", "Администратор сервиса"},
	{"irina", "Ирина Соколова", "lead", "Платформа", "Руководитель команды"},
	{"alexey", "Алексей Петров", "user", "Платформа", "Backend-разработчик"},
	{"maria", "Мария Иванова", "user", "Платформа", "Backend-разработчик"},
	{"dmitry", "Дмитрий Орлов", "user", "Платформа", "DevOps-инженер"},
	{"sergey", "Сергей Волков", "lead", "Мобайл", "Руководитель команды"},
	{"anna", "Анна Козлова", "user", "Мобайл", "iOS-разработчик"},
	{"pavel", "Павел Смирнов", "user", "Мобайл", "Android-разработчик"},
}

type seedTask struct {
	project, title, desc string
	points               int
	priority, state      string
	assignee, creator    string // logins; creator "" = the team lead
	agreed               bool
	createdAgo           int  // days
	approvedAgo          int  // days, for done
	due                  *int // days from now (negative = past)
}

func dp(n int) *int { return &n }

var seedTasks = []seedTask{
	{"PLT", "Схема тарифов в БД", "Таблицы тарифов, периодов и цен. Миграции и индексы.", 5, "P2", "done", "alexey", "", true, 44, 38, nil},
	{"PLT", "API выставления счёта", "POST /invoices с идемпотентностью и валидацией.", 8, "P1", "done", "alexey", "", true, 40, 33, nil},
	{"OPS", "Мониторинг очередей", "Метрики глубины и возраста сообщений, дашборд.", 3, "P3", "done", "maria", "", true, 36, 30, nil},
	{"PLT", "Ретраи вебхуков платёжного шлюза", "Экспоненциальная задержка, журнал попыток.", 5, "P2", "done", "maria", "", true, 32, 26, nil},
	{"OPS", "Бэкап Postgres в S3", "Ежедневный бэкап и проверка восстановления.", 3, "P2", "done", "dmitry", "", true, 28, 22, nil},
	{"PLT", "Экспорт счетов в CSV", "Выгрузка за период с фильтрами.", 2, "P3", "done", "dmitry", "", true, 24, 19, nil},
	{"PLT", "Пересчёт НДС для юрлиц", "Новые ставки с 1 числа, пересчёт черновиков.", 5, "P1", "done", "alexey", "", true, 21, 15, nil},
	{"OPS", "Алерты на ошибки 5xx", "Порог по доле ошибок, уведомление в дежурный канал.", 2, "P3", "done", "maria", "", true, 17, 12, nil},
	{"OPS", "Ротация TLS-сертификатов", "Автоматический выпуск и перезагрузка без простоя.", 2, "P2", "done", "irina", "irina", true, 14, 8, nil},
	{"PLT", "Скидочные купоны", "Купон на процент или сумму, срок действия.", 3, "P2", "done", "alexey", "", true, 14, 9, nil},
	{"OPS", "Обновить Go до 1.26", "Сборка, линтеры, прогон тестов.", 1, "P4", "done", "dmitry", "", true, 10, 6, nil},
	{"PLT", "Кеш справочника валют", "TTL 10 минут, прогрев при старте.", 3, "P3", "done", "maria", "", true, 9, 4, nil},
	{"PLT", "Отчёт по дебиторке", "Сводка просроченных счетов по клиентам.", 5, "P2", "done", "alexey", "", true, 7, 2, nil},
	{"PLT", "Уведомления о просрочке оплаты", "Письмо клиенту через 1, 3 и 7 дней просрочки.", 3, "P2", "review", "alexey", "", true, 6, 0, dp(2)},
	{"OPS", "Лимиты подключений pgbouncer", "Пулы по сервисам, алерт на исчерпание.", 2, "P2", "review", "irina", "irina", true, 5, 0, dp(1)},
	{"PLT", "Импорт платежей из банка", "Разбор выписки, сопоставление со счетами.", 5, "P1", "review", "maria", "", true, 8, 0, dp(-1)},
	{"PLT", "Частичные возвраты", "Возврат части суммы с пересчётом налога.", 5, "P2", "in_progress", "alexey", "", true, 4, 0, dp(3)},
	{"OPS", "Логи в единый формат", "JSON-логи с trace_id во всех сервисах.", 3, "P3", "in_progress", "dmitry", "", true, 5, 0, dp(6)},
	{"PLT", "Валидация ИНН при регистрации", "Контрольная сумма и проверка длины.", 2, "P3", "ready", "", "", true, 3, 0, dp(5)},
	{"PLT", "Вебхук для CRM", "Событие «счёт оплачен» в CRM.", 3, "P2", "ready", "", "", true, 3, 0, dp(7)},
	{"OPS", "Дашборд задержек API", "p50/p95/p99 по ручкам.", 2, "P3", "ready", "maria", "", true, 2, 0, nil},
	{"PLT", "Мультивалютные счета", "Счёт в валюте клиента, курс на дату.", 8, "P3", "backlog", "", "", true, 2, 0, nil},
	{"PLT", "Шаблоны писем", "Редактируемые шаблоны уведомлений.", 1, "P4", "backlog", "", "dmitry", false, 1, 0, nil},
	{"OPS", "Автоскейлинг воркеров", "Масштабирование по глубине очереди.", 5, "P3", "backlog", "", "maria", false, 1, 0, nil},

	{"MOB", "Экран входа по коду", "Ввод кода из SMS, повторная отправка.", 3, "P2", "done", "anna", "", true, 34, 28, nil},
	{"MOB", "Push-уведомления", "Регистрация токена, категории уведомлений.", 5, "P1", "done", "pavel", "", true, 27, 21, nil},
	{"MOB", "Тёмная тема", "Системная и ручная, все экраны.", 2, "P3", "done", "anna", "", true, 18, 14, nil},
	{"MOB", "Офлайн-кеш каталога", "Кеш на устройстве, синхронизация при сети.", 8, "P2", "done", "pavel", "", true, 16, 7, nil},
	{"MOB", "Crash-репорты", "Сбор и группировка падений.", 3, "P2", "done", "anna", "", true, 8, 3, nil},
	{"MOB", "Deep links из писем", "Открытие нужного экрана по ссылке.", 3, "P2", "review", "pavel", "", true, 5, 0, dp(2)},
	{"MOB", "Оплата через СБП", "QR и переход в банковское приложение.", 5, "P1", "in_progress", "anna", "", true, 4, 0, dp(4)},
	{"MOB", "Экран профиля", "Аватар, имя, настройки уведомлений.", 2, "P3", "ready", "", "", true, 2, 0, nil},
	{"MOB", "Виджет на главный экран", "Баланс и последние операции.", 5, "P3", "backlog", "", "pavel", false, 1, 0, nil},
}

type seedBuilding struct {
	kind         string
	level, x, y  int
	buildingSecs int // > 0: under construction to level+1, finishing in that many seconds
}

func ring(level int, cells ...[2]int) []seedBuilding {
	out := []seedBuilding{}
	for _, c := range cells {
		out = append(out, seedBuilding{"wall", level, c[0], c[1], 0})
	}
	return out
}

var inner = [][2]int{{4, 4}, {5, 4}, {6, 4}, {4, 5}, {6, 5}, {4, 6}, {5, 6}, {6, 6}}

func seedCities() map[string][]seedBuilding {
	return map[string][]seedBuilding{
		"alexey": append([]seedBuilding{{"townhall", 3, 5, 5, 0}, {"barracks", 3, 2, 2, 0}, {"barracks", 1, 2, 4, 0}, {"storage", 2, 8, 2, 0}, {"storage", 2, 9, 4, 0},
			{"tower", 3, 3, 8, 0}, {"tower", 2, 8, 8, 0}, {"tower", 2, 5, 1, 0}}, ring(3, inner...)...),
		"maria": append([]seedBuilding{{"townhall", 2, 5, 5, 0}, {"barracks", 2, 2, 3, 0}, {"storage", 2, 8, 3, 0}, {"tower", 2, 3, 8, 0}, {"tower", 1, 8, 8, 1}},
			ring(2, inner[:6]...)...),
		"dmitry": append([]seedBuilding{{"townhall", 1, 5, 5, 0}, {"barracks", 1, 2, 3, 0}, {"storage", 1, 8, 3, 0}, {"tower", 1, 8, 8, 0}},
			ring(1, [2]int{4, 4}, [2]int{5, 4})...),
		"irina": append([]seedBuilding{{"townhall", 2, 5, 5, 0}, {"barracks", 1, 2, 2, 0}, {"storage", 1, 8, 2, 0}, {"tower", 2, 7, 8, 0}},
			ring(1, inner[:4]...)...),
		"sergey": append([]seedBuilding{{"townhall", 2, 5, 5, 0}, {"barracks", 2, 2, 2, 0}, {"storage", 1, 8, 2, 0}, {"tower", 1, 3, 8, 0}, {"tower", 1, 8, 8, 0}},
			ring(1, inner[:5]...)...),
		"anna": append([]seedBuilding{{"townhall", 2, 5, 5, 0}, {"barracks", 2, 3, 2, 0}, {"storage", 2, 8, 2, 0}, {"tower", 2, 3, 8, 0}},
			ring(2, inner[:6]...)...),
		"pavel": append([]seedBuilding{{"townhall", 3, 5, 5, 0}, {"barracks", 3, 2, 2, 0}, {"storage", 2, 8, 2, 0}, {"storage", 1, 9, 4, 0},
			{"tower", 3, 3, 8, 0}, {"tower", 2, 8, 8, 0}}, ring(2, inner...)...),
	}
}

var seedTroops = map[string]map[string]int{
	"alexey": {"guard": 10, "raider": 8, "ram": 3},
	"maria":  {"guard": 6, "raider": 4, "ram": 1},
	"dmitry": {"guard": 4, "raider": 2},
	"irina":  {"guard": 5, "raider": 3},
	"sergey": {"guard": 6, "raider": 4, "ram": 1},
	"anna":   {"guard": 8, "raider": 5, "ram": 2},
	"pavel":  {"guard": 10, "raider": 6, "ram": 4},
}

// seedDuelists accept friendly duels from teammates in the demo.
var seedDuelists = map[string]bool{"alexey": true, "maria": true, "anna": true}

// Seed fills an empty database. It is a no-op once any user exists.
func (s *Server) Seed(ctx context.Context) (bool, error) {
	var n int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n); err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}
	return true, s.inTx(ctx, func(tx pgx.Tx) error { return seed(ctx, tx) })
}

func seed(ctx context.Context, tx pgx.Tx) error {
	now := time.Now()
	day := func(ago int) time.Time { return now.AddDate(0, 0, -ago).Add(-time.Duration(ago%5+1) * time.Hour) }
	teams := map[string]string{}
	for _, name := range []string{"Платформа", "Мобайл"} {
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO teams(name) VALUES($1) RETURNING id::text`, name).Scan(&id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO fortresses(team_id) VALUES($1) ON CONFLICT DO NOTHING`, id); err != nil {
			return err
		}
		teams[name] = id
	}
	users := map[string]actor{}
	for _, u := range seedUsers {
		var id string
		var team any
		if u.team != "" {
			team = teams[u.team]
		}
		if err := tx.QueryRow(ctx, `INSERT INTO users(login,display_name,role,team_id,title,accept_friendly) VALUES($1,$2,$3,$4,$5,$6) RETURNING id::text`,
			u.login, u.name, u.role, team, u.title, seedDuelists[u.login]).Scan(&id); err != nil {
			return err
		}
		a := actor{ID: id, Name: u.name, Login: u.login, Role: u.role}
		if u.team != "" {
			a.TeamID = teams[u.team]
			a.TeamName = u.team
		}
		users[u.login] = a
	}
	projects := map[string]string{}
	for _, p := range []struct{ key, name, team string }{{"PLT", "Биллинг", "Платформа"}, {"OPS", "Инфраструктура", "Платформа"}, {"MOB", "Мобильное приложение", "Мобайл"}} {
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO projects(team_id,key,name) VALUES($1,$2,$3) RETURNING id::text`, teams[p.team], p.key, p.name).Scan(&id); err != nil {
			return err
		}
		projects[p.key] = id
	}
	// Players and cities first: storage caps apply to the historical rewards below.
	for login, layout := range seedCities() {
		u := users[login]
		if _, err := tx.Exec(ctx, `INSERT INTO players(user_id) VALUES($1)`, u.ID); err != nil {
			return err
		}
		if _, err := applyLedger(ctx, tx, ledgerEntry{UserID: u.ID, Kind: "seed", Gold: 300, Energy: 60, Note: "Стартовый запас города", At: ptrTime(now.AddDate(0, 0, -45))}); err != nil {
			return err
		}
		for _, b := range layout {
			if b.buildingSecs > 0 {
				if _, err := tx.Exec(ctx, `INSERT INTO buildings(user_id,type,level,target_level,busy_until,x,y) VALUES($1,$2,$3,$4,now()+$5*interval '1 second',$6,$7)`,
					u.ID, b.kind, b.level, b.level+1, b.buildingSecs*20, b.x, b.y); err != nil {
					return err
				}
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO buildings(user_id,type,level,x,y) VALUES($1,$2,$3,$4,$5)`, u.ID, b.kind, b.level, b.x, b.y); err != nil {
				return err
			}
		}
		for kind, count := range seedTroops[login] {
			if _, err := tx.Exec(ctx, `INSERT INTO troops(user_id,type,count) VALUES($1,$2,$3)`, u.ID, kind, count); err != nil {
				return err
			}
		}
	}
	leadOf := map[string]string{"PLT": "irina", "OPS": "irina", "MOB": "sergey"}
	type pending struct {
		id string
		at time.Time
		by actor
	}
	approvals := []pending{}
	for i, st := range seedTasks {
		lead := users[leadOf[st.project]]
		creator := lead
		if st.creator != "" {
			creator = users[st.creator]
		}
		created := day(st.createdAgo)
		var assignee, agreedBy any
		if st.assignee != "" {
			assignee = users[st.assignee].ID
		}
		if st.agreed {
			agreedBy = lead.ID
		}
		var due any
		if st.due != nil {
			due = now.UTC().AddDate(0, 0, *st.due).Format("2006-01-02") // the fortress counts overdue days in UTC
		} else if st.state == "done" {
			due = day(st.approvedAgo - 1).Format("2006-01-02")
			if i%3 == 0 {
				due = day(st.approvedAgo + 2).Format("2006-01-02") // some were late
			}
		}
		state := st.state
		if state == "done" {
			state = "review"
		}
		var id string
		err := tx.QueryRow(ctx, `WITH p AS (UPDATE projects SET next_number=next_number+1 WHERE id=$1 RETURNING key,next_number-1 n)
 INSERT INTO tasks(number,project_id,title,description,state,assignee_id,priority,points,points_approved_by,points_approved_at,due_date,rank,created_by,created_at,updated_at)
 SELECT p.key||'-'||p.n,$1,$2,$3,$4,$5,$6,$7,$8::uuid,CASE WHEN $8::uuid IS NULL THEN NULL ELSE $11::timestamptz END,$9::date,$10,$12,$11,$11 FROM p RETURNING id::text`,
			projects[st.project], st.title, st.desc, state, assignee, st.priority, st.points, agreedBy, due, int64(i+1)*1024, created, creator.ID).Scan(&id)
		if err != nil {
			return fmt.Errorf("seed task %q: %w", st.title, err)
		}
		note := "Сложность " + itoa(st.points) + " согласована при создании"
		if !st.agreed {
			note = "Сложность " + itoa(st.points) + " — ждёт согласования руководителем"
		}
		if err = history(ctx, tx, id, creator.ID, "created", nil, sp(state), note, created); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO scope_events(team_id,project_id,task_id,kind,delta,created_at) SELECT p.team_id,p.id,$2,'added',$3,$4 FROM projects p WHERE p.id=$1`,
			projects[st.project], id, st.points, created); err != nil {
			return err
		}
		if st.state == "done" {
			by := lead
			if st.assignee == leadOf[st.project] {
				by = users["admin"] // a lead's own task is confirmed by someone else
			}
			approvals = append(approvals, pending{id, day(st.approvedAgo), by})
		}
	}
	sort.Slice(approvals, func(i, j int) bool { return approvals[i].at.Before(approvals[j].at) })
	for _, p := range approvals {
		t, err := loadTask(ctx, tx, p.by, p.id, true)
		if err != nil {
			return err
		}
		if _, err = approveTask(ctx, tx, p.by, t, p.at); err != nil {
			return fmt.Errorf("seed approve %s: %w", t.Number, err)
		}
	}
	// The feed shows historical events at the time they happened.
	if _, err := tx.Exec(ctx, `UPDATE audit_events e SET created_at=l.created_at FROM ledger l
 WHERE e.action='task.approved' AND l.task_id::text=e.target_id AND l.kind='task_reward'`); err != nil {
		return err
	}
	// The current wave of each team, with the enemies on the board as its plan.
	for _, id := range teams {
		if _, err := ensureWave(ctx, tx, id, now); err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `INSERT INTO audit_events(action,target_type,summary) VALUES('seed','system','Созданы демонстрационные данные')`)
	return err
}

func ptrTime(t time.Time) *time.Time { return &t }
