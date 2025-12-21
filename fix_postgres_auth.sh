#!/bin/bash
# Script to set up PostgreSQL authentication for Safecast
# This creates a password for the postgres user or creates a new user

set -e

echo "🔐 Setting up PostgreSQL authentication..."
echo ""

# Option 1: Set password for postgres user
echo "Setting password for 'postgres' user..."
echo "Please enter a password for the postgres user (or press Enter to skip):"
read -s POSTGRES_PASSWORD

if [ -n "$POSTGRES_PASSWORD" ]; then
    sudo -u postgres psql -c "ALTER USER postgres WITH PASSWORD '$POSTGRES_PASSWORD';"
    echo "✅ Password set for postgres user"
    echo ""
    echo "Now set the environment variable:"
    echo "  export DB_PASS='$POSTGRES_PASSWORD'"
    echo ""
    echo "Or use connection string:"
    echo "  ./safecast-new-map -db-conn 'postgres://postgres:$POSTGRES_PASSWORD@localhost:5432/safecast'"
else
    echo "Skipping password setup..."
    echo ""
    echo "Option 2: Create a new user without password (trust authentication)"
    echo "This allows local connections without a password."
    echo ""
    read -p "Create safecast user? (y/n): " CREATE_USER
    if [ "$CREATE_USER" = "y" ]; then
        sudo -u postgres psql <<EOF
-- Create user if it doesn't exist
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_user WHERE usename = 'safecast') THEN
    CREATE USER safecast;
  END IF;
END
\$\$;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE safecast TO safecast;
\c safecast
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO safecast;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO safecast;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO safecast;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO safecast;
EOF
        echo "✅ User 'safecast' created"
        echo ""
        echo "Now run with:"
        echo "  export DB_USER=safecast"
        echo "  ./safecast-new-map"
    fi
fi

echo ""
echo "✅ Setup complete!"

