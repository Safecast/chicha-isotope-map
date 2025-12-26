package database

import (
	"context"
	"fmt"
	"time"
)

// Upload represents a file upload record for tracking purposes.
type Upload struct {
	ID            int64  `json:"id"`
	Filename      string `json:"filename"`
	FileType      string `json:"fileType"`
	TrackID       string `json:"trackID"`
	FileSize      int64  `json:"fileSize"`
	UploadIP      string `json:"uploadIP"`
	CreatedAt     int64  `json:"createdAt"`
	RecordingDate int64  `json:"recordingDate,omitempty"` // Earliest marker date for this track
	Source        string `json:"source,omitempty"`        // Import source (e.g., "safecast-api", "user-upload")
	SourceID      string `json:"sourceID,omitempty"`      // External reference ID (e.g., Safecast import ID)
	SourceURL     string `json:"sourceURL,omitempty"`     // Source file URL (e.g., S3 URL)
	UserID        string `json:"userID,omitempty"`        // User ID from source (e.g., Safecast user ID)
	Username      string `json:"username,omitempty"`      // Username from source (fetched from API)
}

// InsertUpload records a file upload in the uploads table.
func (db *Database) InsertUpload(ctx context.Context, upload Upload) (int64, error) {
	createdAt := upload.CreatedAt
	if createdAt == 0 {
		createdAt = time.Now().Unix()
	}

	var query string
	var args []interface{}
	var id int64

	switch db.Driver {
	case "pgx":
		query = `
			INSERT INTO uploads (filename, file_type, track_id, file_size, upload_ip, created_at, recording_date, source, source_id, source_url, user_id, username)
			VALUES ($1, $2, $3, $4, $5, to_timestamp($6), to_timestamp($7), $8, $9, $10, $11, $12)
			RETURNING id
		`
		recordingDateArg := upload.RecordingDate
		if recordingDateArg == 0 {
			recordingDateArg = createdAt
		}
		args = []interface{}{
			upload.Filename, upload.FileType, upload.TrackID,
			upload.FileSize, upload.UploadIP, createdAt, recordingDateArg,
			upload.Source, upload.SourceID, upload.SourceURL, upload.UserID, upload.Username,
		}
		err := db.DB.QueryRowContext(ctx, query, args...).Scan(&id)
		return id, err

	case "duckdb":
		query = `
			INSERT INTO uploads (filename, file_type, track_id, file_size, upload_ip, created_at, recording_date, source, source_id, source_url, user_id, username)
			VALUES ($1, $2, $3, $4, $5, to_timestamp($6), to_timestamp($7), $8, $9, $10, $11, $12)
			RETURNING id
		`
		recordingDateArg := upload.RecordingDate
		if recordingDateArg == 0 {
			recordingDateArg = createdAt
		}
		args = []interface{}{
			upload.Filename, upload.FileType, upload.TrackID,
			upload.FileSize, upload.UploadIP, createdAt, recordingDateArg,
			upload.Source, upload.SourceID, upload.SourceURL, upload.UserID, upload.Username,
		}
		err := db.DB.QueryRowContext(ctx, query, args...).Scan(&id)
		return id, err

	case "sqlite", "chai":
		query = `
			INSERT INTO uploads (filename, file_type, track_id, file_size, upload_ip, created_at, recording_date, source, source_id, source_url, user_id, username)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		recordingDateArg := upload.RecordingDate
		if recordingDateArg == 0 {
			recordingDateArg = createdAt
		}
		args = []interface{}{
			upload.Filename, upload.FileType, upload.TrackID,
			upload.FileSize, upload.UploadIP, createdAt, recordingDateArg,
			upload.Source, upload.SourceID, upload.SourceURL, upload.UserID, upload.Username,
		}
		result, err := db.DB.ExecContext(ctx, query, args...)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()

	default:
		return 0, fmt.Errorf("unsupported database driver: %s", db.Driver)
	}
}

// GetUploads retrieves upload records, ordered by most recent first.
// If userID is not empty, only returns uploads from that user.
func (db *Database) GetUploads(ctx context.Context, limit int, userID string) ([]Upload, error) {
	return db.GetUploadsPaginated(ctx, limit, 0, userID, "")
}

func (db *Database) GetUploadsPaginated(ctx context.Context, limit int, offset int, userID string, search string) ([]Upload, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var query string
	var args []interface{}
	var whereConditions []string
	paramCount := 0

	// Build WHERE conditions
	if userID != "" {
		paramCount++
		if db.Driver == "pgx" || db.Driver == "duckdb" {
			whereConditions = append(whereConditions, fmt.Sprintf("u.user_id = $%d", paramCount))
		} else {
			whereConditions = append(whereConditions, "u.user_id = ?")
		}
	}

	if search != "" {
		paramCount++
		if db.Driver == "pgx" || db.Driver == "duckdb" {
			// PostgreSQL: use ILIKE for case-insensitive search, also search numeric fields by converting to text
			whereConditions = append(whereConditions, fmt.Sprintf(
				"(u.track_id ILIKE $%d OR u.filename ILIKE $%d OR COALESCE(u.user_id, '') ILIKE $%d OR COALESCE(u.source, '') ILIKE $%d OR COALESCE(u.source_id, '') ILIKE $%d OR CAST(u.id AS TEXT) ILIKE $%d)",
				paramCount, paramCount, paramCount, paramCount, paramCount, paramCount))
		} else {
			// SQLite: use LIKE (case-insensitive by default), also search numeric fields by converting to text
			whereConditions = append(whereConditions,
				"(u.track_id LIKE ? OR u.filename LIKE ? OR COALESCE(u.user_id, '') LIKE ? OR COALESCE(u.source, '') LIKE ? OR COALESCE(u.source_id, '') LIKE ? OR CAST(u.id AS TEXT) LIKE ?)")
		}
	}

	// Build the base query
	baseSelect := `
		SELECT u.id, u.filename, u.file_type, u.track_id, u.file_size, u.upload_ip,`

	if db.Driver == "pgx" || db.Driver == "duckdb" {
		baseSelect += `
		       EXTRACT(EPOCH FROM u.created_at)::BIGINT,
		       COALESCE(EXTRACT(EPOCH FROM u.recording_date)::BIGINT, 0) as recording_date,
		       u.source, u.source_id, u.source_url, u.user_id, u.username
		FROM uploads u`
	} else {
		baseSelect += `
		       u.created_at,
		       COALESCE(u.recording_date, 0) as recording_date,
		       u.source, u.source_id, u.source_url, u.user_id, u.username
		FROM uploads u`
	}

	// Add WHERE clause if we have conditions
	if len(whereConditions) > 0 {
		query = baseSelect + "\nWHERE " + whereConditions[0]
		for i := 1; i < len(whereConditions); i++ {
			query += " AND " + whereConditions[i]
		}
	} else {
		query = baseSelect
	}

	query += "\nORDER BY u.created_at DESC\n"

	// Add LIMIT and OFFSET with correct parameter numbering
	if db.Driver == "pgx" || db.Driver == "duckdb" {
		paramCount++
		limitParam := paramCount
		paramCount++
		offsetParam := paramCount
		query += fmt.Sprintf("LIMIT $%d OFFSET $%d", limitParam, offsetParam)
	} else {
		query += "LIMIT ? OFFSET ?"
	}

	// Build args array
	if userID != "" {
		args = append(args, userID)
	}
	if search != "" {
		searchPattern := "%" + search + "%"
		if db.Driver == "pgx" || db.Driver == "duckdb" {
			args = append(args, searchPattern)
		} else {
			// SQLite needs the pattern repeated 6 times (for each field)
			for i := 0; i < 6; i++ {
				args = append(args, searchPattern)
			}
		}
	}
	args = append(args, limit, offset)

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query uploads: %w", err)
	}
	defer rows.Close()

	var uploads []Upload
	for rows.Next() {
		var u Upload
		var source, sourceID, sourceURL, userID, username *string
		err := rows.Scan(
			&u.ID, &u.Filename, &u.FileType, &u.TrackID,
			&u.FileSize, &u.UploadIP, &u.CreatedAt, &u.RecordingDate,
			&source, &sourceID, &sourceURL, &userID, &username,
		)
		if err != nil {
			continue
		}
		if source != nil {
			u.Source = *source
		}
		if sourceID != nil {
			u.SourceID = *sourceID
		}
		if sourceURL != nil {
			u.SourceURL = *sourceURL
		}
		if userID != nil {
			u.UserID = *userID
		}
		if username != nil {
			u.Username = *username
		}
		uploads = append(uploads, u)
	}

	return uploads, nil
}

// DeleteTrack removes all data associated with a track ID.
// This includes markers, spectra, and upload records.
func (db *Database) DeleteTrack(ctx context.Context, trackID string) error {
	// Delete from spectra first (via CASCADE from markers)
	// Then delete markers, tracks, and upload records

	queries := []string{
		"DELETE FROM markers WHERE trackID = ?",
		"DELETE FROM tracks WHERE trackID = ?",
		"DELETE FROM uploads WHERE track_id = ?",
	}

	if db.Driver == "pgx" || db.Driver == "duckdb" {
		queries = []string{
			"DELETE FROM markers WHERE trackID = $1",
			"DELETE FROM tracks WHERE trackID = $1",
			"DELETE FROM uploads WHERE track_id = $1",
		}
	}

	for _, query := range queries {
		_, err := db.DB.ExecContext(ctx, query, trackID)
		if err != nil {
			return fmt.Errorf("delete track data: %w", err)
		}
	}

	return nil
}

// CheckImportExists returns true if a Safecast import ID has already been imported.
func (db *Database) CheckImportExists(ctx context.Context, sourceType string, importID int64) (bool, error) {
	var query string
	var args []interface{}

	sourceIDStr := fmt.Sprintf("%d", importID)

	switch db.Driver {
	case "pgx", "duckdb":
		query = `SELECT COUNT(*) FROM uploads WHERE source = $1 AND source_id = $2`
		args = []interface{}{sourceType, sourceIDStr}
	case "sqlite", "chai":
		query = `SELECT COUNT(*) FROM uploads WHERE source = ? AND source_id = ?`
		args = []interface{}{sourceType, sourceIDStr}
	default:
		return false, fmt.Errorf("unsupported database driver: %s", db.Driver)
	}

	var count int
	err := db.DB.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check import exists: %w", err)
	}

	return count > 0, nil
}

// GetLastImportedSafecastID returns the highest Safecast import ID that has been processed.
func (db *Database) GetLastImportedSafecastID(ctx context.Context, sourceType string) (int64, error) {
	var query string
	var args []interface{}

	switch db.Driver {
	case "pgx", "duckdb":
		// PostgreSQL and DuckDB: CAST to INTEGER
		query = `
			SELECT COALESCE(MAX(CAST(source_id AS INTEGER)), 0)
			FROM uploads
			WHERE source = $1 AND source_id IS NOT NULL AND source_id != ''
		`
		args = []interface{}{sourceType}
	case "sqlite", "chai":
		// SQLite: CAST to INTEGER
		query = `
			SELECT COALESCE(MAX(CAST(source_id AS INTEGER)), 0)
			FROM uploads
			WHERE source = ? AND source_id IS NOT NULL AND source_id != ''
		`
		args = []interface{}{sourceType}
	default:
		return 0, fmt.Errorf("unsupported database driver: %s", db.Driver)
	}

	var lastID int64
	err := db.DB.QueryRowContext(ctx, query, args...).Scan(&lastID)
	if err != nil {
		return 0, fmt.Errorf("get last imported safecast ID: %w", err)
	}

	return lastID, nil
}

// CountUploads returns the total number of uploads, optionally filtered by user_id and search term
func (db *Database) CountUploads(ctx context.Context, userID string, search string) (int, error) {
	var query string
	var args []interface{}
	var whereConditions []string
	paramCount := 0

	// Build WHERE conditions
	if userID != "" {
		paramCount++
		if db.Driver == "pgx" || db.Driver == "duckdb" {
			whereConditions = append(whereConditions, fmt.Sprintf("user_id = $%d", paramCount))
		} else {
			whereConditions = append(whereConditions, "user_id = ?")
		}
		args = append(args, userID)
	}

	if search != "" {
		paramCount++
		searchPattern := "%" + search + "%"
		if db.Driver == "pgx" || db.Driver == "duckdb" {
			whereConditions = append(whereConditions, fmt.Sprintf(
				"(track_id ILIKE $%d OR filename ILIKE $%d OR COALESCE(user_id, '') ILIKE $%d OR COALESCE(source, '') ILIKE $%d OR COALESCE(source_id, '') ILIKE $%d OR CAST(id AS TEXT) ILIKE $%d)",
				paramCount, paramCount, paramCount, paramCount, paramCount, paramCount))
			args = append(args, searchPattern)
		} else {
			whereConditions = append(whereConditions,
				"(track_id LIKE ? OR filename LIKE ? OR COALESCE(user_id, '') LIKE ? OR COALESCE(source, '') LIKE ? OR COALESCE(source_id, '') LIKE ? OR CAST(id AS TEXT) LIKE ?)")
			for i := 0; i < 6; i++ {
				args = append(args, searchPattern)
			}
		}
	}

	// Build query
	query = "SELECT COUNT(*) FROM uploads"
	if len(whereConditions) > 0 {
		query += " WHERE " + whereConditions[0]
		for i := 1; i < len(whereConditions); i++ {
			query += " AND " + whereConditions[i]
		}
	}

	var count int
	err := db.DB.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count uploads: %w", err)
	}

	return count, nil
}
