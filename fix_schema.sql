-- Fix PostgreSQL schema to match application expectations
-- Run with: psql -U postgres -d safecast -f fix_schema.sql

-- Add missing columns if they don't exist
DO $$
BEGIN
    -- Add 'zoom' column if it doesn't exist (rename from zoomLevel if needed)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='zoom') THEN
        IF EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='zoomLevel') THEN
            -- Rename zoomLevel to zoom
            ALTER TABLE markers RENAME COLUMN zoomLevel TO zoom;
        ELSE
            -- Add zoom column
            ALTER TABLE markers ADD COLUMN zoom INTEGER;
        END IF;
    END IF;

    -- Add 'speed' column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='speed') THEN
        ALTER TABLE markers ADD COLUMN speed DOUBLE PRECISION;
    END IF;

    -- Add other missing columns that might be needed
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='altitude') THEN
        ALTER TABLE markers ADD COLUMN altitude DOUBLE PRECISION;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='detector') THEN
        ALTER TABLE markers ADD COLUMN detector TEXT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='radiation') THEN
        ALTER TABLE markers ADD COLUMN radiation TEXT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='temperature') THEN
        ALTER TABLE markers ADD COLUMN temperature DOUBLE PRECISION;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='humidity') THEN
        ALTER TABLE markers ADD COLUMN humidity DOUBLE PRECISION;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='device_id') THEN
        ALTER TABLE markers ADD COLUMN device_id TEXT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='transport') THEN
        ALTER TABLE markers ADD COLUMN transport TEXT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='device_name') THEN
        ALTER TABLE markers ADD COLUMN device_name TEXT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='tube') THEN
        ALTER TABLE markers ADD COLUMN tube TEXT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='country') THEN
        ALTER TABLE markers ADD COLUMN country TEXT;
    END IF;

    -- Add geom column if it doesn't exist (for PostGIS)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='markers' AND column_name='geom') THEN
        ALTER TABLE markers ADD COLUMN geom GEOMETRY(POINT, 4326);
        
        -- Create spatial index
        CREATE INDEX IF NOT EXISTS idx_markers_geom_gist ON markers USING GIST(geom);
        
        -- Populate geom from lat/lon
        UPDATE markers SET geom = ST_SetSRID(ST_MakePoint(lon, lat), 4326) 
        WHERE lat IS NOT NULL AND lon IS NOT NULL AND geom IS NULL;
        
        -- Create trigger function if it doesn't exist
        CREATE OR REPLACE FUNCTION update_marker_geom()
        RETURNS TRIGGER AS $$
        BEGIN
          IF NEW.lat IS NOT NULL AND NEW.lon IS NOT NULL THEN
            NEW.geom := ST_SetSRID(ST_MakePoint(NEW.lon, NEW.lat), 4326);
          ELSE
            NEW.geom := NULL;
          END IF;
          RETURN NEW;
        END;
        $$ LANGUAGE plpgsql;
        
        -- Create trigger
        DROP TRIGGER IF EXISTS trigger_update_marker_geom ON markers;
        CREATE TRIGGER trigger_update_marker_geom
          BEFORE INSERT OR UPDATE OF lat, lon ON markers
          FOR EACH ROW
          EXECUTE FUNCTION update_marker_geom();
    END IF;
END $$;

-- Show current schema
\d markers

