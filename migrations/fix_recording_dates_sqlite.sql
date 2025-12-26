-- Fix recording_date in uploads table to use actual measurement dates
-- This script is for SQLite/DuckDB databases

UPDATE uploads
SET recording_date = (
    SELECT MIN(date) FROM markers WHERE markers.trackID = uploads.track_id
)
WHERE track_id IN (
    SELECT DISTINCT trackID FROM markers
);

-- Note: This only updates uploads that have markers
-- Uploads without markers (e.g., spectrum-only files) will keep their current recording_date
