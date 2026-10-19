<div align="center">

<img src="docs/assets/hero.svg" alt="Task Bastion: a kanban board for real work and a city the whole team defends" width="100%">

<br>

[![CI](https://github.com/Artibaldi/Kanban-game/actions/workflows/ci.yml/badge.svg)](https://github.com/Artibaldi/Kanban-game/actions/workflows/ci.yml)
[![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue?style=flat-square)](LICENSE)
[![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white)](server/go.mod)
[![Vue 3.5](https://img.shields.io/badge/Vue-3.5-42b883?style=flat-square&logo=vuedotjs&logoColor=white)](web/package.json)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.8-3178C6?style=flat-square&logo=typescript&logoColor=white)](web/tsconfig.json)
[![PostgreSQL 18](https://img.shields.io/badge/PostgreSQL-18-4169E1?style=flat-square&logo=postgresql&logoColor=white)](compose.local.yml)
[![Status: MVP](https://img.shields.io/badge/status-MVP-d4a23a?style=flat-square)](#status-and-honest-limits)

**English** · [Русский](README.ru.md) · [Docs](docs/README.md) · [Roadmap](ROADMAP.md) · [Changelog](CHANGELOG.md)

[**See it**](#see-it) ·
[**Quick start**](#quick-start) ·
[**Rules**](#rules) ·
[**How it works**](#how-it-works) ·
[**Honest limits**](#status-and-honest-limits)

</div>

<br>

**Task Bastion** (*Бастион задач*) is a kanban board for a team's real work with a city-building strategy
game on top. Finish tasks, earn resources, grow your city — and defend the team's fortress together against
the one enemy everybody shares: **the backlog**.

The board stays a work tool. One switch, **Work / Game**, removes the game entirely: no city, no resources, no
animations, no reward pop-ups.

> [!NOTE]
> The interface is in Russian. This README is the English entry point; [README.ru.md](README.ru.md) is the
> full Russian version.

## Why

Gamified work tools usually end up ranking people. Task Bastion is built the other way round:

| When you add a game to a team's work… | …Task Bastion's answer |
|---|---|
| 🏆 **Will it turn into a leaderboard of employees?** | No. There is no table comparing people. Everyone sees their own progress and the team's shared progress. Game data is never used to evaluate work. |
| 🤝 **Who is the opponent?** | The backlog. Planned tasks are enemies, a confirmed task is a defeated enemy, overdue ones hit the walls. The whole team wins or loses a weekly **wave** together. |
| ⚔️ **Can colleagues fight each other?** | Only in **friendly** battles, only by consent: no cost, no losses, no trophies. |
| 💰 **Can someone farm rewards?** | A reward is paid exactly once, only for *In review → Done*, and only after the team lead confirms. You cannot confirm your own task. Every write is idempotent. |
| 🙈 **What if I just want the board?** | Switch to **Work** mode — the game disappears, the kanban stays. |

## See it

| Board | Task card |
|---|---|
| <img src="docs/screenshots/board.png" alt="Kanban board with five columns"> | <img src="docs/screenshots/task.png" alt="Task card with reward, due date and agreed estimate"> |
| **Lead review** | **Barracks** |
| <img src="docs/screenshots/review.png" alt="Lead's review queue: tasks to confirm and estimates to agree"> | <img src="docs/screenshots/army.png" alt="The army camp in 3D and the troop cards"> |
| **Fortress** | **Friendly battle** |
| <img src="docs/screenshots/fortress.png" alt="The 3D fortress: a horde made of the team's tasks marches down the priority lanes toward the walls"> | <img src="docs/screenshots/battle.png" alt="Friendly battle replay: the squad breaks the walls and towers of a colleague's city"> |
| **City** | **Work mode, no game** |
| <img src="docs/screenshots/city.png" alt="Player's 3D city: a walled town hall, towers, storages and barracks"> | <img src="docs/screenshots/work-mode.png" alt="Home page in Work mode: only tasks and events"> |

<details>
<summary>Mobile</summary>

<img src="docs/screenshots/mobile-board.png" alt="Board on a phone" width="300"> <img src="docs/screenshots/mobile-city.png" alt="City on a phone in the dark theme" width="300">

</details>

## What you get

<table>
<tr>
<td width="33%" valign="top">

#### 📋 A real kanban board

Five columns, drag and drop, filters, task cards with history. Estimates (1, 2, 3, 5, 8) are agreed with the
lead *before* work starts; the lead confirms completion.

</td>
<td width="33%" valign="top">

#### 🧾 An honest reward ledger

Gold, energy and XP per confirmed task, by a public formula. Paid exactly once, reversed by a separate
entry, every change in the resource journal.

</td>
<td width="33%" valign="top">

#### 🏙️ Your own city

A 3D 16×16 island: town hall up to level 7, storage, market, builder's huts, walls, towers, mortars,
traps. Construction runs on server timers.

</td>
</tr>
<tr>
<td valign="top">

#### 🏰 Team fortress and waves

The backlog marches on the fortress by priority rows. Each week a **wave** is scored in crowns and the whole
team gets the same reward.

</td>
<td valign="top">

#### ⚔️ Friendly battles

Challenge a colleague or the whole team. Deterministic server-side battle, replayed in 3D round by round.
Five troop types, a workshop and a team banner with reinforcements.

</td>
<td valign="top">

#### 🧘 Work / Game switch

One toggle hides the whole game layer. Game resources never affect work points or wave results.

</td>
</tr>
</table>

## Quick start

You need **Docker**, **Go 1.26+** and **Node.js 22+**.

```bash
git clone https://github.com/Artibaldi/Kanban-game.git
cd Kanban-game

docker compose -f compose.local.yml up -d --wait   # PostgreSQL 18 on 127.0.0.1:15433
cd web && npm install && npm run build && cd ..    # builds the client into web/dist
cd server && go run ./cmd/server                   # app: http://127.0.0.1:8090
```

On first start the server applies migrations and seeds demo data.

| Script | What it does |
|---|---|
| `scripts/dev.sh` | Development: API on :8090 and Vite on http://localhost:5173 with hot reload |
| `scripts/reset-db.sh` | Wipes all data and recreates the demo database |
| `scripts/test.sh` | Every check: Go tests, end-to-end API test, client tests, type check, build |

<details>
<summary>Server environment variables</summary>

| Variable | Default |
|---|---|
| `DATABASE_URL` | `postgres://kanban:kanban-local-only@127.0.0.1:15433/kanban?sslmode=disable` |
| `ADDR` | `127.0.0.1:8090` |
| `ORIGINS` | `http://localhost:5173,http://127.0.0.1:5173,http://localhost:8090,http://127.0.0.1:8090` |
| `STATIC_DIR` | `../web/dist` |

</details>

> [!WARNING]
> **Demo login has no password:** you just pick a user from the list. This is fine for a local prototype only.
> There are no passwords, SSO or MFA yet. See [SECURITY.md](SECURITY.md).

### Demo users

| Login | Role | Team | What to look at |
|---|---|---|---|
| `alexey` | User | Platform | Town hall 3, army ready — good for the city, fortress and friendly battles |
| `maria` | User | Platform | Town hall 2, a tower under construction |
| `dmitry` | User | Platform | Town hall 1, proposed an estimate waiting for approval |
| `irina` | Lead | Platform | Reviews tasks; her own task is in review and she cannot confirm it herself |
| `sergey` | Lead | Mobile | Reviews the second team |
| `anna`, `pavel` | Users | Mobile | The second team — its tasks and colleagues are isolated from the first |
| `admin` | Admin | — | Users and economy coefficients; has no city |

### The full loop in five minutes

1. Log in as **alexey** → **Board** → on the “Валидация ИНН” card press “In progress”, then “In review”.
2. Log out, log in as **irina** → **Review** → “Confirm”. A reward notification appears.
3. Back to **alexey**: resources in the header went up, **Progress** shows points, and in the **Fortress**
   enemy PLT-12 has fallen.
4. **City** → “Build” → “Wall” → pick a free tile. Construction runs on a server timer and survives a reload.
5. **Fortress**: the task horde by priority rows, overdue PLT-10 at the wall, past waves with crowns.
6. **Barracks** → train troops → **Fortress** → “Friendly battles” → pick a colleague and a squad → fight.

## Rules

<details>
<summary><b>Tasks</b></summary>

- Columns: **Backlog → To do → In progress → In review → Done**.
- Estimates use **1, 2, 3, 5, 8** and are **agreed with the lead before work starts**. An unapproved task
  cannot be taken into work.
- The team lead confirms completion. **You cannot confirm your own task.**
- After confirmation, estimate, priority, due date and assignee are frozen.
- Only the lead can reopen a confirmed task, and only with a reason.

</details>

<details>
<summary><b>Rewards</b></summary>

```
gold   = 40 × points × priority multiplier × 1.1 (if confirmed on time)
energy = 10 × points
xp     = 25 × points
```

Priority multipliers: P1 — 1.2, P2 — 1.1, P3 — 1.0, P4 — 0.9.

| Task | Gold | Energy | XP |
|---|---|---|---|
| 5 points, P2, on time | 242 | 50 | 125 |
| 5 points, P2, late | 220 | 50 | 125 |
| 8 points, P1, on time | 422 | 80 | 200 |
| 1 point, P4, no due date | 36 | 10 | 25 |

All coefficients are set by the admin.

- A reward is granted only on *In review → Done*. A unique index prevents a second active reward per task.
- Every mutating request carries an `Idempotency-Key`; a retry returns the stored response and changes nothing.
- Reopening a done task does not delete the reward: it writes a separate reversal for exactly the amount paid.
- **If the resources were already spent**, the balance goes negative. Buildings and troops stay, but you
  cannot build, train or fight until new rewards pay off the debt.
- Income above storage capacity is lost; the journal shows both the formula amount and what fit.

</details>

<details>
<summary><b>Player level, city and troops</b></summary>

- **Player level** grows with XP: level n+1 needs `100 × n^1.5`. Levels 2, 3, 5, 7, 9 and 11 unlock town hall
  2–7, so the city grows only with work.
- **Buildings:**
  - **City:** town hall (city level, up to 7), barracks (army housing), storage and crystal well (gold and
    energy capacity), builder's hut (+1 builder), market (gold ↔ energy with a fee), tavern (challenges and
    daily battles).
  - **Army:** archery range (archers), stable (riders), workshop (troop levels: +10 % HP and attack per level).
  - **Defense:** walls, towers, mortars (area damage), crossbow towers (target heavy units), traps (fire once,
    hidden from the opponent), team banner (a garrison of colleagues' troops).
  - How many of each you can build depends on the town hall level. Two builders to start. Timers are demo-length.
- **Troops:**

  | Unit | Role | HP | Attack | vs. defenses | Housing |
  |---|---|---|---|---|---|
  | Guard | Balanced | 100 | 20 | ×0.8 | 1 |
  | Raider | Strong attack, weak defense | 45 | 42 | ×0.35 | 1 |
  | Archer | Shoots over walls, fragile | 35 | 26 | ×0.5 | 1 |
  | Rider | Rides around walls, hits towers first | 110 | 30 | ×1 | 2 |
  | Ram | Breaks walls and towers | 150 | 10 | ×6 | 2 |

</details>

<details>
<summary><b>Fortress and waves</b></summary>

- **Enemies** are tasks in *To do*, *In progress* and *In review* with an approved estimate. Size: 1–2 points —
  goblin, 3 — knight, 5 — giant, 8 — dragon. Row = priority: P1 is closest to the walls. Backlog tasks are fog.
- **Damage.** On each working day every overdue enemy hits the fortress once: 2 % durability per point. It
  never drops below 10 %. No names are shown next to enemies.
- **A wave** is a calendar week (Mon–Sun, UTC); the admin can make it up to 4 weeks long. The wave plan is
  fixed: a task that became an enemy stays in the plan even if it is moved back to the backlog.
- **Crowns:** 1 — ≥ 50 % of the plan closed; 2 — ≥ 80 % and no overdue P1; 3 — the whole plan and no overdue tasks.
- **Shared reward** for every team member, including the lead, people on leave and those in Work mode:
  60 / 140 / 250 gold and 15 / 35 / 60 energy for 1 / 2 / 3 crowns, plus a defense bonus of up to 20 % from the
  team's average city and army strength. Fortress repair: +10 % / +25 % / full.
- A background worker scores waves; a repeated or concurrent scoring never doubles anything.

</details>

<details>
<summary><b>Friendly battles and the team banner</b></summary>

- Only within your team and only with colleagues who turned on “Accept friendly battles”.
- Free: no energy spent, no troops lost, no trophies. Reward: 5 XP, up to 3 battles a day; the tavern adds more.
  A battle can start from a challenge — direct or open to the whole team; a challenge expires after a day.
- Asynchronous, up to 6 rounds. Each round towers fire first, then troops hit walls, towers, the town hall and the
  rest. Randomness is capped at ±10 % per hit; the battle seed is stored.
- **Team banner.** Colleagues send troops to your banner. The garrison takes hits after the walls, strikes back
  and never dies; you can release it at any time.

</details>

<details>
<summary><b>Progress</b></summary>

- **My progress:** my points per week over 8 weeks, best week, total. No comparison with colleagues.
- **Team:** team points per week, overdue tasks now, wave results.
- **Backlog volume:** a 30-day chart. Added tasks, re-estimates, reopenings and completions are shown
  separately, so you can see whether the work grows or the team burns it down.

</details>

## How it works

```
server/                  Go 1.26 · chi · pgx/v5 · PostgreSQL 18
  cmd/server/            startup: migrations, demo data, timers, HTTP
  db/migrations/         SQL migrations, embedded into the binary
  internal/game/         pure rules: economy, buildings, troops, battle, waves (unit tests)
  internal/app/          HTTP API, transactions, journals, end-to-end test
web/                     Vue 3 · TypeScript · Vite · Pinia · vue-router · three.js
  src/pages/             screens
  src/components/        board, task card, 3D city and fortress, troops, charts
  src/ui/                design tokens, light and dark themes, icons
scripts/                 dev, reset-db, test
```

- **All rules live on the server.** Rewards, construction, training, waves, fortress damage and battles are
  computed server-side; the client only shows the result.
- **One transaction per action.** Debit, credit, journal entry and state change commit together.
- **Journals:** resource journal (every reward linked to its task), task history, team event feed, progress
  events, backlog volume events.
- **Server timers.** A background worker finishes construction and training every second, and every 30 seconds
  opens new waves, closes finished ones and applies overdue damage. Each request first catches up the user's timers.
- **Prototype security:** sessions in an HttpOnly cookie (only the token hash is stored), CSRF token and Origin
  check, roles checked on the server, users see only their own team's tasks, fortress and colleagues.

| Screen | Path | What's there |
|---|---|---|
| Home | `/` | My tasks, weekly points, mini city, fortress durability, event feed |
| Board | `/board` | Five columns, drag and drop, filters, new task, task card |
| Review | `/review` | Lead only: confirm, return, approve estimates |
| Fortress | `/fortress` | Team fortress, task horde, current wave, wave history |
| City | `/city` | 3D 16×16 island: build, upgrade and move buildings, decorations |
| Barracks | `/army` | Training, workshop, reinforcements to colleagues' banners |
| Friendly battles | `/fortress/duels` | Challenges, opponent's city preview, squad, history |
| Battle | `/battles/:id` | 3D replay by server rounds, log and result |
| Progress | `/progress` | My progress, team progress, waves, backlog volume chart |
| Settings | `/settings` | Work / Game mode, friendly battles, theme, resource journal |
| Admin | `/admin` | Roles, teams, economy coefficients, working days and wave length |

## Testing

```bash
scripts/test.sh
```

- **`internal/game`** — reward and level formulas, build and upgrade rules, battle determinism, crowns, fortress
  damage and repair, wave windows, market, banner, challenges.
- **End-to-end API test** on a throwaway database that is created and dropped: a reward is paid exactly once,
  a retried request returns the stored response, self-confirmation and cross-team access are refused,
  construction finishes on its timer, friendly battles are free and limited, wave plans are fixed and
  scored once, a reversal of a spent reward leads to debt. The suite also runs under `TZ=America/New_York`
  to prove the API does not depend on the server's timezone.
- **Client** — board, city, army, fortress and battle-view rules, `vue-tsc`, production build.

CI runs the same checks on every push and pull request ([`ci.yml`](.github/workflows/ci.yml)).

## Status and honest limits

Task Bastion is an **MVP**. What it does not do yet:

- Login is demo-only, without passwords.
- Data refreshes on actions and every 15 seconds; there are no live notifications.
- Battles are computed from aggregate strength of defenses and buildings; troops do not path across the map.
- The interface is Russian only.

What's next is in the [roadmap](ROADMAP.md).

## Contributing, security, license

- How to run, test and send a change: [CONTRIBUTING.md](CONTRIBUTING.md).
- Security issues go through [SECURITY.md](SECURITY.md), not public issues.
- Licensed under [Apache-2.0](LICENSE). © 2026 Arti.
