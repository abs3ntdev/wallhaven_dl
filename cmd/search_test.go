package cmd

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/config"
	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/wallhaven"
)

type mockAPI struct {
	lastPage       int64
	requestedPages []int64
}

func (m *mockAPI) SearchWallpapers(ctx context.Context, search *wallhaven.Search) (*wallhaven.SearchResults, error) {
	m.requestedPages = append(m.requestedPages, search.Page)
	return &wallhaven.SearchResults{
		Data: []wallhaven.Wallpaper{{Path: "https://example.com/test.jpg"}},
		Meta: wallhaven.Meta{CurrentPage: search.Page, LastPage: m.lastPage},
	}, nil
}

func (m *mockAPI) DownloadWallpaper(ctx context.Context, wallpaper *wallhaven.Wallpaper, dir string) error {
	return os.WriteFile(filepath.Join(dir, filepath.Base(wallpaper.Path)), []byte("img"), 0o644)
}

type stubCache struct{}

func (s *stubCache) AddWallpaper(*wallhaven.Wallpaper, string, string, string) error { return nil }
func (s *stubCache) MarkAsUsed(string) error                                         { return nil }
func (s *stubCache) RemoveWallpaper(string) error                                    { return nil }
func (s *stubCache) CleanupInvalidEntries() error                                    { return nil }
func (s *stubCache) GetCurrent() *wallhaven.WallpaperMetadata                        { return nil }
func (s *stubCache) GetPrevious() *wallhaven.WallpaperMetadata                       { return nil }
func (s *stubCache) GetNext() *wallhaven.WallpaperMetadata                           { return nil }
func (s *stubCache) GetByID(string) *wallhaven.WallpaperMetadata                     { return nil }
func (s *stubCache) GetHistory(int) []*wallhaven.WallpaperMetadata                   { return nil }
func (s *stubCache) FindDuplicate(string) *wallhaven.WallpaperMetadata               { return nil }
func (s *stubCache) GetStatistics() *wallhaven.Statistics                            { return nil }
func (s *stubCache) SetCurrentView(string) error                                     { return nil }
func (s *stubCache) GetCurrentView() string                                          { return "" }
func (s *stubCache) GetOldWallpapers(time.Duration) []*wallhaven.WallpaperMetadata   { return nil }
func (s *stubCache) GetUnusedWallpapers() []*wallhaven.WallpaperMetadata             { return nil }
func (s *stubCache) ToggleFavorite(string) error                                     { return nil }
func (s *stubCache) SetRating(string, int) error                                     { return nil }
func (s *stubCache) GetFavorites() []*wallhaven.WallpaperMetadata                    { return nil }
func (s *stubCache) GetRandomFavorite() *wallhaven.WallpaperMetadata                 { return nil }
func (s *stubCache) GetByRating(int) []*wallhaven.WallpaperMetadata                  { return nil }
func (s *stubCache) AddTags(string, []string) error                                  { return nil }
func (s *stubCache) RemoveTags(string, []string) error                               { return nil }
func (s *stubCache) GetByTags([]string) []*wallhaven.WallpaperMetadata               { return nil }

func testConfig(t *testing.T, maxPages int) *config.Config {
	t.Helper()
	cfg := config.NewConfig()
	cfg.MaxPages = maxPages
	cfg.DownloadPath = t.TempDir()
	return cfg
}

func newTestHandler(api *mockAPI) *SearchHandler {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewSearchHandler(&stubCache{}, api, logger)
}

func TestSearchClampsToAvailablePages(t *testing.T) {
	api := &mockAPI{lastPage: 2}
	h := newTestHandler(api)

	for range 25 {
		_, _, err := h.searchAndDownload(context.Background(), testConfig(t, 10), "")
		if err != nil {
			t.Fatalf("searchAndDownload() error = %v", err)
		}
	}

	for _, page := range api.requestedPages {
		if page < 1 || page > api.lastPage {
			t.Errorf("Requested page %d outside available range [1, %d]", page, api.lastPage)
		}
	}
}

func TestSearchSamplesMultiplePages(t *testing.T) {
	api := &mockAPI{lastPage: 100}
	h := newTestHandler(api)

	seen := map[int64]bool{}
	for range 50 {
		_, _, err := h.searchAndDownload(context.Background(), testConfig(t, 5), "")
		if err != nil {
			t.Fatalf("searchAndDownload() error = %v", err)
		}
	}

	for _, page := range api.requestedPages {
		if page < 1 || page > 5 {
			t.Errorf("Requested page %d outside max pages range [1, 5]", page)
		}
		seen[page] = true
	}

	if len(seen) < 2 {
		t.Errorf("Expected random sampling across pages, only saw %v", seen)
	}
}

func TestSearchSinglePageMakesOneRequest(t *testing.T) {
	api := &mockAPI{lastPage: 100}
	h := newTestHandler(api)

	_, _, err := h.searchAndDownload(context.Background(), testConfig(t, 1), "")
	if err != nil {
		t.Fatalf("searchAndDownload() error = %v", err)
	}

	if len(api.requestedPages) != 1 || api.requestedPages[0] != 1 {
		t.Errorf("Expected single request for page 1, got %v", api.requestedPages)
	}
}
