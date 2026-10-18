-- Challenges to friendly duels and the tavern.
-- A challenge is the consent: a direct one to a colleague (to_id) must be accepted before the attack;
-- an open one (to_id NULL) is the author's own city, posted for the team.
ALTER TABLE buildings DROP CONSTRAINT buildings_type_check;
ALTER TABLE buildings ADD CONSTRAINT buildings_type_check CHECK (type IN ('townhall','barracks','storage','wall','tower','tavern'));

CREATE TABLE friendly_challenges (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 team_id uuid NOT NULL REFERENCES teams(id),
 from_id uuid NOT NULL REFERENCES users(id),
 to_id uuid REFERENCES users(id),
 status text NOT NULL CHECK (status IN ('pending','accepted','declined','cancelled','done','open','closed')),
 created_at timestamptz NOT NULL DEFAULT now(),
 expires_at timestamptz NOT NULL,
 CHECK (to_id IS NULL OR to_id <> from_id)
);
-- At most one live challenge to the same colleague, and one open challenge per author.
CREATE UNIQUE INDEX challenge_one_direct ON friendly_challenges(from_id, to_id) WHERE to_id IS NOT NULL AND status IN ('pending','accepted');
CREATE UNIQUE INDEX challenge_one_open ON friendly_challenges(from_id) WHERE to_id IS NULL AND status = 'open';
CREATE INDEX challenge_team ON friendly_challenges(team_id, status);

ALTER TABLE battles ADD COLUMN challenge_id uuid REFERENCES friendly_challenges(id);
-- One battle per attacker and challenge.
CREATE UNIQUE INDEX battle_one_per_challenge ON battles(challenge_id, attacker_id) WHERE challenge_id IS NOT NULL;
