package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type MigrationStats struct {
	TableName    string
	RowsMigrated int64
	StartTime    time.Time
	Duration     time.Duration
}

func main() {
	// Configuration
	sqlitePath := "./database-8765.sqlite"
	postgresConnStr := os.Getenv("POSTGRES_URL")
	if postgresConnStr == "" {
		postgresConnStr = "host=localhost port=5432 dbname=safecast user=safecast password=your_password sslmode=disable"
		fmt.Println("⚠️  Using default PostgreSQL connection string")
		fmt.Println("   Set POSTGRES_URL environment variable to customize")
		fmt.Println("   Example: export POSTGRES_URL='host=localhost port=5432 dbname=safecast user=safecast password=yourpassword sslmode=disable'")
		fmt.Println()
	}

	// Open SQLite source database
	fmt.Println("📂 Opening SQLite database...")
	sqliteDB, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer sqliteDB.Close()

	// Open PostgreSQL target database
	fmt.Println("🐘 Connecting to PostgreSQL...")
	pgDB, err := sql.Open("pgx", postgresConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgDB.Close()

	// Test connections
	if err := sqliteDB.Ping(); err != nil {
		log.Fatalf("SQLite connection failed: %v", err)
	}
	if err := pgDB.Ping(); err != nil {
		log.Fatalf("PostgreSQL connection failed: %v", err)
	}

	fmt.Println("✅ Database connections established")
	fmt.Println()

	// Create PostgreSQL schema
	fmt.Println("🏗️  Creating PostgreSQL schema...")
	if err := createPostgreSQLSchema(pgDB); err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}
	fmt.Println("✅ Schema created")
	fmt.Println()

	// Migrate tables
	stats := []MigrationStats{}

	// Migrate in order (respecting foreign keys)
	tables := []string{"tracks", "markers", "spectra", "uploads"}

	for _, table := range tables {
		stat, err := migrateTable(sqliteDB, pgDB, table)
		if err != nil {
			log.Printf("❌ Failed to migrate table %s: %v", table, err)
			continue
		}
		stats = append(stats, stat)
	}

	// Print summary
	fmt.Println()
	fmt.Println(strings.Repeat("=", 61))
	fmt.Println("📊 Migration Summary")
	fmt.Println(strings.Repeat("=", 61))
	totalRows := int64(0)
	totalDuration := time.Duration(0)
	for _, stat := range stats {
		fmt.Printf("%-15s: %10d rows migrated in %v\n", stat.TableName, stat.RowsMigrated, stat.Duration.Round(time.Second))
		totalRows += stat.RowsMigrated
		totalDuration += stat.Duration
	}
	fmt.Println(strings.Repeat("-", 61))
	fmt.Printf("%-15s: %10d rows in %v\n", "TOTAL", totalRows, totalDuration.Round(time.Second))
	fmt.Println(strings.Repeat("=", 61))
	fmt.Println()
	fmt.Println("✅ Migration completed successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Verify row counts match SQLite")
	fmt.Println("2. Update application to use PostgreSQL")
	fmt.Println("3. Test the application")
	fmt.Println("4. Consider adding PostGIS spatial indexes for better performance")
}

func createPostgreSQLSchema(db *sql.DB) error {
	schema := `
-- Create tracks table
CREATE TABLE IF NOT EXISTS tracks (
  trackID TEXT PRIMARY KEY,
  minLat DOUBLE PRECISION,
  minLon DOUBLE PRECISION,
  maxLat DOUBLE PRECISION,
  maxLon DOUBLE PRECISION,
  created TIMESTAMPTZ,
  zoomLevel INTEGER,
  UNIQUE(trackID, zoomLevel)
);

-- Create markers table
CREATE TABLE IF NOT EXISTS markers (
  id BIGSERIAL PRIMARY KEY,
  doseRate DOUBLE PRECISION,
  date BIGINT,
  lon DOUBLE PRECISION,
  lat DOUBLE PRECISION,
  countRate DOUBLE PRECISION,
  trackID TEXT,
  zoomLevel INTEGER DEFAULT 10,
  has_spectrum BOOLEAN DEFAULT FALSE
);

-- Create spectra table
CREATE TABLE IF NOT EXISTS spectra (
  id BIGSERIAL PRIMARY KEY,
  marker_id BIGINT,
  detector_type TEXT,
  device_serial TEXT,
  device_model TEXT,
  calibration TEXT,
  source_format TEXT,
  filename TEXT,
  raw_data BYTEA,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  FOREIGN KEY (marker_id) REFERENCES markers(id) ON DELETE CASCADE
);

-- Create uploads table
CREATE TABLE IF NOT EXISTS uploads (
  id BIGSERIAL PRIMARY KEY,
  filename TEXT NOT NULL,
  file_type TEXT,
  track_id TEXT,
  file_size BIGINT,
  upload_ip TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  source TEXT,
  source_id TEXT,
  source_url TEXT,
  user_id TEXT
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_markers_trackid ON markers(trackID);
CREATE INDEX IF NOT EXISTS idx_markers_lat ON markers(lat);
CREATE INDEX IF NOT EXISTS idx_markers_lon ON markers(lon);
CREATE INDEX IF NOT EXISTS idx_markers_zoomlevel ON markers(zoomLevel);
CREATE INDEX IF NOT EXISTS idx_markers_lat_lon ON markers(lat, lon);
CREATE INDEX IF NOT EXISTS idx_markers_has_spectrum ON markers(has_spectrum) WHERE has_spectrum = TRUE;

CREATE INDEX IF NOT EXISTS idx_tracks_trackid_zoom ON tracks(trackID, zoomLevel);
CREATE INDEX IF NOT EXISTS idx_tracks_bounds ON tracks(minLat, minLon, maxLat, maxLon);

CREATE INDEX IF NOT EXISTS idx_spectra_marker_id ON spectra(marker_id);

CREATE INDEX IF NOT EXISTS idx_uploads_track_id ON uploads(track_id);
CREATE INDEX IF NOT EXISTS idx_uploads_created_at ON uploads(created_at);
CREATE INDEX IF NOT EXISTS idx_uploads_user_id ON uploads(user_id);
CREATE INDEX IF NOT EXISTS idx_uploads_source_id ON uploads(source, source_id);
`

	_, err := db.Exec(schema)
	return err
}

func migrateTable(sqliteDB, pgDB *sql.DB, tableName string) (MigrationStats, error) {
	stat := MigrationStats{
		TableName: tableName,
		StartTime: time.Now(),
	}

	fmt.Printf("📋 Migrating table: %s...\n", tableName)

	// Count total rows
	var totalRows int64
	err := sqliteDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&totalRows)
	if err != nil {
		return stat, fmt.Errorf("count rows: %w", err)
	}
	fmt.Printf("   Total rows to migrate: %d\n", totalRows)

	// Start transaction
	tx, err := pgDB.Begin()
	if err != nil {
		return stat, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Migrate based on table
	switch tableName {
	case "markers":
		err = migrateMarkers(sqliteDB, tx, &stat)
	case "tracks":
		err = migrateTracks(sqliteDB, tx, &stat)
	case "spectra":
		err = migrateSpectra(sqliteDB, tx, &stat)
	case "uploads":
		err = migrateUploads(sqliteDB, tx, &stat)
	default:
		return stat, fmt.Errorf("unknown table: %s", tableName)
	}

	if err != nil {
		return stat, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return stat, fmt.Errorf("commit: %w", err)
	}

	stat.Duration = time.Since(stat.StartTime)
	fmt.Printf("   ✅ Migrated %d rows in %v\n", stat.RowsMigrated, stat.Duration.Round(time.Second))

	return stat, nil
}

func migrateMarkers(sqliteDB *sql.DB, tx *sql.Tx, stat *MigrationStats) error {
	rows, err := sqliteDB.Query("SELECT id, doseRate, date, lon, lat, countRate, trackID, zoomLevel, has_spectrum FROM markers ORDER BY id")
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	stmt, err := tx.Prepare("INSERT INTO markers (id, doseRate, date, lon, lat, countRate, trackID, zoomLevel, has_spectrum) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)")
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	batchSize := 10000
	count := 0
	for rows.Next() {
		var id int64
		var doseRate, lon, lat, countRate float64
		var date int64
		var trackID string
		var zoomLevel int
		var hasSpectrum bool

		if err := rows.Scan(&id, &doseRate, &date, &lon, &lat, &countRate, &trackID, &zoomLevel, &hasSpectrum); err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		if _, err := stmt.Exec(id, doseRate, date, lon, lat, countRate, trackID, zoomLevel, hasSpectrum); err != nil {
			return fmt.Errorf("insert: %w", err)
		}

		count++
		if count%batchSize == 0 {
			fmt.Printf("   Progress: %d rows migrated...\n", count)
		}
	}

	stat.RowsMigrated = int64(count)
	return rows.Err()
}

func migrateTracks(sqliteDB *sql.DB, tx *sql.Tx, stat *MigrationStats) error {
	rows, err := sqliteDB.Query("SELECT trackID, minLat, minLon, maxLat, maxLon, created, zoomLevel FROM tracks")
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	stmt, err := tx.Prepare("INSERT INTO tracks (trackID, minLat, minLon, maxLat, maxLon, created, zoomLevel) VALUES ($1, $2, $3, $4, $5, to_timestamp($6), $7) ON CONFLICT (trackID, zoomLevel) DO NOTHING")
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	count := 0
	for rows.Next() {
		var trackID string
		var minLat, minLon, maxLat, maxLon float64
		var created int64
		var zoomLevel int

		if err := rows.Scan(&trackID, &minLat, &minLon, &maxLat, &maxLon, &created, &zoomLevel); err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		if _, err := stmt.Exec(trackID, minLat, minLon, maxLat, maxLon, created, zoomLevel); err != nil {
			return fmt.Errorf("insert: %w", err)
		}

		count++
	}

	stat.RowsMigrated = int64(count)
	return rows.Err()
}

func migrateSpectra(sqliteDB *sql.DB, tx *sql.Tx, stat *MigrationStats) error {
	rows, err := sqliteDB.Query("SELECT id, marker_id, detector_type, device_serial, device_model, calibration, source_format, filename, raw_data, created_at FROM spectra ORDER BY id")
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	stmt, err := tx.Prepare("INSERT INTO spectra (id, marker_id, detector_type, device_serial, device_model, calibration, source_format, filename, raw_data, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, to_timestamp($10))")
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	count := 0
	for rows.Next() {
		var id, markerID int64
		var detectorType, deviceSerial, deviceModel, calibration, sourceFormat, filename sql.NullString
		var rawData []byte
		var createdAt int64

		if err := rows.Scan(&id, &markerID, &detectorType, &deviceSerial, &deviceModel, &calibration, &sourceFormat, &filename, &rawData, &createdAt); err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		if _, err := stmt.Exec(id, markerID, detectorType, deviceSerial, deviceModel, calibration, sourceFormat, filename, rawData, createdAt); err != nil {
			return fmt.Errorf("insert: %w", err)
		}

		count++
	}

	stat.RowsMigrated = int64(count)
	return rows.Err()
}

func migrateUploads(sqliteDB *sql.DB, tx *sql.Tx, stat *MigrationStats) error {
	rows, err := sqliteDB.Query("SELECT id, filename, file_type, track_id, file_size, upload_ip, created_at, source, source_id, source_url, user_id FROM uploads ORDER BY id")
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	stmt, err := tx.Prepare("INSERT INTO uploads (id, filename, file_type, track_id, file_size, upload_ip, created_at, source, source_id, source_url, user_id) VALUES ($1, $2, $3, $4, $5, $6, to_timestamp($7), $8, $9, $10, $11)")
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	count := 0
	for rows.Next() {
		var id int64
		var filename string
		var fileType, trackID, uploadIP sql.NullString
		var fileSize sql.NullInt64
		var createdAt int64
		var source, sourceID, sourceURL, userID sql.NullString

		if err := rows.Scan(&id, &filename, &fileType, &trackID, &fileSize, &uploadIP, &createdAt, &source, &sourceID, &sourceURL, &userID); err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		if _, err := stmt.Exec(id, filename, fileType, trackID, fileSize, uploadIP, createdAt, source, sourceID, sourceURL, userID); err != nil {
			return fmt.Errorf("insert: %w", err)
		}

		count++
	}

	stat.RowsMigrated = int64(count)
	return rows.Err()
}
