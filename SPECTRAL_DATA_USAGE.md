# Spectral Data Feature - Usage Guide

## Overview

The Chicha Isotope Map now supports gamma spectroscopy data from scintillator-based radiation detectors. This feature allows you to view detailed energy spectrum graphs for individual measurement points, enabling isotope identification and more precise radiation analysis.

## Supported Formats

- **RadiaCode .rctrk** files with embedded spectrum data
- **ANSI N42.42** XML format (standard nuclear instrumentation format)

## Supported Devices

- RadiaCode 101/102/103
- AtomFast
- Any scintillator-based detector outputting N42 or compatible formats

## Features

### 1. Visual Identification

Markers with spectral data are displayed with a distinctive **gold dashed border** (#FFD700), making them easy to spot on the map.

### 2. Interactive Spectrum Visualization

Click any marker with spectral data to open a popup containing:
- Standard radiation measurement data (dose rate, location, etc.)
- **Gamma spectrum graph** showing:
  - X-axis: Energy in keV (0-3000 keV typical range)
  - Y-axis: Counts (logarithmic scale)
  - Interactive tooltips showing energy and counts
  - Auto-loading on popup open

### 3. Data Export

Each spectrum can be downloaded in multiple formats:
- **JSON**: Full spectrum data with metadata
- **CSV**: Simple tabular format (Channel, Energy_keV, Counts)
- **N42**: Original N42 XML format (if available)

## API Endpoints

### Get Spectrum Data
```
GET /api/spectrum/{markerID}
```
Returns complete spectrum data including channels, calibration, and metadata.

**Response:**
```json
{
  "id": 123,
  "markerID": 456,
  "channels": [0, 2, 5, 10, ...],  // 1024 values
  "channelCount": 1024,
  "energyMinKeV": 0,
  "energyMaxKeV": 3000,
  "liveTimeSec": 60.0,
  "realTimeSec": 61.2,
  "deviceModel": "RadiaCode-102",
  "calibration": {
    "a": 0,
    "b": 2.93,
    "c": 0
  },
  "sourceFormat": "rctrk",
  "createdAt": 1701234567
}
```

### Download Spectrum
```
GET /api/spectrum/{markerID}/download?format=json|csv|n42
```
Downloads spectrum in specified format.

### Get Markers with Spectra
```
GET /api/markers/spectra?minLat=...&maxLat=...&minLon=...&maxLon=...
```
Returns all markers that have associated spectral data within the bounding box.

## File Upload

### Current Status
The spectrum parsers are implemented in `pkg/spectrum/`:
- `rctrk.go` - RadiaCode .rctrk parser
- `n42.go` - ANSI N42.42 XML parser
- `spectrum.go` - Common utilities

**Note**: File upload integration is pending. To add spectrum data currently:
1. Use the API endpoints directly
2. Manually insert spectrum data via database
3. Wait for upload handler integration (coming soon)

## Technical Details

### Database Schema

**Spectra Table:**
```sql
CREATE TABLE spectra (
  id              BIGSERIAL PRIMARY KEY,
  marker_id       BIGINT NOT NULL,
  channels        TEXT,              -- JSON array of 1024 counts
  channel_count   INTEGER DEFAULT 1024,
  energy_min_kev  DOUBLE PRECISION,
  energy_max_kev  DOUBLE PRECISION,
  live_time_sec   DOUBLE PRECISION,
  real_time_sec   DOUBLE PRECISION,
  device_model    TEXT,
  calibration     TEXT,              -- JSON: {a, b, c} coefficients
  source_format   TEXT,              -- "rctrk", "n42", etc.
  raw_data        BYTEA,             -- Original file bytes
  created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

**Markers Table Addition:**
```sql
ALTER TABLE markers ADD COLUMN has_spectrum BOOLEAN DEFAULT FALSE;
```

### Energy Calibration

Energy calibration uses polynomial coefficients:
```
Energy(keV) = A + B × channel + C × channel²
```

Where:
- **A**: Offset term (keV)
- **B**: Linear coefficient (keV/channel)
- **C**: Quadratic coefficient (keV/channel²)

Typical RadiaCode calibration for 1024 channels over 0-3000 keV:
```
A = 0
B = 2.93
C = 0
```

## Usage Examples

### View Spectrum Data
1. Navigate to the map
2. Look for markers with **gold dashed borders**
3. Click the marker to open the popup
4. The spectrum graph loads automatically
5. Hover over the graph to see energy/counts details

### Download Spectrum
1. Click a spectrum marker
2. In the popup, click **Download JSON** or **Download CSV**
3. File downloads automatically as `spectrum_{markerID}.{format}`

### API Integration
```javascript
// Fetch spectrum data
const response = await fetch('/api/spectrum/123');
const spectrum = await response.json();

// Calculate energy for channel 512
const cal = spectrum.calibration;
const energy = cal.a + cal.b * 512 + cal.c * 512 * 512;

// Find counts at 662 keV (Cs-137 peak)
const channel = Math.round((662 - cal.a) / cal.b);
const counts = spectrum.channels[channel];
```

## Future Enhancements

Potential additions for future versions:
- **Automatic upload integration** for .rctrk and N42 files
- **Isotope identification** with automatic peak labeling
- **Energy range filtering** to show only markers with activity in specific energy windows
- **Background subtraction** for cleaner spectra
- **Peak detection** with automatic annotation
- **Region of Interest (ROI)** analysis for specific isotopes
- **Spectrum comparison** between multiple measurements
- **3D visualization** showing spectrum evolution along tracks

## Troubleshooting

### Spectrum not loading
- Check browser console for errors
- Verify marker ID is valid
- Ensure spectrum data exists in database

### Chart not displaying
- Verify Chart.js library loaded (check `<script src="https://cdn.jsdelivr.net/npm/chart.js@4">`)
- Check for JavaScript errors in console
- Ensure canvas element exists in DOM

### Download not working
- Check API endpoint accessibility
- Verify spectrum data includes raw_data for N42 format
- Check browser download permissions

## For Developers

### Adding New Spectrum Formats

1. Create parser in `pkg/spectrum/{format}.go`
2. Implement parser function returning `[]database.Spectrum`
3. Add format handler in upload flow
4. Update API to support format in download endpoint

Example parser structure:
```go
package spectrum

func ParseNewFormat(data []byte) ([]database.Spectrum, []database.Marker, error) {
    // Parse format
    // Convert to Spectrum structs
    // Return spectra and associated markers
}
```

### Database Operations

See `pkg/database/spectrum.go` for:
- `InsertSpectrum(ctx, spectrum)` - Store spectrum
- `GetSpectrum(ctx, markerID)` - Retrieve spectrum
- `GetMarkersWithSpectra(ctx, bounds)` - Find spectrum markers
- `DeleteSpectrum(ctx, markerID)` - Remove spectrum

## Credits

Spectral data support implemented for the Chicha Isotope Map project.
Compatible with Safecast, RadiaCode, AtomFast, and standard N42 formats.

---

For questions or issues, please refer to the main project documentation or file an issue on the project repository.
