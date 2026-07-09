// Package interfaces defines interfaces for dependency injection
package interfaces

import (
	"context"
	"time"

	"git.asdf.cafe/abs3nt/wallhaven_dl/pkg/wallhaven"
)

// WallpaperCache defines the interface for wallpaper caching operations
type WallpaperCache interface {
	// Basic operations
	AddWallpaper(wallpaper *wallhaven.Wallpaper, filePath, categories, purities string) error
	MarkAsUsed(id string) error
	RemoveWallpaper(id string) error
	CleanupInvalidEntries() error

	// Retrieval operations
	GetCurrent() *wallhaven.WallpaperMetadata
	GetPrevious() *wallhaven.WallpaperMetadata
	GetNext() *wallhaven.WallpaperMetadata
	GetByID(id string) *wallhaven.WallpaperMetadata
	GetHistory(limit int) []*wallhaven.WallpaperMetadata
	FindDuplicate(hash string) *wallhaven.WallpaperMetadata
	GetStatistics() *wallhaven.Statistics

	// View state management
	SetCurrentView(wallpaperID string) error
	GetCurrentView() string

	// Cleanup operations
	GetOldWallpapers(olderThan time.Duration) []*wallhaven.WallpaperMetadata
	GetUnusedWallpapers() []*wallhaven.WallpaperMetadata

	// Favorites and rating
	ToggleFavorite(id string) error
	SetRating(id string, rating int) error
	GetFavorites() []*wallhaven.WallpaperMetadata
	GetRandomFavorite() *wallhaven.WallpaperMetadata
	GetByRating(minRating int) []*wallhaven.WallpaperMetadata

	// Tags
	AddTags(id string, tags []string) error
	RemoveTags(id string, tags []string) error
	GetByTags(tags []string) []*wallhaven.WallpaperMetadata
}

// WallpaperAPI defines the interface for wallpaper API operations
type WallpaperAPI interface {
	SearchWallpapers(ctx context.Context, search *wallhaven.Search) (*wallhaven.SearchResults, error)
	DownloadWallpaper(ctx context.Context, wallpaper *wallhaven.Wallpaper, dir string) error
}

// ScriptExecutor defines the interface for script execution
type ScriptExecutor interface {
	Execute(scriptPath, imagePath string) error
}
