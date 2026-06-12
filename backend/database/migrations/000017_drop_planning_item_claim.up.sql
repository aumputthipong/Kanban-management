-- Drop the planning-item "claim" feature. The soft "I'm looking at this" flag
-- (added in 000013) was over-engineered for the solo / small-team persona —
-- removed from the UI, the endpoints, and now the columns. The claim data is
-- ephemeral working state with no lasting value, so the drop is non-destructive
-- in any meaningful sense.
--
-- Historical `activities` rows for planning.item_claimed / item_released /
-- claim_auto_released_on_promote are left intact (the audit log is immutable);
-- the frontend still renders those event types.
ALTER TABLE planning_items
    DROP COLUMN IF EXISTS claimed_at,
    DROP COLUMN IF EXISTS claimed_by_user_id;
