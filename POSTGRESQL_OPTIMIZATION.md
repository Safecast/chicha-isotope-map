# PostgreSQL Multithreading & Multi-Core Optimization Report

## Summary

This document outlines the optimizations implemented and recommendations for improving PostgreSQL performance and multithreading support.

## ✅ Completed Optimizations

### 1. PostgreSQL Connection Pool Configuration
**Status:** ✅ **COMPLETED**

- **Location:** `pkg/database/database.go` (lines ~602-608)
- **Changes:**
  - Added PostgreSQL-specific connection pool configuration
  - Set `MaxOpenConns` to `2×CPU cores` (minimum 8)
  - Set `MaxIdleConns` to match `MaxOpenConns`
  - Configured connection lifetime: 5 minutes
  - Configured idle timeout: 2 minutes
- **Impact:** Enables better concurrent query execution and prevents connection exhaustion

### 2. PostGIS Spatial Index (GIST)
**Status:** ✅ **COMPLETED**

- **Location:** `migrate_to_postgres.go` (schema creation)
- **Changes:**
  - Added PostGIS extension (`CREATE EXTENSION IF NOT EXISTS postgis`)
  - Added `geom GEOMETRY(POINT, 4326)` column to `markers` table
  - Created GIST spatial index: `CREATE INDEX idx_markers_geom_gist ON markers USING GIST(geom)`
  - Added trigger to automatically populate `geom` column from `lat`/`lon`
  - Updated existing rows to populate `geom` column
- **Impact:** Dramatically improves spatial query performance using spatial indexes

### 3. PostGIS Spatial Queries with && Bounding Box
**Status:** ✅ **COMPLETED**

- **Locations:**
  - `pkg/database/stream.go` - StreamMarkersByZoomAndBounds, StreamMarkersByTrackIDZoomAndBounds
  - `pkg/database/database.go` - All GetMarkers* functions with bounds filtering
- **Changes:**
  - Created `buildSpatialWhereClause()` helper function
  - Updated queries to use `ST_Intersects(geom, ST_MakeEnvelope(...))` with `&&` operator for PostgreSQL
  - Maintains backward compatibility with other databases (SQLite, DuckDB, ClickHouse)
- **Impact:** 
  - Uses GIST index efficiently via `&&` bounding box operator
  - `ST_Intersects` provides exact spatial filtering
  - Significantly faster than `lat BETWEEN` and `lon BETWEEN` for large datasets

## ⚠️ Remaining Optimizations (Require Larger Refactoring)

### 1. Migrate to Native pgxpool
**Status:** ⚠️ **RECOMMENDED BUT REQUIRES REFACTORING**

**Current State:**
- Using `database/sql` with `github.com/jackc/pgx/v5/stdlib` wrapper
- Works but doesn't leverage pgx's native connection pooling features

**Benefits of Migration:**
- Better concurrency control
- Lower overhead (no wrapper layer)
- Native prepared statement support
- Better error handling and type support

**Required Changes:**
- Replace `*sql.DB` with `*pgxpool.Pool` in `Database` struct
- Update all query methods to use `pgxpool` API
- Update transaction handling
- This is a significant refactoring affecting many files

**Recommendation:** Consider this for a future major version update.

### 2. Prepared Statements
**Status:** ⚠️ **RECOMMENDED BUT REQUIRES REFACTORING**

**Current State:**
- Queries are built dynamically as strings
- No prepared statement caching

**Benefits:**
- Reduced query parsing overhead
- Better query plan caching
- Protection against SQL injection (already handled via placeholders)

**Required Changes:**
- Identify frequently executed queries
- Create prepared statement cache
- Use `Prepare()` or pgxpool's prepared statement support
- Update query execution to use prepared statements

**Recommendation:** Implement for high-frequency queries first (e.g., `GetMarkersByZoomAndBounds`).

## Performance Impact

### Expected Improvements

1. **Connection Pooling:**
   - Better utilization of multiple CPU cores
   - Reduced connection overhead
   - Better handling of concurrent requests

2. **Spatial Indexes:**
   - 10-100x faster spatial queries (depending on dataset size)
   - Efficient bounding box filtering
   - Better query plan optimization

3. **PostGIS Queries:**
   - Optimal use of GIST indexes
   - Faster spatial filtering
   - Better scalability with large datasets

## Testing Recommendations

1. **Load Testing:**
   - Test with concurrent requests (2×CPU cores)
   - Monitor connection pool usage
   - Verify no connection exhaustion

2. **Spatial Query Performance:**
   - Compare query times before/after PostGIS changes
   - Test with various bounding box sizes
   - Verify index usage with `EXPLAIN ANALYZE`

3. **Migration Testing:**
   - Test migration script on sample database
   - Verify `geom` column population
   - Test trigger functionality

## Migration Steps

1. **Run Migration Script:**
   ```bash
   export POSTGRES_URL='host=localhost port=5432 dbname=safecast user=safecast password=yourpassword sslmode=disable'
   go run migrate_to_postgres.go
   ```

2. **Verify Spatial Index:**
   ```sql
   SELECT indexname, indexdef 
   FROM pg_indexes 
   WHERE tablename = 'markers' AND indexname = 'idx_markers_geom_gist';
   ```

3. **Verify Query Performance:**
   ```sql
   EXPLAIN ANALYZE
   SELECT * FROM markers 
   WHERE geom && ST_MakeEnvelope(-180, -90, 180, 90, 4326)
     AND ST_Intersects(geom, ST_MakeEnvelope(-180, -90, 180, 90, 4326));
   ```

## Notes

- The code maintains backward compatibility with SQLite, DuckDB, and ClickHouse
- PostGIS optimizations only apply when using PostgreSQL (`dbType == "pgx"`)
- Connection pool size automatically scales with CPU cores
- Spatial index is automatically maintained via trigger

