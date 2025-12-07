package spectrum

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"chicha-isotope-map/pkg/database"
)

// N42 XML structures based on ANSI N42.42 standard
// Reference: https://www.nist.gov/document/annexbn42xml

// N42RadInstrumentData is the root element of an N42 document.
type N42RadInstrumentData struct {
	XMLName      xml.Name         `xml:"N42InstrumentData"`
	Measurements []N42Measurement `xml:"Measurement"`
}

// N42Measurement represents a single measurement in N42 format.
type N42Measurement struct {
	InstrumentInfo *N42InstrumentInfo `xml:"InstrumentInformation"`
	DetectorData   *N42DetectorData   `xml:"DetectorData"`
	StartTime      string             `xml:"StartTime"`
}

// N42DetectorData contains detector measurement data.
type N42DetectorData struct {
	StartTime          string                 `xml:"StartTime"`
	SampleRealTime     string                 `xml:"SampleRealTime"`
	DetectorMeasurement *N42DetectorMeasurement `xml:"DetectorMeasurement"`
}

// N42DetectorMeasurement contains spectrum measurements.
type N42DetectorMeasurement struct {
	SpectrumMeasurement *N42SpectrumMeasurement `xml:"SpectrumMeasurement"`
}

// N42SpectrumMeasurement contains one or more spectra.
type N42SpectrumMeasurement struct {
	Spectra []N42Spectrum `xml:"Spectrum"`
}

// N42Spectrum represents gamma spectrum data.
type N42Spectrum struct {
	RealTime        string                `xml:"RealTime"`
	LiveTime        string                `xml:"LiveTime"`
	ChannelData     string                `xml:"ChannelData"`
	Calibration     []N42Calibration      `xml:"Calibration"`
}

// N42Calibration represents energy calibration coefficients.
type N42Calibration struct {
	Type         string `xml:"Type,attr"`
	EnergyUnits  string `xml:"EnergyUnits,attr"`
	Equation     *N42Equation `xml:"Equation"`
}

// N42Equation represents calibration equation.
type N42Equation struct {
	Model        string `xml:"Model,attr"`
	Coefficients string `xml:"Coefficients"`
}

// N42Coordinates represents GPS location.
type N42Coordinates struct {
	Latitude  float64 `xml:"Latitude"`
	Longitude float64 `xml:"Longitude"`
}

// N42DoseRate represents dose rate information.
type N42DoseRate struct {
	Value float64 `xml:"Value"`
	Units string  `xml:"Units"`
}

// N42InstrumentInfo contains instrument metadata.
type N42InstrumentInfo struct {
	InstrumentType  string `xml:"InstrumentType"`
	Manufacturer    string `xml:"Manufacturer"`
	InstrumentModel string `xml:"InstrumentModel"`
	InstrumentID    string `xml:"InstrumentID"`
}

// ParseN42 parses ANSI N42.42 XML format and extracts spectrum data.
// Returns a slice of Spectrum objects with their associated marker data.
func ParseN42(data []byte) ([]database.Spectrum, []database.Marker, error) {
	var doc N42RadInstrumentData
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("parse n42 XML: %w", err)
	}

	var spectra []database.Spectrum
	var markers []database.Marker

	fmt.Printf("N42 Debug: Found %d measurements\n", len(doc.Measurements))

	// Process measurements
	for i, m := range doc.Measurements {
		deviceModel := "Unknown"
		if m.InstrumentInfo != nil && m.InstrumentInfo.InstrumentModel != "" {
			deviceModel = m.InstrumentInfo.InstrumentModel
		}

		// Extract spectra from detector data
		if m.DetectorData == nil || m.DetectorData.DetectorMeasurement == nil ||
			m.DetectorData.DetectorMeasurement.SpectrumMeasurement == nil {
			fmt.Printf("N42 Debug: Skipping measurement[%d]: no detector/spectrum data\n", i)
			continue
		}

		specMeas := m.DetectorData.DetectorMeasurement.SpectrumMeasurement
		fmt.Printf("N42 Debug: Processing measurement[%d] with %d spectra\n", i, len(specMeas.Spectra))

		for j, spec := range specMeas.Spectra {
			spectrum, marker, err := convertN42Spectrum(&spec, &m, deviceModel, data)
			if err != nil {
				fmt.Printf("N42 Debug: Skipping spectrum[%d][%d]: %v\n", i, j, err)
				continue
			}
			if spectrum != nil {
				spectra = append(spectra, *spectrum)
			}
			if marker != nil {
				markers = append(markers, *marker)
			}
		}
	}

	if len(spectra) == 0 {
		return nil, nil, fmt.Errorf("no valid spectra found in N42 file (found %d measurements)",
			len(doc.Measurements))
	}

	return spectra, markers, nil
}

// convertN42Spectrum converts an N42 spectrum to database format.
func convertN42Spectrum(spec *N42Spectrum, m *N42Measurement, deviceModel string, rawData []byte) (*database.Spectrum, *database.Marker, error) {
	// Parse channel data
	channels, err := parseChannelData(spec.ChannelData)
	if err != nil {
		return nil, nil, fmt.Errorf("parse channel data: %w", err)
	}

	if len(channels) == 0 {
		return nil, nil, fmt.Errorf("empty channel data")
	}

	// Parse time durations
	liveTime, err := parseN42Duration(spec.LiveTime)
	if err != nil {
		liveTime = 0
	}

	realTime, err := parseN42Duration(spec.RealTime)
	if err != nil {
		realTime = liveTime
	}

	// Parse energy calibration - look for Energy type calibration
	calibration := &database.EnergyCalibration{
		A: 0,
		B: 3000.0 / float64(len(channels)),
		C: 0,
	}

	for _, cal := range spec.Calibration {
		if cal.Type == "Energy" && cal.Equation != nil {
			if parsedCal, err := parseN42CalibrationEquation(cal.Equation.Coefficients); err == nil {
				calibration = parsedCal
				break
			}
		}
	}

	// Parse measurement time
	timestamp := time.Now().Unix()
	if m.DetectorData != nil && m.DetectorData.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, m.DetectorData.StartTime); err == nil {
			timestamp = t.Unix()
		}
	} else if m.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, m.StartTime); err == nil {
			timestamp = t.Unix()
		}
	}

	spectrum := &database.Spectrum{
		Channels:     channels,
		ChannelCount: len(channels),
		EnergyMinKeV: 0,
		EnergyMaxKeV: 3000.0, // Standard range, adjust if needed
		LiveTimeSec:  liveTime,
		RealTimeSec:  realTime,
		DeviceModel:  deviceModel,
		Calibration:  calibration,
		SourceFormat: "n42",
		RawData:      rawData,
		CreatedAt:    time.Now().Unix(),
	}

	// Create a basic marker (no coordinates in this N42 format)
	// User can add coordinates manually or we'll skip marker creation
	doseRate := 0.0
	if liveTime > 0 {
		// Estimate dose rate from spectrum
		doseRate = CalculateDoseRate(channels, liveTime, calibration)
	}

	countRate := 0.0
	if liveTime > 0 {
		countRate = float64(sumChannels(channels)) / liveTime
	}

	marker := &database.Marker{
		Lat:         0, // No coordinates in this N42 format
		Lon:         0,
		DoseRate:    doseRate,
		CountRate:   countRate,
		Date:        timestamp,
		HasSpectrum: true,
		Detector:    deviceModel,
	}

	return spectrum, marker, nil
}

// parseChannelData parses space-separated channel counts from N42 format.
func parseChannelData(data string) ([]int, error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil, fmt.Errorf("empty channel data")
	}

	fields := strings.Fields(data)
	channels := make([]int, 0, len(fields))

	for _, field := range fields {
		count, err := strconv.Atoi(field)
		if err != nil {
			return nil, fmt.Errorf("parse channel count %q: %w", field, err)
		}
		channels = append(channels, count)
	}

	return channels, nil
}

// parseN42Duration parses ISO 8601 duration (e.g., "PT60S" = 60 seconds).
func parseN42Duration(duration string) (float64, error) {
	duration = strings.TrimSpace(duration)
	if duration == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// Simple parser for PT{n}S format
	if strings.HasPrefix(duration, "PT") && strings.HasSuffix(duration, "S") {
		seconds := strings.TrimPrefix(duration, "PT")
		seconds = strings.TrimSuffix(seconds, "S")
		val, err := strconv.ParseFloat(seconds, 64)
		if err != nil {
			return 0, err
		}
		return val, nil
	}

	// Handle PT{n}M format (minutes)
	if strings.HasPrefix(duration, "PT") && strings.HasSuffix(duration, "M") {
		minutes := strings.TrimPrefix(duration, "PT")
		minutes = strings.TrimSuffix(minutes, "M")
		val, err := strconv.ParseFloat(minutes, 64)
		if err != nil {
			return 0, err
		}
		return val * 60, nil
	}

	return 0, fmt.Errorf("unsupported duration format: %s", duration)
}

// parseN42CalibrationEquation parses energy calibration coefficients from equation string.
func parseN42CalibrationEquation(coeffStr string) (*database.EnergyCalibration, error) {
	if coeffStr == "" {
		return nil, fmt.Errorf("no calibration coefficients")
	}

	fields := strings.Fields(coeffStr)
	if len(fields) < 2 {
		return nil, fmt.Errorf("insufficient calibration coefficients")
	}

	a, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return nil, err
	}

	b, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return nil, err
	}

	c := 0.0
	if len(fields) >= 3 {
		c, _ = strconv.ParseFloat(fields[2], 64)
	}

	return &database.EnergyCalibration{A: a, B: b, C: c}, nil
}
