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
  report_id uuid REFERENCES reports(id) NOT NULL,
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
    created_at TIMESTAMPTZ DEFAULT current_timestamp
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
