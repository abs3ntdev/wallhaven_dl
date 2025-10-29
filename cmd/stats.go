// Package cmd provides command handlers for the CLI
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/urfave/cli/v3"

	"git.asdf.cafe/abs3nt/wallhaven_dl/config"
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

	fmt.Printf("\n╔═══════════════════════════════════════════════════╗\n")
	fmt.Printf("║         Wallpaper Statistics & Insights          ║\n")
	fmt.Printf("╚═══════════════════════════════════════════════════╝\n\n")

	// Basic Stats
	fmt.Printf("📊 Collection Overview\n")
	fmt.Printf("─────────────────────────────────────────────────────\n")
	fmt.Printf("  Total wallpapers:     %v\n", stats["total_wallpapers"])
	fmt.Printf("  Valid wallpapers:     %v\n", stats["valid_wallpapers"])
	fmt.Printf("  Invalid/missing:      %v\n", stats["invalid_wallpapers"])
	fmt.Printf("  Favorite wallpapers:  %v\n", stats["favorite_count"])
	fmt.Printf("  Total storage used:   %.2f MB\n", stats["total_size_mb"])
	if avgRating, ok := stats["average_rating"].(float64); ok && avgRating > 0 {
		fmt.Printf("  Average rating:       %.1f / 5\n", avgRating)
	}
	fmt.Printf("\n")

	// Timeline
	fmt.Printf("📅 Timeline\n")
	fmt.Printf("─────────────────────────────────────────────────────\n")
	if oldest, ok := stats["oldest_download"].(time.Time); ok && !oldest.IsZero() {
		fmt.Printf("  Oldest download:      %s\n", oldest.Format("2006-01-02 15:04:05"))
	}
	if newest, ok := stats["newest_download"].(time.Time); ok && !newest.IsZero() {
		fmt.Printf("  Newest download:      %s\n", newest.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("\n")

	// Recent Activity
	fmt.Printf("⚡ Recent Activity\n")
	fmt.Printf("─────────────────────────────────────────────────────\n")
	fmt.Printf("  Unique wallpapers used (last 7 days):  %v\n", stats["unique_wallpapers_last_week"])
	fmt.Printf("  Unique wallpapers used (last 30 days): %v\n", stats["unique_wallpapers_last_month"])
	fmt.Printf("  Total history entries:                 %v\n", stats["total_history_entries"])
	fmt.Printf("\n")

	// Current State
	if current, ok := stats["current_wallpaper"].(string); ok && current != "" {
		fmt.Printf("🖼️  Current State\n")
		fmt.Printf("─────────────────────────────────────────────────────\n")
		fmt.Printf("  Current wallpaper ID:  %s\n", current)
		if previous, ok := stats["previous_wallpaper"].(string); ok && previous != "" {
			fmt.Printf("  Previous wallpaper ID: %s\n", previous)
		}
		fmt.Printf("\n")
	}

	// Most Used Wallpapers - Using reflection/sprintf since we can't type assert the struct from GetStatistics
	fmt.Printf("⭐ Top 5 Most Used Wallpapers\n")
	fmt.Printf("─────────────────────────────────────────────────────\n")
	if mostUsedRaw, ok := stats["most_used"]; ok {
		// Use fmt to print the value - it will handle the struct slice
		fmt.Printf("  %v\n", mostUsedRaw)
	} else {
		fmt.Printf("  No data available\n")
	}
	fmt.Printf("\n")

	// Top Tags
	fmt.Printf("🏷️  Top 10 Most Common Tags\n")
	fmt.Printf("─────────────────────────────────────────────────────\n")
	if topTagsRaw, ok := stats["top_tags"]; ok {
		fmt.Printf("  %v\n", topTagsRaw)
	} else {
		fmt.Printf("  No tags found\n")
	}
	fmt.Printf("\n")

	// Resolution Distribution
	fmt.Printf("📐 Resolution Distribution\n")
	fmt.Printf("─────────────────────────────────────────────────────\n")
	if resolutionsRaw, ok := stats["resolutions"]; ok {
		fmt.Printf("  %v\n", resolutionsRaw)
	} else {
		fmt.Printf("  No resolution data\n")
	}
	fmt.Printf("\n")

	return nil
}

// GetFlags returns the CLI flags for the stats command
func (h *StatsHandler) GetFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:      "downloadPath",
			Aliases:   []string{"dp"},
			Value:     config.GetDefaultDownloadPath(),
			TakesFile: true,
			Usage:     "Absolute path to download directory",
		},
	}
}
