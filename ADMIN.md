# 🔐 Admin Panel Documentation

The admin panel provides powerful tools for managing your radiation map instance, including bulk track operations, upload tracking, and database management.

---

## 🚀 Quick Start

### Enabling Admin Access

To enable the admin panel, start the server with the `-admin-password` flag:

```bash
./chicha-isotope-map -admin-password YOUR_SECURE_PASSWORD
```

**Important Security Notes:**
- Choose a strong, unique password
- The password is checked server-side on every admin request
- Admin pages are protected - unauthorized access returns to the main map
- For production deployments, always use HTTPS with `-domain` flag

### Accessing the Admin Panel

Once enabled, access admin features at (include password in URL):
- **All Tracks:** `http://localhost:8765/api/admin/tracks?password=YOUR_PASSWORD`
- **Uploads:** `http://localhost:8765/api/admin/uploads?password=YOUR_PASSWORD`

**Important:** The password must be included as a URL parameter. Replace `YOUR_PASSWORD` with your actual admin password. There is no separate login form - authentication is via the URL query parameter.

---

## 📊 All Tracks Page

**URL:** `/api/admin/tracks?password=YOUR_PASSWORD`

The All Tracks page provides a comprehensive overview of all radiation tracks stored in your database, with powerful bulk management capabilities.

### Features

#### Track Overview
- **Track listing** with pagination (50 tracks per page)
- **Real-time statistics:**
  - Total number of tracks
  - Total markers (data points)
  - Database size
- **Track details** for each entry:
  - Track ID (unique identifier)
  - Point count (number of markers)
  - Date range (first to last measurement)
  - Bounding box coordinates (geographic coverage)

#### Bulk Operations
- **Select All** checkbox to quickly select all tracks on current page
- **Individual checkboxes** for selective deletion
- **Delete Selected** button (only appears when tracks are selected)
- **Visual feedback** - delete button updates dynamically based on selection

#### Track Actions
Each track row provides:
- **View on Map** - Jump directly to the track visualization
- **Delete** - Remove individual track with confirmation
- **Selection checkbox** - Include in bulk operations

### Usage Examples

**View a specific track:**
1. Navigate to `/api/admin/tracks?password=YOUR_PASSWORD`
2. Find the track in the list
3. Click "View on Map" to see it visualized

**Delete multiple tracks:**
1. Check the boxes next to tracks you want to remove
2. Click "Delete Selected"
3. Confirm the deletion

**Delete all visible tracks:**
1. Click "Select All" checkbox in the header
2. Click "Delete Selected"
3. Confirm the mass deletion

---

## 📁 Uploads Page

**URL:** `/api/admin/uploads?password=YOUR_PASSWORD`

The Uploads page tracks all file uploads to your radiation map, providing visibility into data sources and enabling bulk cleanup operations.

### Features

#### Upload Tracking
Every file upload is automatically logged with:
- **Filename** - Original name of uploaded file
- **File Type** - Format (`.kml`, `.kmz`, `.json`, `.rctrk`, `.csv`, `.gpx`, `.n42`, `.spe`, etc.)
- **Track ID** - Associated track identifier
- **File Size** - Upload size in human-readable format (KB, MB, GB)
- **Upload IP** - Source IP address for audit trail
- **Upload Date** - Timestamp of upload

#### Bulk Operations
- **Select All** functionality across all visible uploads
- **Individual selection** checkboxes
- **Delete Selected** - Remove multiple upload records and associated tracks at once
- **Smart deletion** - Removes tracks, markers, spectra, and upload records together

#### Database Schema

The uploads table structure:
```sql
CREATE TABLE uploads (
    id SERIAL PRIMARY KEY,
    filename TEXT NOT NULL,
    file_type TEXT,
    track_id TEXT NOT NULL,
    file_size BIGINT,
    upload_ip TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Usage Examples

**Review recent uploads:**
1. Navigate to `/api/admin/uploads?password=YOUR_PASSWORD`
2. View the chronological list (most recent first)
3. Check file types, sizes, and sources

**Clean up test uploads:**
1. Select uploads you want to remove
2. Click "Delete Selected"
3. Confirm - this removes both the upload records AND associated track data

**Audit trail:**
- Check upload IP addresses to understand data sources
- Review file types to ensure proper format handling
- Monitor upload patterns over time

---

## 🎨 User Interface

### Theme Support
Both admin pages support light and dark themes that automatically sync with your system preferences:
- **Dark theme** - Comfortable for extended use
- **Light theme** - Clear visibility in bright environments
- **Automatic switching** - Respects `prefers-color-scheme`

### Navigation
- **Back to Map** button - Return to main radiation map
- **Clear breadcrumbs** - Always know where you are
- **Responsive design** - Works on desktop and mobile devices

### Visual Feedback
- **Dynamic button states** - Delete buttons enable/disable based on selection
- **Loading indicators** - Visual feedback during operations
- **Confirmation dialogs** - Prevent accidental deletions
- **Success/error messages** - Clear operation results

---

## 🔒 Security Considerations

### Password Protection
- Admin password is required for all admin operations
- Password is transmitted with each request (use HTTPS in production)
- No session cookies - simple stateless authentication
- Failed authentication redirects to main map

### Best Practices
1. **Use HTTPS in production:**
   ```bash
   ./chicha-isotope-map -domain your-domain.org -admin-password STRONG_PASSWORD
   ```

2. **Choose a strong password:**
   - Minimum 12 characters
   - Mix of letters, numbers, symbols
   - Avoid common words or patterns

3. **Limit admin access:**
   - Use firewall rules to restrict admin page access
   - Consider VPN or IP whitelisting for sensitive deployments

4. **Regular backups:**
   - Export data regularly via `/api/json/weekly.tgz`
   - Test restore procedures
   - Keep backups secure and encrypted

### Audit Trail
The uploads table provides basic audit capabilities:
- Track WHO uploaded (via IP address)
- Track WHAT was uploaded (filename, type, size)
- Track WHEN uploads occurred (created_at timestamp)

---

## 🛠️ Advanced Usage

### Database Backends

Admin features work seamlessly across all supported databases:

- **PostgreSQL (pgx)** - Recommended for production, best concurrent access
  ```bash
  ./chicha-isotope-map -db-type pgx \
    -db-conn "postgres://user:pass@host:5432/database" \
    -admin-password STRONG_PASSWORD
  ```

- **DuckDB** - Fast embedded database for single-user instances
  ```bash
  ./chicha-isotope-map -db-type duckdb \
    -db-path /path/to/database.duckdb \
    -admin-password STRONG_PASSWORD
  ```

- **SQLite** - Default, good for personal use
  ```bash
  ./chicha-isotope-map -admin-password STRONG_PASSWORD
  ```

### Combined with Other Features

**Admin + Realtime Safecast data:**
```bash
./chicha-isotope-map \
  -safecast-realtime \
  -admin-password ADMIN_PASS \
  -port 8765
```

**Admin + Custom domain with HTTPS:**
```bash
./chicha-isotope-map \
  -domain maps.example.org \
  -admin-password ADMIN_PASS
```

**Admin + Import existing data:**
```bash
./chicha-isotope-map \
  -import-tgz-url https://pelora.org/api/json/weekly.tgz \
  -admin-password ADMIN_PASS
```

### API Integration

While the admin panel provides a web interface, you can also manage data programmatically:

**Delete a track via command line:**
```bash
curl -X POST "http://localhost:8765/api/admin/delete" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "password=YOUR_PASSWORD&trackID=TRACK_ID"
```

**Note:** Always use HTTPS for API calls in production to protect the admin password.

---

## 📝 Spectrum File Support

The admin system includes special handling for spectrum files:

### Supported Formats
- **N42 files** (`.n42`) - ANSI N42.42 standard format
- **SPE files** (`.spe`) - Spectroscopy data format

### GPS Coordinate Handling
When uploading spectrum files without GPS coordinates:
- User is prompted to enter coordinates manually
- Map view stays at current position (doesn't jump to 0,0)
- Zoom level set to 18 for precise location viewing
- Upload is tracked with spectrum-specific metadata

### Spectrum Data Storage
Each spectrum upload creates:
- **Upload record** - In uploads table with file metadata
- **Marker record** - In markers table with GPS coordinates
- **Spectrum record** - In spectra table with channel data, calibration, device info
- **Association** - Marker's `has_spectrum` flag set to true

---

## 🐛 Troubleshooting

### "Unauthorized" redirect
**Problem:** Admin pages redirect to main map
**Solution:** Check that `-admin-password` flag is set when starting the server

### Delete operations not working
**Problem:** Clicking delete doesn't remove tracks
**Solution:**
- Verify admin password is correct
- Check browser console for JavaScript errors
- Ensure database has write permissions

### Upload tracking not appearing
**Problem:** Uploads work but don't appear in admin panel
**Solution:**
- Ensure server was started with upload tracking support (version with pkg/database/uploads.go)
- Check that database schema includes uploads table
- Verify uploads are completing successfully

### Multiple server instances
**Problem:** Changes not appearing after rebuild
**Solution:**
```bash
# Kill all instances on port 8765
pkill -9 -f "chicha-isotope-map.*8765"

# Verify all killed
ps aux | grep chicha-isotope-map

# Start fresh instance
./chicha-isotope-map -admin-password YOUR_PASSWORD -port 8765
```

---

## 💡 Tips and Tricks

1. **Regular cleanup:** Use the uploads page to identify and remove test uploads periodically

2. **Bulk operations:** When cleaning up large numbers of tracks, use "Select All" on All Tracks page

3. **Backup before bulk delete:** Always export your data before performing mass deletions:
   ```bash
   wget http://localhost:8765/api/json/weekly.tgz
   ```

4. **Monitor disk usage:** Check database size on All Tracks page to plan storage

5. **Track naming:** Consider implementing track naming conventions based on upload source

6. **Geographic filtering:** View tracks on map first to verify geographic coverage before deletion

---

## 🔄 Future Enhancements

Potential admin features under consideration:
- User management and role-based access control
- Track merging and splitting tools
- Bulk export with filtering
- Advanced search and filtering
- Statistics and analytics dashboard
- Automated cleanup rules
- Email notifications for uploads
- API key management

---

## 📚 Related Documentation

- [README.md](README.md) - General usage and quick start
- [DUCKDB_PERFORMANCE.md](doc/DUCKDB_PERFORMANCE.md) - Database optimization
- [API Documentation](#) - API endpoints and integration

---

## 🤝 Contributing

Found a bug or have a feature request for the admin panel?

- Open an issue on GitHub
- Submit a pull request with improvements
- Share your admin workflow tips with the community

The admin panel is designed to make radiation map management accessible to everyone. Your feedback helps us improve!
