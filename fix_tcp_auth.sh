#!/bin/bash
# Fix TCP/IP authentication for PostgreSQL
# This adds trust authentication for localhost TCP connections

set -e

PG_HBA="/etc/postgresql/16/main/pg_hba.conf"

echo "🔐 Configuring TCP/IP trust authentication for localhost..."
echo ""

# Backup
BACKUP="${PG_HBA}.backup.tcp.$(date +%Y%m%d_%H%M%S)"
sudo cp "$PG_HBA" "$BACKUP"
echo "📋 Backup created: $BACKUP"

# Check if already configured
if sudo grep -q "^host.*all.*safecast.*127.0.0.1/32.*trust" "$PG_HBA" 2>/dev/null; then
    echo "✅ TCP trust authentication already configured for safecast"
else
    echo "✏️  Adding TCP trust authentication for safecast user..."
    # Add before IPv4 local connections line
    sudo sed -i '/^host.*all.*all.*127.0.0.1\/32/i host    all             safecast            127.0.0.1/32            trust' "$PG_HBA"
    echo "✅ Added TCP trust for safecast"
fi

if sudo grep -q "^host.*all.*postgres.*127.0.0.1/32.*trust" "$PG_HBA" 2>/dev/null; then
    echo "✅ TCP trust authentication already configured for postgres"
else
    echo "✏️  Adding TCP trust authentication for postgres user..."
    sudo sed -i '/^host.*all.*all.*127.0.0.1\/32/i host    all             postgres            127.0.0.1/32            trust' "$PG_HBA"
    echo "✅ Added TCP trust for postgres"
fi

echo ""
echo "🔄 Reloading PostgreSQL..."
sudo systemctl reload postgresql

echo ""
echo "✅ TCP authentication configured!"
echo ""
echo "Now test the connection:"
echo "  export DB_USER=safecast"
echo "  ./safecast-new-map -safecast-realtime -safecast-fetcher -admin-password test123"

