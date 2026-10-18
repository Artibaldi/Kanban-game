# Security

## Status

Task Bastion is an **MVP for local use**. It is not meant to be exposed to the internet as is:

- **Login has no password.** Anyone who can open the app can sign in as any demo user.
- There is no SSO, MFA or rate limiting.
- Demo credentials (`kanban` / `kanban-local-only`) are in `compose.local.yml` and bind to `127.0.0.1` only.

What *is* in place: sessions in an HttpOnly cookie (only the token hash is stored), CSRF token and Origin
check, server-side role checks, team isolation for tasks, fortress and colleagues, idempotent writes.

## Supported versions

Only the latest commit on `main` is supported.

## Reporting a vulnerability

Please **do not open a public issue.** Report it privately through
[GitHub Security Advisories](https://github.com/Artibaldi/Kanban-game/security/advisories/new).

Include what is affected, how to reproduce it (a demo user and steps are ideal) and the impact you expect.
Issues that follow only from the documented demo login above are known
and do not need a report.
