-- Cluster markers table for faster spatial queries
-- This physically reorders the table data on disk according to the spatial index
-- Expected time: 2-5 minutes depending on table size
-- Expected speedup: 2-5x for bbox queries

\echo '========================================='
\echo 'Clustering markers table by spatial index'
\echo 'This will take 2-5 minutes...'
\echo '========================================='

-- Cluster the table
CLUSTER markers USING idx_markers_zoom_bounds;

-- Update statistics
ANALYZE markers;

\echo ''
\echo '✅ Clustering complete!'
\echo 'Queries filtering by zoom + lat/lon should now be 2-5x faster'
\echo ''
\echo 'Note: Re-run this periodically after bulk imports to maintain performance'
