package safecastfetcher

import (
	"context"
	"fmt"
	"time"

	"safecast-new-map/pkg/database"
)

const (
	// SourceTypeSafecastAPI is the source identifier for API-imported files
	SourceTypeSafecastAPI = "safecast-api"
)

// Fetcher handles periodic polling and importing from Safecast API
type Fetcher struct {
	client      *Client
	db          *database.Database
	dbType      string
	batchSize   int
	startDate   string
	importer    ImporterFunc
	logf        func(string, ...any)
}

// Config contains configuration for the fetcher
type Config struct {
	DB          *database.Database
	DBType      string
	Interval    time.Duration
	BatchSize   int
	StartDate   string
	Importer    ImporterFunc
	Logf        func(string, ...any)
}

// Start launches the background polling service
func Start(ctx context.Context, cfg Config) {
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}

	cfg.Logf("[safecast-fetcher] start: interval=%s batch=%d start_date=%s",
		cfg.Interval, cfg.BatchSize, cfg.StartDate)

	fetcher := &Fetcher{
		client:    NewClient(),
		db:        cfg.DB,
		dbType:    cfg.DBType,
		batchSize: cfg.BatchSize,
		startDate: cfg.StartDate,
		importer:  cfg.Importer,
		logf:      cfg.Logf,
	}

	// Launch background polling goroutine
	go fetcher.run(ctx, cfg.Interval)
}

// run is the main polling loop
func (f *Fetcher) run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Poll immediately on startup
	if err := f.poll(ctx); err != nil {
		f.logf("[safecast-fetcher] initial poll error: %v", err)
	}

	// Then poll on each tick
	for {
		select {
		case <-ctx.Done():
			f.logf("[safecast-fetcher] stopped")
			return
		case <-ticker.C:
			if err := f.poll(ctx); err != nil {
				f.logf("[safecast-fetcher] poll error: %v", err)
			}
		}
	}
}

// poll fetches and imports new approved files
func (f *Fetcher) poll(ctx context.Context) error {
	// Get the last imported Safecast ID
	lastID, err := f.db.GetLastImportedSafecastID(ctx, SourceTypeSafecastAPI)
	if err != nil {
		return fmt.Errorf("get last imported ID: %w", err)
	}

	f.logf("[safecast-fetcher] poll: checking for imports after ID %d", lastID)

	// Fetch approved imports from API with pagination
	allImports, err := f.fetchNewImports(ctx, lastID)
	if err != nil {
		return fmt.Errorf("fetch imports: %w", err)
	}

	f.logf("[safecast-fetcher] poll: found %d new approved imports", len(allImports))

	if len(allImports) == 0 {
		return nil
	}

	// Process each import
	imported := 0
	skipped := 0
	errors := 0

	for _, imp := range allImports {
		// Check context cancellation
		select {
		case <-ctx.Done():
			f.logf("[safecast-fetcher] poll cancelled: imported %d/%d, skipped %d, errors %d",
				imported, len(allImports), skipped, errors)
			return ctx.Err()
		default:
		}

		// Double-check if already imported
		exists, err := f.db.CheckImportExists(ctx, SourceTypeSafecastAPI, imp.ID)
		if err != nil {
			f.logf("[safecast-fetcher] import #%d: check failed: %v", imp.ID, err)
			errors++
			continue
		}
		if exists {
			f.logf("[safecast-fetcher] import #%d: already imported, skipping", imp.ID)
			skipped++
			continue
		}

		// Download log file
		f.logf("[safecast-fetcher] import #%d: downloading %s", imp.ID, imp.Name)
		content, filename, err := DownloadLogFile(ctx, imp.SourceURL)
		if err != nil {
			f.logf("[safecast-fetcher] import #%d: download failed: %v", imp.ID, err)
			errors++
			continue
		}

		// Import the file
		result, err := ImportSafecastFile(ctx, content, filename, imp.ID, imp.SourceURL, fmt.Sprintf("%d", imp.UserID), f.db, f.dbType, f.importer)
		if err != nil {
			f.logf("[safecast-fetcher] import #%d: import failed: %v", imp.ID, err)
			errors++
			continue
		}

		f.logf("[safecast-fetcher] import #%d: imported track %s with %d markers",
			imp.ID, result.TrackID, result.MarkerCount)
		imported++
	}

	f.logf("[safecast-fetcher] summary: imported %d/%d, skipped %d, errors %d",
		imported, len(allImports), skipped, errors)

	return nil
}

// fetchNewImports retrieves all new imports from the API with pagination
func (f *Fetcher) fetchNewImports(ctx context.Context, lastID int64) ([]SafecastImport, error) {
	var allImports []SafecastImport
	page := 1
	const perPageEstimate = 50 // Typical page size

	for {
		// Fetch page
		imports, err := f.client.FetchApprovedImports(ctx, f.startDate, page)
		if err != nil {
			return nil, fmt.Errorf("fetch page %d: %w", page, err)
		}

		// No more results
		if len(imports) == 0 {
			f.logf("[safecast-fetcher] page %d: no results, stopping", page)
			break
		}

		// Log what we got
		if len(imports) > 0 {
			f.logf("[safecast-fetcher] page %d: fetched %d imports (IDs %d-%d)",
				page, len(imports), imports[0].ID, imports[len(imports)-1].ID)
		}

		// Filter out already-processed imports
		newImports := 0
		for _, imp := range imports {
			if imp.ID > lastID {
				allImports = append(allImports, imp)
				newImports++
			}
		}

		f.logf("[safecast-fetcher] page %d: found %d new imports (> %d)", page, newImports, lastID)

		// Check if we've hit our batch size limit
		if f.batchSize > 0 && len(allImports) >= f.batchSize {
			allImports = allImports[:f.batchSize]
			break
		}

		// Move to next page
		page++

		// Safety: don't fetch more than 10 pages in one poll cycle
		if page > 10 {
			f.logf("[safecast-fetcher] warning: stopped at page 10, consider increasing poll interval")
			break
		}
	}

	return allImports, nil
}
