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
			                     source_format, filename, raw_data, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, to_timestamp($13))
			RETURNING id
		`
		args = []interface{}{
			spectrum.MarkerID, string(channelsJSON), spectrum.ChannelCount,
			spectrum.EnergyMinKeV, spectrum.EnergyMaxKeV, spectrum.LiveTimeSec,
			spectrum.RealTimeSec, spectrum.DeviceModel, string(calibrationJSON),
			spectrum.SourceFormat, spectrum.Filename, spectrum.RawData, createdAt,
		}

		var id int64
		err := conn.QueryRowContext(ctx, query, args...).Scan(&id)
		return id, err

	case "duckdb":
		query = `
			INSERT INTO spectra (marker_id, channels, channel_count, energy_min_kev, energy_max_kev,
			                     live_time_sec, real_time_sec, device_model, calibration,
			                     source_format, filename, raw_data, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, to_timestamp($13))
		`
		args = []interface{}{
			spectrum.MarkerID, string(channelsJSON), spectrum.ChannelCount,
			spectrum.EnergyMinKeV, spectrum.EnergyMaxKeV, spectrum.LiveTimeSec,
			spectrum.RealTimeSec, spectrum.DeviceModel, string(calibrationJSON),
			spectrum.SourceFormat, spectrum.Filename, spectrum.RawData, createdAt,
		}

		result, err := conn.ExecContext(ctx, query, args...)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()

	case "sqlite", "chai":
		query = `
			INSERT INTO spectra (marker_id, channels, channel_count, energy_min_kev, energy_max_kev,
			                     live_time_sec, real_time_sec, device_model, calibration,
			                     source_format, filename, raw_data, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		args = []interface{}{
			spectrum.MarkerID, string(channelsJSON), spectrum.ChannelCount,
			spectrum.EnergyMinKeV, spectrum.EnergyMaxKeV, spectrum.LiveTimeSec,
			spectrum.RealTimeSec, spectrum.DeviceModel, string(calibrationJSON),
			spectrum.SourceFormat, spectrum.Filename, spectrum.RawData, createdAt,
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
			                     source_format, filename, raw_data, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now())
		`
		args = []interface{}{
			id, spectrum.MarkerID, string(channelsJSON), spectrum.ChannelCount,
			spectrum.EnergyMinKeV, spectrum.EnergyMaxKeV, spectrum.LiveTimeSec,
			spectrum.RealTimeSec, spectrum.DeviceModel, string(calibrationJSON),
			spectrum.SourceFormat, spectrum.Filename, string(spectrum.RawData),
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
// It searches for spectrum by marker_id first, then falls back to finding spectrum
// for any marker at the same location/time (handles multi-zoom duplicates).
func (db *Database) getSpectrumSQL(ctx context.Context, conn *sql.DB, markerID int64) (*Spectrum, error) {
	// Try direct marker_id lookup first
	query := `
		SELECT s.id, s.marker_id, s.channels, s.channel_count, s.energy_min_kev, s.energy_max_kev,
		       s.live_time_sec, s.real_time_sec, s.device_model, s.calibration,
		       s.source_format, s.filename, s.raw_data, s.created_at
		FROM spectra s
		WHERE s.marker_id = ?
		LIMIT 1
	`
	args := []interface{}{markerID}

	if db.Driver == "pgx" || db.Driver == "duckdb" {
		placeholder := "$1"
		query = `
			SELECT s.id, s.marker_id, s.channels, s.channel_count, s.energy_min_kev, s.energy_max_kev,
			       s.live_time_sec, s.real_time_sec, s.device_model, s.calibration,
			       s.source_format, s.filename, s.raw_data, EXTRACT(EPOCH FROM s.created_at)::BIGINT
			FROM spectra s
			WHERE s.marker_id = ` + placeholder + `
			LIMIT 1
		`
	}

	var spectrum Spectrum
	var channelsJSON, calibrationJSON string
	var createdAt int64

	err := conn.QueryRowContext(ctx, query, args...).Scan(
		&spectrum.ID, &spectrum.MarkerID, &channelsJSON, &spectrum.ChannelCount,
		&spectrum.EnergyMinKeV, &spectrum.EnergyMaxKeV, &spectrum.LiveTimeSec,
		&spectrum.RealTimeSec, &spectrum.DeviceModel, &calibrationJSON,
		&spectrum.SourceFormat, &spectrum.Filename, &spectrum.RawData, &createdAt,
	)

	// If not found by marker_id, try finding by location/time
	if err == sql.ErrNoRows {
		// Get marker coordinates to search for spectrum at same location/time
		var lat, lon float64
		var date int64
		markerQuery := "SELECT lat, lon, date FROM markers WHERE id = ?"
		markerArgs := []interface{}{markerID}
		if db.Driver == "pgx" || db.Driver == "duckdb" {
			markerQuery = "SELECT lat, lon, date FROM markers WHERE id = $1"
		}

		err = conn.QueryRowContext(ctx, markerQuery, markerArgs...).Scan(&lat, &lon, &date)
		if err != nil {
			return nil, fmt.Errorf("marker not found: %w", err)
		}

		// Find spectrum for any marker at this location/time
		query = `
			SELECT s.id, s.marker_id, s.channels, s.channel_count, s.energy_min_kev, s.energy_max_kev,
			       s.live_time_sec, s.real_time_sec, s.device_model, s.calibration,
			       s.source_format, s.filename, s.raw_data, s.created_at
			FROM spectra s
			JOIN markers m ON s.marker_id = m.id
			WHERE m.lat = ? AND m.lon = ? AND m.date = ?
			LIMIT 1
		`
		args = []interface{}{lat, lon, date}

		if db.Driver == "pgx" || db.Driver == "duckdb" {
			query = `
				SELECT s.id, s.marker_id, s.channels, s.channel_count, s.energy_min_kev, s.energy_max_kev,
				       s.live_time_sec, s.real_time_sec, s.device_model, s.calibration,
				       s.source_format, s.filename, s.raw_data, EXTRACT(EPOCH FROM s.created_at)::BIGINT
				FROM spectra s
				JOIN markers m ON s.marker_id = m.id
				WHERE m.lat = $1 AND m.lon = $2 AND m.date = $3
				LIMIT 1
			`
		}

		err = conn.QueryRowContext(ctx, query, args...).Scan(
			&spectrum.ID, &spectrum.MarkerID, &channelsJSON, &spectrum.ChannelCount,
			&spectrum.EnergyMinKeV, &spectrum.EnergyMaxKeV, &spectrum.LiveTimeSec,
			&spectrum.RealTimeSec, &spectrum.DeviceModel, &calibrationJSON,
			&spectrum.SourceFormat, &spectrum.Filename, &spectrum.RawData, &createdAt,
		)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no spectrum found")
		}
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

	spectrum.CreatedAt = createdAt

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

	if db.Driver == "pgx" || db.Driver == "duckdb" {
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
	if db.Driver == "pgx" || db.Driver == "duckdb" {
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

	if db.Driver == "pgx" || db.Driver == "duckdb" {
		query = "UPDATE markers SET has_spectrum = $1 WHERE id = $2"
	}

	_, err := conn.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update marker spectrum flag: %w", err)
	}

	return nil
}
