-- Enables fast PostGIS proximity queries for destination-based group search.
-- Safe to run repeatedly.

CREATE INDEX IF NOT EXISTS idx_community_groups_destination_location
  ON community_groups
  USING GIST (destination_location)
  WHERE is_deleted = FALSE;

