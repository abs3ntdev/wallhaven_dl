// Package cmd provides command handlers for the CLI
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/config"
	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/interfaces"
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
	fmt.Printf("║          Wallpaper Statistics & Insights          ║\n")
	fmt.Printf("╚═══════════════════════════════════════════════════╝\n\n")

	fmt.Printf("📊 Collection Overview\n")
	fmt.Printf("─────────────────────────────────────────────────────\n")
	fmt.Printf("  Total wallpapers:     %d\n", stats.TotalWallpapers)
	fmt.Printf("  Valid wallpapers:     %d\n", stats.ValidWallpapers)
	fmt.Printf("  Invalid/missing:      %d\n", stats.InvalidWallpapers)
	fmt.Printf("  Favorite wallpapers:  %d\n", stats.FavoriteCount)
	fmt.Printf("  Total storage used:   %.2f MB\n", stats.TotalSizeMB)
	if stats.AverageRating > 0 {
		fmt.Printf("  Average rating:       %.1f / 5\n", stats.AverageRating)
	}
	fmt.Printf("\n")

	if !stats.OldestDownload.IsZero() || !stats.NewestDownload.IsZero() {
		fmt.Printf("📅 Timeline\n")
		fmt.Printf("─────────────────────────────────────────────────────\n")
		if !stats.OldestDownload.IsZero() {
			fmt.Printf("  Oldest download:      %s\n", stats.OldestDownload.Format("2006-01-02 15:04:05"))
		}
		if !stats.NewestDownload.IsZero() {
			fmt.Printf("  Newest download:      %s\n", stats.NewestDownload.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("\n")
	}

	fmt.Printf("⚡ Recent Activity\n")
	fmt.Printf("─────────────────────────────────────────────────────\n")
	fmt.Printf("  Unique wallpapers used (last 7 days):  %d\n", stats.UniqueLastWeek)
	fmt.Printf("  Unique wallpapers used (last 30 days): %d\n", stats.UniqueLastMonth)
	fmt.Printf("  Total history entries:                 %d\n", stats.HistoryEntries)
	fmt.Printf("\n")

	if stats.CurrentWallpaper != "" {
		fmt.Printf("🖼️  Current State\n")
		fmt.Printf("─────────────────────────────────────────────────────\n")
		fmt.Printf("  Current wallpaper ID:  %s\n", stats.CurrentWallpaper)
		if stats.PreviousWallpaper != "" {
			fmt.Printf("  Previous wallpaper ID: %s\n", stats.PreviousWallpaper)
		}
		fmt.Printf("\n")
	}

	if len(stats.MostUsed) > 0 {
		fmt.Printf("⭐ Top %d Most Used Wallpapers\n", len(stats.MostUsed))
		fmt.Printf("─────────────────────────────────────────────────────\n")
		for i, entry := range stats.MostUsed {
			fmt.Printf("  %d. %s (%d uses)\n", i+1, filepath.Base(entry.Path), entry.UseCount)
		}
		fmt.Printf("\n")
	}

	if len(stats.TopTags) > 0 {
		fmt.Printf("🏷️  Top %d Most Common Tags\n", len(stats.TopTags))
		fmt.Printf("─────────────────────────────────────────────────────\n")
		for _, tc := range stats.TopTags {
			fmt.Printf("  %-30s %d\n", tc.Tag, tc.Count)
		}
		fmt.Printf("\n")
	}

	if len(stats.Resolutions) > 0 {
		fmt.Printf("📐 Resolution Distribution\n")
		fmt.Printf("─────────────────────────────────────────────────────\n")
		for _, rc := range stats.Resolutions {
			fmt.Printf("  %-30s %d\n", rc.Resolution, rc.Count)
		}
		fmt.Printf("\n")
	}

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
