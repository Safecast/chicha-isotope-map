#!/bin/bash
# Diagnostic script to check PostgreSQL authentication configuration

echo "🔍 Checking PostgreSQL authentication configuration..."
echo ""

PG_HBA="/etc/postgresql/16/main/pg_hba.conf"

echo "📋 Current pg_hba.conf entries for 127.0.0.1:"
echo "----------------------------------------"
sudo grep "127.0.0.1" "$PG_HBA" | grep -v "^#" || echo "No entries found"
echo ""

echo "📋 Current pg_hba.conf entries for safecast user:"
echo "----------------------------------------"
sudo grep "safecast" "$PG_HBA" | grep -v "^#" || echo "No entries found"
echo ""

echo "📋 Current pg_hba.conf entries for postgres user:"
echo "----------------------------------------"
sudo grep "^local\|^host" "$PG_HBA" | grep "postgres" | grep -v "^#" | head -5
echo ""

echo "💡 Note: PostgreSQL uses the FIRST matching line in pg_hba.conf"
echo "   Make sure trust entries come BEFORE scram-sha-256 entries"
echo ""

echo "🔄 Try restarting PostgreSQL (instead of reload):"
echo "   sudo systemctl restart postgresql"
echo ""

echo "🧪 Test connection:"
echo "   psql -h 127.0.0.1 -U safecast -d safecast -c 'SELECT 1;'"

