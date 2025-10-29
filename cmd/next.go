// Package cmd provides command handlers for the CLI
package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"

	"git.asdf.cafe/abs3nt/wallhaven_dl/config"
	"git.asdf.cafe/abs3nt/wallhaven_dl/executor"
	"git.asdf.cafe/abs3nt/wallhaven_dl/interfaces"
)

// NextHandler handles next wallpaper command
type NextHandler struct {
	cache    interfaces.WallpaperCache
	executor interfaces.ScriptExecutor
	logger   *slog.Logger
}

// NewNextHandler creates a new next handler
func NewNextHandler(cache interfaces.WallpaperCache, logger *slog.Logger) *NextHandler {
	return &NextHandler{
		cache:    cache,
		executor: executor.NewScriptExecutor(logger),
		logger:   logger,
	}
}

// Handle processes the next command
func (h *NextHandler) Handle(ctx context.Context, c *cli.Command) error {
	next := h.cache.GetNext()
	if next == nil {
		h.logger.Info("No next wallpaper found")
		return fmt.Errorf("no next wallpaper available")
	}

	h.logger.Info("Switching to next wallpaper", "path", next.Path)

	scriptPath := c.String("scriptPath")
	if scriptPath != "" {
		if err := h.executor.Execute(scriptPath, next.Path); err != nil {
			return err
		}
	}

	// Update the current view to this wallpaper so next/previous calls work correctly
	if err := h.cache.SetCurrentView(next.ID); err != nil {
		h.logger.Warn("Failed to update current view", "error", err)
	}

	return nil
}

// GetFlags returns the CLI flags for the next command
func (h *NextHandler) GetFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:      "downloadPath",
			Aliases:   []string{"dp"},
			Value:     config.GetDefaultDownloadPath(),
			TakesFile: true,
			Usage:     "Absolute path to download directory",
		},
		&cli.StringFlag{
			Name:      "scriptPath",
			Aliases:   []string{"sp"},
			Value:     "",
			TakesFile: true,
			Usage:     "Path to the script to run after switching",
		},
	}
}
