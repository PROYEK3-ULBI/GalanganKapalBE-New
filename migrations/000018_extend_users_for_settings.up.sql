-- Add personal-profile and notification-preference columns to users.
-- These are owned by each user and managed through Settings.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS phone                   VARCHAR(50),
    ADD COLUMN IF NOT EXISTS position                VARCHAR(100),
    ADD COLUMN IF NOT EXISTS notification_preferences JSONB NOT NULL DEFAULT '{}'::jsonb;
