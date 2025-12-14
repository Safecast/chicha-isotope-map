package spectrum

import (
	"encoding/json"
	"fmt"
	"math"

	"chicha-isotope-map/pkg/database"
)

// ChannelToEnergy converts a channel number to energy (keV) using calibration coefficients.
// Formula: Energy = A + B*channel + C*channel^2
func ChannelToEnergy(channel int, cal *database.EnergyCalibration) float64 {
	if cal == nil {
		return 0
	}
	ch := float64(channel)
	return cal.A + cal.B*ch + cal.C*ch*ch
}

// EnergyToChannel converts energy (keV) to approximate channel number using calibration.
// This is the inverse of ChannelToEnergy, solving the quadratic equation.
func EnergyToChannel(energyKeV float64, cal *database.EnergyCalibration) int {
	if cal == nil {
		return 0
	}

	// For quadratic: c*x^2 + b*x + (a-E) = 0
	// Using quadratic formula: x = (-b + sqrt(b^2 - 4ac)) / 2c
	if cal.C != 0 {
		discriminant := cal.B*cal.B - 4*cal.C*(cal.A-energyKeV)
		if discriminant < 0 {
			return 0
		}
		return int((-cal.B + math.Sqrt(discriminant)) / (2 * cal.C))
	}

	// Linear calibration: b*x + a = E
	if cal.B != 0 {
		return int((energyKeV - cal.A) / cal.B)
	}

	return 0
}

// ChannelsToJSON serializes channel data to JSON for database storage.
func ChannelsToJSON(channels []int) (string, error) {
	if len(channels) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal(channels)
	if err != nil {
		return "", fmt.Errorf("marshal channels: %w", err)
	}
	return string(data), nil
}

// JSONToChannels deserializes channel data from JSON string.
func JSONToChannels(jsonData string) ([]int, error) {
	if jsonData == "" || jsonData == "[]" {
		return []int{}, nil
	}
	var channels []int
	if err := json.Unmarshal([]byte(jsonData), &channels); err != nil {
		return nil, fmt.Errorf("unmarshal channels: %w", err)
	}
	return channels, nil
}

// CalibrationToJSON serializes energy calibration to JSON for database storage.
func CalibrationToJSON(cal *database.EnergyCalibration) (string, error) {
	if cal == nil {
		return "", nil
	}
	data, err := json.Marshal(cal)
	if err != nil {
		return "", fmt.Errorf("marshal calibration: %w", err)
	}
	return string(data), nil
}

// JSONToCalibration deserializes energy calibration from JSON string.
func JSONToCalibration(jsonData string) (*database.EnergyCalibration, error) {
	if jsonData == "" {
		return nil, nil
	}
	var cal database.EnergyCalibration
	if err := json.Unmarshal([]byte(jsonData), &cal); err != nil {
		return nil, fmt.Errorf("unmarshal calibration: %w", err)
	}
	return &cal, nil
}

// Peak represents a detected peak in the spectrum.
type Peak struct {
	Channel int     // Channel number of the peak
	Energy  float64 // Energy in keV (if calibration is available)
	Counts  int     // Peak height (counts in the channel)
}

// DetectPeaks performs simple peak detection on spectrum data.
// A peak is defined as a local maximum that exceeds the threshold (as a fraction of max counts).
// This is a basic implementation - more sophisticated algorithms exist for real peak analysis.
func DetectPeaks(channels []int, threshold float64, cal *database.EnergyCalibration) []Peak {
	if len(channels) < 3 {
		return nil
	}

	// Find maximum counts for threshold calculation
	maxCounts := 0
	for _, c := range channels {
		if c > maxCounts {
			maxCounts = c
		}
	}

	if maxCounts == 0 {
		return nil
	}

	minCounts := int(float64(maxCounts) * threshold)
	var peaks []Peak

	// Scan for local maxima
	for i := 1; i < len(channels)-1; i++ {
		if channels[i] > channels[i-1] && channels[i] > channels[i+1] && channels[i] >= minCounts {
			peak := Peak{
				Channel: i,
				Counts:  channels[i],
			}
			if cal != nil {
				peak.Energy = ChannelToEnergy(i, cal)
			}
			peaks = append(peaks, peak)
		}
	}

	return peaks
}

// IntegrateEnergyRange calculates the total counts in a specified energy range.
// Returns the sum of counts in channels corresponding to energies between minKeV and maxKeV.
func IntegrateEnergyRange(channels []int, minKeV, maxKeV float64, cal *database.EnergyCalibration) int {
	if cal == nil || len(channels) == 0 {
		return 0
	}

	minChannel := EnergyToChannel(minKeV, cal)
	maxChannel := EnergyToChannel(maxKeV, cal)

	if minChannel < 0 {
		minChannel = 0
	}
	if maxChannel >= len(channels) {
		maxChannel = len(channels) - 1
	}

	total := 0
	for i := minChannel; i <= maxChannel; i++ {
		total += channels[i]
	}

	return total
}

// CalculateDoseRate estimates dose rate from spectrum using energy-dependent conversion.
// This is a simplified calculation - real dose rate calculations are more complex.
// Returns dose rate in µSv/h.
func CalculateDoseRate(channels []int, liveTimeSec float64, cal *database.EnergyCalibration) float64 {
	if len(channels) == 0 || liveTimeSec == 0 || cal == nil {
		return 0
	}

	// Simple approach: sum all counts and apply average conversion factor
	// Real implementation would use energy-dependent dose coefficients
	totalCounts := 0
	for _, c := range channels {
		totalCounts += c
	}

	// Average conversion factor (approximate, varies by detector and energy)
	// Typical value for gamma rays: ~0.01 µSv/h per CPS
	cps := float64(totalCounts) / liveTimeSec
	doseRate := cps * 0.01

	return doseRate
}
