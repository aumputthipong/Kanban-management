-- The renumbering is not reverted: the old duplicate positions carried no meaning, and
-- 1..n in display order is valid for code on either side of this migration.
ALTER TABLE card_subtasks
    DROP CONSTRAINT IF EXISTS card_subtasks_card_id_position_key;
