# PostgreSQL Setup Guide

## Overview

The application now defaults to PostgreSQL (`pgx`) instead of SQLite. SQLite is still fully supported and can be used by specifying `-db-type=sqlite`.

## Quick Start

### 1. Install PostgreSQL

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib postgis
```

**macOS (Homebrew):**
```bash
brew install postgresql postgis
```

**Docker:**
```bash
docker run -d \
  --name safecast-postgres \
  -e POSTGRES_PASSWORD=yourpassword \
  -e POSTGRES_DB=safecast \
  -p 5432:5432 \
  postgis/postgis:15-3.3
```

### 2. Create Database and User

```bash
# Connect to PostgreSQL
sudo -u postgres psql

# Create database
CREATE DATABASE safecast;

# Enable PostGIS extension
\c safecast
CREATE EXTENSION IF NOT EXISTS postgis;

# Create user (optional, can use 'postgres' user)
CREATE USER safecast WITH PASSWORD 'yourpassword';
GRANT ALL PRIVILEGES ON DATABASE safecast TO safecast;
\q
```

### 3. Run the Application

**Option 1: Using defaults (localhost, postgres user, safecast database)**
```bash
./safecast-new-map
```

**Option 2: Using environment variables**
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=safecast
export DB_PASS=yourpassword
export DB_NAME=safecast
export DB_SSLMODE=prefer
./safecast-new-map
```

**Option 3: Using connection string**
```bash
./safecast-new-map -db-conn "postgres://safecast:yourpassword@localhost:5432/safecast?sslmode=prefer"
```

**Option 4: Using SQLite (fallback)**
```bash
./safecast-new-map -db-type=sqlite
```

## Configuration Options

### Command-Line Flags

- `-db-type`: Database driver (default: `pgx`)
  - Options: `pgx`, `sqlite`, `chai`, `duckdb`, `clickhouse`
- `-db-conn`: Full connection URI (overrides individual settings)
  - Example: `postgres://user:pass@host:5432/dbname?sslmode=prefer`

### Environment Variables

These are used as defaults when `-db-conn` is not provided:

- `DB_HOST`: PostgreSQL host (default: `127.0.0.1`)
- `DB_PORT`: PostgreSQL port (default: `5432`)
- `DB_USER`: PostgreSQL user (default: `postgres`)
- `DB_PASS`: PostgreSQL password (default: empty)
- `DB_NAME`: Database name (default: `safecast`)
- `DB_SSLMODE`: SSL mode (default: `prefer`)
  - Options: `disable`, `allow`, `prefer`, `require`, `verify-ca`, `verify-full`

### Priority Order

1. `-db-conn` flag (highest priority - overrides everything)
2. Environment variables
3. Hard-coded defaults

## Migration from SQLite

If you have an existing SQLite database and want to migrate to PostgreSQL:

1. **Run the migration script:**
   ```bash
   export POSTGRES_URL='postgres://safecast:yourpassword@localhost:5432/safecast?sslmode=prefer'
   go run migrate_to_postgres.go
   ```

2. **Or manually migrate:**
   - The migration script will:
     - Create PostGIS extension
     - Create all tables with spatial indexes
     - Migrate data from SQLite
     - Set up triggers for automatic geometry updates

## Verification

### Check PostgreSQL Connection

```bash
psql -h localhost -U safecast -d safecast -c "SELECT version();"
```

### Check PostGIS Extension

```bash
psql -h localhost -U safecast -d safecast -c "SELECT PostGIS_version();"
```

### Check Spatial Index

```bash
psql -h localhost -U safecast -d safecast -c "\d markers"
# Should show: idx_markers_geom_gist (GIST index on geom column)
```

## Troubleshooting

### Connection Refused

- Check PostgreSQL is running: `sudo systemctl status postgresql`
- Verify port: `netstat -tlnp | grep 5432`
- Check `pg_hba.conf` for authentication settings

### Authentication Failed

- Verify username/password
- Check PostgreSQL logs: `sudo tail -f /var/log/postgresql/postgresql-*.log`
- Ensure user has database access: `GRANT ALL PRIVILEGES ON DATABASE safecast TO safecast;`

### PostGIS Extension Missing

```sql
-- Connect to database
\c safecast

-- Install PostGIS
CREATE EXTENSION IF NOT EXISTS postgis;
```

### Database Does Not Exist

```sql
CREATE DATABASE safecast;
\c safecast
CREATE EXTENSION IF NOT EXISTS postgis;
```

## Performance Tips

1. **Connection Pooling**: Already configured (2×CPU cores)
2. **Spatial Indexes**: Automatically created via migration script
3. **Query Optimization**: Uses PostGIS `ST_Intersects` with `&&` operator

## Security Considerations

- **Production**: Use `sslmode=require` or `verify-full`
- **Development**: `sslmode=prefer` or `disable` is acceptable
- **Password**: Never commit passwords to version control
- **User Permissions**: Use dedicated database user with minimal required privileges

## Switching Back to SQLite

If you need to use SQLite instead:

```bash
./safecast-new-map -db-type=sqlite
```

All SQLite functionality remains unchanged and fully supported.

