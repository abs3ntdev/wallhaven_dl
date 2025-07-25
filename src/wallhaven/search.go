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
	"sync"
	"time"

	"git.asdf.cafe/abs3nt/wallhaven_dl/constants"
	"git.asdf.cafe/abs3nt/wallhaven_dl/errors"
)

// WallpaperID is a string representing a wallpaper
type WallpaperID string

// Q is used to hold the Q params for various fulltext options that the WH Search supports
type Q struct {
	Tags       []string
	ExcludeTags []string
	UserName   string
	TagID      int
	Type       string // Type is one of png/jpg
	Like       WallpaperID
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
	val := sb.String()
	if len(val) > 0 {
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
		v.Add("resolutions", strings.Join(s.Ratios, ","))
	}
	if len(s.Ratios) > 0 {
		v.Add("ratios", strings.Join(s.Ratios, ","))
	}
	if len(s.Colors) > 0 {
		v.Add("colors", strings.Join([]string(s.Colors), ","))
	}
	if s.Page > 0 {
		v.Add("page", strconv.FormatInt(s.Page, 10))
	}
	return v
}

// SearchWallpapers performs a search on WH given a set of criteria.
// Note that this API behaves slightly differently than the various
// single item apis as it also includes the metadata for paging purposes
func SearchWallpapers(search *Search) (*SearchResults, error) {
	return SearchWallpapersWithContext(context.Background(), search)
}

// SearchWallpapersWithContext performs a search on WH given a set of criteria with context support.
func SearchWallpapersWithContext(ctx context.Context, search *Search) (*SearchResults, error) {
	slog.Debug("Making API request to wallhaven", "endpoint", "/search/")
	resp, err := getWithValuesAndContext(ctx, "/search/", search.toQuery())
	if err != nil {
		return nil, err
	}

	out := &SearchResults{}
	err = processResponse(resp, out)
	if err != nil {
		return nil, err
	}
	slog.Debug("API request successful", "results_count", len(out.Data))
	return out, nil
}

func processResponse(resp *http.Response, out interface{}) error {
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

// SearchResults a wrapper containing search results from wh
type SearchResults struct {
	Data []Wallpaper `json:"data"`
}

// Wallpaper information about a given wallpaper
type Wallpaper struct {
	Path string `json:"path"`
}

// Tag full data on a given wallpaper tag
type Tag struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Alias      string `json:"alias"`
	CategoryID int    `json:"category_id"`
	Category   string `json:"category"`
	Purity     string `json:"purity"`
	CreatedAt  string `json:"created_at"`
}

const baseURL = "https://wallhaven.cc/api/v1"

func getWithBase(p string) string {
	return baseURL + p
}

func getWithValues(p string, v url.Values) (*http.Response, error) {
	return getWithValuesAndContext(context.Background(), p, v)
}

func getWithValuesAndContext(ctx context.Context, p string, v url.Values) (*http.Response, error) {
	u, err := url.Parse(getWithBase(p))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	u.RawQuery = v.Encode()
	return getAuthedResponseWithContext(ctx, u.String())
}

func getAuthedResponse(url string) (*http.Response, error) {
	return getAuthedResponseWithContext(context.Background(), url)
}

func getAuthedResponseWithContext(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	if apiKey := os.Getenv("WH_API_KEY"); apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	req.Header.Set("User-Agent", constants.UserAgent)
	
	for attempt := 0; attempt < maxRetries; attempt++ {
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

	// downloadPool limits concurrent downloads
	downloadPool = make(chan struct{}, 3)
	downloadMutex sync.Mutex
)

func download(filepath string, resp *http.Response) error {
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Get content length for progress tracking
	size := resp.ContentLength
	if size > 0 {
		slog.Info("Starting download", "size_mb", fmt.Sprintf("%.2f", float64(size)/1024/1024))
	}

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("%w: %v", errors.ErrDownloadFailed, err)
	}

	slog.Info("Download completed", "bytes_written", written)
	return nil
}

// Download downloads a wallpaper given the local filepath to save the wallpaper to
func (w *Wallpaper) Download(dir string) error {
	return w.DownloadWithContext(context.Background(), dir)
}

func (w *Wallpaper) DownloadWithContext(ctx context.Context, dir string) error {
	if w.Path == "" {
		return fmt.Errorf("wallpaper path is empty")
	}
	
	// Acquire download slot to limit concurrent downloads
	select {
	case downloadPool <- struct{}{}:
		defer func() { <-downloadPool }()
	case <-ctx.Done():
		return ctx.Err()
	}
	
	filePath := filepath.Join(dir, path.Base(w.Path))
	slog.Debug("Downloading wallpaper", "url", w.Path, "destination", filePath)
	
	resp, err := getAuthedResponseWithContext(ctx, w.Path)
	if err != nil {
		return fmt.Errorf("failed to get wallpaper: %w", err)
	}
	
	return download(filePath, resp)
}
