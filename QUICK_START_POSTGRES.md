# Quick Start: PostgreSQL Setup

PostgreSQL is installed! Now let's set up the database.

## Option 1: Using sudo (Recommended for first-time setup)

Run these commands to create the database and enable PostGIS:

```bash
# Switch to postgres user and open psql
sudo -u postgres psql

# Then run these SQL commands:
CREATE DATABASE safecast;
\c safecast
CREATE EXTENSION IF NOT EXISTS postgis;
SELECT PostGIS_version();
\q
```

## Option 2: Create a dedicated user (More secure)

```bash
# Switch to postgres user
sudo -u postgres psql

# Create user and database
CREATE USER safecast WITH PASSWORD 'your_secure_password';
CREATE DATABASE safecast OWNER safecast;
\c safecast
CREATE EXTENSION IF NOT EXISTS postgis;
GRANT ALL PRIVILEGES ON DATABASE safecast TO safecast;
\q
```

Then set environment variables:
```bash
export DB_USER=safecast
export DB_PASS=your_secure_password
```

## Option 3: Use the setup script

If you have PostgreSQL access configured:

```bash
# Set your PostgreSQL user (defaults to 'postgres')
export DB_USER=postgres

# Run the setup script
./setup_postgres.sh
```

## Verify Setup

Check that everything is working:

```bash
# Test connection
psql -U postgres -d safecast -c "SELECT PostGIS_version();"

# Or if using a custom user:
psql -U safecast -d safecast -c "SELECT PostGIS_version();"
```

## Run the Application

Once the database is set up, you can run:

```bash
# Using defaults (postgres user, safecast database)
./safecast-new-map

# Or with environment variables
export DB_USER=safecast
export DB_PASS=yourpassword
./safecast-new-map

# Or with connection string
./safecast-new-map -db-conn "postgres://safecast:yourpassword@localhost:5432/safecast"
```

## Troubleshooting

**If you get "Peer authentication failed":**
- Use `sudo -u postgres psql` to connect as the postgres user
- Or configure `pg_hba.conf` to allow password authentication

**If PostGIS is not found:**
```bash
# Install PostGIS extension
sudo apt-get install postgresql-postgis-3

# Then enable it in the database:
sudo -u postgres psql -d safecast -c "CREATE EXTENSION IF NOT EXISTS postgis;"
```

**Check PostgreSQL is running:**
```bash
sudo systemctl status postgresql
# Or
pg_isready
```

