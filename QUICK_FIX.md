# Quick Fix: Enable Local Authentication

The `safecast` user is set up but needs authentication. For local development, the easiest solution is to enable trust authentication.

## Option 1: Run the Setup Script (Easiest)

```bash
bash setup_trust_auth.sh
```

This will:
- Configure PostgreSQL to allow local connections without passwords
- Reload PostgreSQL configuration
- Make the application work immediately

## Option 2: Manual Configuration

Edit PostgreSQL config:
```bash
sudo nano /etc/postgresql/*/main/pg_hba.conf
```

Add these lines near the top (before other `local` lines):
```
local   all             safecast                                trust
local   all             postgres                                trust
```

Then reload PostgreSQL:
```bash
sudo systemctl reload postgresql
```

## Option 3: Set a Password Instead

If you prefer password authentication:

```bash
sudo -u postgres psql -c "ALTER USER safecast WITH PASSWORD 'your_password';"
```

Then run:
```bash
export DB_USER=safecast
export DB_PASS='your_password'
./safecast-new-map -safecast-realtime -safecast-fetcher -admin-password test123
```

## After Setup

Run your application:
```bash
export DB_USER=safecast
./safecast-new-map -safecast-realtime -safecast-fetcher -admin-password test123
```

You should see:
```
PostgreSQL connection pool configured: MaxOpenConns=32 (2×16 CPU cores)
Using database driver: pgx with DSN: postgres://safecast@127.0.0.1:5432/safecast?sslmode=prefer
```

No more SQLite! 🎉

