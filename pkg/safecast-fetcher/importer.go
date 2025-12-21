package safecastfetcher

import (
	"context"
	"fmt"

	"safecast-new-map/pkg/database"
)

// ImporterFunc is a function type that handles importing a bGeigie file
// This function is expected to be provided by the main application
// It should parse the file, store markers, and record the upload
type ImporterFunc func(
	ctx context.Context,
	fileContent []byte,
	filename string,
	safecastImportID int64,
	sourceURL string,
	userID string,
	db *database.Database,
	dbType string,
) (trackID string, markerCount int, err error)

// DefaultImporter provides access to the default import functionality
// This will be set by the main application to avoid circular dependencies
var DefaultImporter ImporterFunc

// ImportResult contains the results of an import operation
type ImportResult struct {
	TrackID      string
	MarkerCount  int
	Filename     string
	ImportID     int64
	Error        error
}

// ImportSafecastFile imports a downloaded log file using the provided importer function
func ImportSafecastFile(
	ctx context.Context,
	fileContent []byte,
	filename string,
	safecastImportID int64,
	sourceURL string,
	userID string,
	db *database.Database,
	dbType string,
	importer ImporterFunc,
) (ImportResult, error) {
	result := ImportResult{
		Filename: filename,
		ImportID: safecastImportID,
	}

	// Use provided importer or default
	importFunc := importer
	if importFunc == nil {
		importFunc = DefaultImporter
	}
	if importFunc == nil {
		return result, fmt.Errorf("no importer function available")
	}

	// Execute import
	trackID, markerCount, err := importFunc(ctx, fileContent, filename, safecastImportID, sourceURL, userID, db, dbType)
	if err != nil {
		result.Error = err
		return result, fmt.Errorf("import failed: %w", err)
	}

	result.TrackID = trackID
	result.MarkerCount = markerCount

	return result, nil
}
