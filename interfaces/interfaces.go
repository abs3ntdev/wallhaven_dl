// Package interfaces defines interfaces for dependency injection
package interfaces

import (
	"context"
	"time"

	"git.asdf.cafe/abs3nt/wallhaven_dl/src/wallhaven"
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
	GetStatistics() map[string]interface{}

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

// Logger defines the interface for logging operations
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// FileSystem defines the interface for file system operations
type FileSystem interface {
	Exists(path string) bool
	MkdirAll(path string, perm int) error
	Remove(path string) error
	Stat(path string) (FileInfo, error)
}

// FileInfo represents file information
type FileInfo interface {
	Size() int64
	ModTime() time.Time
	IsDir() bool
	Name() string
}

// Validator defines the interface for input validation
type Validator interface {
	ValidateRange(value string) error
	ValidatePurity(value string) error
	ValidateCategories(value string) error
	ValidateSort(value string) error
	ValidateOrder(value string) error
	ValidateRating(value int) error
	ValidateCleanupMode(value string) error
}