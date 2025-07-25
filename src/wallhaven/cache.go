// Package wallhaven provides wallpaper caching functionality
package wallhaven

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"git.asdf.cafe/abs3nt/wallhaven_dl/internal/constants"
)

// WallpaperMetadata contains metadata about a cached wallpaper
type WallpaperMetadata struct {
	ID           string    `json:"id"`
	Path         string    `json:"path"`
	OriginalURL  string    `json:"original_url"`
	Hash         string    `json:"hash"`
	Size         int64     `json:"size"`
	DownloadedAt time.Time `json:"downloaded_at"`
	LastUsed     time.Time `json:"last_used"`
	UseCount     int       `json:"use_count"`
	Categories   string    `json:"categories"`
	Purities     string    `json:"purities"`
	Resolution   string    `json:"resolution"`
	IsFavorite   bool      `json:"is_favorite"`
	Tags         []string  `json:"tags"`
	Rating       int       `json:"rating"` // 1-5 star rating
}

// WallpaperCache manages wallpaper metadata and history
type WallpaperCache struct {
	metadataPath string
	wallpapers   map[string]*WallpaperMetadata
	history      []string // ordered list of wallpaper IDs by usage
}

// NewWallpaperCache creates a new wallpaper cache instance
func NewWallpaperCache(cacheDir string) (*WallpaperCache, error) {
	metadataPath := filepath.Join(cacheDir, "metadata.json")
	
	cache := &WallpaperCache{
		metadataPath: metadataPath,
		wallpapers:   make(map[string]*WallpaperMetadata),
		history:      make([]string, 0),
	}
	
	if err := cache.load(); err != nil {
		slog.Debug("Failed to load cache metadata", "error", err)
	}
	
	return cache, nil
}

func (c *WallpaperCache) load() error {
	data, err := os.ReadFile(c.metadataPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %w", err)
	}
	
	var cacheData struct {
		Wallpapers map[string]*WallpaperMetadata `json:"wallpapers"`
		History    []string                      `json:"history"`
	}
	
	if err := json.Unmarshal(data, &cacheData); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	
	c.wallpapers = cacheData.Wallpapers
	c.history = cacheData.History
	
	return nil
}

func (c *WallpaperCache) save() error {
	if err := os.MkdirAll(filepath.Dir(c.metadataPath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	cacheData := struct {
		Wallpapers map[string]*WallpaperMetadata `json:"wallpapers"`
		History    []string                      `json:"history"`
	}{
		Wallpapers: c.wallpapers,
		History:    c.history,
	}
	
	data, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	if err := os.WriteFile(c.metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}
	
	return nil
}

// AddWallpaper adds a new wallpaper to the cache
func (c *WallpaperCache) AddWallpaper(wallpaper *Wallpaper, filePath, categories, purities string) error {
	hash, size, err := CalculateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}
	
	id := GenerateID(wallpaper.Path)
	metadata := &WallpaperMetadata{
		ID:           id,
		Path:         filePath,
		OriginalURL:  wallpaper.Path,
		Hash:         hash,
		Size:         size,
		DownloadedAt: time.Now(),
		LastUsed:     time.Now(),
		UseCount:     1,
		Categories:   categories,
		Purities:     purities,
	}
	
	c.wallpapers[id] = metadata
	c.addToHistory(id)
	
	return c.save()
}

func (c *WallpaperCache) MarkAsUsed(id string) error {
	if metadata, exists := c.wallpapers[id]; exists {
		metadata.LastUsed = time.Now()
		metadata.UseCount++
		c.addToHistory(id)
		return c.save()
	}
	return fmt.Errorf("wallpaper not found in cache: %s", id)
}

func (c *WallpaperCache) addToHistory(id string) {
	for i, historyID := range c.history {
		if historyID == id {
			c.history = append(c.history[:i], c.history[i+1:]...)
			break
		}
	}
	c.history = append([]string{id}, c.history...)
	
	if len(c.history) > constants.MaxHistorySize {
		c.history = c.history[:constants.MaxHistorySize]
	}
}

func (c *WallpaperCache) GetPrevious() *WallpaperMetadata {
	if len(c.history) < 2 {
		return nil
	}
	
	prevID := c.history[1]
	if metadata, exists := c.wallpapers[prevID]; exists {
		if _, err := os.Stat(metadata.Path); err == nil {
			return metadata
		}
	}
	return nil
}

func (c *WallpaperCache) GetCurrent() *WallpaperMetadata {
	if len(c.history) == 0 {
		return nil
	}
	
	currentID := c.history[0]
	if metadata, exists := c.wallpapers[currentID]; exists {
		if _, err := os.Stat(metadata.Path); err == nil {
			return metadata
		}
	}
	return nil
}

func (c *WallpaperCache) FindDuplicate(hash string) *WallpaperMetadata {
	for _, metadata := range c.wallpapers {
		if metadata.Hash == hash {
			if _, err := os.Stat(metadata.Path); err == nil {
				return metadata
			}
		}
	}
	return nil
}

func (c *WallpaperCache) GetStatistics() map[string]interface{} {
	totalCount := len(c.wallpapers)
	var totalSize int64
	var oldestDownload, newestDownload time.Time
	
	validWallpapers := 0
	for _, metadata := range c.wallpapers {
		if _, err := os.Stat(metadata.Path); err == nil {
			validWallpapers++
			totalSize += metadata.Size
			
			if oldestDownload.IsZero() || metadata.DownloadedAt.Before(oldestDownload) {
				oldestDownload = metadata.DownloadedAt
			}
			if newestDownload.IsZero() || metadata.DownloadedAt.After(newestDownload) {
				newestDownload = metadata.DownloadedAt
			}
		}
	}
	
	stats := map[string]interface{}{
		"total_wallpapers":   totalCount,
		"valid_wallpapers":   validWallpapers,
		"invalid_wallpapers": totalCount - validWallpapers,
		"total_size_mb":      float64(totalSize) / 1024 / 1024,
		"oldest_download":    oldestDownload,
		"newest_download":    newestDownload,
	}
	
	if len(c.history) > 0 {
		stats["current_wallpaper"] = c.history[0]
		if len(c.history) > 1 {
			stats["previous_wallpaper"] = c.history[1]
		}
	}
	
	return stats
}

func (c *WallpaperCache) GetOldWallpapers(olderThan time.Duration) []*WallpaperMetadata {
	cutoff := time.Now().Add(-olderThan)
	var old []*WallpaperMetadata
	
	for _, metadata := range c.wallpapers {
		if metadata.LastUsed.Before(cutoff) {
			if _, err := os.Stat(metadata.Path); err == nil {
				old = append(old, metadata)
			}
		}
	}
	
	sort.Slice(old, func(i, j int) bool {
		return old[i].LastUsed.Before(old[j].LastUsed)
	})
	
	return old
}

func (c *WallpaperCache) GetUnusedWallpapers() []*WallpaperMetadata {
	var unused []*WallpaperMetadata
	
	for _, metadata := range c.wallpapers {
		if metadata.UseCount <= 1 {
			if _, err := os.Stat(metadata.Path); err == nil {
				unused = append(unused, metadata)
			}
		}
	}
	
	sort.Slice(unused, func(i, j int) bool {
		return unused[i].DownloadedAt.Before(unused[j].DownloadedAt)
	})
	
	return unused
}

func (c *WallpaperCache) RemoveWallpaper(id string) error {
	if metadata, exists := c.wallpapers[id]; exists {
		if err := os.Remove(metadata.Path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove wallpaper file: %w", err)
		}
		
		delete(c.wallpapers, id)
		
		for i, historyID := range c.history {
			if historyID == id {
				c.history = append(c.history[:i], c.history[i+1:]...)
				break
			}
		}
		
		return c.save()
	}
	return fmt.Errorf("wallpaper not found in cache: %s", id)
}

func (c *WallpaperCache) CleanupInvalidEntries() error {
	var toRemove []string
	
	for id, metadata := range c.wallpapers {
		if _, err := os.Stat(metadata.Path); os.IsNotExist(err) {
			toRemove = append(toRemove, id)
		}
	}
	
	for _, id := range toRemove {
		delete(c.wallpapers, id)
		for i, historyID := range c.history {
			if historyID == id {
				c.history = append(c.history[:i], c.history[i+1:]...)
				break
			}
		}
	}
	
	if len(toRemove) > 0 {
		slog.Info("Cleaned up invalid cache entries", "count", len(toRemove))
		return c.save()
	}
	
	return nil
}

func (c *WallpaperCache) ToggleFavorite(id string) error {
	if metadata, exists := c.wallpapers[id]; exists {
		metadata.IsFavorite = !metadata.IsFavorite
		slog.Info("Toggled favorite status", "id", id, "is_favorite", metadata.IsFavorite)
		return c.save()
	}
	return fmt.Errorf("wallpaper not found in cache: %s", id)
}

func (c *WallpaperCache) SetRating(id string, rating int) error {
	if rating < constants.MinRating || rating > constants.MaxRating {
		return fmt.Errorf("rating must be between %d and %d", constants.MinRating, constants.MaxRating)
	}
	
	if metadata, exists := c.wallpapers[id]; exists {
		metadata.Rating = rating
		slog.Info("Set wallpaper rating", "id", id, "rating", rating)
		return c.save()
	}
	return fmt.Errorf("wallpaper not found in cache: %s", id)
}

func (c *WallpaperCache) AddTags(id string, tags []string) error {
	if metadata, exists := c.wallpapers[id]; exists {
		for _, tag := range tags {
			found := false
			for _, existingTag := range metadata.Tags {
				if existingTag == tag {
					found = true
					break
				}
			}
			if !found {
				metadata.Tags = append(metadata.Tags, tag)
			}
		}
		slog.Info("Added tags to wallpaper", "id", id, "tags", tags)
		return c.save()
	}
	return fmt.Errorf("wallpaper not found in cache: %s", id)
}

func (c *WallpaperCache) RemoveTags(id string, tags []string) error {
	if metadata, exists := c.wallpapers[id]; exists {
		for _, tagToRemove := range tags {
			for i, tag := range metadata.Tags {
				if tag == tagToRemove {
					metadata.Tags = append(metadata.Tags[:i], metadata.Tags[i+1:]...)
					break
				}
			}
		}
		slog.Info("Removed tags from wallpaper", "id", id, "tags", tags)
		return c.save()
	}
	return fmt.Errorf("wallpaper not found in cache: %s", id)
}

func (c *WallpaperCache) GetFavorites() []*WallpaperMetadata {
	var favorites []*WallpaperMetadata
	
	for _, metadata := range c.wallpapers {
		if metadata.IsFavorite {
			if _, err := os.Stat(metadata.Path); err == nil {
				favorites = append(favorites, metadata)
			}
		}
	}
	
	sort.Slice(favorites, func(i, j int) bool {
		if favorites[i].Rating != favorites[j].Rating {
			return favorites[i].Rating > favorites[j].Rating
		}
		return favorites[i].LastUsed.After(favorites[j].LastUsed)
	})
	
	return favorites
}

func (c *WallpaperCache) GetByRating(minRating int) []*WallpaperMetadata {
	var rated []*WallpaperMetadata
	
	for _, metadata := range c.wallpapers {
		if metadata.Rating >= minRating {
			if _, err := os.Stat(metadata.Path); err == nil {
				rated = append(rated, metadata)
			}
		}
	}
	
	sort.Slice(rated, func(i, j int) bool {
		if rated[i].Rating != rated[j].Rating {
			return rated[i].Rating > rated[j].Rating
		}
		return rated[i].LastUsed.After(rated[j].LastUsed)
	})
	
	return rated
}

func (c *WallpaperCache) GetByTags(tags []string) []*WallpaperMetadata {
	var tagged []*WallpaperMetadata
	
	for _, metadata := range c.wallpapers {
		hasAllTags := true
		for _, requiredTag := range tags {
			found := false
			for _, tag := range metadata.Tags {
				if tag == requiredTag {
					found = true
					break
				}
			}
			if !found {
				hasAllTags = false
				break
			}
		}
		
		if hasAllTags {
			if _, err := os.Stat(metadata.Path); err == nil {
				tagged = append(tagged, metadata)
			}
		}
	}
	
	sort.Slice(tagged, func(i, j int) bool {
		return tagged[i].LastUsed.After(tagged[j].LastUsed)
	})
	
	return tagged
}

func (c *WallpaperCache) GetRandomFavorite() *WallpaperMetadata {
	favorites := c.GetFavorites()
	if len(favorites) == 0 {
		return nil
	}
	
	return favorites[0] // Already sorted by rating and usage
}

// CalculateFileHash calculates SHA256 hash and size of a file
func CalculateFileHash(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	
	return fmt.Sprintf("%x", hash.Sum(nil)), size, nil
}

// GenerateID generates a unique ID from a URL
func GenerateID(url string) string {
	hash := sha256.Sum256([]byte(url))
	return fmt.Sprintf("%x", hash)[:16]
}