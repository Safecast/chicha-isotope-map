# Database Setup Instructions

Since I can't run sudo commands directly, please run one of these options:

## Option 1: Quick Setup (Recommended)

Run this single command:

```bash
sudo -u postgres psql < setup_database.sql
```

This will:
- Create the `safecast` database
- Enable PostGIS extension
- Verify everything is working

## Option 2: Manual Setup

Run these commands step by step:

```bash
# 1. Open PostgreSQL prompt
sudo -u postgres psql

# 2. In the psql prompt, run:
CREATE DATABASE safecast;
\c safecast
CREATE EXTENSION IF NOT EXISTS postgis;
SELECT PostGIS_version();
\q
```

## Option 3: Interactive Script

Run the interactive script:

```bash
bash create_database.sh
```

(You'll be prompted for your sudo password)

## Verify Setup

After running any of the above, verify it worked:

```bash
psql -U postgres -d safecast -c "SELECT PostGIS_version();"
```

Or:

```bash
sudo -u postgres psql -d safecast -c "SELECT PostGIS_version();"
```

## Once Database is Created

You can immediately run the application:

```bash
./safecast-new-map
```

The application will automatically:
- Connect to the `safecast` database
- Create all necessary tables
- Set up spatial indexes
- Be ready to use!

## Troubleshooting

**If PostGIS is not installed:**
```bash
sudo apt-get install postgresql-postgis-3
```

**If you get permission errors:**
- Make sure you're using `sudo -u postgres` for database creation
- The application can connect as your user if permissions are set correctly

