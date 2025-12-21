package safecastfetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// SafecastImport represents a bGeigie import from api.safecast.org
type SafecastImport struct {
	ID                int64     `json:"id"`
	SourceURL         string    // Extracted from nested source.url
	UserID            int64     `json:"user_id"`
	Approved          bool      `json:"approved"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	MeasurementsCount int       `json:"measurements_count"`
	MD5Sum            string    `json:"md5sum"`
	Name              string    `json:"name"` // Original filename
	Status            string    `json:"status"`
}

// sourceWrapper is used to unmarshal the nested source.url structure
type sourceWrapper struct {
	URL string `json:"url"`
}

// safecastImportRaw is used for initial JSON unmarshaling
type safecastImportRaw struct {
	ID                int64         `json:"id"`
	Source            sourceWrapper `json:"source"`
	UserID            int64         `json:"user_id"`
	Approved          bool          `json:"approved"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
	MeasurementsCount int           `json:"measurements_count"`
	MD5Sum            string        `json:"md5sum"`
	Name              string        `json:"name"`
	Status            string        `json:"status"`
}

// Client handles communication with api.safecast.org
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Safecast API client
func NewClient() *Client {
	return &Client{
		baseURL: "https://api.safecast.org",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchApprovedImports queries the API for approved imports
// Parameters:
//   - uploadedAfter: ISO date string (YYYY-MM-DD) to get imports after this date (empty for no filter)
//   - page: pagination page number (1-indexed)
func (c *Client) FetchApprovedImports(ctx context.Context, uploadedAfter string, page int) ([]SafecastImport, error) {
	// Build URL: GET /bgeigie_imports.json?status=approved&page=N
	u, err := url.Parse(fmt.Sprintf("%s/bgeigie_imports.json", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	query := u.Query()
	query.Set("status", "approved")
	query.Set("order", "created_at desc")
	if uploadedAfter != "" {
		query.Set("uploaded_after", uploadedAfter)
	}
	if page > 0 {
		query.Set("page", fmt.Sprintf("%d", page))
	}
	u.RawQuery = query.Encode()

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse JSON response
	var rawImports []safecastImportRaw
	if err := json.NewDecoder(resp.Body).Decode(&rawImports); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to SafecastImport (flatten source.url)
	imports := make([]SafecastImport, 0, len(rawImports))
	for _, raw := range rawImports {
		imports = append(imports, SafecastImport{
			ID:                raw.ID,
			SourceURL:         raw.Source.URL,
			UserID:            raw.UserID,
			Approved:          raw.Approved,
			CreatedAt:         raw.CreatedAt,
			UpdatedAt:         raw.UpdatedAt,
			MeasurementsCount: raw.MeasurementsCount,
			MD5Sum:            raw.MD5Sum,
			Name:              raw.Name,
			Status:            raw.Status,
		})
	}

	return imports, nil
}
