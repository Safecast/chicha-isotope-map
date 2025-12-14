package spectrum

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"chicha-isotope-map/pkg/database"
)

// SPEFile represents parsed SPE format data
// SPE is a text-based format commonly used for gamma spectroscopy
// Reference: IAEA SPE format specification
type SPEFile struct {
	SpecID       string                    // Spectrum identifier
	SpecRem      []string                  // Remarks/comments
	DateMea      string                    // Measurement date/time
	MeasTim      []float64                 // Measurement timing [live_time, real_time]
	Data         []int                     // Channel counts
	MCACalib     *database.EnergyCalibration // Energy calibration coefficients
	EnerFit      []float64                 // Energy fitting parameters
	DeviceModel  string                    // Detector model extracted from remarks
}

// ParseSPE parses IAEA SPE format and extracts spectrum data.
// Returns a slice of Spectrum objects with their associated marker data.
func ParseSPE(data []byte) ([]database.Spectrum, []database.Marker, error) {
	spe, err := parseSPEFile(data)
	if err != nil {
		return nil, nil, fmt.Errorf("parse SPE file: %w", err)
	}

	// Convert to database format
	spectrum, marker, err := convertSPEToSpectrum(spe, data)
	if err != nil {
		return nil, nil, fmt.Errorf("convert SPE to spectrum: %w", err)
	}

	spectra := []database.Spectrum{*spectrum}
	markers := []database.Marker{*marker}

	return spectra, markers, nil
}

// parseSPEFile parses the SPE file format into an SPEFile structure
func parseSPEFile(data []byte) (*SPEFile, error) {
	spe := &SPEFile{
		SpecRem: make([]string, 0),
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	var currentSection string
	var dataStart, dataEnd int
	var dataLines []string // Lines from $DATA section
	var calLines []string  // Lines from $MCA_CAL section

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for section markers
		if strings.HasPrefix(line, "$") {
			currentSection = line
			continue
		}

		// Process section data
		switch currentSection {
		case "$SPEC_ID:":
			spe.SpecID = line

		case "$SPEC_REM:":
			spe.SpecRem = append(spe.SpecRem, line)
			// Try to extract device model from remarks
			if strings.Contains(strings.ToLower(line), "detector") ||
				strings.Contains(strings.ToLower(line), "device") ||
				strings.Contains(strings.ToLower(line), "model") {
				spe.DeviceModel = line
			}

		case "$DATE_MEA:":
			spe.DateMea = line

		case "$MEAS_TIM:":
			// Parse measurement timing (typically two values: live_time real_time)
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				liveTime, err1 := strconv.ParseFloat(fields[0], 64)
				realTime, err2 := strconv.ParseFloat(fields[1], 64)
				if err1 == nil && err2 == nil {
					spe.MeasTim = []float64{liveTime, realTime}
				}
			} else if len(fields) == 1 {
				// Only one time value provided, use it for both
				timeVal, err := strconv.ParseFloat(fields[0], 64)
				if err == nil {
					spe.MeasTim = []float64{timeVal, timeVal}
				}
			}

		case "$DATA:":
			// First line after $DATA: contains start and end channel numbers
			if dataStart == 0 && dataEnd == 0 {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					dataStart, _ = strconv.Atoi(fields[0])
					dataEnd, _ = strconv.Atoi(fields[1])
				}
			} else {
				// Subsequent lines contain channel counts
				dataLines = append(dataLines, line)
			}

		case "$MCA_CAL:":
			// First line is number of coefficients
			// Following lines are the coefficients
			calLines = append(calLines, line)

		case "$ENER_FIT:":
			// Energy fitting parameters (not currently used)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan SPE file: %w", err)
	}

	// Parse accumulated data sections
	if err := parseSPEDataSection(spe, dataLines, dataStart, dataEnd); err != nil {
		return nil, err
	}

	if err := parseSPECalibration(spe, calLines); err != nil {
		// Non-fatal: use default calibration if parsing fails
		fmt.Printf("SPE Debug: Could not parse calibration, using defaults: %v\n", err)
	}

	// Validate we got spectrum data
	if len(spe.Data) == 0 {
		return nil, fmt.Errorf("no spectrum data found in SPE file")
	}

	return spe, nil
}

// parseSPEDataSection parses the $DATA section
func parseSPEDataSection(spe *SPEFile, lines []string, start, end int) error {
	if start == 0 && end == 0 {
		return fmt.Errorf("no $DATA section found")
	}

	expectedChannels := end - start + 1
	spe.Data = make([]int, 0, expectedChannels)

	// Re-parse data lines for channel counts
	scanner := bufio.NewScanner(bytes.NewReader([]byte(strings.Join(lines, "\n"))))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Skip the start/end line
		fields := strings.Fields(line)
		if len(fields) == 2 {
			// Check if this looks like the start/end declaration
			if val1, err1 := strconv.Atoi(fields[0]); err1 == nil {
				if val2, err2 := strconv.Atoi(fields[1]); err2 == nil {
					if val1 == start && val2 == end {
						continue // Skip this line
					}
				}
			}
		}

		// Parse channel count
		count, err := strconv.Atoi(line)
		if err != nil {
			return fmt.Errorf("parse channel count %q: %w", line, err)
		}
		spe.Data = append(spe.Data, count)
	}

	if len(spe.Data) == 0 {
		return fmt.Errorf("no channel data parsed")
	}

	return nil
}

// parseSPECalibration parses the $MCA_CAL section
func parseSPECalibration(spe *SPEFile, lines []string) error {
	if len(lines) == 0 {
		return fmt.Errorf("no calibration data")
	}

	// First line is number of coefficients
	coeffs := make([]float64, 0)

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if i == 0 {
			// First line is number of coefficients (we just skip it)
			continue
		} else {
			// Coefficient values
			val, err := strconv.ParseFloat(line, 64)
			if err != nil {
				continue
			}
			coeffs = append(coeffs, val)
		}
	}

	// Need at least 2 coefficients for linear calibration (a, b)
	if len(coeffs) < 2 {
		return fmt.Errorf("insufficient calibration coefficients: got %d, need at least 2", len(coeffs))
	}

	spe.MCACalib = &database.EnergyCalibration{
		A: coeffs[0], // Offset
		B: coeffs[1], // Linear
		C: 0,         // Quadratic (default to 0 if not present)
	}

	if len(coeffs) >= 3 {
		spe.MCACalib.C = coeffs[2]
	}

	return nil
}

// convertSPEToSpectrum converts an SPEFile to database format
func convertSPEToSpectrum(spe *SPEFile, rawData []byte) (*database.Spectrum, *database.Marker, error) {
	if len(spe.Data) == 0 {
		return nil, nil, fmt.Errorf("empty spectrum data")
	}

	// Extract timing information
	liveTime := 0.0
	realTime := 0.0
	if len(spe.MeasTim) >= 2 {
		liveTime = spe.MeasTim[0]
		realTime = spe.MeasTim[1]
	} else if len(spe.MeasTim) == 1 {
		liveTime = spe.MeasTim[0]
		realTime = spe.MeasTim[0]
	}

	// Set up energy calibration
	calibration := spe.MCACalib
	if calibration == nil {
		// Default calibration: assume 0-3000 keV range
		calibration = &database.EnergyCalibration{
			A: 0,
			B: 3000.0 / float64(len(spe.Data)),
			C: 0,
		}
	}

	// Calculate energy range
	energyMin := calibration.A
	lastChannel := float64(len(spe.Data) - 1)
	energyMax := calibration.A + calibration.B*lastChannel + calibration.C*lastChannel*lastChannel

	// Determine device model
	deviceModel := spe.DeviceModel
	if deviceModel == "" {
		deviceModel = "Unknown SPE Device"
	}

	// Parse measurement date
	timestamp := time.Now().Unix()
	if spe.DateMea != "" {
		// Try common date formats
		formats := []string{
			"01/02/2006 15:04:05", // MM/DD/YYYY HH:MM:SS
			"02/01/2006 15:04:05", // DD/MM/YYYY HH:MM:SS
			"2006-01-02 15:04:05", // YYYY-MM-DD HH:MM:SS
			time.RFC3339,
		}
		for _, format := range formats {
			if t, err := time.Parse(format, spe.DateMea); err == nil {
				timestamp = t.Unix()
				break
			}
		}
	}

	spectrum := &database.Spectrum{
		Channels:     spe.Data,
		ChannelCount: len(spe.Data),
		EnergyMinKeV: energyMin,
		EnergyMaxKeV: energyMax,
		LiveTimeSec:  liveTime,
		RealTimeSec:  realTime,
		DeviceModel:  deviceModel,
		Calibration:  calibration,
		SourceFormat: "spe",
		RawData:      rawData,
		CreatedAt:    time.Now().Unix(),
	}

	// Calculate dose rate and count rate
	doseRate := 0.0
	if liveTime > 0 {
		doseRate = CalculateDoseRate(spe.Data, liveTime, calibration)
	}

	countRate := 0.0
	if liveTime > 0 {
		countRate = float64(sumChannels(spe.Data)) / liveTime
	}

	marker := &database.Marker{
		Lat:         0, // Will be set by caller
		Lon:         0, // Will be set by caller
		DoseRate:    doseRate,
		CountRate:   countRate,
		Date:        timestamp,
		HasSpectrum: true,
		Detector:    deviceModel,
	}

	return spectrum, marker, nil
}
