-- Migration: Add user_id column to uploads table
-- Date: 2025-12-21
-- Description: Adds user_id column to track which Safecast user uploaded each file

-- For SQLite
ALTER TABLE uploads ADD COLUMN user_id TEXT;
CREATE INDEX IF NOT EXISTS idx_uploads_user_id ON uploads(user_id);

-- For PostgreSQL (if using):
-- ALTER TABLE uploads ADD COLUMN user_id TEXT;
-- CREATE INDEX IF NOT EXISTS idx_uploads_user_id ON uploads(user_id);

-- For DuckDB (if using):
-- ALTER TABLE uploads ADD COLUMN user_id TEXT;
-- CREATE INDEX IF NOT EXISTS idx_uploads_user_id ON uploads(user_id);
