-- Co-op against the backlog: one fortress per team, weekly waves with a frozen plan, fortress damage and repair.
-- Raids between colleagues give way to optional friendly duels; raid trophies and shields are reset.

ALTER TABLE teams
 ADD COLUMN work_days smallint[] NOT NULL DEFAULT '{1,2,3,4,5}',
 ADD COLUMN wave_weeks smallint NOT NULL DEFAULT 1 CHECK (wave_weeks BETWEEN 1 AND 4);

ALTER TABLE users
 ADD COLUMN accept_friendly boolean NOT NULL DEFAULT false,
 ADD COLUMN game_rules_ack_at timestamptz;

CREATE TABLE fortresses (
 team_id uuid PRIMARY KEY REFERENCES teams(id),
 hp integer NOT NULL DEFAULT 100 CHECK (hp BETWEEN 0 AND 100),
 level integer NOT NULL DEFAULT 1,
 updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO fortresses(team_id) SELECT id FROM teams ON CONFLICT DO NOTHING;

-- A wave is one period of a team; it is settled once, after ends_at.
CREATE TABLE waves (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 team_id uuid NOT NULL REFERENCES teams(id),
 starts_at timestamptz NOT NULL,
 ends_at timestamptz NOT NULL,
 planned_points integer,
 closed_points integer,
 overdue_count integer,
 overdue_p1_count integer,
 crowns integer CHECK (crowns BETWEEN 0 AND 3),
 defense_bonus numeric(4,3),
 summary text,
 settled_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE (team_id, starts_at)
);
CREATE INDEX waves_open_idx ON waves(ends_at) WHERE settled_at IS NULL;

-- The frozen plan: a task that became an enemy during the wave stays in it, whatever happens to it later.
CREATE TABLE wave_plan (
 wave_id uuid NOT NULL REFERENCES waves(id),
 task_id uuid NOT NULL REFERENCES tasks(id),
 points integer NOT NULL,
 joined_at timestamptz NOT NULL,
 PRIMARY KEY (wave_id, task_id)
);

-- Every change of the fortress: daily damage per overdue task (once per task and day) and the repair after a wave.
CREATE TABLE fortress_events (
 id bigserial PRIMARY KEY,
 team_id uuid NOT NULL REFERENCES teams(id),
 kind text NOT NULL CHECK (kind IN ('overdue_damage','wave_repair')),
 task_id uuid REFERENCES tasks(id),
 day date,
 wave_id uuid REFERENCES waves(id),
 delta integer NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE (task_id, day)
);
CREATE UNIQUE INDEX fortress_one_repair ON fortress_events(wave_id) WHERE kind = 'wave_repair';

ALTER TABLE ledger ADD COLUMN wave_id uuid REFERENCES waves(id);
ALTER TABLE ledger DROP CONSTRAINT ledger_kind_check;
ALTER TABLE ledger ADD CONSTRAINT ledger_kind_check CHECK (kind IN ('seed','task_reward','task_reward_reversal','build','upgrade','train','attack','battle_reward','admin','wave_reward'));
-- At most one team chest per member and wave.
CREATE UNIQUE INDEX ledger_one_wave_reward ON ledger(user_id, wave_id) WHERE kind = 'wave_reward';

ALTER TABLE battles ADD COLUMN kind text NOT NULL DEFAULT 'raid' CHECK (kind IN ('raid','friendly'));

UPDATE players SET trophies = 0, shield_until = NULL;
