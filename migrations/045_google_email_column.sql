-- Migration 045: Add google_email column to idosos table
-- Stores the linked Google account email for display in the UI

ALTER TABLE idosos ADD COLUMN IF NOT EXISTS google_email TEXT;

-- Safety net: ensure Google OAuth columns exist (they should from earlier migrations)
ALTER TABLE idosos ADD COLUMN IF NOT EXISTS google_refresh_token TEXT;
ALTER TABLE idosos ADD COLUMN IF NOT EXISTS google_access_token TEXT;
ALTER TABLE idosos ADD COLUMN IF NOT EXISTS google_token_expiry TIMESTAMPTZ;
