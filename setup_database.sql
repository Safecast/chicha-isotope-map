-- Safecast PostgreSQL Database Setup Script
-- Run with: sudo -u postgres psql < setup_database.sql
-- Or: psql -U postgres -f setup_database.sql

-- Create database if it doesn't exist
SELECT 'CREATE DATABASE safecast'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'safecast')\gexec

-- Connect to the safecast database
\c safecast

-- Enable PostGIS extension
CREATE EXTENSION IF NOT EXISTS postgis;

-- Verify PostGIS installation
SELECT PostGIS_version() AS postgis_version;

-- Show database info
\dt

\echo ''
\echo '✅ Database setup complete!'
\echo '   Database: safecast'
\echo '   PostGIS: Enabled'
\echo ''
\echo 'You can now run the application with:'
\echo '  ./safecast-new-map'

