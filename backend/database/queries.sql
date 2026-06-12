-- name: GetAllBoards :many
SELECT id, title, created_at
FROM boards
ORDER BY created_at DESC;

-- name: GetBoardIDByColumn :one
SELECT board_id FROM columns WHERE id = $1;

-- name: GetBoardIDByCard :one
SELECT col.board_id
FROM cards c
JOIN columns col ON col.id = c.column_id
WHERE c.id = $1;

-- name: GetMyTasks :many
-- Cards in the user's "work inbox": assigned to them, plus (when
-- include_unassigned=true) unassigned cards on boards they're a member of.
-- Done cards and stashed boards are excluded.
--
-- Returns a derived `status`: cards in the first TODO column of their board
-- are "todo"; cards in any later TODO column are "in_progress".
--
-- `work_group` buckets the card relative to `today` (passed in by the service
-- in the user's timezone — Asia/Bangkok hardcoded in S.1). Buckets:
--   overdue   — due_date < today
--   today     — due_date = today
--   this_week — due_date within the next 6 days after today
--   later     — due_date further out
--   no_date   — due_date IS NULL
WITH first_todo AS (
    SELECT board_id, MIN(position) AS first_pos
    FROM columns
    WHERE category = 'TODO'
    GROUP BY board_id
)
SELECT
    c.id,
    c.title,
    c.priority,
    c.due_date,
    c.estimated_hours,
    c.is_done,
    col.board_id,
    b.title AS board_name,
    col.title AS column_name,
    (SELECT COUNT(*) FROM card_subtasks cs WHERE cs.card_id = c.id) AS total_subtasks,
    (SELECT COUNT(*) FROM card_subtasks cs WHERE cs.card_id = c.id AND cs.is_done) AS completed_subtasks,
    CASE
        WHEN col.category = 'TODO' AND col.position = ft.first_pos THEN 'todo'
        WHEN col.category = 'TODO' THEN 'in_progress'
        ELSE 'todo'
    END::text AS status,
    CASE
        WHEN c.due_date IS NULL                                        THEN 'no_date'
        WHEN c.due_date <  sqlc.arg(today)::date                       THEN 'overdue'
        WHEN c.due_date =  sqlc.arg(today)::date                       THEN 'today'
        WHEN c.due_date <= (sqlc.arg(today)::date + INTERVAL '6 days') THEN 'this_week'
        ELSE 'later'
    END::text AS work_group
FROM cards c
JOIN columns col ON col.id = c.column_id
JOIN boards  b   ON b.id  = col.board_id
LEFT JOIN first_todo ft ON ft.board_id = col.board_id
WHERE c.is_done = FALSE
  AND b.deleted_at IS NULL
  AND (
        c.assignee_id = sqlc.arg(user_id)::uuid
     OR (
            sqlc.arg(include_unassigned)::boolean = TRUE
        AND c.assignee_id IS NULL
        AND col.board_id IN (
                SELECT board_id FROM board_members WHERE user_id = sqlc.arg(user_id)::uuid
            )
        )
      )
ORDER BY
    CASE WHEN c.due_date IS NULL THEN 1 ELSE 0 END,
    c.due_date ASC,
    c.created_at DESC;

-- name: CompleteCardAsAssignee :execrows
-- Atomic complete + move-to-DONE-column. Only succeeds if the caller is the
-- card's assignee. Returns affected row count so the handler can 404 on miss.
UPDATE cards
SET column_id = $2,
    is_done = TRUE,
    completed_at = NOW()
WHERE id = $1
  AND assignee_id = $3;

-- name: GetColumnsByBoardID :many
SELECT id, board_id, title, position, category, color, created_at, updated_at
FROM columns
WHERE board_id = $1
ORDER BY position ASC;

-- name: GetCardsByColumnIDs :many
SELECT
    c.id,
    c.column_id,
    c.title,
    c.description,
    c.position,
    c.due_date,
    c.estimated_hours,
    c.assignee_id,
    c.priority,
    c.is_done,
    c.completed_at,
    c.created_at,
    c.created_by,
    u.full_name AS assignee_name,
    COUNT(cs.id) AS total_subtasks,
    COUNT(cs.id) FILTER (WHERE cs.is_done) AS completed_subtasks
FROM cards c
LEFT JOIN users u ON c.assignee_id = u.id
LEFT JOIN card_subtasks cs ON cs.card_id = c.id
WHERE c.column_id = ANY($1::uuid[])
GROUP BY c.id, u.full_name
ORDER BY c.position ASC;
-- name: CreateBoard :one
-- Appearance is optional at create time: a NULL narg falls back to the column
-- default (description '', color '#1E40AF', icon 'board') so the board always
-- has a sensible identity even when the client omits these.
INSERT INTO boards (id, title, description, color, icon, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    sqlc.arg('title'),
    COALESCE(sqlc.narg('description'), ''),
    COALESCE(sqlc.narg('color'), '#1E40AF'),
    COALESCE(sqlc.narg('icon'), 'board'),
    CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
RETURNING id, title;

-- name: CreateColumn :one
INSERT INTO columns (id, board_id, title, position, category, color, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
RETURNING id, board_id, title, position, category, color;

-- name: GetMaxColumnPositionInBoard :one
SELECT COALESCE(MAX(position), 0)
FROM columns
WHERE board_id = $1;

-- name: GetMaxColumnPositionBeforeDone :one
SELECT COALESCE(MAX(position), 0)
FROM columns
WHERE board_id = $1 AND category != 'DONE';

-- name: UpdateColumn :exec
UPDATE columns
SET title    = $2,
    category = $3,
    color    = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: RenameColumn :exec
UPDATE columns
SET title = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteColumn :exec
DELETE FROM columns WHERE id = $1;


-- name: GetMaxPositionInColumn :one
SELECT COALESCE(MAX(position), 0)
FROM cards
WHERE column_id = $1;

-- name: CreateCard :one
-- acceptance_criteria and implementation_note are accepted as optional
-- inserts so PromoteItem can carry the planning row's values onto the
-- resulting card in one statement. User-initiated card creation passes
-- nil and the columns stay NULL.
INSERT INTO cards (column_id, title, position, due_date, assignee_id, priority, created_by, acceptance_criteria, implementation_note, description)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, column_id, title, description, position, due_date, assignee_id, priority, created_by, acceptance_criteria, implementation_note;

-- name: UpdateCardColumn :exec
UPDATE cards
SET 
    column_id = $1,
    position = $2,
    is_done = $3,
    completed_at = $4,
    updated_at = NOW()
WHERE id = $5;

-- name: DeleteCard :exec
DELETE FROM cards WHERE id = $1;


-- name: StashBoard :exec
-- "Stash" = recoverable soft-delete. Reuses the generic deleted_at column;
-- a non-null deleted_at means the board is in the stash (คลังบอร์ด).
UPDATE boards
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: GetAllActiveBoards :many
SELECT id, title, created_at
FROM boards
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetActiveBoardsWithStats :many
-- Sort order is "most recently opened by this user". Boards the user has
-- never opened fall back to created_at (a write-once timestamp) instead
-- of updated_at — otherwise another member editing a never-opened board
-- would shuffle this user's list, which is the exact UX the per-user
-- tracking is supposed to prevent. created_at is exposed for the
-- "Newest" / "Oldest" client-side sort options.
SELECT
    b.id,
    b.title,
    b.description,
    b.color,
    b.icon,
    b.created_at,
    b.updated_at,
    me.last_accessed_at,
    COALESCE(COUNT(DISTINCT c.id), 0)::int                                  AS total_cards,
    COALESCE(COUNT(DISTINCT c.id) FILTER (WHERE c.is_done = TRUE), 0)::int  AS done_cards
FROM boards b
JOIN board_members me ON me.board_id = b.id AND me.user_id = $1
LEFT JOIN columns col ON col.board_id = b.id
LEFT JOIN cards   c   ON c.column_id  = col.id
WHERE b.deleted_at IS NULL
GROUP BY b.id, b.title, b.description, b.color, b.icon, b.created_at, b.updated_at, me.last_accessed_at
ORDER BY COALESCE(me.last_accessed_at, b.created_at) DESC;

-- name: GetMembersForActiveBoards :many
SELECT
    bm.board_id::text AS board_id,
    u.id::text        AS user_id,
    u.full_name
FROM board_members bm
JOIN users  u ON u.id  = bm.user_id
JOIN boards b ON b.id  = bm.board_id
WHERE b.deleted_at IS NULL
  AND b.id IN (
    SELECT bm2.board_id FROM board_members bm2 WHERE bm2.user_id = $1
  )
ORDER BY bm.joined_at ASC;

-- name: GetStashedBoardsForOwner :many
-- Appearance (description/color/icon) is returned so the คลังบอร์ด rows render
-- the same glyph + description as the project-list cards.
SELECT b.id, b.title, b.description, b.color, b.icon, b.deleted_at
FROM boards b
JOIN board_members bm ON bm.board_id = b.id AND bm.user_id = $1 AND bm.role = 'owner'
WHERE b.deleted_at IS NOT NULL
ORDER BY b.deleted_at DESC;

-- name: HardDeleteBoard :exec
-- ลบข้อมูลออกจากตารางจริง (ถ้าตั้ง ON DELETE CASCADE ไว้ ลูกๆ จะหายไปด้วย)
DELETE FROM boards 
WHERE id = $1;

-- name: RestoreStashedBoard :exec
UPDATE boards
SET deleted_at = NULL
WHERE id = $1;


-- name: UpdateBoard :one
-- The service reads the existing row first and passes through unchanged
-- values, so every column is set unconditionally here (PUT-like in spirit).
UPDATE boards
SET title = $2,
    budget = $3,
    description = $4,
    color = $5,
    icon = $6,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetBoardByID :one
SELECT * FROM boards 
WHERE id = $1 LIMIT 1;

-- name: UpdateCard :one
-- title/description/due_date/assignee_id/priority/estimated_hours are
-- overwritten (the handler reads the existing row first; this is PUT-like
-- in spirit even though the route is PATCH). acceptance_criteria and
-- implementation_note use COALESCE so a card update that doesn't touch
-- them keeps whatever PromoteItem copied in — without this guard, any
-- edit of title would silently wipe the AC the dev rely on.
UPDATE cards
SET
    title               = $2,
    description         = $3,
    due_date            = $4,
    assignee_id         = $5,
    priority            = $6,
    estimated_hours     = $7,
    acceptance_criteria = COALESCE(sqlc.narg(acceptance_criteria)::text, acceptance_criteria),
    implementation_note = COALESCE(sqlc.narg(implementation_note)::text, implementation_note),
    updated_at          = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;



-- name: CreateUser :one
INSERT INTO users (email, full_name, password_hash, provider, provider_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT id, email, full_name FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserByProviderID :one
SELECT * FROM users 
WHERE provider = $1 AND provider_id = $2 
LIMIT 1;

-- name: UpsertOAuthUser :one
INSERT INTO users (email, full_name, provider, provider_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (email) 
DO UPDATE SET 
    full_name = EXCLUDED.full_name,
    provider_id = EXCLUDED.provider_id
RETURNING *;

-- name: GetAllUsers :many
-- Capped at 500 rows. The assignee-picker dropdown calls this; beyond ~500
-- users it needs a search endpoint, not a full dump. Bump only with a UX plan.
SELECT id, email, full_name FROM users ORDER BY full_name ASC LIMIT 500;



-- name: AddBoardMember :one
INSERT INTO board_members (board_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetBoardMembers :many
SELECT 
    bm.id,
    bm.role,
    bm.joined_at,
    u.id       AS user_id,
    u.email,
    u.full_name
FROM board_members bm
JOIN users u ON bm.user_id = u.id
WHERE bm.board_id = $1
ORDER BY bm.joined_at ASC;

-- name: RemoveBoardMember :exec
DELETE FROM board_members
WHERE board_id = $1 AND user_id = $2;

-- name: UpdateBoardMemberRole :one
UPDATE board_members
SET role = $3
WHERE board_id = $1 AND user_id = $2
RETURNING *;

-- name: GetBoardMemberRole :one
-- Active boards only: a stashed board (deleted_at set) reports no membership so
-- the board-access gate 404s — stashed boards are unreachable via normal routes
-- (reads and mutations alike), consistent with anti-enumeration.
SELECT bm.role
FROM board_members bm
JOIN boards b ON b.id = bm.board_id
WHERE bm.board_id = $1 AND bm.user_id = $2 AND b.deleted_at IS NULL;

-- name: GetStashedBoardMemberRole :one
-- The mirror of GetBoardMemberRole for the /api/stash routes: only matches when
-- the board IS stashed, so restore / permanent-delete operate on stashed boards
-- while normal board routes keep 404ing for them.
SELECT bm.role
FROM board_members bm
JOIN boards b ON b.id = bm.board_id
WHERE bm.board_id = $1 AND bm.user_id = $2 AND b.deleted_at IS NOT NULL;

-- name: TouchBoardMemberIfStale :exec
-- Update the membership's last_accessed_at, but skip writes that happened
-- within the throttle window (5 min). Called from a goroutine after the
-- user opens a board — the throttle keeps the write rate sane when a user
-- refreshes or bounces between sub-pages.
UPDATE board_members
SET last_accessed_at = now()
WHERE board_id = $1
  AND user_id  = $2
  AND (last_accessed_at IS NULL OR last_accessed_at < now() - INTERVAL '5 minutes');


-- name: GetCard :one
SELECT * FROM cards 
WHERE id = $1 LIMIT 1;


-- name: GetSubtasksByCardID :many
SELECT * FROM card_subtasks
WHERE card_id = $1
ORDER BY position ASC;

-- name: GetSubtasksByCardIDs :many
SELECT * FROM card_subtasks
WHERE card_id = ANY($1::uuid[])
ORDER BY card_id, position ASC;

-- name: CreateSubtask :one
INSERT INTO card_subtasks (card_id, title, position)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateSubtask :one
UPDATE card_subtasks
SET 
    title = COALESCE($2, title),
    is_done = COALESCE($3, is_done),
    position = COALESCE($4, position),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteSubtask :exec
DELETE FROM card_subtasks
WHERE id = $1;


-- name: GetColumnCategory :one
SELECT category
FROM columns
WHERE id = $1;

-- name: GetColumnByBoardAndCategory :one
-- ใช้หา DONE column หรือ TODO column ของ board นั้นๆ
-- ORDER BY position เพื่อได้ column แรกสุดของ category นั้น
SELECT id, position
FROM columns
WHERE board_id = $1 AND category = $2
ORDER BY position ASC
LIMIT 1;

-- name: GetSubtask :one
SELECT * FROM card_subtasks WHERE id = $1 LIMIT 1;

-- name: UpdateSubtaskDone :exec
UPDATE card_subtasks
SET
    is_done = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;


-- name: GetTagsByBoardID :many
SELECT * FROM tags WHERE board_id = $1 ORDER BY name ASC;

-- name: GetTagsByCardIDs :many
SELECT ct.card_id, t.id, t.board_id, t.name, t.color, t.created_at
FROM tags t
JOIN card_tags ct ON ct.tag_id = t.id
WHERE ct.card_id = ANY($1::uuid[])
ORDER BY ct.card_id, t.name ASC;

-- name: CreateTag :one
INSERT INTO tags (board_id, name, color)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = $1 AND board_id = $2;

-- name: ClearCardTags :exec
DELETE FROM card_tags WHERE card_id = $1;

-- name: InsertCardTag :exec
INSERT INTO card_tags (card_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;
-- name: CreateActivity :one
INSERT INTO activities (board_id, actor_id, event_type, entity_type, entity_id, payload)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, board_id, actor_id, event_type, entity_type, entity_id, payload, created_at;

-- name: ListActivitiesByBoard :many
SELECT
    a.id,
    a.board_id,
    a.actor_id,
    a.event_type,
    a.entity_type,
    a.entity_id,
    a.payload,
    a.created_at,
    u.full_name AS actor_name
FROM activities a
LEFT JOIN users u ON a.actor_id = u.id
WHERE a.board_id = $1
ORDER BY a.created_at DESC
LIMIT $2;

-- name: ListActivitiesByBoardBefore :many
SELECT
    a.id,
    a.board_id,
    a.actor_id,
    a.event_type,
    a.entity_type,
    a.entity_id,
    a.payload,
    a.created_at,
    u.full_name AS actor_name
FROM activities a
LEFT JOIN users u ON a.actor_id = u.id
WHERE a.board_id = $1 AND a.created_at < $2
ORDER BY a.created_at DESC
LIMIT $3;


-- name: InsertRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent, ip)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: GetRefreshTokenByHash :one
SELECT id, user_id, token_hash, expires_at, revoked_at, replaced_by, created_at
FROM refresh_tokens
WHERE token_hash = $1
LIMIT 1;

-- name: RevokeRefreshToken :exec
-- Single-token revoke. replaced_by is set on rotation to track lineage so
-- replay of an already-rotated token can be detected and the whole family
-- revoked.
UPDATE refresh_tokens
SET revoked_at = now(), replaced_by = $2
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeAllRefreshTokensForUser :exec
-- Called on replay detection (a revoked token presented again) and on logout-
-- all-sessions. Idempotent: already-revoked rows are skipped.
UPDATE refresh_tokens
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;


-- =============================================================
-- Planning sessions & items
-- =============================================================

-- name: ListPlanningSessionsByBoard :many
SELECT
    ps.id, ps.board_id, ps.title, ps.label, ps.meeting_at,
    ps.created_by, ps.created_at, ps.updated_at,
    COUNT(pi.id) FILTER (WHERE pi.type = 'REQ' AND pi.status NOT IN ('dropped','promoted')) AS req_count,
    COUNT(pi.id) FILTER (WHERE pi.type = 'DEC' AND pi.status NOT IN ('dropped','promoted')) AS dec_count,
    COUNT(pi.id) FILTER (WHERE pi.type = 'Q'   AND pi.status NOT IN ('dropped','promoted')) AS q_count,
    COUNT(pi.id) FILTER (WHERE pi.status = 'promoted') AS promoted_count,
    COUNT(pi.id) FILTER (WHERE pi.status = 'dropped') AS dropped_count
FROM planning_sessions ps
LEFT JOIN planning_items pi ON pi.session_id = ps.id
WHERE ps.board_id = $1
GROUP BY ps.id
ORDER BY COALESCE(ps.meeting_at, ps.created_at) DESC;

-- name: CreatePlanningSession :one
INSERT INTO planning_sessions (board_id, title, label, meeting_at, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPlanningSession :one
SELECT * FROM planning_sessions WHERE id = $1;

-- name: UpdatePlanningSession :one
-- PATCH semantics for planning: nil/omitted JSON fields preserve the existing
-- value (COALESCE-driven). An empty string is treated as a real value — for
-- nullable text columns (label) it stores ''. Required columns (title) must
-- be rejected at the handler layer because the validator's `omitempty`
-- short-circuits min=1 on *string pointing to "".
UPDATE planning_sessions
SET title      = COALESCE(sqlc.narg(title)::varchar, title),
    label      = COALESCE(sqlc.narg(label)::text, label),
    meeting_at = COALESCE(sqlc.narg(meeting_at)::timestamptz, meeting_at),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeletePlanningSession :exec
DELETE FROM planning_sessions WHERE id = $1;

-- name: ListPlanningItemsBySession :many
SELECT * FROM planning_items
WHERE session_id = $1
ORDER BY position ASC;

-- name: GetMaxPlanningItemPosition :one
SELECT COALESCE(MAX(position), 0)::float8 FROM planning_items WHERE session_id = $1;

-- name: CreatePlanningItem :one
INSERT INTO planning_items (session_id, type, title, description, position)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdatePlanningItem :one
-- See UpdatePlanningSession's comment block on PATCH semantics. description,
-- acceptance_criteria, and implementation_note are nullable columns —
-- omitted via nil arg keeps the existing value, "" stores "" (treated as
-- NULL by the app layer). Required ones (type/title/status) go through
-- the handler-level "" check.
UPDATE planning_items
SET type                = COALESCE(sqlc.narg(type)::varchar, type),
    title               = COALESCE(sqlc.narg(title)::text, title),
    description         = COALESCE(sqlc.narg(description)::text, description),
    status              = COALESCE(sqlc.narg(status)::varchar, status),
    position            = COALESCE(sqlc.narg(position)::float8, position),
    acceptance_criteria = COALESCE(sqlc.narg(acceptance_criteria)::text, acceptance_criteria),
    implementation_note = COALESCE(sqlc.narg(implementation_note)::text, implementation_note)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: SetPlanningItemPromoted :exec
UPDATE planning_items
SET status = 'promoted',
    promoted_to_card_id = $2
WHERE id = $1;

-- name: DeletePlanningItem :exec
DELETE FROM planning_items WHERE id = $1;

-- name: GetBoardIDByPlanningSession :one
SELECT board_id FROM planning_sessions WHERE id = $1;

-- name: GetBoardIDByPlanningItem :one
SELECT ps.board_id
FROM planning_items pi
JOIN planning_sessions ps ON ps.id = pi.session_id
WHERE pi.id = $1;

-- name: GetPlanningItem :one
SELECT * FROM planning_items WHERE id = $1;

-- LockPlanningItemForUpdate takes a row-level write lock on the planning
-- item so concurrent PromoteItem callers serialize. Without this, two
-- transactions running at READ COMMITTED can both read status='live',
-- both pass the "already promoted?" check, and both create a card —
-- producing duplicates. Always pair with a transaction.
-- name: LockPlanningItemForUpdate :one
SELECT * FROM planning_items WHERE id = $1 FOR UPDATE;

-- GetPlanningSourceByCard returns the planning item + session that a card
-- was promoted from, or zero rows if the card was never promoted (or the
-- source item/session was deleted). Used by the card detail modal's
-- "ที่มา" (source) section. The partial index idx_planning_items_promoted_card
-- keeps this lookup cheap even when the cards table grows.
-- name: GetPlanningSourceByCard :one
SELECT
    pi.id          AS item_id,
    pi.type        AS item_type,
    pi.title       AS item_title,
    pi.status      AS item_status,
    ps.id          AS session_id,
    ps.title       AS session_title,
    ps.label       AS session_label,
    ps.meeting_at  AS session_meeting_at,
    ps.board_id    AS session_board_id
FROM planning_items pi
JOIN planning_sessions ps ON ps.id = pi.session_id
WHERE pi.promoted_to_card_id = $1;

-- ListPendingQuestionsBySession returns up to N open questions from a
-- planning session — type='Q' and status NOT IN (dropped, promoted). Used
-- alongside GetPlanningSourceByCard to surface "you still have N open
-- questions from this meeting" on the card detail.
-- name: ListPendingQuestionsBySession :many
SELECT id, title
FROM planning_items
WHERE session_id = $1
  AND type = 'Q'
  AND status NOT IN ('dropped', 'promoted')
ORDER BY position ASC
LIMIT $2;

-- Planning item comments — CRUD with soft delete + author name joined in
-- so the list endpoint stays a single round-trip per item.
-- Listing keeps deleted comments visible (the UI renders them as
-- "ถูกลบแล้ว") so the thread's position doesn't shift on delete.

-- name: ListPlanningItemComments :many
SELECT
    c.id,
    c.item_id,
    c.author_id,
    u.full_name AS author_name,
    c.body,
    c.created_at,
    c.updated_at,
    c.deleted_at
FROM planning_item_comments c
JOIN users u ON u.id = c.author_id
WHERE c.item_id = $1
ORDER BY c.created_at ASC;

-- name: GetPlanningItemComment :one
SELECT id, item_id, author_id, body, created_at, updated_at, deleted_at
FROM planning_item_comments
WHERE id = $1;

-- GetBoardIDByPlanningComment resolves comment → item → session → board
-- in one round-trip. Used by edit/delete handlers to re-check membership
-- before touching the row.
-- name: GetBoardIDByPlanningComment :one
SELECT ps.board_id
FROM planning_item_comments c
JOIN planning_items pi ON pi.id = c.item_id
JOIN planning_sessions ps ON ps.id = pi.session_id
WHERE c.id = $1;

-- name: CreatePlanningItemComment :one
INSERT INTO planning_item_comments (item_id, author_id, body)
VALUES ($1, $2, $3)
RETURNING *;

-- UpdatePlanningItemComment edits body. updated_at refreshes so the UI can
-- show an "(edited X ago)" hint. deleted_at is unchanged — the soft-delete
-- path has its own query.
-- name: UpdatePlanningItemComment :one
UPDATE planning_item_comments
SET body       = $2,
    updated_at = now()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePlanningItemComment :exec
UPDATE planning_item_comments
SET deleted_at = now()
WHERE id = $1
  AND deleted_at IS NULL;

-- Claim / release. Both queries return the row count via :execrows so the
-- service layer can map "0 rows updated" to the right sentinel — claim
-- needs to distinguish "row didn't exist" from "someone else holds it",
-- and release needs to distinguish "you didn't own it" from "no claim
-- existed in the first place".

-- ClaimPlanningItem succeeds only if the item is currently unclaimed.
-- The "claimed_by_user_id IS NULL" guard is what gives us the 409
-- semantics for free — no row matched = someone got there first.
-- name: ClaimPlanningItem :execrows
UPDATE planning_items
SET claimed_by_user_id = sqlc.arg(user_id),
    claimed_at         = now()
WHERE id = sqlc.arg(id)
  AND claimed_by_user_id IS NULL;

-- ReleasePlanningItemAsOwner clears the claim only if the caller is the
-- current claimer. Used by the "เลิกดู" button on the row owner side.
-- name: ReleasePlanningItemAsOwner :execrows
UPDATE planning_items
SET claimed_by_user_id = NULL,
    claimed_at         = NULL
WHERE id = sqlc.arg(id)
  AND claimed_by_user_id = sqlc.arg(user_id);

-- ReleasePlanningItemForce clears the claim regardless of who holds it.
-- Used by board owner/manager moderation and by PromoteItem's
-- auto-release path (the planning row is becoming a card; whoever was
-- looking at it doesn't need the claim anymore).
-- name: ReleasePlanningItemForce :exec
UPDATE planning_items
SET claimed_by_user_id = NULL,
    claimed_at         = NULL
WHERE id = $1;

-- name: GetUserSettings :one
-- Returns the caller's workspace preferences. Pair with UpsertUserSettings
-- in the service so first-read materializes a default row and the API
-- never has to branch on "missing".
SELECT user_id, default_landing, show_all_cards, timezone, updated_at
FROM user_settings
WHERE user_id = $1;

-- name: UpsertUserSettings :one
-- INSERT-or-UPDATE on the user_settings row. NULL params keep the existing
-- value (or the column default on first insert), so PATCH /api/me/settings
-- can omit fields it doesn't want to touch.
INSERT INTO user_settings (user_id, default_landing, show_all_cards, timezone, updated_at)
VALUES (
    sqlc.arg(user_id),
    COALESCE(sqlc.narg(default_landing)::varchar, 'today'),
    COALESCE(sqlc.narg(show_all_cards)::boolean,  FALSE),
    COALESCE(sqlc.narg(timezone)::varchar,        'Asia/Bangkok'),
    now()
)
ON CONFLICT (user_id) DO UPDATE SET
    default_landing = COALESCE(sqlc.narg(default_landing)::varchar, user_settings.default_landing),
    show_all_cards  = COALESCE(sqlc.narg(show_all_cards)::boolean,  user_settings.show_all_cards),
    timezone        = COALESCE(sqlc.narg(timezone)::varchar,        user_settings.timezone),
    updated_at      = now()
RETURNING user_id, default_landing, show_all_cards, timezone, updated_at;
