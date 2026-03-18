-- Migration: Rename/introduce group_memberships.last_read_at for unread tracking.
-- - If legacy column last_read_timestamp exists, rename it.
-- - Ensure last_read_at is NOT NULL with DEFAULT NOW().
-- - Backfill any NULLs to NOW().

DO $$
BEGIN
  -- Rename legacy column if it exists and new one doesn't.
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'group_memberships'
      AND column_name = 'last_read_timestamp'
  ) AND NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'group_memberships'
      AND column_name = 'last_read_at'
  ) THEN
    ALTER TABLE group_memberships RENAME COLUMN last_read_timestamp TO last_read_at;
  END IF;

  -- Add last_read_at if missing (fresh installs).
  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'group_memberships'
      AND column_name = 'last_read_at'
  ) THEN
    ALTER TABLE group_memberships ADD COLUMN last_read_at TIMESTAMPTZ;
  END IF;
END $$;

-- Backfill and enforce defaults/constraints.
UPDATE group_memberships
SET last_read_at = NOW()
WHERE last_read_at IS NULL;

ALTER TABLE group_memberships
  ALTER COLUMN last_read_at SET DEFAULT NOW(),
  ALTER COLUMN last_read_at SET NOT NULL;

