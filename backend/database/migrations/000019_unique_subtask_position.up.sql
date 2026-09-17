-- Subtask positions were client-computed as count+1, which repeats a position after a
-- middle subtask is deleted; tied rows then swap on every update. Renumber each card
-- to 1..n in its current display order, then make a repeat impossible.
WITH ordered AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY card_id ORDER BY position, created_at, id) AS rn
    FROM card_subtasks
)
UPDATE card_subtasks s
SET position = o.rn
FROM ordered o
WHERE s.id = o.id;

ALTER TABLE card_subtasks
    ADD CONSTRAINT card_subtasks_card_id_position_key UNIQUE (card_id, position);
