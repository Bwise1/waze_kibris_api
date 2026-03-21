-- Device tokens for Firebase Cloud Messaging (push notifications).
CREATE TABLE IF NOT EXISTS user_fcm_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token text NOT NULL,
    platform varchar(16) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_fcm_tokens_platform_check CHECK (platform IN ('android', 'ios', 'web')),
    CONSTRAINT user_fcm_tokens_user_token_unique UNIQUE (user_id, token)
);

CREATE INDEX IF NOT EXISTS idx_user_fcm_tokens_user_id ON user_fcm_tokens(user_id);
