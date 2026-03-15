-- Backfill profile_icon for existing users who don't have one.
-- Uses the same 9 bundled icon filenames as the app (see auth_helper.go defaultProfileIcons).
-- Assigns one at random per user. Safe to run multiple times (only updates WHERE profile_icon IS NULL).

UPDATE users
SET profile_icon = (
  ARRAY[
    'buddy_buggy.png',
    'camper.png',
    'chill_buddy.png',
    'chill_wheels.png',
    'lone_rider.png',
    'peepers.png',
    'smooth_operator.png',
    'solo_driver.png',
    'the_roadtripper.png'
  ]
)[1 + floor(random() * 9)::int]
WHERE profile_icon IS NULL;
