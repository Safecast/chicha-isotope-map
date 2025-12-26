-- Fix recording_date in uploads table to use actual measurement dates
-- This script updates all uploads to use the earliest marker date instead of import date

-- For PostgreSQL
UPDATE uploads u
SET recording_date = to_timestamp(
    (SELECT MIN(date) FROM markers m WHERE m.trackID = u.track_id)
)
WHERE EXISTS (
    SELECT 1 FROM markers m WHERE m.trackID = u.track_id
);

-- Note: This only updates uploads that have markers
-- Uploads without markers (e.g., spectrum-only files) will keep their current recording_date
