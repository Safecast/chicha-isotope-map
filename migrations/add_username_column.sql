-- Add username column to uploads table
ALTER TABLE uploads ADD COLUMN IF NOT EXISTS username TEXT;

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_uploads_username ON uploads(username);
