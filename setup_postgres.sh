#!/bin/bash
# Setup script for Safecast PostgreSQL database
# This script creates the database, enables PostGIS, and sets up the schema

set -e

# Configuration
DB_NAME="${DB_NAME:-safecast}"
DB_USER="${DB_USER:-postgres}"
PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5432}"

echo "🔧 Setting up PostgreSQL database for Safecast..."
echo "   Database: $DB_NAME"
echo "   User: $DB_USER"
echo "   Host: $PGHOST:$PGPORT"
echo ""

# Check if PostGIS is available
echo "📦 Checking PostGIS extension..."
psql -h "$PGHOST" -p "$PGPORT" -U "$DB_USER" -d postgres -c "SELECT 1 FROM pg_available_extensions WHERE name = 'postgis';" > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "⚠️  Warning: PostGIS extension may not be installed"
    echo "   Install with: sudo apt-get install postgresql-postgis-3"
fi

# Create database if it doesn't exist
echo "📂 Creating database '$DB_NAME'..."
psql -h "$PGHOST" -p "$PGPORT" -U "$DB_USER" -d postgres <<EOF
SELECT 'CREATE DATABASE $DB_NAME'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$DB_NAME')\gexec
EOF

# Enable PostGIS extension
echo "🗺️  Enabling PostGIS extension..."
psql -h "$PGHOST" -p "$PGPORT" -U "$DB_USER" -d "$DB_NAME" <<EOF
CREATE EXTENSION IF NOT EXISTS postgis;
SELECT PostGIS_version();
EOF

echo ""
echo "✅ Database setup complete!"
echo ""
echo "You can now run the application with:"
echo "  ./safecast-new-map"
echo ""
echo "Or with custom settings:"
echo "  export DB_NAME=$DB_NAME"
echo "  export DB_USER=$DB_USER"
echo "  ./safecast-new-map"
echo ""
echo "To migrate existing SQLite data, run:"
echo "  export POSTGRES_URL='postgres://$DB_USER@$PGHOST:$PGPORT/$DB_NAME'"
echo "  go run migrate_to_postgres.go"

