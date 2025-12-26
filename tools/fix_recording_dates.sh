#!/bin/bash

# Script to fix recording dates in the uploads table
# Usage: ./fix_recording_dates.sh [database_type] [connection_string_or_path]

set -e

DB_TYPE="${1:-pgx}"
DB_CONN="${2}"

echo "Fixing recording dates in uploads table..."
echo "Database type: $DB_TYPE"

case "$DB_TYPE" in
    pgx|postgres|postgresql)
        if [ -z "$DB_CONN" ]; then
            echo "Error: Connection string required for PostgreSQL"
            echo "Usage: $0 pgx 'postgres://user:pass@host:5432/dbname'"
            exit 1
        fi
        echo "Applying PostgreSQL fix..."
        psql "$DB_CONN" -f migrations/fix_recording_dates.sql
        echo "Done! Recording dates updated."
        ;;
    
    sqlite)
        DB_PATH="${DB_CONN:-database-8765.sqlite}"
        echo "Applying SQLite fix to: $DB_PATH"
        sqlite3 "$DB_PATH" < migrations/fix_recording_dates_sqlite.sql
        echo "Done! Recording dates updated."
        ;;
    
    duckdb)
        DB_PATH="${DB_CONN:-database-8765.duckdb}"
        echo "Applying DuckDB fix to: $DB_PATH"
        duckdb "$DB_PATH" < migrations/fix_recording_dates_sqlite.sql
        echo "Done! Recording dates updated."
        ;;
    
    *)
        echo "Error: Unsupported database type: $DB_TYPE"
        echo "Supported types: pgx, sqlite, duckdb"
        exit 1
        ;;
esac

echo ""
echo "Recording dates have been updated to reflect the actual measurement dates"
echo "from the log files instead of the import dates."
