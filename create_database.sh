#!/bin/bash
# Script to create Safecast PostgreSQL database
# Run with: bash create_database.sh

set -e

echo "🔧 Creating Safecast PostgreSQL database..."
echo ""

# Check if database already exists
if psql -U postgres -lqt | cut -d \| -f 1 | grep -qw safecast; then
    echo "✅ Database 'safecast' already exists"
else
    echo "📂 Creating database 'safecast'..."
    psql -U postgres -c "CREATE DATABASE safecast;" || {
        echo "❌ Failed to create database. Trying with sudo..."
        sudo -u postgres psql -c "CREATE DATABASE safecast;"
    }
fi

echo "🗺️  Enabling PostGIS extension..."
psql -U postgres -d safecast -c "CREATE EXTENSION IF NOT EXISTS postgis;" || {
    echo "⚠️  Trying with sudo..."
    sudo -u postgres psql -d safecast -c "CREATE EXTENSION IF NOT EXISTS postgis;"
}

echo ""
echo "✅ Verifying PostGIS installation..."
psql -U postgres -d safecast -c "SELECT PostGIS_version();" || {
    echo "⚠️  Trying with sudo..."
    sudo -u postgres psql -d safecast -c "SELECT PostGIS_version();"
}

echo ""
echo "✅ Database setup complete!"
echo ""
echo "You can now run the application with:"
echo "  ./safecast-new-map"
echo ""
echo "Or with custom settings:"
echo "  export DB_USER=postgres"
echo "  export DB_NAME=safecast"
echo "  ./safecast-new-map"

