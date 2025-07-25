// Package cmd provides command handlers for the CLI
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"git.asdf.cafe/abs3nt/wallhaven_dl/executor"
	"git.asdf.cafe/abs3nt/wallhaven_dl/interfaces"
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

	if err := h.cache.MarkAsUsed(previous.ID); err != nil {
		h.logger.Warn("Failed to mark wallpaper as used", "error", err)
	}

	return nil
}

// GetFlags returns the CLI flags for the previous command
func (h *PreviousHandler) GetFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:      "downloadPath",
			Aliases:   []string{"dp"},
			Value:     filepath.Join(os.Getenv("HOME"), "Pictures", "Wallpapers"),
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