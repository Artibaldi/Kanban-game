-- Team banner with colleagues' reinforcements, and decor the team earns in waves.
ALTER TABLE buildings DROP CONSTRAINT buildings_type_check;
ALTER TABLE buildings ADD CONSTRAINT buildings_type_check CHECK (type IN ('townhall','barracks','storage','wall','tower','tavern','hut','market','well','archery','workshop','mortar','trap','stable','crossbow','banner','road','tree','flowers','lantern','pond','flags','arch','fountain','statue'));

-- Troops a colleague lent to someone's banner. They are the donor's, kept away from home: released, or when either
-- of the two leaves the team, they go back to the donor's camp.
CREATE TABLE reinforcements (
 owner_id uuid NOT NULL REFERENCES users(id),
 donor_id uuid NOT NULL REFERENCES users(id),
 type text NOT NULL CHECK (type IN ('guard','raider','archer','rider','ram')),
 count integer NOT NULL CHECK (count > 0),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (owner_id, donor_id, type),
 CHECK (owner_id <> donor_id)
);
CREATE INDEX reinforcements_donor_idx ON reinforcements(donor_id);
