-- City economy: builder hut, market and crystal well; the market's trades are their own ledger kind.
ALTER TABLE buildings DROP CONSTRAINT buildings_type_check;
ALTER TABLE buildings ADD CONSTRAINT buildings_type_check CHECK (type IN ('townhall','barracks','storage','wall','tower','tavern','hut','market','well','road','tree','flowers','lantern','pond'));
ALTER TABLE ledger DROP CONSTRAINT ledger_kind_check;
ALTER TABLE ledger ADD CONSTRAINT ledger_kind_check CHECK (kind IN ('seed','task_reward','task_reward_reversal','build','upgrade','train','attack','battle_reward','admin','wave_reward','market'));
