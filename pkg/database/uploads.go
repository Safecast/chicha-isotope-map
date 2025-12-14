package database

import (
	"context"
	"fmt"
	"time"
)

// Upload represents a file upload record for tracking purposes.
type Upload struct {
	ID        int64  `json:"id"`
	Filename  string `json:"filename"`
	FileType  string `json:"fileType"`
	TrackID   string `json:"trackID"`
	FileSize  int64  `json:"fileSize"`
	UploadIP  string `json:"uploadIP"`
	CreatedAt int64  `json:"createdAt"`
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
			INSERT INTO uploads (filename, file_type, track_id, file_size, upload_ip, created_at)
			VALUES ($1, $2, $3, $4, $5, to_timestamp($6))
			RETURNING id
		`
		args = []interface{}{
			upload.Filename, upload.FileType, upload.TrackID,
			upload.FileSize, upload.UploadIP, createdAt,
		}
		err := db.DB.QueryRowContext(ctx, query, args...).Scan(&id)
		return id, err

	case "duckdb":
		query = `
			INSERT INTO uploads (filename, file_type, track_id, file_size, upload_ip, created_at)
			VALUES ($1, $2, $3, $4, $5, to_timestamp($6))
			RETURNING id
		`
		args = []interface{}{
			upload.Filename, upload.FileType, upload.TrackID,
			upload.FileSize, upload.UploadIP, createdAt,
		}
		err := db.DB.QueryRowContext(ctx, query, args...).Scan(&id)
		return id, err

	case "sqlite", "chai":
		query = `
			INSERT INTO uploads (filename, file_type, track_id, file_size, upload_ip, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`
		args = []interface{}{
			upload.Filename, upload.FileType, upload.TrackID,
			upload.FileSize, upload.UploadIP, createdAt,
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

// GetUploads retrieves all upload records, ordered by most recent first.
func (db *Database) GetUploads(ctx context.Context, limit int) ([]Upload, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, filename, file_type, track_id, file_size, upload_ip, created_at
		FROM uploads
		ORDER BY created_at DESC
		LIMIT ?
	`
	args := []interface{}{limit}

	if db.Driver == "pgx" || db.Driver == "duckdb" {
		query = `
			SELECT id, filename, file_type, track_id, file_size, upload_ip,
			       EXTRACT(EPOCH FROM created_at)::BIGINT
			FROM uploads
			ORDER BY created_at DESC
			LIMIT $1
		`
	}

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query uploads: %w", err)
	}
	defer rows.Close()

	var uploads []Upload
	for rows.Next() {
		var u Upload
		err := rows.Scan(
			&u.ID, &u.Filename, &u.FileType, &u.TrackID,
			&u.FileSize, &u.UploadIP, &u.CreatedAt,
		)
		if err != nil {
			continue
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
