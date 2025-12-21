# Fix TCP/IP Authentication

The application connects via TCP/IP (`127.0.0.1:5432`), but trust authentication was only configured for local sockets. You need to add TCP trust authentication.

## Quick Fix

Run this command to add TCP trust authentication:

```bash
sudo bash fix_tcp_auth.sh
```

## Manual Fix

1. Edit PostgreSQL config:
   ```bash
   sudo nano /etc/postgresql/16/main/pg_hba.conf
   ```

2. Find the line that looks like:
   ```
   host    all             all             127.0.0.1/32            scram-sha-256
   ```

3. Add these lines **BEFORE** that line:
   ```
   host    all             safecast        127.0.0.1/32            trust
   host    all             postgres        127.0.0.1/32            trust
   ```

4. Save and reload PostgreSQL:
   ```bash
   sudo systemctl reload postgresql
   ```

## Verify

After fixing, test the connection:
```bash
export DB_USER=safecast
./safecast-new-map -safecast-realtime -safecast-fetcher -admin-password test123
```

You should see:
```
PostgreSQL connection pool configured: MaxOpenConns=32 (2×16 CPU cores)
Using database driver: pgx with DSN: postgres://safecast@127.0.0.1:5432/safecast?sslmode=prefer
```

No more authentication errors! 🎉

