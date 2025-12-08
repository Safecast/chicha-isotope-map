#!/usr/bin/env python3
"""
Insert a test spectrum at Mitsue onsen, Nara, Japan coordinates
"""

import sqlite3
import json
import time

# Mitsue onsen coordinates
LAT = 34.4883891
LON = 136.1659156
TRACK_ID = "TEST_MITSUE_ONSEN"

# Connect to database
conn = sqlite3.connect('database-8765.sqlite')
cursor = conn.cursor()

# Create test spectrum with Cs-137 and Ba-133 peaks
channels = [0] * 1024

# Add background
for i in range(len(channels)):
    if i < 50:
        channels[i] = 5 + i // 10
    else:
        channels[i] = 3

# Cs-137 peak at ~662 keV (channel 226)
cs137_ch = 226
channels[cs137_ch] = 1500
channels[cs137_ch-1] = 800
channels[cs137_ch+1] = 800
channels[cs137_ch-2] = 300
channels[cs137_ch+2] = 300
channels[cs137_ch-3] = 100
channels[cs137_ch+3] = 100

# Ba-133 peaks
ba133_peaks = [28, 94, 103, 122, 131]
for ch in ba133_peaks:
    channels[ch] = 400 + ch * 2
    if ch > 0:
        channels[ch-1] = 150
    if ch < len(channels) - 1:
        channels[ch+1] = 150

# Calculate stats
total_counts = sum(channels)
live_time = 300.0  # 5 minutes
real_time = 305.0
count_rate = total_counts / live_time
dose_rate = count_rate * 0.001  # Simplified conversion
timestamp = int(time.time())

print(f"Creating test spectrum at Mitsue onsen coordinates:")
print(f"  Location: {LAT}, {LON}")
print(f"  Total counts: {total_counts}")
print(f"  Count rate: {count_rate:.2f} CPS")
print(f"  Estimated dose rate: {dose_rate:.3f} µSv/h")

# Insert markers for all zoom levels
for zoom in range(21):
    cursor.execute('''
        INSERT INTO markers (lat, lon, doseRate, countRate, date, zoom, trackID, detector, has_spectrum, speed)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ''', (LAT, LON, dose_rate, count_rate, timestamp, zoom, TRACK_ID, "Test Spectrum Generator", 1, 0.0))

    marker_id = cursor.lastrowid
    print(f"Inserted marker at zoom {zoom} with ID {marker_id}")

    # Insert spectrum only for zoom 0
    if zoom == 0:
        # Create spectrum JSON
        calibration = {
            "a": 0.0,
            "b": 3000.0 / 1024.0,
            "c": 0.0
        }

        spectrum_data = {
            "id": 0,  # Will be auto-assigned
            "markerID": marker_id,
            "channels": channels,
            "channelCount": len(channels),
            "energyMinKeV": 0.0,
            "energyMaxKeV": 3000.0,
            "liveTimeSec": live_time,
            "realTimeSec": real_time,
            "deviceModel": "Test Spectrum Generator",
            "calibration": calibration,
            "sourceFormat": "test",
            "rawData": b"Test spectrum data",
            "createdAt": timestamp
        }

        cursor.execute('''
            INSERT INTO spectra (
                marker_id, channels, channel_count, energy_min_kev, energy_max_kev,
                live_time_sec, real_time_sec, device_model, calibration,
                source_format, raw_data, created_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ''', (
            marker_id,
            json.dumps(channels),
            len(channels),
            0.0,
            3000.0,
            live_time,
            real_time,
            "Test Spectrum Generator",
            json.dumps(calibration),
            "test",
            b"Test spectrum data",
            timestamp
        ))

        spectrum_id = cursor.lastrowid
        print(f"Inserted spectrum with ID {spectrum_id} for marker {marker_id}")

conn.commit()
conn.close()

print("\nTest spectrum successfully inserted!")
print(f"Open map at: http://localhost:8765/#15/{LAT}/{LON}")
