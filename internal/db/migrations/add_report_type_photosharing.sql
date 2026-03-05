-- Migration: Allow PHOTOSHARING (image) report type in reports.type
-- The app sends type PHOTOSHARING for photo/image reports.

-- Drop the existing CHECK constraint on reports.type (PostgreSQL default name)
ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_type_check;

-- Re-add the constraint including PHOTOSHARING
ALTER TABLE reports ADD CONSTRAINT reports_type_check CHECK (
  type IN (
    'TRAFFIC',
    'POLICE',
    'ACCIDENT',
    'HAZARD',
    'ROAD_CLOSED',
    'PHOTOSHARING'
  )
);
