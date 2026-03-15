-- Profile icon: either a URL (user-uploaded image) or an asset filename (e.g. buddy_buggy.png) for bundled icons.
ALTER TABLE users ADD COLUMN IF NOT EXISTS profile_icon TEXT;
