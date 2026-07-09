// Package wallhaven provides functionality for interacting with the Wallhaven API
package wallhaven

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/constants"
	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/errors"
)

// WallpaperID is a string representing a wallpaper
type WallpaperID string

// Q is used to hold the Q params for various fulltext options that the WH Search supports
type Q struct {
	Tags        []string
	ExcludeTags []string
	UserName    string
	TagID       int
	Type        string // Type is one of png/jpg
	Like        WallpaperID
}

func (q Q) toQuery() url.Values {
	var sb strings.Builder

	for _, tag := range q.Tags {
		sb.WriteString("+")
		sb.WriteString(tag)
	}
	for _, etag := range q.ExcludeTags {
		sb.WriteString("-")
		sb.WriteString(etag)
	}
	if len(q.UserName) > 0 {
		sb.WriteString("@")
		sb.WriteString(q.UserName)
	}
	if len(q.Type) > 0 {
		sb.WriteString("type:")
		sb.WriteString(q.Type)
	}
	out := url.Values{}
	if val := sb.String(); len(val) > 0 {
		out.Set("q", val)
	}
	return out
}

// Search provides various parameters to search for on wallhaven
type Search struct {
	Query       Q
	Categories  string
	Purities    string
	Sorting     string
	Order       string
	TopRange    string
	AtLeast     string
	Resolutions []string
	Ratios      []string
	Colors      []string // Colors is an array of hex colors represented as strings in #RRGGBB format
	Page        int64
}

func (s Search) toQuery() url.Values {
	v := s.Query.toQuery()
	if s.Categories != "" {
		v.Add("categories", s.Categories)
	}
	if s.Purities != "" {
		v.Add("purity", s.Purities)
	}
	if s.Sorting != "" {
		v.Add("sorting", s.Sorting)
	}
	if s.Order != "" {
		v.Add("order", s.Order)
	}
	if s.TopRange != "" && s.Sorting == "toplist" {
		v.Add("topRange", s.TopRange)
	}
	if s.AtLeast != "" {
		v.Add("atleast", s.AtLeast)
	}
	if len(s.Resolutions) > 0 {
		v.Add("resolutions", strings.Join(s.Resolutions, ","))
	}
	if len(s.Ratios) > 0 {
		v.Add("ratios", strings.Join(s.Ratios, ","))
	}
	if len(s.Colors) > 0 {
		v.Add("colors", strings.Join(s.Colors, ","))
	}
	if s.Page > 0 {
		v.Add("page", strconv.FormatInt(s.Page, 10))
	}
	return v
}

// SearchWallpapers performs a search on WH given a set of criteria.
// The response includes paging metadata in Meta.
func SearchWallpapers(ctx context.Context, search *Search) (*SearchResults, error) {
	slog.Debug("Making API request to wallhaven", "endpoint", "/search/", "page", search.Page)
	resp, err := getWithValues(ctx, "/search/", search.toQuery())
	if err != nil {
		return nil, err
	}

	out := &SearchResults{}
	if err := processResponse(resp, out); err != nil {
		return nil, err
	}
	slog.Debug("API request successful", "results_count", len(out.Data), "last_page", out.Meta.LastPage)
	return out, nil
}

func processResponse(resp *http.Response, out any) error {
	defer resp.Body.Close()

	byt, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(byt, out); err != nil {
		return fmt.Errorf("%w: %v", errors.ErrInvalidResponse, err)
	}

	return nil
}

// Result Structs -- server responses

// Meta contains paging metadata returned by the search endpoint
type Meta struct {
	CurrentPage int64 `json:"current_page"`
	LastPage    int64 `json:"last_page"`
	PerPage     int64 `json:"per_page"`
	Total       int64 `json:"total"`
}

// SearchResults a wrapper containing search results from wh
type SearchResults struct {
	Data []Wallpaper `json:"data"`
	Meta Meta        `json:"meta"`
}

// Wallpaper information about a given wallpaper
type Wallpaper struct {
	Path string `json:"path"`
}

const baseURL = "https://wallhaven.cc/api/v1"

func getWithValues(ctx context.Context, p string, v url.Values) (*http.Response, error) {
	u, err := url.Parse(baseURL + p)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	u.RawQuery = v.Encode()
	return getAuthedResponse(ctx, u.String())
}

func getAuthedResponse(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if apiKey := os.Getenv("WH_API_KEY"); apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	req.Header.Set("User-Agent", constants.UserAgent)

	for attempt := range maxRetries {
		if attempt > 0 {
			slog.Debug("Retrying request", "attempt", attempt+1, "url", url)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay * time.Duration(attempt)):
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			if attempt == maxRetries-1 {
				return nil, fmt.Errorf("%w: %v", errors.ErrAPIRequest, err)
			}
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		resp.Body.Close()

		if resp.StatusCode >= 500 && attempt < maxRetries-1 {
			slog.Debug("Server error, retrying", "status_code", resp.StatusCode)
			continue
		}

		return nil, errors.NewAPIError(url, resp.StatusCode, "HTTP request failed")
	}

	return nil, errors.NewAPIError(url, 0, "max retries exceeded")
}

var (
	client = &http.Client{
		Timeout: constants.RequestTimeout * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        constants.MaxIdleConns,
			MaxIdleConnsPerHost: constants.MaxIdleConnsPerHost,
			IdleConnTimeout:     constants.IdleConnTimeout * time.Second,
		},
	}
	maxRetries = constants.MaxRetries
	retryDelay = constants.RetryDelaySeconds * time.Second
)

func writeToFile(dest string, resp *http.Response) error {
	defer resp.Body.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dest), ".wallhaven_dl-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	if size := resp.ContentLength; size > 0 {
		slog.Info("Starting download", "size_mb", fmt.Sprintf("%.2f", float64(size)/1024/1024))
	}

	written, err := io.Copy(tmp, resp.Body)
	if err != nil {
		return fmt.Errorf("%w: %v", errors.ErrDownloadFailed, err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmp.Name(), dest); err != nil {
		return fmt.Errorf("failed to move downloaded file into place: %w", err)
	}

	slog.Info("Download completed", "bytes_written", written)
	return nil
}

// Download downloads a wallpaper into the given directory
func (w *Wallpaper) Download(ctx context.Context, dir string) error {
	if w.Path == "" {
		return fmt.Errorf("wallpaper path is empty")
	}

	filePath := filepath.Join(dir, path.Base(w.Path))
	slog.Debug("Downloading wallpaper", "url", w.Path, "destination", filePath)

	resp, err := getAuthedResponse(ctx, w.Path)
	if err != nil {
		return fmt.Errorf("failed to get wallpaper: %w", err)
	}

	return writeToFile(filePath, resp)
}
