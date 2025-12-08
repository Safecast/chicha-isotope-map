// insert_test_spectrum.go
// Utility to insert a test spectrum at Mitsue onsen, Nara, Japan coordinates
// Run with: go run insert_test_spectrum.go

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"chicha-isotope-map/pkg/database"

	_ "modernc.org/sqlite"
)

func main() {
	// Mitsue onsen coordinates (Mitsue-mura, Uda-gun, Nara Prefecture, Japan)
	const (
		lat = 34.4883891
		lon = 136.1659156
	)

	// Open database
	dbPath := "database-8765.sqlite"
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()

	db := &database.Database{DB: sqlDB}
	ctx := context.Background()

	// Create a simple test spectrum with Cs-137 peak at 662 keV
	// This simulates a 1024-channel detector with 0-3000 keV range
	channels := make([]int, 1024)

	// Add some background counts
	for i := range channels {
		if i < 50 {
			channels[i] = 5 + i/10 // Low energy background
		} else {
			channels[i] = 3 // Flat background
		}
	}

	// Add Cs-137 peak at ~662 keV
	// Channel = Energy / (3000/1024) ≈ Energy / 2.93
	// 662 keV ≈ channel 226
	cs137Channel := 226
	channels[cs137Channel] = 1500    // Peak
	channels[cs137Channel-1] = 800   // Left shoulder
	channels[cs137Channel+1] = 800   // Right shoulder
	channels[cs137Channel-2] = 300
	channels[cs137Channel+2] = 300
	channels[cs137Channel-3] = 100
	channels[cs137Channel+3] = 100

	// Add Ba-133 peaks (81, 276, 303, 356, 384 keV)
	ba133Peaks := []int{28, 94, 103, 122, 131} // Approximate channels
	for _, ch := range ba133Peaks {
		if ch < len(channels) {
			channels[ch] = 400 + ch*2
			if ch > 0 {
				channels[ch-1] = 150
			}
			if ch < len(channels)-1 {
				channels[ch+1] = 150
			}
		}
	}

	// Energy calibration: E = a + b*channel + c*channel^2
	calibration := &database.EnergyCalibration{
		A: 0.0,                    // Offset
		B: 3000.0 / 1024.0,        // Linear coefficient (~2.93 keV/channel)
		C: 0.0,                    // No quadratic term
	}

	timestamp := time.Now().Unix()
	liveTime := 300.0  // 5 minutes
	realTime := 305.0  // 5 min 5 sec

	// Calculate dose rate (simplified)
	totalCounts := 0
	for _, count := range channels {
		totalCounts += count
	}
	countRate := float64(totalCounts) / liveTime
	doseRate := countRate * 0.001 // Simplified conversion to µSv/h

	fmt.Printf("Creating test spectrum at Mitsue onsen coordinates:\n")
	fmt.Printf("  Location: %.6f, %.6f\n", lat, lon)
	fmt.Printf("  Total counts: %d\n", totalCounts)
	fmt.Printf("  Count rate: %.2f CPS\n", countRate)
	fmt.Printf("  Estimated dose rate: %.3f µSv/h\n", doseRate)

	// Create markers for all zoom levels
	trackID := "TEST_MITSUE_ONSEN"
	var markers []database.Marker

	for zoom := 0; zoom <= 20; zoom++ {
		marker := database.Marker{
			Lat:         lat,
			Lon:         lon,
			DoseRate:    doseRate,
			CountRate:   countRate,
			Date:        timestamp,
			Zoom:        zoom,
			TrackID:     trackID,
			Detector:    "Test Spectrum Generator",
			HasSpectrum: true,
		}
		markers = append(markers, marker)
	}

	// Insert all markers in a transaction
	tx, err := sqlDB.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	progress := make(chan database.MarkerBatchProgress, 100)
	go func() {
		for range progress {
			// Discard progress messages
		}
	}()

	err = db.InsertMarkersBulk(ctx, tx, markers, "sqlite", 1000, progress, database.WorkloadUserUpload)
	close(progress)
	if err != nil {
		log.Fatalf("Failed to insert markers: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	fmt.Printf("Inserted %d markers across all zoom levels\n", len(markers))

	// Query back the marker ID for zoom 0 to link the spectrum
	query := "SELECT id FROM markers WHERE lat = ? AND lon = ? AND date = ? AND zoom = 0"
	var markerID int64
	err = sqlDB.QueryRow(query, lat, lon, timestamp).Scan(&markerID)
	if err != nil {
		log.Fatalf("Failed to query marker ID: %v", err)
	}

	// Insert spectrum for zoom 0 marker
	spectrum := database.Spectrum{
		MarkerID:     markerID,
		Channels:     channels,
		ChannelCount: len(channels),
		EnergyMinKeV: 0.0,
		EnergyMaxKeV: 3000.0,
		LiveTimeSec:  liveTime,
		RealTimeSec:  realTime,
		DeviceModel:  "Test Spectrum Generator",
		Calibration:  calibration,
		SourceFormat: "test",
		RawData:      []byte("Test spectrum data"),
		CreatedAt:    timestamp,
	}

	spectrumID, err := db.InsertSpectrum(ctx, spectrum)
	if err != nil {
		log.Fatalf("Failed to insert spectrum: %v", err)
	}

	fmt.Printf("Inserted spectrum with ID %d for marker %d\n", spectrumID, markerID)

	fmt.Println("\nTest spectrum successfully inserted!")
	fmt.Printf("Open map at: http://localhost:8765/#15/%.6f/%.6f\n", lat, lon)
}
