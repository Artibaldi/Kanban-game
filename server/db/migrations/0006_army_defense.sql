-- Army and defence, part one: archery range, workshop, mortar, traps; the archer; troop levels from the workshop.
ALTER TABLE buildings DROP CONSTRAINT buildings_type_check;
ALTER TABLE buildings ADD CONSTRAINT buildings_type_check CHECK (type IN ('townhall','barracks','storage','wall','tower','tavern','hut','market','well','archery','workshop','mortar','trap','road','tree','flowers','lantern','pond'));
ALTER TABLE troops DROP CONSTRAINT troops_type_check;
ALTER TABLE troops ADD CONSTRAINT troops_type_check CHECK (type IN ('guard','raider','archer','ram'));
ALTER TABLE training_queue DROP CONSTRAINT training_queue_type_check;
ALTER TABLE training_queue ADD CONSTRAINT training_queue_type_check CHECK (type IN ('guard','raider','archer','ram'));
ALTER TABLE ledger DROP CONSTRAINT ledger_kind_check;
ALTER TABLE ledger ADD CONSTRAINT ledger_kind_check CHECK (kind IN ('seed','task_reward','task_reward_reversal','build','upgrade','train','attack','battle_reward','admin','wave_reward','market','research'));

-- A troop type's workshop level; a row exists once it was first raised. One research runs at a time per player.
CREATE TABLE troop_levels (
 user_id uuid NOT NULL REFERENCES users(id),
 type text NOT NULL CHECK (type IN ('guard','raider','archer','ram')),
 level integer NOT NULL DEFAULT 1 CHECK (level BETWEEN 1 AND 4),
 target_level integer CHECK (target_level BETWEEN 2 AND 4),
 started_at timestamptz,
 busy_until timestamptz,
 PRIMARY KEY (user_id, type),
 CHECK ((target_level IS NULL) = (busy_until IS NULL))
);
CREATE UNIQUE INDEX troop_one_research ON troop_levels(user_id) WHERE busy_until IS NOT NULL;

-- The attacker's troop levels in a battle, so a replay shows what fought.
ALTER TABLE battles ADD COLUMN troop_levels jsonb NOT NULL DEFAULT '{}';
