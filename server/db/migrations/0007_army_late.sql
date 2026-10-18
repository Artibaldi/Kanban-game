-- Army and defence, part two: stable and rider, crossbow tower. Town hall 6 and 7 need no schema change.
ALTER TABLE buildings DROP CONSTRAINT buildings_type_check;
ALTER TABLE buildings ADD CONSTRAINT buildings_type_check CHECK (type IN ('townhall','barracks','storage','wall','tower','tavern','hut','market','well','archery','workshop','mortar','trap','stable','crossbow','road','tree','flowers','lantern','pond'));
ALTER TABLE troops DROP CONSTRAINT troops_type_check;
ALTER TABLE troops ADD CONSTRAINT troops_type_check CHECK (type IN ('guard','raider','archer','rider','ram'));
ALTER TABLE training_queue DROP CONSTRAINT training_queue_type_check;
ALTER TABLE training_queue ADD CONSTRAINT training_queue_type_check CHECK (type IN ('guard','raider','archer','rider','ram'));
ALTER TABLE troop_levels DROP CONSTRAINT troop_levels_type_check;
ALTER TABLE troop_levels ADD CONSTRAINT troop_levels_type_check CHECK (type IN ('guard','raider','archer','rider','ram'));
