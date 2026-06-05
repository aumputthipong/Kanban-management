-- Board appearance: a per-board icon, accent colour, and short description.
-- These drive the project-list cards and the sidebar identity. All three are
-- NOT NULL with sensible defaults so list queries + cards always have a value
-- (description falls back to a placeholder line in the UI, not NULL).
ALTER TABLE boards
    ADD COLUMN IF NOT EXISTS description TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS color       VARCHAR(20) NOT NULL DEFAULT '#1E40AF',
    ADD COLUMN IF NOT EXISTS icon        VARCHAR(20) NOT NULL DEFAULT 'board';
