-- City planning: the island grows to 16×16 with the old 12×12 field in its centre, and decor buildings.
ALTER TABLE buildings DROP CONSTRAINT buildings_type_check;
ALTER TABLE buildings ADD CONSTRAINT buildings_type_check CHECK (type IN ('townhall','barracks','storage','wall','tower','tavern','road','tree','flowers','lantern','pond'));
-- Every existing city moves two cells toward the centre of the bigger island; layouts stay as they were.
-- Two steps: the (user, x, y) uniqueness is checked row by row, so a direct +2 would collide mid-update.
UPDATE buildings SET x = x + 100, y = y + 100;
UPDATE buildings SET x = x - 98, y = y - 98;
