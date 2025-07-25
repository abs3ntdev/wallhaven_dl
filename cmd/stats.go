// Package cmd provides command handlers for the CLI
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v3"

	"git.asdf.cafe/abs3nt/wallhaven_dl/interfaces"
)

// StatsHandler handles statistics command
type StatsHandler struct {
	cache  interfaces.WallpaperCache
	logger *slog.Logger
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(cache interfaces.WallpaperCache, logger *slog.Logger) *StatsHandler {
	return &StatsHandler{
		cache:  cache,
		logger: logger,
	}
}

// Handle processes the stats command
func (h *StatsHandler) Handle(ctx context.Context, c *cli.Command) error {
	stats := h.cache.GetStatistics()
	favorites := h.cache.GetFavorites()

	fmt.Printf("Wallpaper Statistics:\n")
	fmt.Printf("==================\n\n")
	fmt.Printf("Total wallpapers: %v\n", stats["total_wallpapers"])
	fmt.Printf("Valid wallpapers: %v\n", stats["valid_wallpapers"])
	fmt.Printf("Invalid/missing wallpapers: %v\n", stats["invalid_wallpapers"])
	fmt.Printf("Favorite wallpapers: %d\n", len(favorites))
	fmt.Printf("Total storage used: %.2f MB\n", stats["total_size_mb"])

	if oldest, ok := stats["oldest_download"].(time.Time); ok && !oldest.IsZero() {
		fmt.Printf("Oldest download: %s\n", oldest.Format("2006-01-02 15:04:05"))
	}
	if newest, ok := stats["newest_download"].(time.Time); ok && !newest.IsZero() {
		fmt.Printf("Newest download: %s\n", newest.Format("2006-01-02 15:04:05"))
	}

	if current, ok := stats["current_wallpaper"].(string); ok {
		fmt.Printf("Current wallpaper ID: %s\n", current)
	}
	if previous, ok := stats["previous_wallpaper"].(string); ok {
		fmt.Printf("Previous wallpaper ID: %s\n", previous)
	}

	return nil
}

// GetFlags returns the CLI flags for the stats command
func (h *StatsHandler) GetFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:      "downloadPath",
			Aliases:   []string{"dp"},
			Value:     filepath.Join(os.Getenv("HOME"), "Pictures", "Wallpapers"),
			TakesFile: true,
			Usage:     "Absolute path to download directory",
		},
	}
}