package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// InsertSpectrum stores a spectrum record linked to a marker in the database.
// The function handles JSON serialization of channels and calibration data,
// and automatically updates the marker's has_spectrum flag.
func (db *Database) InsertSpectrum(ctx context.Context, spectrum Spectrum) (int64, error) {
	// Serialize channels to JSON
	channelsJSON, err := json.Marshal(spectrum.Channels)
	if err != nil {
		return 0, fmt.Errorf("marshal channels: %w", err)
	}

	// Serialize calibration to JSON
	var calibrationJSON []byte
	if spectrum.Calibration != nil {
		calibrationJSON, err = json.Marshal(spectrum.Calibration)
		if err != nil {
			return 0, fmt.Errorf("marshal calibration: %w", err)
		}
	}

	// Determine created_at timestamp
	createdAt := spectrum.CreatedAt
	if createdAt == 0 {
		createdAt = time.Now().Unix()
	}

	var spectrumID int64

	// Use direct execution - simplified approach
	spectrumID, err = db.insertSpectrumSQL(ctx, db.DB, spectrum, channelsJSON, calibrationJSON, createdAt)
	if err != nil {
		return 0, err
	}

	// Update marker's has_spectrum flag
	if err := db.UpdateMarkerSpectrumFlag(ctx, spectrum.MarkerID, true); err != nil {
		// Log warning but don't fail the insert
		fmt.Printf("Warning: failed to update marker spectrum flag: %v\n", err)
	}

	return spectrumID, nil
}

// insertSpectrumSQL performs the actual SQL insert operation.
func (db *Database) insertSpectrumSQL(ctx context.Context, conn *sql.DB, spectrum Spectrum, channelsJSON, calibrationJSON []byte, createdAt int64) (int64, error) {
	var query string
	var args []interface{}

	switch db.Driver {
	case "pgx":
		query = `
			INSERT INTO spectra (marker_id, channels, channel_count, energy_min_kev, energy_max_kev,
			                     live_time_sec, real_time_sec, device_model, calibration,
			                     source_format, raw_data, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, to_timestamp($12))
			RETURNING id
		`
		args = []interface{}{
			spectrum.MarkerID, string(channelsJSON), spectrum.ChannelCount,
			spectrum.EnergyMinKeV, spectrum.EnergyMaxKeV, spectrum.LiveTimeSec,
			spectrum.RealTimeSec, spectrum.DeviceModel, string(calibrationJSON),
			spectrum.SourceFormat, spectrum.RawData, createdAt,
		}

		var id int64
		err := conn.QueryRowContext(ctx, query, args...).Scan(&id)
		return id, err

	case "sqlite", "chai", "duckdb":
		query = `
			INSERT INTO spectra (marker_id, channels, channel_count, energy_min_kev, energy_max_kev,
			                     live_time_sec, real_time_sec, device_model, calibration,
			                     source_format, raw_data, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		args = []interface{}{
			spectrum.MarkerID, string(channelsJSON), spectrum.ChannelCount,
			spectrum.EnergyMinKeV, spectrum.EnergyMaxKeV, spectrum.LiveTimeSec,
			spectrum.RealTimeSec, spectrum.DeviceModel, string(calibrationJSON),
			spectrum.SourceFormat, spectrum.RawData, createdAt,
		}

		result, err := conn.ExecContext(ctx, query, args...)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()

	case "clickhouse":
		// ClickHouse doesn't have auto-increment, generate ID manually
		id := time.Now().UnixNano()
		query = `
			INSERT INTO spectra (id, marker_id, channels, channel_count, energy_min_kev, energy_max_kev,
			                     live_time_sec, real_time_sec, device_model, calibration,
			                     source_format, raw_data, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now())
		`
		args = []interface{}{
			id, spectrum.MarkerID, string(channelsJSON), spectrum.ChannelCount,
			spectrum.EnergyMinKeV, spectrum.EnergyMaxKeV, spectrum.LiveTimeSec,
			spectrum.RealTimeSec, spectrum.DeviceModel, string(calibrationJSON),
			spectrum.SourceFormat, string(spectrum.RawData),
		}

		_, err := conn.ExecContext(ctx, query, args...)
		return id, err

	default:
		return 0, fmt.Errorf("unsupported database driver: %s", db.Driver)
	}
}

// GetSpectrum retrieves spectrum data for a specific marker by marker ID.
func (db *Database) GetSpectrum(ctx context.Context, markerID int64) (*Spectrum, error) {
	return db.getSpectrumSQL(ctx, db.DB, markerID)
}

// getSpectrumSQL performs the actual SQL query.
func (db *Database) getSpectrumSQL(ctx context.Context, conn *sql.DB, markerID int64) (*Spectrum, error) {
	query := `
		SELECT id, marker_id, channels, channel_count, energy_min_kev, energy_max_kev,
		       live_time_sec, real_time_sec, device_model, calibration,
		       source_format, raw_data, created_at
		FROM spectra
		WHERE marker_id = ?
		LIMIT 1
	`

	if db.Driver == "pgx" {
		query = `
			SELECT id, marker_id, channels, channel_count, energy_min_kev, energy_max_kev,
			       live_time_sec, real_time_sec, device_model, calibration,
			       source_format, raw_data, EXTRACT(EPOCH FROM created_at)::BIGINT
			FROM spectra
			WHERE marker_id = $1
			LIMIT 1
		`
	}

	var spectrum Spectrum
	var channelsJSON, calibrationJSON string
	var rawData []byte
	var createdAt sql.NullInt64

	err := conn.QueryRowContext(ctx, query, markerID).Scan(
		&spectrum.ID, &spectrum.MarkerID, &channelsJSON, &spectrum.ChannelCount,
		&spectrum.EnergyMinKeV, &spectrum.EnergyMaxKeV, &spectrum.LiveTimeSec,
		&spectrum.RealTimeSec, &spectrum.DeviceModel, &calibrationJSON,
		&spectrum.SourceFormat, &rawData, &createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No spectrum found
	}
	if err != nil {
		return nil, fmt.Errorf("query spectrum: %w", err)
	}

	// Deserialize channels
	if err := json.Unmarshal([]byte(channelsJSON), &spectrum.Channels); err != nil {
		return nil, fmt.Errorf("unmarshal channels: %w", err)
	}

	// Deserialize calibration
	if calibrationJSON != "" {
		var cal EnergyCalibration
		if err := json.Unmarshal([]byte(calibrationJSON), &cal); err != nil {
			return nil, fmt.Errorf("unmarshal calibration: %w", err)
		}
		spectrum.Calibration = &cal
	}

	spectrum.RawData = rawData
	if createdAt.Valid {
		spectrum.CreatedAt = createdAt.Int64
	}

	return &spectrum, nil
}

// GetMarkersWithSpectra returns all markers that have associated spectral data within a bounding box.
func (db *Database) GetMarkersWithSpectra(ctx context.Context, bounds Bounds) ([]Marker, error) {
	return db.getMarkersWithSpectraSQL(ctx, db.DB, bounds)
}

// getMarkersWithSpectraSQL performs the actual SQL query.
func (db *Database) getMarkersWithSpectraSQL(ctx context.Context, conn *sql.DB, bounds Bounds) ([]Marker, error) {
	query := `
		SELECT id, doseRate, date, lon, lat, countRate, zoom, speed, trackID,
		       altitude, detector, radiation, temperature, humidity, has_spectrum
		FROM markers
		WHERE has_spectrum = ?
		  AND lat BETWEEN ? AND ?
		  AND lon BETWEEN ? AND ?
		ORDER BY date DESC
		LIMIT 1000
	`

	args := []interface{}{
		true, bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon,
	}

	if db.Driver == "pgx" {
		query = `
			SELECT id, doseRate, date, lon, lat, countRate, zoom, speed, trackID,
			       altitude, detector, radiation, temperature, humidity, has_spectrum
			FROM markers
			WHERE has_spectrum = $1
			  AND lat BETWEEN $2 AND $3
			  AND lon BETWEEN $4 AND $5
			ORDER BY date DESC
			LIMIT 1000
		`
	}

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query markers with spectra: %w", err)
	}
	defer rows.Close()

	var markers []Marker
	for rows.Next() {
		var m Marker
		err := rows.Scan(
			&m.ID, &m.DoseRate, &m.Date, &m.Lon, &m.Lat, &m.CountRate,
			&m.Zoom, &m.Speed, &m.TrackID, &m.Altitude, &m.Detector,
			&m.Radiation, &m.Temperature, &m.Humidity, &m.HasSpectrum,
		)
		if err != nil {
			continue
		}
		markers = append(markers, m)
	}

	return markers, nil
}

// DeleteSpectrum removes spectrum data associated with a marker.
func (db *Database) DeleteSpectrum(ctx context.Context, markerID int64) error {
	return db.deleteSpectrumSQL(ctx, db.DB, markerID)
}

// deleteSpectrumSQL performs the actual SQL delete operation.
func (db *Database) deleteSpectrumSQL(ctx context.Context, conn *sql.DB, markerID int64) error {
	query := "DELETE FROM spectra WHERE marker_id = ?"
	if db.Driver == "pgx" {
		query = "DELETE FROM spectra WHERE marker_id = $1"
	}

	_, err := conn.ExecContext(ctx, query, markerID)
	if err != nil {
		return fmt.Errorf("delete spectrum: %w", err)
	}

	// Update marker's has_spectrum flag
	return db.UpdateMarkerSpectrumFlag(ctx, markerID, false)
}

// UpdateMarkerSpectrumFlag updates the has_spectrum flag for a marker.
func (db *Database) UpdateMarkerSpectrumFlag(ctx context.Context, markerID int64, hasSpectrum bool) error {
	return db.updateMarkerSpectrumFlagSQL(ctx, db.DB, markerID, hasSpectrum)
}

// updateMarkerSpectrumFlagSQL performs the actual SQL update.
func (db *Database) updateMarkerSpectrumFlagSQL(ctx context.Context, conn *sql.DB, markerID int64, hasSpectrum bool) error {
	query := "UPDATE markers SET has_spectrum = ? WHERE id = ?"
	args := []interface{}{hasSpectrum, markerID}

	if db.Driver == "pgx" {
		query = "UPDATE markers SET has_spectrum = $1 WHERE id = $2"
	}

	_, err := conn.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update marker spectrum flag: %w", err)
	}

	return nil
}
