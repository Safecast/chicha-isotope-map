# PostgreSQL Authentication Fix

The application is now using PostgreSQL, but needs authentication configured. Choose one option:

## Option 1: Set Password for Postgres User (Recommended)

```bash
# Set a password for the postgres user
sudo -u postgres psql -c "ALTER USER postgres WITH PASSWORD 'your_password';"

# Then run the application with:
export DB_PASS='your_password'
./safecast-new-map -safecast-realtime -safecast-fetcher -admin-password test123
```

Or use connection string:
```bash
./safecast-new-map -db-conn "postgres://postgres:your_password@localhost:5432/safecast" -safecast-realtime -safecast-fetcher -admin-password test123
```

## Option 2: Create a New User (No Password Required)

```bash
sudo -u postgres psql <<EOF
CREATE USER safecast;
GRANT ALL PRIVILEGES ON DATABASE safecast TO safecast;
\c safecast
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO safecast;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO safecast;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO safecast;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO safecast;
EOF

# Then run with:
export DB_USER=safecast
./safecast-new-map -safecast-realtime -safecast-fetcher -admin-password test123
```

## Option 3: Use Interactive Script

```bash
bash fix_postgres_auth.sh
```

## Option 4: Configure Trust Authentication (Development Only)

**⚠️ WARNING: Only for local development, not production!**

Edit PostgreSQL config:
```bash
sudo nano /etc/postgresql/*/main/pg_hba.conf
```

Change this line:
```
local   all             postgres                                peer
```

To:
```
local   all             postgres                                trust
```

Then restart PostgreSQL:
```bash
sudo systemctl restart postgresql
```

Now you can run without a password:
```bash
./safecast-new-map -safecast-realtime -safecast-fetcher -admin-password test123
```

## Verify Connection

Test the connection:
```bash
psql -U postgres -d safecast -c "SELECT version();"
```

Or with password:
```bash
PGPASSWORD=your_password psql -U postgres -d safecast -c "SELECT version();"
```

