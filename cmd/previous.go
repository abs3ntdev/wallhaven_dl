// Package cmd provides command handlers for the CLI
package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"

	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/config"
	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/executor"
	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/interfaces"
)

// PreviousHandler handles previous wallpaper command
type PreviousHandler struct {
	cache    interfaces.WallpaperCache
	executor interfaces.ScriptExecutor
	logger   *slog.Logger
}

// NewPreviousHandler creates a new previous handler
func NewPreviousHandler(cache interfaces.WallpaperCache, logger *slog.Logger) *PreviousHandler {
	return &PreviousHandler{
		cache:    cache,
		executor: executor.NewScriptExecutor(logger),
		logger:   logger,
	}
}

// Handle processes the previous command
func (h *PreviousHandler) Handle(ctx context.Context, c *cli.Command) error {
	previous := h.cache.GetPrevious()
	if previous == nil {
		h.logger.Info("No previous wallpaper found")
		return fmt.Errorf("no previous wallpaper available")
	}

	h.logger.Info("Switching to previous wallpaper", "path", previous.Path)

	scriptPath := c.String("scriptPath")
	if scriptPath != "" {
		if err := h.executor.Execute(scriptPath, previous.Path); err != nil {
			return err
		}
	}

	// Update the current view to this wallpaper so next 'previous' call goes further back
	if err := h.cache.SetCurrentView(previous.ID); err != nil {
		h.logger.Warn("Failed to update current view", "error", err)
	}

	return nil
}

// GetFlags returns the CLI flags for the previous command
func (h *PreviousHandler) GetFlags() []cli.Flag {
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
