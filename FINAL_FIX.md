# Final Fix: PostgreSQL Authentication

If you're still getting authentication errors, try these steps:

## Step 1: Check Configuration

Run the diagnostic script:
```bash
bash check_auth.sh
```

This will show you what's currently in pg_hba.conf.

## Step 2: Verify pg_hba.conf Order

**Important:** PostgreSQL uses the FIRST matching line. Trust entries must come BEFORE scram-sha-256 entries.

Check the file:
```bash
sudo nano /etc/postgresql/16/main/pg_hba.conf
```

Make sure these lines appear BEFORE any `scram-sha-256` lines for 127.0.0.1:
```
host    all             safecast        127.0.0.1/32            trust
host    all             postgres        127.0.0.1/32            trust
```

## Step 3: Restart PostgreSQL (Not Just Reload)

Sometimes a full restart is needed:
```bash
sudo systemctl restart postgresql
```

## Step 4: Test Connection

Test the connection directly:
```bash
psql -h 127.0.0.1 -U safecast -d safecast -c "SELECT 1;"
```

If this works, the application should work too.

## Alternative: Use Unix Socket Instead

If TCP authentication is problematic, you can configure the app to use Unix sockets by setting the host to empty or "localhost" without specifying 127.0.0.1. However, the current code uses 127.0.0.1, so TCP auth is needed.

## Last Resort: Set a Password

If trust authentication continues to be problematic:

```bash
sudo -u postgres psql -c "ALTER USER safecast WITH PASSWORD 'safecast123';"
```

Then run:
```bash
export DB_USER=safecast
export DB_PASS='safecast123'
./safecast-new-map -safecast-realtime -safecast-fetcher -admin-password test123
```

