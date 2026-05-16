ALTER TABLE users
    DROP COLUMN IF EXISTS notification_preferences,
    DROP COLUMN IF EXISTS position,
    DROP COLUMN IF EXISTS phone;
