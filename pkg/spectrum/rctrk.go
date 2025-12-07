package spectrum

import (
	"encoding/json"
	"fmt"
	"time"

	"chicha-isotope-map/pkg/database"
)

// RCTRKSpectrum represents a spectrum in the RadiaCode .rctrk format.
// RadiaCode devices can save spectrum data alongside track points.
type RCTRKSpectrum struct {
	Channels    []int              `json:"channels"`    // Array of counts per channel (typically 1024)
	Duration    float64            `json:"duration"`    // Measurement duration in seconds
	RealTime    float64            `json:"realTime"`    // Real time including dead time
	Calibration RCTRKCalibration   `json:"calibration"` // Energy calibration
	Timestamp   int64              `json:"timestamp"`   // UNIX timestamp
	Coordinates *RCTRKCoordinates  `json:"coordinates"` // GPS coordinates
}

// RCTRKCalibration represents energy calibration in RadiaCode format.
type RCTRKCalibration struct {
	A float64 `json:"a"` // Offset
	B float64 `json:"b"` // Linear
	C float64 `json:"c"` // Quadratic
}

// RCTRKCoordinates represents GPS coordinates in RadiaCode format.
type RCTRKCoordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// RCTRKMarkerWithSpectrum extends the standard marker format with optional spectrum.
type RCTRKMarkerWithSpectrum struct {
	Lat       float64        `json:"lat"`
	Lon       float64        `json:"lon"`
	DoseRate  float64        `json:"doseRate"`
	CountRate float64        `json:"countRate"`
	Date      int64          `json:"date"`
	Spectrum  *RCTRKSpectrum `json:"spectrum,omitempty"` // Optional spectrum data
}

// RCTRKFile represents a complete .rctrk file that may contain spectra.
type RCTRKFile struct {
	ID       string                     `json:"id"`
	Markers  []RCTRKMarkerWithSpectrum  `json:"markers"`
	Spectra  []RCTRKSpectrum            `json:"spectra,omitempty"` // Standalone spectra array
}

// ParseRCTRK parses a RadiaCode .rctrk file and extracts spectrum data.
// Returns a slice of Spectrum objects with their associated marker coordinates.
func ParseRCTRK(data []byte) ([]database.Spectrum, []database.Marker, error) {
	var file RCTRKFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, nil, fmt.Errorf("parse rctrk JSON: %w", err)
	}

	var spectra []database.Spectrum
	var markers []database.Marker

	// Parse markers with embedded spectra
	for _, m := range file.Markers {
		marker := database.Marker{
			Lat:         m.Lat,
			Lon:         m.Lon,
			DoseRate:    m.DoseRate,
			CountRate:   m.CountRate,
			Date:        m.Date,
			HasSpectrum: m.Spectrum != nil,
			Detector:    "RadiaCode",
		}
		markers = append(markers, marker)

		if m.Spectrum != nil {
			spectrum, err := convertRCTRKSpectrum(m.Spectrum)
			if err != nil {
				continue // Skip invalid spectra
			}
			spectra = append(spectra, spectrum)
		}
	}

	// Parse standalone spectra array
	for _, s := range file.Spectra {
		spectrum, err := convertRCTRKSpectrum(&s)
		if err != nil {
			continue
		}

		// Create a marker for standalone spectrum
		if s.Coordinates != nil {
			marker := database.Marker{
				Lat:         s.Coordinates.Lat,
				Lon:         s.Coordinates.Lon,
				Date:        s.Timestamp,
				HasSpectrum: true,
				Detector:    "RadiaCode",
			}

			// Calculate dose rate from spectrum if not provided
			if marker.DoseRate == 0 && len(s.Channels) > 0 {
				cal := &database.EnergyCalibration{
					A: s.Calibration.A,
					B: s.Calibration.B,
					C: s.Calibration.C,
				}
				marker.DoseRate = CalculateDoseRate(s.Channels, s.Duration, cal)
				marker.CountRate = float64(sumChannels(s.Channels)) / s.Duration
			}

			markers = append(markers, marker)
		}

		spectra = append(spectra, spectrum)
	}

	return spectra, markers, nil
}

// convertRCTRKSpectrum converts a RadiaCode spectrum to the database format.
func convertRCTRKSpectrum(rcSpectrum *RCTRKSpectrum) (database.Spectrum, error) {
	if rcSpectrum == nil {
		return database.Spectrum{}, fmt.Errorf("nil spectrum")
	}

	if len(rcSpectrum.Channels) == 0 {
		return database.Spectrum{}, fmt.Errorf("empty channels")
	}

	channelCount := len(rcSpectrum.Channels)

	// RadiaCode typically uses 0-3000 keV range for 1024 channels
	energyMax := 3000.0
	if channelCount != 1024 {
		// Adjust energy range proportionally
		energyMax = 3000.0 * float64(channelCount) / 1024.0
	}

	calibration := &database.EnergyCalibration{
		A: rcSpectrum.Calibration.A,
		B: rcSpectrum.Calibration.B,
		C: rcSpectrum.Calibration.C,
	}

	// If no calibration provided, use linear default
	if calibration.A == 0 && calibration.B == 0 && calibration.C == 0 {
		calibration.A = 0
		calibration.B = energyMax / float64(channelCount)
		calibration.C = 0
	}

	spectrum := database.Spectrum{
		Channels:     rcSpectrum.Channels,
		ChannelCount: channelCount,
		EnergyMinKeV: 0,
		EnergyMaxKeV: energyMax,
		LiveTimeSec:  rcSpectrum.Duration,
		RealTimeSec:  rcSpectrum.RealTime,
		DeviceModel:  "RadiaCode",
		Calibration:  calibration,
		SourceFormat: "rctrk",
		CreatedAt:    time.Now().Unix(),
	}

	return spectrum, nil
}

// sumChannels calculates the total counts across all channels.
func sumChannels(channels []int) int {
	total := 0
	for _, c := range channels {
		total += c
	}
	return total
}
