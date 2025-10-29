// Package cmd provides command handlers for the CLI
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"

	"git.asdf.cafe/abs3nt/wallhaven_dl/executor"
	"git.asdf.cafe/abs3nt/wallhaven_dl/interfaces"
)

// HistoryHandler handles history browsing
type HistoryHandler struct {
	cache    interfaces.WallpaperCache
	executor interfaces.ScriptExecutor
	logger   *slog.Logger
}

// NewHistoryHandler creates a new history handler
func NewHistoryHandler(cache interfaces.WallpaperCache, logger *slog.Logger) *HistoryHandler {
	return &HistoryHandler{
		cache:    cache,
		executor: executor.NewScriptExecutor(logger),
		logger:   logger,
	}
}

// Handle processes the history command
func (h *HistoryHandler) Handle(ctx context.Context, c *cli.Command) error {
	history := h.cache.GetHistory(50)

	if len(history) == 0 {
		fmt.Println("No wallpaper history found.")
		fmt.Println("Use 'search' to download some wallpapers first!")
		return nil
	}

	fmt.Printf("\n📜 Wallpaper History (last %d)\n", len(history))
	fmt.Println(strings.Repeat("=", 80))

	for i, wp := range history {
		fmt.Printf("\n%d. %s\n", i+1, filepath.Base(wp.Path))
		fmt.Printf("   Resolution: %s\n", wp.Resolution)
		fmt.Printf("   Used: %d times", wp.UseCount)

		if wp.IsFavorite {
			fmt.Printf(" | ⭐ Favorite")
		}
		if wp.Rating > 0 {
			fmt.Printf(" | Rating: %s", strings.Repeat("★", wp.Rating))
		}
		fmt.Println()

		if len(wp.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(wp.Tags, ", "))
		}
	}

	fmt.Println()

	// Interactive selection
	scriptPath := c.String("scriptPath")
	if scriptPath == "" {
		return nil
	}

	fmt.Print("Enter number to apply wallpaper (or press Enter to cancel): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		fmt.Println("Cancelled.")
		return nil
	}

	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(history) {
		return fmt.Errorf("invalid selection: %s", input)
	}

	selected := history[selection-1]
	fmt.Printf("Applying wallpaper: %s\n", filepath.Base(selected.Path))

	if err := h.executor.Execute(scriptPath, selected.Path); err != nil {
		return err
	}

	// Update view state
	if err := h.cache.SetCurrentView(selected.ID); err != nil {
		h.logger.Warn("Failed to update current view", "error", err)
	}

	fmt.Println("✓ Wallpaper applied successfully!")
	return nil
}

// GetFlags returns the CLI flags for the history command
func (h *HistoryHandler) GetFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:      "scriptPath",
			Aliases:   []string{"sp"},
			Value:     "",
			TakesFile: true,
			Usage:     "Path to the script to run after selecting a wallpaper",
		},
	}
}
