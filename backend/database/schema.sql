CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- 1. ตาราง Users (เก็บข้อมูลนักพัฒนาและค่าแรง)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    hourly_rate DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    password_hash VARCHAR(255),
    provider VARCHAR(50) NOT NULL DEFAULT 'credentials',
    provider_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- 2. ตาราง Boards (โปรเจกต์)
CREATE TABLE boards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    budget DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    -- งบประมาณรวมของโปรเจกต์ (ส่วนของ ERP Finance)
    description TEXT NOT NULL DEFAULT '',
    -- คำอธิบายสั้น ๆ ของบอร์ด แสดงบนการ์ดในหน้ารายการโปรเจกต์
    color VARCHAR(20) NOT NULL DEFAULT '#1E40AF',
    -- สีประจำบอร์ด (hex) ใช้เป็น accent บน glyph / แถบการ์ด / sidebar
    icon VARCHAR(20) NOT NULL DEFAULT 'board',
    -- คีย์ไอคอนประจำบอร์ด (board/rocket/target/bolt/bug)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
);
-- 3. ตาราง Columns (สถานะงาน เช่น Backlog, In Progress, Done)
CREATE TABLE columns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    position DOUBLE PRECISION NOT NULL,
    category VARCHAR(50) NOT NULL DEFAULT 'TODO',
    color VARCHAR(50) DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE cards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    column_id UUID NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    estimated_hours DECIMAL(5, 2) DEFAULT 0.00,
    priority VARCHAR(20) DEFAULT NULL,
    due_date DATE,
    position DOUBLE PRECISION NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    is_done BOOLEAN NOT NULL DEFAULT FALSE,
    acceptance_criteria TEXT,
    implementation_note TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);
-- 5. ตาราง Time_Logs (บันทึกเวลาทำงาน - หัวใจของการคำนวณ Costing)
CREATE TABLE time_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    card_id UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    hours_spent DECIMAL(5, 2) NOT NULL CHECK (hours_spent > 0),
    cost_incurred DECIMAL(10, 2) NOT NULL,
    -- ต้นทุนที่เกิดขึ้น (hours_spent * hourly_rate ณ เวลานั้น)
    logged_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Index สำหรับเพิ่มความเร็วในการดึงข้อมูล
CREATE INDEX idx_columns_board_id ON columns(board_id);
CREATE INDEX idx_cards_column_id ON cards(column_id);
CREATE INDEX idx_time_logs_card_id ON time_logs(card_id);
CREATE INDEX idx_time_logs_user_id ON time_logs(user_id);
-- รันใน database
CREATE TABLE board_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'manager', 'member')),
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMPTZ,
    UNIQUE (board_id, user_id)
);
CREATE INDEX idx_board_members_board_id ON board_members(board_id);
CREATE INDEX idx_board_members_user_id ON board_members(user_id);
ALTER TABLE board_members
ADD CONSTRAINT board_members_role_check CHECK (role IN ('owner', 'manager', 'member'));
CREATE TABLE card_subtasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    is_done BOOLEAN NOT NULL DEFAULT false,
    position DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
-- สร้าง Index เพื่อให้ Query หา Subtask ของ Card นั้นๆ ได้เร็วขึ้น
CREATE INDEX idx_card_subtasks_card_id ON card_subtasks(card_id);

CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    color VARCHAR(20) NOT NULL DEFAULT 'slate',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    UNIQUE(board_id, name)
);

CREATE TABLE card_tags (
    card_id UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (card_id, tag_id)
);

CREATE INDEX idx_tags_board_id ON tags(board_id);
CREATE INDEX idx_card_tags_card_id ON card_tags(card_id);
CREATE TABLE activities (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id    UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    actor_id    UUID NOT NULL REFERENCES users(id),
    event_type  VARCHAR(40) NOT NULL,
    entity_type VARCHAR(20) NOT NULL,
    entity_id   UUID,
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_activities_board_time ON activities(board_id, created_at DESC);

CREATE TABLE planning_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id    UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    title       VARCHAR(255) NOT NULL,
    label       TEXT,
    meeting_at  TIMESTAMPTZ,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE planning_items (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id            UUID NOT NULL REFERENCES planning_sessions(id) ON DELETE CASCADE,
    type                  VARCHAR(8) NOT NULL CHECK (type IN ('REQ','DEC','Q')),
    title                 TEXT NOT NULL,
    description           TEXT,
    status                VARCHAR(12) NOT NULL DEFAULT 'live'
        CHECK (status IN ('live','selected','dropped','promoted')),
    promoted_to_card_id   UUID REFERENCES cards(id) ON DELETE SET NULL,
    position              DOUBLE PRECISION NOT NULL,
    acceptance_criteria   TEXT,
    implementation_note   TEXT,
    claimed_by_user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    claimed_at            TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_planning_sessions_board ON planning_sessions(board_id);
CREATE INDEX idx_planning_items_session ON planning_items(session_id);

CREATE TABLE planning_item_comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id    UUID NOT NULL REFERENCES planning_items(id) ON DELETE CASCADE,
    author_id  UUID NOT NULL REFERENCES users(id),
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_planning_comments_item
    ON planning_item_comments(item_id, created_at)
    WHERE deleted_at IS NULL;

CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(64) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    replaced_by UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    user_agent  TEXT,
    ip          TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

CREATE TABLE user_settings (
    user_id          UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    default_landing  VARCHAR(20) NOT NULL DEFAULT 'today'
        CHECK (default_landing IN ('today', 'my_work', 'all_boards')),
    show_all_cards   BOOLEAN     NOT NULL DEFAULT FALSE,
    timezone         VARCHAR(50) NOT NULL DEFAULT 'Asia/Bangkok',
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
