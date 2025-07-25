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

	"git.asdf.cafe/abs3nt/wallhaven_dl/constants"
	"git.asdf.cafe/abs3nt/wallhaven_dl/interfaces"
	"git.asdf.cafe/abs3nt/wallhaven_dl/validator"
	"git.asdf.cafe/abs3nt/wallhaven_dl/src/wallhaven"
)

// CleanupHandler handles cleanup command
type CleanupHandler struct {
	cache     interfaces.WallpaperCache
	validator interfaces.Validator
	logger    *slog.Logger
}

// NewCleanupHandler creates a new cleanup handler
func NewCleanupHandler(cache interfaces.WallpaperCache, logger *slog.Logger) *CleanupHandler {
	return &CleanupHandler{
		cache:     cache,
		validator: validator.NewValidator(),
		logger:    logger,
	}
}

// Handle processes the cleanup command
func (h *CleanupHandler) Handle(ctx context.Context, c *cli.Command) error {
	mode := c.String("mode")
	dryRun := c.Bool("dryRun")

	if err := h.validator.ValidateCleanupMode(mode); err != nil {
		return err
	}

	var toRemove []*wallhaven.WallpaperMetadata

	switch mode {
	case constants.CleanupModeUnused:
		toRemove = h.cache.GetUnusedWallpapers()
		fmt.Printf("Found %d unused wallpapers\n", len(toRemove))
	case constants.CleanupModeOld:
		olderThanStr := c.String("olderThan")
		duration, err := h.parseDuration(olderThanStr)
		if err != nil {
			return fmt.Errorf("invalid olderThan duration: %w", err)
		}
		toRemove = h.cache.GetOldWallpapers(duration)
		fmt.Printf("Found %d wallpapers older than %s\n", len(toRemove), olderThanStr)
	case constants.CleanupModeInvalid:
		if err := h.cache.CleanupInvalidEntries(); err != nil {
			return fmt.Errorf("failed to cleanup invalid entries: %w", err)
		}
		fmt.Printf("Cleaned up invalid cache entries\n")
		return nil
	default:
		return fmt.Errorf("invalid cleanup mode: %s", mode)
	}

	if len(toRemove) == 0 {
		fmt.Printf("No wallpapers to remove\n")
		return nil
	}

	return h.processRemoval(toRemove, dryRun)
}

func (h *CleanupHandler) processRemoval(toRemove []*wallhaven.WallpaperMetadata, dryRun bool) error {
	var totalSize int64
	for _, wallpaper := range toRemove {
		totalSize += wallpaper.Size
		if dryRun {
			fmt.Printf("Would remove: %s (%.2f MB, last used: %s)\n",
				wallpaper.Path,
				float64(wallpaper.Size)/1024/1024,
				wallpaper.LastUsed.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("Removing: %s\n", wallpaper.Path)
			if err := h.cache.RemoveWallpaper(wallpaper.ID); err != nil {
				h.logger.Error("Failed to remove wallpaper", "error", err, "path", wallpaper.Path)
			}
		}
	}

	if dryRun {
		fmt.Printf("\nWould free %.2f MB of storage\n", float64(totalSize)/1024/1024)
		fmt.Printf("Run without --dryRun to actually remove these wallpapers\n")
	} else {
		fmt.Printf("\nFreed %.2f MB of storage\n", float64(totalSize)/1024/1024)
	}

	return nil
}

func (h *CleanupHandler) parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	unit := s[len(s)-1:]
	valueStr := s[:len(s)-1]

	value, err := time.ParseDuration(valueStr + "h")
	if err != nil {
		return 0, err
	}

	switch unit {
	case "d":
		return value * 24, nil
	case "w":
		return value * 24 * 7, nil
	case "M":
		return value * 24 * 30, nil
	case "y":
		return value * 24 * 365, nil
	default:
		return time.ParseDuration(s)
	}
}

// GetFlags returns the CLI flags for the cleanup command
func (h *CleanupHandler) GetFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:      "downloadPath",
			Aliases:   []string{"dp"},
			Value:     filepath.Join(os.Getenv("HOME"), "Pictures", "Wallpapers"),
			TakesFile: true,
			Usage:     "Absolute path to download directory",
		},
		&cli.StringFlag{
			Name:  "mode",
			Value: constants.CleanupModeUnused,
			Usage: "Cleanup mode: " + joinValidValues(constants.ValidCleanupModes),
		},
		&cli.StringFlag{
			Name:  "olderThan",
			Value: constants.DefaultCleanupOlderThan,
			Usage: "Remove wallpapers older than this duration (e.g., '30d', '1w')",
		},
		&cli.BoolFlag{
			Name:  "dryRun",
			Value: false,
			Usage: "Show what would be removed without actually removing",
		},
	}
}