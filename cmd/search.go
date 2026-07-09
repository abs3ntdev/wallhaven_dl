// Package cmd provides command handlers for the CLI
package cmd

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"os"
	"path"
	"strings"

	"github.com/urfave/cli/v3"

	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/config"
	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/constants"
	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/errors"
	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/executor"
	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/interfaces"
	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/wallhaven"
	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/validator"
)

// SearchHandler handles search-related commands
type SearchHandler struct {
	cache    interfaces.WallpaperCache
	api      interfaces.WallpaperAPI
	executor interfaces.ScriptExecutor
	logger   *slog.Logger
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(cache interfaces.WallpaperCache, api interfaces.WallpaperAPI, logger *slog.Logger) *SearchHandler {
	return &SearchHandler{
		cache:    cache,
		api:      api,
		executor: executor.NewScriptExecutor(logger),
		logger:   logger,
	}
}

// Handle processes the search command
func (h *SearchHandler) Handle(ctx context.Context, c *cli.Command) error {
	h.logger.Info("Starting wallpaper search")

	cfg := h.buildConfig(c)

	if err := h.cache.CleanupInvalidEntries(); err != nil {
		h.logger.Warn("Failed to cleanup invalid cache entries", "error", err)
	}

	wallpaper, filePath, err := h.searchAndDownload(ctx, cfg, c.Args().First())
	if err != nil {
		h.logger.Error("Failed to search and download wallpaper", "error", err)
		return err
	}

	h.logger.Info("Wallpaper ready", "path", filePath)

	if err := h.executeScript(cfg.ScriptPath, filePath); err != nil {
		h.logger.Warn("Script execution failed, but wallpaper was downloaded successfully", "error", err)
	}

	if wallpaper != nil {
		id := wallhaven.GenerateID(wallpaper.Path)
		if err := h.cache.MarkAsUsed(id); err != nil {
			h.logger.Warn("Failed to mark wallpaper as used", "error", err)
		}
		if err := h.cache.SetCurrentView(id); err != nil {
			h.logger.Warn("Failed to update current view", "error", err)
		}
	}

	return nil
}

func (h *SearchHandler) buildConfig(c *cli.Command) *config.Config {
	cfg := config.NewConfig()

	cfg.Range = c.String("range")
	cfg.Purity = c.String("purity")
	cfg.Categories = c.String("categories")
	cfg.Sort = c.String("sort")
	cfg.Order = c.String("order")
	cfg.MaxPages = c.Int("page")
	cfg.Ratios = c.StringSlice("ratios")
	cfg.AtLeast = c.String("atLeast")
	cfg.DownloadPath = c.String("downloadPath")
	cfg.ScriptPath = c.String("scriptPath")

	return cfg
}

func (h *SearchHandler) searchAndDownload(ctx context.Context, cfg *config.Config, query string) (*wallhaven.Wallpaper, string, error) {
	search := &wallhaven.Search{
		Categories: cfg.Categories,
		Purities:   cfg.Purity,
		Sorting:    cfg.Sort,
		Order:      cfg.Order,
		TopRange:   cfg.Range,
		AtLeast:    cfg.AtLeast,
		Ratios:     cfg.Ratios,
		Page:       1,
	}

	if query != "" {
		search.Query = wallhaven.Q{
			Tags: []string{query},
		}
	}

	h.logger.Debug("Searching wallpapers", "query", query, "max_pages", cfg.MaxPages)
	results, err := h.api.SearchWallpapers(ctx, search)
	if err != nil {
		return nil, "", err
	}

	maxPage := min(int64(cfg.MaxPages), results.Meta.LastPage)
	if maxPage > 1 {
		if page := rand.Int64N(maxPage) + 1; page > 1 {
			search.Page = page
			h.logger.Debug("Fetching random page", "page", page, "available_pages", results.Meta.LastPage)
			results, err = h.api.SearchWallpapers(ctx, search)
			if err != nil {
				return nil, "", err
			}
		}
	}

	h.logger.Info("Found wallpapers", "count", len(results.Data), "page", search.Page)
	return h.getOrDownloadWithCache(ctx, results, cfg.DownloadPath, cfg.Categories, cfg.Purity)
}

func (h *SearchHandler) getOrDownloadWithCache(
	ctx context.Context,
	results *wallhaven.SearchResults,
	downloadPath, categories, purities string,
) (*wallhaven.Wallpaper, string, error) {
	if len(results.Data) == 0 {
		return nil, "", errors.ErrNoWallpapersFound
	}

	if err := os.MkdirAll(downloadPath, constants.DirPermissions); err != nil {
		return nil, "", err
	}

	result := results.Data[rand.IntN(len(results.Data))]
	fullPath := path.Join(downloadPath, path.Base(result.Path))

	if _, err := os.Stat(fullPath); err == nil {
		h.logger.Info("Using existing wallpaper", "path", fullPath)
		id := wallhaven.GenerateID(result.Path)
		if existing := h.cache.GetByID(id); existing == nil {
			if err := h.cache.AddWallpaper(&result, fullPath, categories, purities); err != nil {
				h.logger.Warn("Failed to add existing wallpaper to cache", "error", err)
			}
		}
		return &result, fullPath, nil
	}

	if err := h.api.DownloadWallpaper(ctx, &result, downloadPath); err != nil {
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
			Name:      "page",
			Aliases:   []string{"pg", "maxPages"},
			Value:     constants.DefaultMaxPages,
			Validator: v.ValidatePage,
			Usage:     "Maximum number of pages to randomly sample from (capped at the pages available)",
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
			Validator: v.ValidateScriptPath,
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
