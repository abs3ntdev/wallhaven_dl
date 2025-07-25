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

	"git.asdf.cafe/abs3nt/wallhaven_dl/internal/constants"
	"git.asdf.cafe/abs3nt/wallhaven_dl/internal/interfaces"
	"git.asdf.cafe/abs3nt/wallhaven_dl/internal/validator"
)

// RateHandler handles rating command
type RateHandler struct {
	cache     interfaces.WallpaperCache
	validator interfaces.Validator
	logger    *slog.Logger
}

// NewRateHandler creates a new rate handler
func NewRateHandler(cache interfaces.WallpaperCache, logger *slog.Logger) *RateHandler {
	return &RateHandler{
		cache:     cache,
		validator: validator.NewValidator(),
		logger:    logger,
	}
}

// Handle processes the rate command
func (h *RateHandler) Handle(ctx context.Context, c *cli.Command) error {
	rating := c.Int("rating")
	if err := h.validator.ValidateRating(rating); err != nil {
		return err
	}

	current := h.cache.GetCurrent()
	if current == nil {
		fmt.Printf("No current wallpaper found\n")
		return fmt.Errorf("no current wallpaper available")
	}

	if err := h.cache.SetRating(current.ID, rating); err != nil {
		h.logger.Error("Failed to set rating", "error", err)
		return err
	}

	fmt.Printf("Rated wallpaper %s: %s\n", filepath.Base(current.Path), strings.Repeat("⭐", rating))
	return nil
}

// GetFlags returns the CLI flags for the rate command
func (h *RateHandler) GetFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:      "downloadPath",
			Aliases:   []string{"dp"},
			Value:     filepath.Join(os.Getenv("HOME"), "Pictures", "Wallpapers"),
			TakesFile: true,
			Usage:     "Absolute path to download directory",
		},
		&cli.IntFlag{
			Name:     "rating",
			Aliases:  []string{"r"},
			Usage:    fmt.Sprintf("Rating from %d to %d stars", constants.MinRating, constants.MaxRating),
			Required: true,
		},
	}
}