-- My Work looks up a user's open cards by assignee; without this it scans every card.
-- Partial on is_done = FALSE because done cards never appear in the inbox.
CREATE INDEX IF NOT EXISTS idx_cards_assignee_open ON cards (assignee_id) WHERE is_done = FALSE;
