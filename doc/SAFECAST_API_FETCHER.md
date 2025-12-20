# Safecast API Auto-Importer

## Overview

The Safecast API Auto-Importer is an automated background service that periodically fetches approved bGeigie radiation measurement log files from api.safecast.org and imports them into your local database. This enables your Safecast map instance to automatically stay synchronized with the global Safecast dataset without manual intervention.

## What Problem Does This Solve?

The Safecast project collects radiation measurement data from volunteers worldwide who upload their bGeigie device logs to api.safecast.org. These logs go through a moderation process before being marked as "approved" and made publicly available.

**Without this feature:**
- You would need to manually check api.safecast.org for new approved files
- Each file would need to be manually downloaded
- Files would need to be individually imported via the upload interface
- Your map would become outdated unless constantly maintained

**With this feature:**
- New approved files are automatically discovered every 5 minutes (configurable)
- Files are automatically downloaded from Safecast's S3 storage
- Files are automatically imported using the same proven import pipeline
- Your map stays current with minimal operational overhead
- Full deduplication prevents re-importing files

## How It Works

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Safecast API Auto-Importer                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ Every 5 minutes (configurable)
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 1. Query api.safecast.org/bgeigie_imports.json                   │
│    - Filter: status=approved                                     │
│    - Filter: uploaded_after={start_date} (optional)              │
│    - Pagination: Fetch all pages                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. Deduplication Check                                           │
│    - Query uploads table: source='safecast-api'                  │
│    - Skip files already imported (by source_id)                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. Download Log Files                                            │
│    - Download from S3 URL (source.url)                           │
│    - Validate content ($BNRDD signature for bGeigie)             │
│    - Extract filename from URL                                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. Import Using Existing Pipeline                                │
│    - Parse bGeigie log format                                    │
│    - Convert CPM to µSv/h                                        │
│    - Calculate bounding box                                      │
│    - Detect duplicates                                           │
│    - Compute zoom levels                                         │
│    - Batch insert markers                                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. Track Import                                                  │
│    - Record in uploads table                                     │
│    - Set source='safecast-api'                                   │
│    - Set source_id={safecast_import_id}                          │
│    - Store filename, track_id, file_size, timestamp              │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **API Query**: The fetcher queries the Safecast API endpoint with filters for approved imports
2. **Response Parsing**: JSON response contains import metadata including S3 URLs for log files
3. **Deduplication**: Check local database to skip already-imported files
4. **Download**: Fetch log file content from Amazon S3
5. **Parsing**: Reuse the same bGeigie log parser used for manual uploads
6. **Storage**: Insert measurement markers into the database
7. **Tracking**: Record the import in the uploads table with source tracking

## Database Schema Changes

The implementation extends the existing `uploads` table with two new columns:

| Column | Type | Purpose | Example Value |
|--------|------|---------|---------------|
| `source` | TEXT | Identifies the import source | `"safecast-api"`, `"user-upload"` |
| `source_id` | TEXT | External reference ID | `"12345"` (Safecast import ID) |

**Index**: `idx_uploads_source_id` on `(source, source_id)` for fast deduplication queries

### Why This Approach?

- ✅ Reuses existing infrastructure (uploads table already tracks files)
- ✅ Simple deduplication queries: `WHERE source='safecast-api' AND source_id='12345'`
- ✅ Backward compatible (NULL values for manually uploaded files)
- ✅ Extensible (can add other sources like "opendata-api" in future)
- ✅ Tracks provenance (know which files came from which source)

## Configuration

### CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-safecast-fetcher-enabled` | bool | `false` | Enable the auto-importer service |
| `-safecast-fetcher-interval` | duration | `5m` | How often to poll for new files (e.g., `1m`, `10m`, `1h`) |
| `-safecast-fetcher-batch-size` | int | `10` | Maximum files to import per polling cycle (0 = unlimited) |
| `-safecast-fetcher-start-date` | string | `""` | Only import files uploaded after this date (YYYY-MM-DD format) |

### Usage Examples

#### Conservative Initial Deployment
Start with a small batch size and longer interval to test the system:
```bash
./safecast-new-map \
  -safecast-fetcher-enabled \
  -safecast-fetcher-interval=10m \
  -safecast-fetcher-batch-size=5 \
  -safecast-fetcher-start-date=2025-12-20
```

#### Production Configuration
Once tested, use typical production settings:
```bash
./safecast-new-map \
  -safecast-fetcher-enabled \
  -safecast-fetcher-interval=5m \
  -safecast-fetcher-batch-size=20 \
  -safecast-fetcher-start-date=2025-01-01
```

#### Catch-Up Mode
Import historical data faster (use temporarily):
```bash
./safecast-new-map \
  -safecast-fetcher-enabled \
  -safecast-fetcher-interval=1m \
  -safecast-fetcher-batch-size=50
```

#### Process All Approved Files
Remove date filter to import everything:
```bash
./safecast-new-map \
  -safecast-fetcher-enabled \
  -safecast-fetcher-batch-size=50
```

### Adjusting the Start Date

The `-safecast-fetcher-start-date` flag can be changed at any time:

1. **Move forward in time**: Only newer files will be fetched
2. **Move backward in time**: Older files will be fetched (if not already imported)
3. **Remove filter**: All approved files will be processed

The system tracks which files have been imported, so changing the date won't cause re-imports.

## Logging and Monitoring

### Log Format

The fetcher produces structured log messages with the `[safecast-fetcher]` prefix:

```
[safecast-fetcher] start: interval=5m0s batch=10 start_date=2025-01-01
[safecast-fetcher] poll: checking for imports after ID 58231
[safecast-fetcher] poll: fetched 8 new approved imports
[safecast-fetcher] import #58232: downloading BGM-200110.LOG
[safecast-fetcher] import #58232: imported track abc123def with 452 markers
[safecast-fetcher] import #58233: already imported, skipping
[safecast-fetcher] import #58234: download failed: context deadline exceeded
[safecast-fetcher] summary: imported 6/8, skipped 1, errors 1
```

### Monitoring Recommendations

**Track these metrics:**
- Number of imports per polling cycle
- Error rates (download failures, parse failures)
- Import duration
- Database growth rate

**Watch for:**
- Repeated download failures (network issues, S3 availability)
- Persistent parse errors (format changes, corrupted files)
- Growing error count (may indicate API changes)

## Error Handling

### Error Categories and Responses

| Error Type | Behavior | Example |
|------------|----------|---------|
| **API unreachable** | Log error, skip poll cycle, retry next cycle | Network timeout, DNS failure |
| **Download failure** | Log error, skip file, continue with next | S3 timeout, 404 Not Found |
| **Parse error** | Log error, skip file, continue with next | Invalid format, corrupted data |
| **Database error** | Log critical error, stop batch, retry next cycle | Connection lost, disk full |
| **Already imported** | Log skip message, continue | File already in uploads table |

### Philosophy

The fetcher is designed to be **resilient and non-blocking**:
- Transient errors don't stop the entire batch
- Failed imports can be manually retried later
- The system keeps running even when individual files fail
- Each polling cycle is independent

## Security and Rate Limiting

### API Access
- **Authentication**: None required (public endpoint)
- **Rate Limits**: None documented by Safecast API
- **Respectful polling**: Default 5-minute interval is conservative

### Download Safety
- Files downloaded from trusted Safecast S3 buckets
- Content validation (checks for $BNRDD signature)
- Same security model as manual file uploads
- Existing import pipeline includes duplicate detection

### Recommendations
- Monitor download bandwidth if importing large historical datasets
- Consider longer intervals (10-15 minutes) if server load is a concern
- Use batch size limits during catch-up to avoid overwhelming the database

## Technical Implementation Details

### Package Structure
```
pkg/safecast-fetcher/
├── fetcher.go      # Main coordinator with time.Ticker polling
├── client.go       # Safecast API client (HTTP, JSON parsing)
├── downloader.go   # S3 file download with validation
└── importer.go     # Integration with existing import pipeline
```

### Integration Points

**Reuses existing code:**
- `processBGeigieZenFile()` - Parse bGeigie log format
- `processAndStoreMarkersWithContext()` - Import pipeline
- `db.InsertUpload()` - Track uploaded files
- `GenerateSerialNumber()` - Generate track IDs

**Follows existing patterns:**
- `pkg/safecast-realtime/fetcher.go` - Polling architecture
- `pkg/jsonarchive/generator.go` - Background service pattern
- `pkg/selfupgrade/manager.go` - Context-based lifecycle

### Concurrency Model
- Single goroutine with `time.Ticker` for polling
- Context-based cancellation for graceful shutdown
- Sequential processing within each batch (no parallel downloads)
- Database transactions per import (not per marker)

## FAQ

### Q: Will this re-import files I've already uploaded manually?
**A:** No. The deduplication check looks at source_id. Manually uploaded files have NULL source_id and won't conflict.

### Q: What happens if I restart the service?
**A:** The fetcher queries the database for the highest imported Safecast ID and resumes from there. No state file is needed.

### Q: Can I run multiple instances?
**A:** Not recommended. The deduplication check happens before download, but concurrent imports could cause duplicate work. Use a single instance.

### Q: What if the Safecast API changes?
**A:** The implementation is based on the current API as of December 2025. Monitor logs for parsing errors, which could indicate API changes.

### Q: How much disk space will this use?
**A:** Depends on your start date and import rate. A typical bGeigie log with 1000 measurements is ~100KB. The database will grow based on marker count.

### Q: Can I filter by specific users or regions?
**A:** Not in the initial implementation. The API supports these filters, but they're not exposed as CLI flags. This could be added later.

### Q: What if I want to stop importing?
**A:** Simply restart without the `-safecast-fetcher-enabled` flag. The uploads table retains the tracking information for if/when you re-enable it.

## Troubleshooting

### No files are being imported

**Check:**
1. Is `-safecast-fetcher-enabled` set?
2. Does the start date filter exclude all files?
3. Are there new approved files on api.safecast.org since your last poll?
4. Check logs for API errors

### Import errors

**Common causes:**
1. Invalid file format (not a bGeigie log)
2. Corrupted download
3. Database constraint violations (duplicate markers)

**Solution:** Check logs for specific error messages. Failed files can be manually re-imported.

### High error rate

**Possible issues:**
1. Network connectivity problems
2. S3 bucket availability issues
3. API format changes
4. Database performance issues

**Action:** Review error logs, reduce batch size, increase polling interval.

## Future Enhancements

Potential features for future versions:
- Filter by user ID (import only specific contributors)
- Filter by geographic region (bounding box)
- Webhooks for real-time notifications instead of polling
- Metrics dashboard for monitoring
- Retry queue for failed imports
- Support for other Safecast data formats (not just bGeigie)

## References

- [Safecast API Documentation](https://api.safecast.org/)
- [Safecast API Source Code](https://github.com/Safecast/safecastapi)
- [bGeigie Device Information](https://github.com/Safecast/bGeigieNanoKit)
- [Implementation Plan](/.claude/plans/staged-enchanting-sprout.md)

## License

This feature is part of the safecast-new-map project. See LICENSE file for details.

---

**Document Version:** 1.0
**Last Updated:** 2025-12-20
**Status:** Pre-implementation documentation
