-- Kanban game: work (teams, projects, tasks) and game (players, city, army, battles) in one schema.
-- Every change of resources is a ledger row; balances on players are the running sum kept in the same transaction.

CREATE TABLE teams (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 name text NOT NULL UNIQUE,
 created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 login text NOT NULL UNIQUE,
 display_name text NOT NULL,
 role text NOT NULL CHECK (role IN ('user','lead','admin')),
 team_id uuid REFERENCES teams(id),
 title text NOT NULL DEFAULT '',
 game_mode boolean NOT NULL DEFAULT true,
 active boolean NOT NULL DEFAULT true,
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK (role = 'admin' OR team_id IS NOT NULL)
);

-- Demo sessions: a random token in an HttpOnly cookie, only its SHA-256 digest is stored.
CREATE TABLE sessions (
 token_digest bytea PRIMARY KEY,
 user_id uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 expires_at timestamptz NOT NULL,
 revoked_at timestamptz
);

CREATE TABLE projects (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 team_id uuid NOT NULL REFERENCES teams(id),
 key text NOT NULL UNIQUE CHECK (key ~ '^[A-Z]{2,6}$'),
 name text NOT NULL,
 next_number integer NOT NULL DEFAULT 1,
 created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tasks (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 number text NOT NULL UNIQUE,
 project_id uuid NOT NULL REFERENCES projects(id),
 title text NOT NULL,
 description text NOT NULL DEFAULT '',
 state text NOT NULL DEFAULT 'backlog' CHECK (state IN ('backlog','ready','in_progress','review','done')),
 assignee_id uuid REFERENCES users(id),
 priority text NOT NULL DEFAULT 'P3' CHECK (priority IN ('P1','P2','P3','P4')),
 points integer NOT NULL CHECK (points IN (1,2,3,5,8)),
 points_approved_by uuid REFERENCES users(id),
 points_approved_at timestamptz,
 due_date date,
 rank bigint NOT NULL,
 result text,
 approved_by uuid REFERENCES users(id),
 approved_at timestamptz,
 version bigint NOT NULL DEFAULT 1,
 created_by uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK ((state = 'done') = (approved_at IS NOT NULL))
);
CREATE INDEX tasks_project_idx ON tasks(project_id, state, rank);
CREATE INDEX tasks_assignee_idx ON tasks(assignee_id, state);

CREATE TABLE task_comments (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 task_id uuid NOT NULL REFERENCES tasks(id),
 author_id uuid NOT NULL REFERENCES users(id),
 body text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX task_comments_task_idx ON task_comments(task_id, created_at);

-- Human-readable history of a task (card timeline).
CREATE TABLE task_history (
 id bigserial PRIMARY KEY,
 task_id uuid NOT NULL REFERENCES tasks(id),
 actor_id uuid REFERENCES users(id),
 kind text NOT NULL,
 from_value text,
 to_value text,
 note text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX task_history_task_idx ON task_history(task_id, id);

-- Game state of a user. Balances may go negative only through a reward reversal (debt).
CREATE TABLE players (
 user_id uuid PRIMARY KEY REFERENCES users(id),
 gold bigint NOT NULL DEFAULT 0,
 energy bigint NOT NULL DEFAULT 0,
 xp bigint NOT NULL DEFAULT 0,
 trophies integer NOT NULL DEFAULT 0 CHECK (trophies >= 0),
 shield_until timestamptz,
 updated_at timestamptz NOT NULL DEFAULT now()
);

-- Journal of every resource operation. Reward rows link to the task; a reversal links to the row it cancels.
CREATE TABLE ledger (
 id bigserial PRIMARY KEY,
 user_id uuid NOT NULL REFERENCES users(id),
 kind text NOT NULL CHECK (kind IN ('seed','task_reward','task_reward_reversal','build','upgrade','train','attack','battle_reward','admin')),
 gold bigint NOT NULL DEFAULT 0,
 energy bigint NOT NULL DEFAULT 0,
 xp bigint NOT NULL DEFAULT 0,
 -- What the formula produced before the storage cap; shown so a cut-off reward is explainable.
 gold_formula bigint,
 energy_formula bigint,
 task_id uuid REFERENCES tasks(id),
 battle_id uuid,
 building_id uuid,
 reverses_id bigint UNIQUE REFERENCES ledger(id),
 reversed boolean NOT NULL DEFAULT false,
 actor_id uuid REFERENCES users(id),
 note text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK (kind NOT IN ('task_reward','task_reward_reversal') OR task_id IS NOT NULL),
 CHECK ((kind = 'task_reward_reversal') = (reverses_id IS NOT NULL))
);
-- At most one live reward per task: a second approval without a reversal fails at the database.
CREATE UNIQUE INDEX ledger_one_live_reward ON ledger(task_id) WHERE kind = 'task_reward' AND NOT reversed;
CREATE INDEX ledger_user_idx ON ledger(user_id, id DESC);

-- Work rating source: + points on approval, − points on reversal. Battles never write here.
CREATE TABLE work_points_events (
 id bigserial PRIMARY KEY,
 user_id uuid NOT NULL REFERENCES users(id),
 team_id uuid NOT NULL REFERENCES teams(id),
 project_id uuid NOT NULL REFERENCES projects(id),
 task_id uuid NOT NULL REFERENCES tasks(id),
 points integer NOT NULL,
 ledger_id bigint REFERENCES ledger(id),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX work_points_team_idx ON work_points_events(team_id, created_at);

-- Backlog scope in points: added / re-estimated / completed / reopened, so scope change is visible apart from burn.
CREATE TABLE scope_events (
 id bigserial PRIMARY KEY,
 team_id uuid NOT NULL REFERENCES teams(id),
 project_id uuid NOT NULL REFERENCES projects(id),
 task_id uuid NOT NULL REFERENCES tasks(id),
 kind text NOT NULL CHECK (kind IN ('added','reestimated','completed','reopened')),
 delta integer NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX scope_events_project_idx ON scope_events(project_id, created_at);

CREATE TABLE buildings (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 user_id uuid NOT NULL REFERENCES users(id),
 type text NOT NULL CHECK (type IN ('townhall','barracks','storage','wall','tower')),
 level integer NOT NULL CHECK (level >= 0),
 target_level integer,
 busy_until timestamptz,
 x integer NOT NULL,
 y integer NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE (user_id, x, y),
 CHECK ((target_level IS NULL) = (busy_until IS NULL))
);
CREATE INDEX buildings_busy_idx ON buildings(busy_until) WHERE busy_until IS NOT NULL;

CREATE TABLE troops (
 user_id uuid NOT NULL REFERENCES users(id),
 type text NOT NULL CHECK (type IN ('guard','raider','ram')),
 count integer NOT NULL DEFAULT 0 CHECK (count >= 0),
 PRIMARY KEY (user_id, type)
);

CREATE TABLE training_queue (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 user_id uuid NOT NULL REFERENCES users(id),
 type text NOT NULL CHECK (type IN ('guard','raider','ram')),
 count integer NOT NULL CHECK (count > 0),
 started_at timestamptz NOT NULL,
 finish_at timestamptz NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX training_queue_finish_idx ON training_queue(finish_at);

CREATE TABLE battles (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 attacker_id uuid NOT NULL REFERENCES users(id),
 defender_id uuid NOT NULL REFERENCES users(id),
 seed bigint NOT NULL,
 army jsonb NOT NULL,
 losses jsonb NOT NULL,
 defense jsonb NOT NULL,
 rounds jsonb NOT NULL,
 destruction integer NOT NULL,
 stars integer NOT NULL,
 gold_reward bigint NOT NULL,
 xp_reward bigint NOT NULL,
 attacker_trophies integer NOT NULL,
 defender_trophies integer NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX battles_pair_idx ON battles(attacker_id, defender_id, created_at DESC);
CREATE INDEX battles_defender_idx ON battles(defender_id, created_at DESC);

-- Economy and battle coefficients, editable by an administrator.
CREATE TABLE settings (
 key text PRIMARY KEY,
 value jsonb NOT NULL,
 updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE idempotency_records (
 actor_id uuid NOT NULL,
 route text NOT NULL,
 key text NOT NULL,
 body_digest bytea NOT NULL,
 status integer NOT NULL,
 response jsonb NOT NULL,
 expires_at timestamptz NOT NULL,
 PRIMARY KEY (actor_id, route, key)
);

-- Operations journal for the feed and for audit: work and game events.
CREATE TABLE audit_events (
 id bigserial PRIMARY KEY,
 actor_id uuid REFERENCES users(id),
 team_id uuid REFERENCES teams(id),
 action text NOT NULL,
 target_type text NOT NULL,
 target_id text,
 summary text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_events_team_idx ON audit_events(team_id, id DESC);
