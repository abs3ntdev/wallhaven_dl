// Package cmd provides command handlers for the CLI
package cmd

import (
	"context"
	"log/slog"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"git.asdf.cafe/abs3nt/wallhaven_dl/config"
	"git.asdf.cafe/abs3nt/wallhaven_dl/constants"
	"git.asdf.cafe/abs3nt/wallhaven_dl/errors"
	"git.asdf.cafe/abs3nt/wallhaven_dl/executor"
	"git.asdf.cafe/abs3nt/wallhaven_dl/interfaces"
	"git.asdf.cafe/abs3nt/wallhaven_dl/src/wallhaven"
	"git.asdf.cafe/abs3nt/wallhaven_dl/validator"
)

// SearchHandler handles search-related commands
type SearchHandler struct {
	cache     interfaces.WallpaperCache
	api       interfaces.WallpaperAPI
	executor  interfaces.ScriptExecutor
	validator interfaces.Validator
	logger    *slog.Logger
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(cache interfaces.WallpaperCache, api interfaces.WallpaperAPI, logger *slog.Logger) *SearchHandler {
	return &SearchHandler{
		cache:     cache,
		api:       api,
		executor:  executor.NewScriptExecutor(logger),
		validator: validator.NewValidator(),
		logger:    logger,
	}
}

// Handle processes the search command
func (h *SearchHandler) Handle(ctx context.Context, c *cli.Command) error {
	h.logger.Info("Starting wallpaper search")

	cfg, err := h.buildConfig(c)
	if err != nil {
		h.logger.Error("Failed to build configuration", "error", err)
		return err
	}

	if err := cfg.Validate(); err != nil {
		h.logger.Error("Configuration validation failed", "error", err)
		return err
	}

	if err := h.cache.CleanupInvalidEntries(); err != nil {
		h.logger.Warn("Failed to cleanup invalid cache entries", "error", err)
	}

	wallpaper, filePath, err := h.searchAndDownload(ctx, cfg, c.Args().First())
	if err != nil {
		h.logger.Error("Failed to search and download wallpaper", "error", err)
		return err
	}

	h.logger.Info("Wallpaper ready", "path", filePath)

	// Execute script if provided - non-fatal if it fails
	if err := h.executeScript(cfg.ScriptPath, filePath); err != nil {
		h.logger.Warn("Script execution failed, but wallpaper was downloaded successfully", "error", err)
	}

	if wallpaper != nil {
		id := wallhaven.GenerateID(wallpaper.Path)
		if err := h.cache.MarkAsUsed(id); err != nil {
			h.logger.Warn("Failed to mark wallpaper as used", "error", err)
		}
		// Set this as the current view so 'previous' works correctly
		if err := h.cache.SetCurrentView(id); err != nil {
			h.logger.Warn("Failed to update current view", "error", err)
		}
	}

	return nil
}

func (h *SearchHandler) buildConfig(c *cli.Command) (*config.Config, error) {
	cfg := config.NewConfig()

	// Override with CLI values
	cfg.Range = c.String("range")
	cfg.Purity = c.String("purity")
	cfg.Categories = c.String("categories")
	cfg.Sort = c.String("sort")
	cfg.Order = c.String("order")
	cfg.Page = c.Int("page")
	cfg.Ratios = c.StringSlice("ratios")
	cfg.AtLeast = c.String("atLeast")
	cfg.DownloadPath = c.String("downloadPath")
	cfg.ScriptPath = c.String("scriptPath")

	return cfg, nil
}

func (h *SearchHandler) searchAndDownload(ctx context.Context, cfg *config.Config, query string) (*wallhaven.Wallpaper, string, error) {
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)

	search := &wallhaven.Search{
		Categories: cfg.Categories,
		Purities:   cfg.Purity,
		Sorting:    cfg.Sort,
		Order:      cfg.Order,
		TopRange:   cfg.Range,
		AtLeast:    cfg.AtLeast,
		Ratios:     cfg.Ratios,
		Page:       int64(r.Intn(cfg.Page) + 1),
	}

	if query != "" {
		search.Query = wallhaven.Q{
			Tags: []string{query},
		}
	}

	h.logger.Debug("Searching wallpapers", "query", query, "page", search.Page)
	results, err := wallhaven.SearchWallpapersWithContext(ctx, search)
	if err != nil {
		return nil, "", err
	}

	h.logger.Info("Found wallpapers", "count", len(results.Data))
	return h.getOrDownloadWithCache(ctx, results, r, cfg.DownloadPath, cfg.Categories, cfg.Purity)
}

func (h *SearchHandler) getOrDownloadWithCache(ctx context.Context, results *wallhaven.SearchResults, r *rand.Rand, downloadPath, categories, purities string) (*wallhaven.Wallpaper, string, error) {
	if len(results.Data) == 0 {
		return nil, "", errors.ErrNoWallpapersFound
	}

	if err := os.MkdirAll(downloadPath, 0o755); err != nil {
		return nil, "", err
	}

	result := results.Data[r.Intn(len(results.Data))]
	fullPath := path.Join(downloadPath, path.Base(result.Path))

	if _, err := os.Stat(fullPath); err == nil {
		h.logger.Info("Using existing wallpaper", "path", fullPath)
		// Ensure the wallpaper is in the cache (may be missing if migrated from old cache)
		id := wallhaven.GenerateID(result.Path)
		if existing := h.cache.GetByID(id); existing == nil {
			if err := h.cache.AddWallpaper(&result, fullPath, categories, purities); err != nil {
				h.logger.Warn("Failed to add existing wallpaper to cache", "error", err)
			}
		}
		return &result, fullPath, nil
	}

	if err := result.DownloadWithContext(ctx, downloadPath); err != nil {
		return nil, "", err
	}

	hash, _, err := wallhaven.CalculateFileHash(fullPath)
	if err != nil {
		h.logger.Warn("Failed to calculate hash for downloaded file", "error", err)
	} else {
		if duplicate := h.cache.FindDuplicate(hash); duplicate != nil {
			h.logger.Info("Duplicate wallpaper detected", "existing", duplicate.Path, "new", fullPath)
			os.Remove(fullPath)
			return &result, duplicate.Path, nil
		}
	}

	if err := h.cache.AddWallpaper(&result, fullPath, categories, purities); err != nil {
		h.logger.Warn("Failed to add wallpaper to cache", "error", err)
	}

	return &result, fullPath, nil
}

func (h *SearchHandler) executeScript(scriptPath, imagePath string) error {
	if scriptPath == "" {
		return nil
	}

	return h.executor.Execute(scriptPath, imagePath)
}

// GetFlags returns the CLI flags for the search command
func (h *SearchHandler) GetFlags() []cli.Flag {
	v := validator.NewValidator()

	return []cli.Flag{
		&cli.StringFlag{
			Name:      "range",
			Aliases:   []string{"r"},
			Value:     constants.DefaultRange,
			Validator: v.ValidateRange,
			Usage:     "Time range for top sorting (" + strings.Join(constants.ValidRanges, ", ") + ")",
		},
		&cli.StringFlag{
			Name:      "purity",
			Aliases:   []string{"p"},
			Value:     constants.DefaultPurity,
			Validator: v.ValidatePurity,
			Usage:     "Purity filter: 3 chars for SFW|Sketchy|NSFW (e.g., '110' for SFW+Sketchy)",
		},
		&cli.StringFlag{
			Name:      "categories",
			Aliases:   []string{"c"},
			Value:     constants.DefaultCategories,
			Validator: v.ValidateCategories,
			Usage:     "Category filter: 3 chars for General|Anime|People (e.g., '010' for Anime only)",
		},
		&cli.StringFlag{
			Name:      "sort",
			Aliases:   []string{"s"},
			Value:     constants.DefaultSort,
			Validator: v.ValidateSort,
			Usage:     "Sort order: " + strings.Join(constants.ValidSorts, ", "),
		},
		&cli.StringFlag{
			Name:      "order",
			Aliases:   []string{"o"},
			Value:     constants.DefaultOrder,
			Validator: v.ValidateOrder,
			Usage:     "Order of the wallpapers: " + strings.Join(constants.ValidOrders, ", "),
		},
		&cli.IntFlag{
			Name:    "page",
			Aliases: []string{"pg"},
			Value:   constants.DefaultMaxPages,
			Usage:   "Max pages to randomly select from (1-100)",
		},
		&cli.StringSliceFlag{
			Name:    "ratios",
			Aliases: []string{"rt"},
			Value:   constants.DefaultRatios,
			Usage:   "Ratios of the wallpapers",
		},
		&cli.StringFlag{
			Name:    "atLeast",
			Aliases: []string{"al"},
			Value:   constants.DefaultAtLeast,
			Usage:   "Minimum resolution",
		},
		&cli.StringFlag{
			Name:      "scriptPath",
			Aliases:   []string{"sp"},
			Value:     "",
			TakesFile: true,
			Usage:     "Path to the script to run after downloading",
		},
		&cli.StringFlag{
			Name:      "downloadPath",
			Aliases:   []string{"dp"},
			Value:     config.GetDefaultDownloadPath(),
			TakesFile: true,
			Usage:     "Absolute path to download directory",
		},
	}
}
