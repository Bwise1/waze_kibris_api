-- Drop table if it exists
DROP TABLE IF EXISTS users;

-- Create the users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    firstname VARCHAR(50),
    lastname VARCHAR(50),
    auth_provider VARCHAR(20) NOT NULL, -- 'email' or 'google'
    auth_provider_id VARCHAR(255), -- Used for Google ID
    preferred_language VARCHAR(10) DEFAULT 'en',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);


-- Create a trigger to automatically update the updated_at field
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- drop email verification table
DROP TABLE IF EXISTS email_verifications;

-- Create email verification table
CREATE TABLE email_verifications (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    email VARCHAR(255) NOT NULL,
    verification_code VARCHAR(4),
    verification_token UUID DEFAULT gen_random_uuid(),
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, email)
);


-- drop email verification table
DROP TABLE IF EXISTS auth_tokens;

-- Auth tokens
CREATE TABLE auth_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    token_type VARCHAR(20) NOT NULL, -- 'access', 'refresh'
    token_value TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    -- created_ip INET,
    -- user_agent TEXT,
    UNIQUE(token_value)
);

-- Index for token lookups and cleanup
CREATE INDEX idx_auth_tokens_value ON auth_tokens(token_value);
CREATE INDEX idx_auth_tokens_user ON auth_tokens(user_id);
CREATE INDEX idx_auth_tokens_expiry ON auth_tokens(expires_at);

-- to invalidate tokens
CREATE TABLE token_blacklist (
    token_hash TEXT PRIMARY KEY,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for cleanup
CREATE INDEX idx_blacklist_expiry ON token_blacklist(expires_at);

-- Cleanup function
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS void AS $$
BEGIN
    DELETE FROM auth_tokens WHERE expires_at < NOW();
    DELETE FROM token_blacklist WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Run cleanup every hour
CREATE EXTENSION IF NOT EXISTS pg_cron;
SELECT cron.schedule('0 * * * *', 'SELECT cleanup_expired_tokens()');




-- Reports table for traffic incidents and hazards
CREATE TABLE reports (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid REFERENCES users(id) NOT NULL,
  type text NOT NULL CHECK (type IN ('TRAFFIC', 'POLICE', 'ACCIDENT', 'HAZARD', 'ROAD_CLOSED')),
  subtype text CHECK (subtype IN ('LIGHT', 'HEAVY', 'STAND_STILL', 'VISIBLE', 'HIDDEN', 'OTHER_SIDE', 'MINOR', 'MAJOR')),
  position geometry(Point, 4326) NOT NULL,
  description text,
  severity integer CHECK (severity BETWEEN 1 AND 5),
  verified_count integer DEFAULT 0,
  active boolean DEFAULT true,
  resolved boolean DEFAULT false,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now(),
  expires_at timestamptz NOT NULL,
  image_url text,
  report_source text CHECK (report_source IN ('USER', 'AUTOMATIC')),
  report_status text CHECK (report_status IN ('PENDING', 'VERIFIED', 'RESOLVED')),
  comments_count integer DEFAULT 0,
  upvotes_count integer DEFAULT 0,
  downvotes_count integer DEFAULT 0,
  CONSTRAINT valid_report_position CHECK (ST_IsValid(position))
);

-- Spatial index for efficient location queries
CREATE INDEX reports_position_idx ON reports USING GIST (position);

-- Index for filtering active reports
CREATE INDEX reports_active_idx ON reports(active) WHERE active = true;

-- Index for expired reports cleanup
CREATE INDEX reports_expires_at_idx ON reports(expires_at);

-- Index for filtering reports by user
CREATE INDEX reports_user_id_idx ON reports(user_id);

-- Index for filtering resolved reports
CREATE INDEX reports_resolved_idx ON reports(resolved);


-- Comments table for storing comments on reports
CREATE TABLE comments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  report_id uuid REFERENCES reports(id) NOT NULL,
  user_id uuid REFERENCES users(id) NOT NULL,
  comment text NOT NULL,
  created_at timestamptz DEFAULT now()
);

-- Index for filtering comments by report
CREATE INDEX comments_report_id_idx ON comments(report_id);

-- Index for filtering comments by user
CREATE INDEX comments_user_id_idx ON comments(user_id);

-- Votes table for storing votes on reports
CREATE TABLE votes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  report_id bigint REFERENCES reports(id) NOT NULL,
  user_id uuid REFERENCES users(id) NOT NULL,
  vote_type text NOT NULL CHECK (vote_type IN ('UPVOTE', 'DOWNVOTE')),
  created_at timestamptz DEFAULT now()
);

-- Index for filtering votes by report
CREATE INDEX votes_report_id_idx ON votes(report_id);

-- Index for filtering votes by user
CREATE INDEX votes_user_id_idx ON votes(user_id);



-- Drop table if it exists
DROP TABLE IF EXISTS saved_locations;

-- Create the saved_locations table with PostGIS support
CREATE TABLE saved_locations (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id UUID REFERENCES users(id),
    name VARCHAR(50) NOT NULL, -- e.g., 'Home', 'Office'
    location GEOMETRY(Point, 4326) NOT NULL,
    place_id VARCHAR(255), -- Google Place ID or other provider place ID
    created_at TIMESTAMPTZ DEFAULT current_timestamp,
    UNIQUE(user_id, name) -- Prevent duplicate location names per user
);


-- reports
-- Drop table if it exists
DROP TABLE IF EXISTS reports;

-- Reports table for traffic incidents and hazards
CREATE TABLE reports (
  id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  user_id uuid REFERENCES users(id) NOT NULL,
  type text NOT NULL CHECK (type IN ('TRAFFIC', 'POLICE', 'ACCIDENT', 'HAZARD', 'ROAD_CLOSED')),
  subtype text CHECK (subtype IN ('LIGHT', 'HEAVY', 'STAND_STILL', 'VISIBLE', 'HIDDEN', 'OTHER_SIDE', 'MINOR', 'MAJOR')),
  position geometry(Point, 4326) NOT NULL,
  description text,
  severity integer CHECK (severity BETWEEN 1 AND 5),
  verified_count integer DEFAULT 0,
  active boolean DEFAULT true,
  resolved boolean DEFAULT false,
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now(),
  expires_at timestamptz NOT NULL,
  image_url text,
  report_source text CHECK (report_source IN ('USER', 'AUTOMATIC')),
  report_status text CHECK (report_status IN ('PENDING', 'VERIFIED', 'RESOLVED')),
  comments_count integer DEFAULT 0,
  upvotes_count integer DEFAULT 0,
  downvotes_count integer DEFAULT 0,
  CONSTRAINT valid_report_position CHECK (ST_IsValid(position))
);

-- Spatial index for efficient location queries
CREATE INDEX reports_position_idx ON reports USING GIST (position);

-- Index for filtering active reports
CREATE INDEX reports_active_idx ON reports(active) WHERE active = true;

-- Index for expired reports cleanup
CREATE INDEX reports_expires_at_idx ON reports(expires_at);

-- Index for filtering reports by user
CREATE INDEX reports_user_id_idx ON reports(user_id);

-- Index for filtering resolved reports
CREATE INDEX reports_resolved_idx ON reports(resolved);


-- --- chats feature ---
CREATE TABLE community_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Unique identifier for the group
    name TEXT NOT NULL, -- Name of the group (e.g., "Traffic to Nicosia Mall", "Concert Attendees")
    description TEXT, -- Description of the group

    -- Group Type and Linking --
    group_type TEXT NOT NULL CHECK (group_type IN ('destination', 'event', 'route', 'general')) DEFAULT 'general', -- Type of group
    destination_place_id TEXT, -- Optional: Link to an external Place ID (e.g., Google Place ID, internal POI ID)
    destination_name TEXT, -- Optional: Human-readable name of the destination/event
    destination_location GEOMETRY(Point, 4326), -- Optional: Coordinates of the destination (Requires PostGIS)

    -- Visibility and Management --
    visibility TEXT NOT NULL CHECK (visibility IN ('public', 'private')) DEFAULT 'public', -- Visibility: public or private
    creator_id UUID REFERENCES users(id) ON DELETE SET NULL, -- User who created the group
    icon_url TEXT, -- Optional: URL for a group icon/avatar

    -- Activity Tracking --
    member_count INT DEFAULT 0, -- Denormalized count of members (update via triggers or app logic)
    last_message_at TIMESTAMPTZ, -- Timestamp of the last message sent in the group (for sorting/activity)

    -- Soft Delete --
    is_deleted BOOLEAN DEFAULT FALSE, -- Flag for soft deletion
    deleted_at TIMESTAMPTZ, -- Timestamp when the group was soft deleted

    -- Timestamps --
    created_at TIMESTAMPTZ DEFAULT NOW(), -- Timestamp when the group was created
    updated_at TIMESTAMPTZ DEFAULT NOW() -- Timestamp when the group was last updated
);

-- Add indexes for community_groups
CREATE INDEX idx_community_groups_visibility ON community_groups (visibility) WHERE is_deleted = FALSE; -- Index only active groups
CREATE INDEX idx_community_groups_type ON community_groups (group_type) WHERE is_deleted = FALSE;
CREATE INDEX idx_community_groups_last_message_at ON community_groups (last_message_at DESC NULLS LAST) WHERE is_deleted = FALSE;
CREATE INDEX idx_community_groups_is_deleted ON community_groups (is_deleted); -- Index for finding deleted groups if needed
-- Add spatial index if using PostGIS for destination_location
-- CREATE INDEX idx_community_groups_destination_location ON community_groups USING GIST (destination_location) WHERE is_deleted = FALSE;

-- --- Enhanced Group Memberships Table ---
CREATE TABLE group_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Unique identifier for the membership
    group_id UUID NOT NULL REFERENCES community_groups(id) ON DELETE CASCADE, -- ID of the group (Ensure NOT NULL)
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- ID of the user (Ensure NOT NULL)
    role TEXT NOT NULL CHECK (role IN ('member', 'moderator', 'admin')) DEFAULT 'member', -- Role in the group

    -- User Status/Preferences --
    last_read_timestamp TIMESTAMPTZ, -- Timestamp of the last message the user read in this group (for unread counts)
    notifications_enabled BOOLEAN DEFAULT TRUE, -- User preference for notifications from this group

    -- Timestamps --
    joined_at TIMESTAMPTZ DEFAULT NOW(), -- Timestamp when the user joined the group
    updated_at TIMESTAMPTZ DEFAULT NOW(), -- Timestamp when the membership was last updated

    -- Constraints --
    UNIQUE (group_id, user_id) -- Ensure a user can only be in a group once
);

-- Add indexes for group_memberships
CREATE INDEX idx_group_memberships_user_id ON group_memberships (user_id);
-- Index on (group_id, user_id) is likely covered by the UNIQUE constraint


-- --- Enhanced Group Messages Table (v2) ---
CREATE TABLE group_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Unique identifier for the message
    group_id UUID NOT NULL REFERENCES community_groups(id) ON DELETE CASCADE, -- ID of the group (Ensure NOT NULL)
    sender_id UUID REFERENCES users(id) ON DELETE SET NULL, -- ID of the user who sent the message (SET NULL if user deleted)

    -- Message Content and Type --
    content TEXT, -- Message content (can be plain text, JSON for structured messages like location pins, or NULL for simple image messages)
    message_type TEXT NOT NULL CHECK (
        message_type IN (
            'text',
            'location_update', -- User's live location update (temporary)
            'eta_update',      -- User sharing their ETA
            'report_share',    -- Sharing an existing report (ID in related_object_id)
            'poll',            -- Poll message (details might be in content or related object)
            'system',          -- System generated message (e.g., user joined/left)
            'image',           -- Message contains an image (URL in attachment_url)
            'location_pin',    -- Pinning a specific map location (details in content as JSON)
            'report_pin'       -- Pinning a specific report (ID in related_object_id)
        )
    ) DEFAULT 'text', -- Type of message
    related_object_id TEXT, -- Optional: ID linking to another object (e.g., report ID for report_share/report_pin, poll ID, user ID for location update)
    attachment_url TEXT, -- Optional: URL for attached files like images (used when message_type='image')

    -- Threading --
    parent_message_id UUID REFERENCES group_messages(id) ON DELETE SET NULL, -- Optional: For replies/threading

    -- Status and Timestamps --
    is_deleted BOOLEAN DEFAULT FALSE, -- Soft delete flag for messages
    created_at TIMESTAMPTZ DEFAULT NOW(), -- Timestamp when the message was sent
    updated_at TIMESTAMPTZ DEFAULT NOW() -- Timestamp when the message was last edited
);

-- Add indexes for group_messages
CREATE INDEX idx_group_messages_group_id_created_at ON group_messages (group_id, created_at DESC);
CREATE INDEX idx_group_messages_sender_id ON group_messages (sender_id);
CREATE INDEX idx_group_messages_parent_id ON group_messages (parent_message_id);
CREATE INDEX idx_group_messages_type ON group_messages (message_type);


-- --- Group Invitations Table (v2 - Unchanged from previous version) ---
CREATE TABLE group_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Unique identifier for the invitation
    group_id UUID NOT NULL REFERENCES community_groups(id) ON DELETE CASCADE, -- ID of the group (Ensure NOT NULL)
    invited_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- ID of the invited user (Ensure NOT NULL)
    invited_by UUID REFERENCES users(id) ON DELETE SET NULL, -- ID of the user who sent the invitation
    status TEXT NOT NULL CHECK (status IN ('pending', 'accepted', 'declined', 'revoked')) DEFAULT 'pending', -- Invitation status (Added 'revoked')
    created_at TIMESTAMPTZ DEFAULT NOW(), -- Timestamp when the invitation was created
    updated_at TIMESTAMPTZ DEFAULT NOW(), -- Timestamp when the invitation was last updated

    -- Constraints --
    UNIQUE (group_id, invited_user_id, status) WHERE status = 'pending' -- Prevent duplicate pending invitations
);

-- Add indexes for group_invitations
CREATE INDEX idx_group_invitations_invited_user_id_status ON group_invitations (invited_user_id, status);

-- --- NEW: Message Reactions Table (v2 - Unchanged from previous version) ---
CREATE TABLE message_reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES group_messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reaction_emoji TEXT NOT NULL, -- The emoji used for reaction (e.g., "👍", "❤️")
    created_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints --
    UNIQUE (message_id, user_id, reaction_emoji) -- Allow a user to react only once with the same emoji per message
);

-- Add indexes for message_reactions
CREATE INDEX idx_message_reactions_message_id ON message_reactions (message_id);
CREATE INDEX idx_message_reactions_user_id ON message_reactions (user_id);

-- --- NEW: Moderation Actions Log Table (v2 - Unchanged from previous version) ---
CREATE TABLE moderation_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES community_groups(id) ON DELETE CASCADE,
    moderator_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- User performing the action
    target_user_id UUID REFERENCES users(id) ON DELETE SET NULL, -- Optional: User being acted upon (e.g., kicked, banned)
    target_message_id UUID REFERENCES group_messages(id) ON DELETE SET NULL, -- Optional: Message being acted upon (e.g., deleted)
    action_type TEXT NOT NULL CHECK (action_type IN ('delete_message', 'kick_user', 'ban_user', 'unban_user', 'promote_moderator', 'demote_moderator')), -- Type of action taken
    reason TEXT, -- Optional: Reason for the action
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add indexes for moderation_actions
CREATE INDEX idx_moderation_actions_group_id ON moderation_actions (group_id);
CREATE INDEX idx_moderation_actions_moderator_id ON moderation_actions (moderator_user_id);
CREATE INDEX idx_moderation_actions_target_user_id ON moderation_actions (target_user_id);
CREATE INDEX idx_moderation_actions_target_message_id ON moderation_actions (target_message_id);
