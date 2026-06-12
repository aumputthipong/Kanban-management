-- Revert: re-add the claim columns (mirrors 000013). Claim data is not
-- recoverable — the columns come back empty (NULL = free), which is the
-- correct "no one is holding anything" starting state.
ALTER TABLE planning_items
    ADD COLUMN IF NOT EXISTS claimed_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS claimed_at         TIMESTAMPTZ;
