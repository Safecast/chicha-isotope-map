-- Add recording_date column to uploads table if it doesn't exist
-- For SQLite/DuckDB

-- Add the column
ALTER TABLE uploads ADD COLUMN recording_date BIGINT;

-- Update all existing uploads with the earliest marker date
UPDATE uploads
SET recording_date = (
    SELECT MIN(date) FROM markers WHERE markers.trackID = uploads.track_id
)
WHERE track_id IN (
    SELECT DISTINCT trackID FROM markers
);

-- For uploads without markers, set recording_date to created_at
UPDATE uploads
SET recording_date = created_at
WHERE recording_date IS NULL;
