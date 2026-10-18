# Contributing

Thanks for helping build Task Bastion! This guide covers how to run the project, how to check a change and
what a good pull request looks like.

## Principles

These shape every rule in the game. A change that breaks one of them will not be merged:

1. **No ranking of people.** Game data is never used to evaluate work; there is no table comparing colleagues.
2. **The board works without the game.** Work mode hides the game layer completely.
3. **All rules live on the server.** The client only shows results the server computed.
4. **One transaction per action.** Resources, journals and state change together, and a retry changes nothing.

## Setup

You need **Docker**, **Go 1.26+** and **Node.js 22+**.

```bash
docker compose -f compose.local.yml up -d --wait
scripts/dev.sh          # API on :8090, Vite on http://localhost:5173
```

`scripts/reset-db.sh` recreates the demo database. Demo users are listed in the [README](README.md#demo-users).

## Checks

```bash
scripts/test.sh
```

It runs `go vet`, the Go unit tests, the end-to-end API test on a throwaway database (also under
`TZ=America/New_York`), the client tests, `vue-tsc` and the production build. CI runs the same steps.

- Game rules go in `server/internal/game` as pure functions with unit tests.
- Anything touching money, timers, waves or permissions gets a case in the end-to-end test in
  `server/internal/app`.
- Client rules (what to show, what is allowed) go in plain TypeScript modules with tests in `web/tests`.

## Database changes

Add a new numbered file in `server/db/migrations/` — never edit a migration that is already on `main`.
Migrations are embedded in the binary and run on startup, so test them against existing demo data, not
only an empty database.

## Branches, commits and pull requests

- Branch from `main`: `feature/<name>` for features, `fix/<name>` for fixes, `chore/<name>` for the rest.
- Commit messages are in Russian, start with the area and describe the result:
  `Крепость: этап задачи и признак «моя» у врага`.
- Larger features go in stages: rules on the server → client → a review pass with fixes.
- Fill in the pull request template. Include screenshots for UI changes (light and dark theme, phone).
- If rules or setup change, update both [README.md](README.md) and [README.ru.md](README.ru.md), and add a line
  to [CHANGELOG.md](CHANGELOG.md) under *Unreleased*.

## Code style

- Go: `gofmt` and `go vet` clean; game rules stay free of database and HTTP code.
- Vue / TypeScript: `<script setup>`, design tokens from `web/src/ui` instead of raw colors, strict types.
- Interface copy is Russian; keep it short and concrete.

## Conduct and security

By taking part you agree to the [Code of Conduct](CODE_OF_CONDUCT.md). Security problems go through
[SECURITY.md](SECURITY.md), not public issues.

By contributing you agree that your contributions are licensed under the [Apache-2.0](LICENSE) license.
