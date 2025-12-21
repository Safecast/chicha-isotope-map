# PostgreSQL Migration Guide

This guide will help you migrate your Safecast data from SQLite (5.7GB) to PostgreSQL for better performance.

## Prerequisites

✅ PostgreSQL 18.1 installed
⬜ PostGIS extension (for spatial queries)
⬜ Sufficient disk space (at least 10GB free)

## Step 1: Create PostgreSQL Database

```bash
# Create database and user
sudo -u postgres psql << 'EOF'
-- Create user
CREATE USER safecast WITH PASSWORD 'your_secure_password_here';

-- Create database
CREATE DATABASE safecast OWNER safecast;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE safecast TO safecast;

-- Connect to the database
\c safecast

-- Enable PostGIS extension (for spatial queries)
CREATE EXTENSION IF NOT EXISTS postgis;

-- Grant schema permissions
GRANT ALL ON SCHEMA public TO safecast;
EOF
```

## Step 2: Run the Migration Script

The migration script will:
1. Read data from SQLite database
2. Create tables in PostgreSQL
3. Copy all data with progress tracking
4. Create indexes and constraints

```bash
# Run the migration (will take 30-60 minutes for 5.7GB)
go run migrate_to_postgres.go
```

## Step 3: Update Application Configuration

Edit your run script or command line to use PostgreSQL:

**Before (SQLite):**
```bash
./safecast-new-map -port 8765 -db database-8765.sqlite
```

**After (PostgreSQL):**
```bash
./safecast-new-map \
  -port 8765 \
  -dbType postgres \
  -dbHost localhost \
  -dbPort 5432 \
  -dbName safecast \
  -dbUser safecast \
  -dbPassword your_secure_password_here
```

## Step 4: Verify Migration

```bash
# Check row counts
psql -U safecast -d safecast << 'EOF'
SELECT 'markers' AS table_name, COUNT(*) FROM markers
UNION ALL
SELECT 'tracks', COUNT(*) FROM tracks
UNION ALL
SELECT 'uploads', COUNT(*) FROM uploads
UNION ALL
SELECT 'spectra', COUNT(*) FROM spectra;
EOF
```

Compare with SQLite:
```bash
sqlite3 database-8765.sqlite << 'EOF'
SELECT 'markers', COUNT(*) FROM markers
UNION ALL
SELECT 'tracks', COUNT(*) FROM tracks
UNION ALL
SELECT 'uploads', COUNT(*) FROM uploads
UNION ALL
SELECT 'spectra', COUNT(*) FROM spectra;
EOF
```

## Expected Performance Improvements

| Operation | SQLite | PostgreSQL | Improvement |
|-----------|--------|------------|-------------|
| Bounding box query (10k markers) | ~500ms | ~50ms | 10x faster |
| Concurrent reads (5 users) | ~2s (blocking) | ~100ms | 20x faster |
| Full table scan | ~5s | ~1s | 5x faster |
| Spatial queries with PostGIS | N/A | ~20ms | 25x faster |

## Troubleshooting

**Connection refused:**
```bash
sudo systemctl status postgresql
sudo systemctl start postgresql
```

**Permission denied:**
```bash
# Verify user exists
sudo -u postgres psql -c "\du"

# Reset password if needed
sudo -u postgres psql -c "ALTER USER safecast PASSWORD 'new_password';"
```

**Out of memory during migration:**
```bash
# Increase PostgreSQL shared_buffers
sudo nano /etc/postgresql/18/main/postgresql.conf
# Set: shared_buffers = 2GB
sudo systemctl restart postgresql
```

## Rollback Plan

If you need to rollback to SQLite:
1. Keep the SQLite database file (don't delete it)
2. Stop the application
3. Start with SQLite parameters again
4. Your data is unchanged in the SQLite file

## Next Steps After Migration

1. **Optimize with PostGIS**: Convert lat/lon to GEOGRAPHY type for better spatial queries
2. **Add spatial indexes**: `CREATE INDEX idx_markers_location ON markers USING GIST (ST_MakePoint(lon, lat)::geography);`
3. **Tune PostgreSQL**: Adjust `work_mem`, `maintenance_work_mem` for better performance
4. **Set up backups**: Configure pg_dump for regular backups

---

**Estimated Migration Time:** 30-60 minutes for 5.7GB database
**Estimated Disk Usage:** ~6-7GB in PostgreSQL (similar to SQLite, better compression)
