// Package main provides the CLI entry point for wallhaven_dl
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"git.asdf.cafe/abs3nt/wallhaven_dl/cmd"
	"git.asdf.cafe/abs3nt/wallhaven_dl/constants"
	"git.asdf.cafe/abs3nt/wallhaven_dl/src/wallhaven"
)

// wallhavenAPI implements the WallpaperAPI interface
type wallhavenAPI struct{}

func (api *wallhavenAPI) SearchWallpapers(ctx context.Context, search *wallhaven.Search) (*wallhaven.SearchResults, error) {
	return wallhaven.SearchWallpapersWithContext(ctx, search)
}

func (api *wallhavenAPI) DownloadWallpaper(ctx context.Context, wallpaper *wallhaven.Wallpaper, dir string) error {
	return wallpaper.DownloadWithContext(ctx, dir)
}

var Version = "dev"

func main() {
	logger := setupLogger()
	slog.SetDefault(logger)

	cache, err := initializeCache()
	if err != nil {
		logger.Error("Failed to initialize cache", "error", err)
		os.Exit(1)
	}

	app := createCLIApp(cache, logger)
	
	if err := app.Run(context.Background(), os.Args); err != nil {
		logger.Error("Application failed", "error", err)
		os.Exit(1)
	}
}

func setupLogger() *slog.Logger {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	}

	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}

func initializeCache() (*wallhaven.WallpaperCache, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return nil, fmt.Errorf("HOME environment variable not set")
	}
	
	cacheDir := filepath.Join(home, "Pictures", "Wallpapers", constants.CacheDir)
	return wallhaven.NewWallpaperCache(cacheDir)
}

func createCLIApp(cache *wallhaven.WallpaperCache, logger *slog.Logger) *cli.Command {
	// Initialize handlers
	searchHandler := cmd.NewSearchHandler(cache, &wallhavenAPI{}, logger)
	previousHandler := cmd.NewPreviousHandler(cache, logger)
	statsHandler := cmd.NewStatsHandler(cache, logger)
	cleanupHandler := cmd.NewCleanupHandler(cache, logger)
	favoritesHandler := cmd.NewFavoritesHandler(cache, logger)
	rateHandler := cmd.NewRateHandler(cache, logger)

	return &cli.Command{
		EnableShellCompletion: true,
		Version:               Version,
		Name:                  constants.AppName,
		Usage:                 "Download wallpapers from wallhaven.cc",
		Commands: []*cli.Command{
			{
				Name:  "search",
				Usage: "Search for wallpapers",
				Flags: searchHandler.GetFlags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					return searchHandler.Handle(ctx, c)
				},
			},
			{
				Name:    "previous",
				Aliases: []string{"prev", "p"},
				Usage:   "Switch back to the previous wallpaper",
				Flags:   previousHandler.GetFlags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					return previousHandler.Handle(ctx, c)
				},
			},
			{
				Name:    "stats",
				Aliases: []string{"statistics"},
				Usage:   "Show wallpaper statistics",
				Flags:   statsHandler.GetFlags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					return statsHandler.Handle(ctx, c)
				},
			},
			{
				Name:    "cleanup",
				Aliases: []string{"clean"},
				Usage:   "Clean up old or unused wallpapers",
				Flags:   cleanupHandler.GetFlags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					return cleanupHandler.Handle(ctx, c)
				},
			},
			{
				Name:    "favorite",
				Aliases: []string{"fav"},
				Usage:   "Manage favorite wallpapers",
				Commands: []*cli.Command{
					{
						Name:  "add",
						Usage: "Add current wallpaper to favorites",
						Flags: favoritesHandler.GetCommonFlags(),
						Action: func(ctx context.Context, c *cli.Command) error {
							return favoritesHandler.HandleAdd(ctx, c)
						},
					},
					{
						Name:  "list",
						Usage: "List all favorite wallpapers",
						Flags: favoritesHandler.GetCommonFlags(),
						Action: func(ctx context.Context, c *cli.Command) error {
							return favoritesHandler.HandleList(ctx, c)
						},
					},
					{
						Name:  "random",
						Usage: "Set a random favorite as wallpaper",
						Flags: favoritesHandler.GetRandomFlags(),
						Action: func(ctx context.Context, c *cli.Command) error {
							return favoritesHandler.HandleRandom(ctx, c)
						},
					},
				},
			},
			{
				Name:  "rate",
				Usage: "Rate current wallpaper (1-5 stars)",
				Flags: rateHandler.GetFlags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					return rateHandler.Handle(ctx, c)
				},
			},
		},
	}
}