// Package cmd provides command handlers for the CLI
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"git.asdf.cafe/abs3nt/wallhaven_dl/internal/executor"
	"git.asdf.cafe/abs3nt/wallhaven_dl/internal/interfaces"
)

// FavoritesHandler handles favorites-related commands
type FavoritesHandler struct {
	cache    interfaces.WallpaperCache
	executor interfaces.ScriptExecutor
	logger   *slog.Logger
}

// NewFavoritesHandler creates a new favorites handler
func NewFavoritesHandler(cache interfaces.WallpaperCache, logger *slog.Logger) *FavoritesHandler {
	return &FavoritesHandler{
		cache:    cache,
		executor: executor.NewScriptExecutor(logger),
		logger:   logger,
	}
}

// HandleAdd adds current wallpaper to favorites
func (h *FavoritesHandler) HandleAdd(ctx context.Context, c *cli.Command) error {
	current := h.cache.GetCurrent()
	if current == nil {
		fmt.Printf("No current wallpaper found\n")
		return fmt.Errorf("no current wallpaper available")
	}

	if err := h.cache.ToggleFavorite(current.ID); err != nil {
		h.logger.Error("Failed to toggle favorite", "error", err)
		return err
	}

	if current.IsFavorite {
		fmt.Printf("Added wallpaper to favorites: %s\n", current.Path)
	} else {
		fmt.Printf("Removed wallpaper from favorites: %s\n", current.Path)
	}

	return nil
}

// HandleList lists all favorite wallpapers
func (h *FavoritesHandler) HandleList(ctx context.Context, c *cli.Command) error {
	favorites := h.cache.GetFavorites()
	if len(favorites) == 0 {
		fmt.Printf("No favorite wallpapers found\n")
		return nil
	}

	fmt.Printf("Favorite Wallpapers (%d total):\n", len(favorites))
	fmt.Printf("====================================\n\n")

	for i, fav := range favorites {
		fmt.Printf("%d. %s\n", i+1, filepath.Base(fav.Path))
		fmt.Printf("   ID: %s\n", fav.ID)
		fmt.Printf("   Path: %s\n", fav.Path)
		if fav.Rating > 0 {
			fmt.Printf("   Rating: %s\n", strings.Repeat("⭐", fav.Rating))
		}
		if len(fav.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(fav.Tags, ", "))
		}
		fmt.Printf("   Last used: %s\n", fav.LastUsed.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Use count: %d\n", fav.UseCount)
		fmt.Printf("\n")
	}

	return nil
}

// HandleRandom sets a random favorite as wallpaper
func (h *FavoritesHandler) HandleRandom(ctx context.Context, c *cli.Command) error {
	favorite := h.cache.GetRandomFavorite()
	if favorite == nil {
		fmt.Printf("No favorite wallpapers found\n")
		return fmt.Errorf("no favorite wallpapers available")
	}

	fmt.Printf("Setting random favorite wallpaper: %s\n", filepath.Base(favorite.Path))

	scriptPath := c.String("scriptPath")
	if scriptPath != "" {
		if err := h.executor.Execute(scriptPath, favorite.Path); err != nil {
			return err
		}
	}

	if err := h.cache.MarkAsUsed(favorite.ID); err != nil {
		h.logger.Warn("Failed to mark wallpaper as used", "error", err)
	}

	return nil
}

// GetCommonFlags returns common flags for favorites commands
func (h *FavoritesHandler) GetCommonFlags() []cli.Flag {
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

// GetRandomFlags returns flags for the random favorites command
func (h *FavoritesHandler) GetRandomFlags() []cli.Flag {
	flags := h.GetCommonFlags()
	flags = append(flags, &cli.StringFlag{
		Name:      "scriptPath",
		Aliases:   []string{"sp"},
		Value:     "",
		TakesFile: true,
		Usage:     "Path to the script to run after switching",
	})
	return flags
}