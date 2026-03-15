-- Migration: Create messages table for group chat (and later DMs).
-- Prerequisites: community_groups and users tables must exist.
-- Run with: psql -d your_database -f internal/db/migrations/create_messages.sql

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID REFERENCES community_groups(id) ON DELETE CASCADE,
    sender_id UUID REFERENCES users(id) ON DELETE SET NULL,

    content TEXT,
    message_type TEXT NOT NULL CHECK (
        message_type IN (
            'text',
            'location_update',
            'eta_update',
            'report_share',
            'poll',
            'system',
            'image',
            'location_pin',
            'report_pin'
        )
    ) DEFAULT 'text',
    related_object_id TEXT,
    attachment_url TEXT,

    parent_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,

    is_deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_group_id_created_at
    ON messages (group_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_sender_id
    ON messages (sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_parent_id
    ON messages (parent_message_id);
CREATE INDEX IF NOT EXISTS idx_messages_type
    ON messages (message_type);
