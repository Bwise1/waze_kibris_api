-- Migration: Add unique constraint to saved_locations table
-- This prevents duplicate location names (Home, Work, etc.) per user

-- Step 1: Remove duplicate entries, keeping only the most recent one
-- This query will identify duplicates and delete all but the one with the highest ID
DELETE FROM saved_locations
WHERE id NOT IN (
    SELECT MAX(id)
    FROM saved_locations
    GROUP BY user_id, name
);

-- Step 2: Add the unique constraint
ALTER TABLE saved_locations
ADD CONSTRAINT saved_locations_user_name_unique UNIQUE(user_id, name);

-- Step 3: Create an index on (user_id, name) for faster lookups
-- Note: The UNIQUE constraint already creates an index, so this step is optional
-- CREATE INDEX IF NOT EXISTS idx_saved_locations_user_name ON saved_locations(user_id, name);
