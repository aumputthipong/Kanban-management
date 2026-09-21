DROP INDEX IF EXISTS idx_users_demo_expiry;

ALTER TABLE users
    DROP COLUMN IF EXISTS is_demo,
    DROP COLUMN IF EXISTS demo_expires_at;
