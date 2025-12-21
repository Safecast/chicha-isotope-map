# Fixed: Spectral Data Upload Issue

## Issue
Spectral file uploads were failing with:
```
ERROR: column "channels" of relation "spectra" does not exist (SQLSTATE 42703)
```

## Root Cause
The `spectra` table was created with an old schema that didn't match what the application expects. The table was missing several required columns.

## Solution
Added the missing columns to the `spectra` table:

```sql
ALTER TABLE spectra 
  ADD COLUMN IF NOT EXISTS channels TEXT,
  ADD COLUMN IF NOT EXISTS channel_count INTEGER DEFAULT 1024,
  ADD COLUMN IF NOT EXISTS energy_min_kev DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS energy_max_kev DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS live_time_sec DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS real_time_sec DOUBLE PRECISION;
```

## Status
✅ **Fixed** - All required columns are now present in the spectra table.

## Next Steps
1. Try uploading a spectral file again (`.spe`, `.n42`, `.rctrk`)
2. The spectrum should now be stored successfully
3. The marker's `has_spectrum` flag should be automatically set to `TRUE`
4. Spectral markers should now display on the map

## Verification
To check if spectra are being stored:
```sql
SELECT m.id, m.has_spectrum, s.source_format, s.filename
FROM markers m
JOIN spectra s ON s.marker_id = m.id
LIMIT 10;
```

