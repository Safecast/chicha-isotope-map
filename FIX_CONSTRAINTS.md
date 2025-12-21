# Fixed: Missing Database Constraints

## Issue
Uploads were failing with error:
```
ERROR: constraint "markers_unique" for table "markers" does not exist (SQLSTATE 42704)
```

## Solution
Added the missing `markers_unique` constraint to the markers table:

```sql
ALTER TABLE markers 
ADD CONSTRAINT markers_unique 
UNIQUE (doseRate, date, lon, lat, countRate, zoom, speed, trackID);
```

This constraint ensures that duplicate markers (same location, time, dose rate, etc.) are not inserted multiple times.

## Status
✅ **Fixed** - The constraint has been added and uploads should now work.

## Test
Try uploading a spectral file again. The upload should now succeed without the constraint error.

