#!/bin/bash
# Script to configure PostgreSQL trust authentication for local development
# WARNING: This is for development only, not production!

set -e

echo "🔐 Configuring PostgreSQL trust authentication for local connections..."
echo "⚠️  WARNING: This is for development only!"
echo ""

# Find pg_hba.conf
PG_HBA=$(sudo find /etc/postgresql -name "pg_hba.conf" 2>/dev/null | head -1)

if [ -z "$PG_HBA" ]; then
    echo "❌ Could not find pg_hba.conf"
    echo "   Please locate it manually and edit it"
    exit 1
fi

echo "Found pg_hba.conf at: $PG_HBA"
echo ""

# Backup the file
BACKUP="${PG_HBA}.backup.$(date +%Y%m%d_%H%M%S)"
echo "📋 Creating backup: $BACKUP"
sudo cp "$PG_HBA" "$BACKUP"

# Check if already configured
if grep -q "^local.*all.*safecast.*trust" "$PG_HBA" 2>/dev/null; then
    echo "✅ Trust authentication already configured for safecast user"
else
    echo "✏️  Adding trust authentication for safecast user..."
    
    # Add trust line for safecast user (before any existing local lines)
    sudo sed -i '/^local.*all.*all/i local   all             safecast                                trust' "$PG_HBA"
    
    echo "✅ Added trust authentication for safecast user"
fi

# Also add for postgres user if not already there
if ! grep -q "^local.*all.*postgres.*trust" "$PG_HBA" 2>/dev/null; then
    echo "✏️  Adding trust authentication for postgres user..."
    sudo sed -i '/^local.*all.*all/i local   all             postgres                                trust' "$PG_HBA"
    echo "✅ Added trust authentication for postgres user"
fi

echo ""
echo "🔄 Reloading PostgreSQL configuration..."
sudo systemctl reload postgresql || sudo service postgresql reload

echo ""
echo "✅ Configuration complete!"
echo ""
echo "You can now run the application without a password:"
echo "  export DB_USER=safecast"
echo "  ./safecast-new-map -safecast-realtime -safecast-fetcher -admin-password test123"
echo ""
echo "Or with postgres user:"
echo "  ./safecast-new-map -safecast-realtime -safecast-fetcher -admin-password test123"

