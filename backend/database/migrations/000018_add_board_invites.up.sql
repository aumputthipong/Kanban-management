-- Shareable invite links. A board can have an active invite link (an opaque
-- random token); anyone who is logged in and opens the link joins the board as
-- a member. Links are multi-use (no per-link seat count) but expire after a
-- fixed window and can be revoked — revoked_at / expires_at gate acceptance.
--
-- Role is always "member" (no role column): manager+ is granted explicitly, not
-- handed out via a shareable URL. board_id ON DELETE CASCADE so deleting a board
-- cleans up its links.
CREATE TABLE IF NOT EXISTS board_invites (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id    UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    token       TEXT NOT NULL UNIQUE,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Lookups: "the active link for this board" (create/show/revoke) hits board_id;
-- "accept this token" hits the unique token index already created above.
CREATE INDEX IF NOT EXISTS idx_board_invites_active
    ON board_invites (board_id)
    WHERE revoked_at IS NULL;
