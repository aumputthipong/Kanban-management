-- Throwaway sandbox accounts behind the "Try demo" button. A visitor who has
-- no interest in registering gets their own disposable user + board clone, so
-- one person's dragging and deleting never degrades the next person's demo.
--
-- demo_expires_at is the purge pivot, not a session length: the access cookie
-- outlives it, and a request from an already-purged user simply 401s.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_demo BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS demo_expires_at TIMESTAMPTZ DEFAULT NULL;

-- Partial: real users are the overwhelming majority and never match the sweep.
CREATE INDEX IF NOT EXISTS idx_users_demo_expiry
    ON users(demo_expires_at)
    WHERE is_demo;
