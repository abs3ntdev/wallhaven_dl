// Package wallhaven provides wallpaper caching functionality
package wallhaven

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"git.asdf.cafe/abs3nt/wallhaven_dl/constants"
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

// WallpaperCache manages wallpaper metadata and history using SQLite
type WallpaperCache struct {
	db *sql.DB
	mu sync.RWMutex // protects database operations
}

// NewWallpaperCache creates a new wallpaper cache instance with SQLite backend
func NewWallpaperCache(cacheDir string) (*WallpaperCache, error) {
	if err := os.MkdirAll(cacheDir, constants.DirPermissions); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	dbPath := filepath.Join(cacheDir, "wallpapers.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	cache := &WallpaperCache{db: db}

	if err := cache.initialize(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return cache, nil
}

// initialize creates the database schema
func (c *WallpaperCache) initialize() error {
	schema := `
	CREATE TABLE IF NOT EXISTS wallpapers (
		id TEXT PRIMARY KEY,
		path TEXT NOT NULL,
		original_url TEXT NOT NULL,
		hash TEXT NOT NULL,
		size INTEGER NOT NULL,
		downloaded_at DATETIME NOT NULL,
		last_used DATETIME NOT NULL,
		use_count INTEGER NOT NULL DEFAULT 1,
		categories TEXT NOT NULL,
		purities TEXT NOT NULL,
		resolution TEXT,
		is_favorite BOOLEAN NOT NULL DEFAULT 0,
		rating INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS wallpaper_tags (
		wallpaper_id TEXT NOT NULL,
		tag TEXT NOT NULL,
		PRIMARY KEY (wallpaper_id, tag),
		FOREIGN KEY (wallpaper_id) REFERENCES wallpapers(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS usage_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		wallpaper_id TEXT NOT NULL,
		used_at DATETIME NOT NULL,
		FOREIGN KEY (wallpaper_id) REFERENCES wallpapers(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS view_state (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		current_wallpaper_id TEXT,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_wallpapers_hash ON wallpapers(hash);
	CREATE INDEX IF NOT EXISTS idx_wallpapers_last_used ON wallpapers(last_used);
	CREATE INDEX IF NOT EXISTS idx_wallpapers_favorite ON wallpapers(is_favorite);
	CREATE INDEX IF NOT EXISTS idx_usage_history_wallpaper_id ON usage_history(wallpaper_id);
	CREATE INDEX IF NOT EXISTS idx_usage_history_used_at ON usage_history(used_at);
	`

	_, err := c.db.Exec(schema)
	return err
}

// Close closes the database connection
func (c *WallpaperCache) Close() error {
	return c.db.Close()
}

// AddWallpaper adds a new wallpaper to the cache
func (c *WallpaperCache) AddWallpaper(wallpaper *Wallpaper, filePath, categories, purities string) error {
	hash, size, err := CalculateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Get image dimensions
	resolution, err := getImageResolution(filePath)
	if err != nil {
		slog.Warn("Failed to get image resolution", "path", filePath, "error", err)
		resolution = "" // Leave empty if we can't determine it
	}

	id := GenerateID(wallpaper.Path)
	now := time.Now()

	c.mu.Lock()
	tx, err := c.db.Begin()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO wallpapers (id, path, original_url, hash, size, downloaded_at, last_used, use_count, categories, purities, resolution)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?)
	`, id, filePath, wallpaper.Path, hash, size, now, now, categories, purities, resolution)
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to insert wallpaper: %w", err)
	}

	// Add to usage history
	_, err = tx.Exec(`INSERT INTO usage_history (wallpaper_id, used_at) VALUES (?, ?)`, id, now)
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to insert usage history: %w", err)
	}

	if err := tx.Commit(); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	c.mu.Unlock()

	// Enforce cache limits after adding new wallpaper (no lock held)
	return c.EnforceCacheLimits()
}

// getImageResolution returns the resolution of an image as "WIDTHxHEIGHT"
func getImageResolution(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%dx%d", img.Width, img.Height), nil
}

// MarkAsUsed updates the last used timestamp and increments use count
func (c *WallpaperCache) MarkAsUsed(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		UPDATE wallpapers
		SET last_used = ?, use_count = use_count + 1
		WHERE id = ?
	`, now, id)
	if err != nil {
		return fmt.Errorf("failed to update wallpaper: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("wallpaper not found in cache: %s", id)
	}

	// Add to usage history
	_, err = tx.Exec(`INSERT INTO usage_history (wallpaper_id, used_at) VALUES (?, ?)`, id, now)
	if err != nil {
		return fmt.Errorf("failed to insert usage history: %w", err)
	}

	return tx.Commit()
}

// SetCurrentView updates the currently viewed wallpaper
func (c *WallpaperCache) SetCurrentView(wallpaperID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.db.Exec(`
		INSERT INTO view_state (id, current_wallpaper_id, updated_at)
		VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			current_wallpaper_id = excluded.current_wallpaper_id,
			updated_at = excluded.updated_at
	`, wallpaperID, time.Now())

	return err
}

// GetCurrentView returns the ID of the currently viewed wallpaper
func (c *WallpaperCache) GetCurrentView() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var wallpaperID string
	err := c.db.QueryRow(`SELECT current_wallpaper_id FROM view_state WHERE id = 1`).Scan(&wallpaperID)
	if err != nil {
		return ""
	}
	return wallpaperID
}

// GetNext returns the wallpaper after the currently viewed one in history
func (c *WallpaperCache) GetNext() *WallpaperMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Get the currently viewed wallpaper
	currentViewID := ""
	c.db.QueryRow(`SELECT current_wallpaper_id FROM view_state WHERE id = 1`).Scan(&currentViewID)

	// If no current view, return the most recent from history
	if currentViewID == "" {
		var wallpaperID string
		err := c.db.QueryRow(`
			SELECT wallpaper_id
			FROM usage_history
			GROUP BY wallpaper_id
			ORDER BY MAX(used_at) DESC
			LIMIT 1
		`).Scan(&wallpaperID)

		if err != nil {
			return nil
		}

		var metadata WallpaperMetadata
		err = c.db.QueryRow(`
			SELECT id, path, original_url, hash, size, downloaded_at, last_used, use_count,
			       categories, purities, COALESCE(resolution, ''), is_favorite, rating
			FROM wallpapers
			WHERE id = ?
		`, wallpaperID).Scan(&metadata.ID, &metadata.Path, &metadata.OriginalURL, &metadata.Hash,
			&metadata.Size, &metadata.DownloadedAt, &metadata.LastUsed, &metadata.UseCount,
			&metadata.Categories, &metadata.Purities, &metadata.Resolution, &metadata.IsFavorite, &metadata.Rating)

		if err != nil {
			return nil
		}

		if _, err := os.Stat(metadata.Path); err != nil {
			return nil
		}

		metadata.Tags = c.getTags(metadata.ID)
		return &metadata
	}

	// Find the wallpaper that comes after the current view in history (more recent)
	var wallpaperID string
	err := c.db.QueryRow(`
		SELECT wallpaper_id
		FROM usage_history
		WHERE wallpaper_id IN (
			SELECT wallpaper_id
			FROM usage_history
			GROUP BY wallpaper_id
			HAVING MAX(used_at) > (
				SELECT MAX(used_at)
				FROM usage_history
				WHERE wallpaper_id = ?
			)
		)
		GROUP BY wallpaper_id
		ORDER BY MAX(used_at) ASC
		LIMIT 1
	`, currentViewID).Scan(&wallpaperID)

	if err != nil {
		return nil
	}

	// Get the wallpaper metadata
	var metadata WallpaperMetadata
	err = c.db.QueryRow(`
		SELECT id, path, original_url, hash, size, downloaded_at, last_used, use_count,
		       categories, purities, COALESCE(resolution, ''), is_favorite, rating
		FROM wallpapers
		WHERE id = ?
	`, wallpaperID).Scan(&metadata.ID, &metadata.Path, &metadata.OriginalURL, &metadata.Hash,
		&metadata.Size, &metadata.DownloadedAt, &metadata.LastUsed, &metadata.UseCount,
		&metadata.Categories, &metadata.Purities, &metadata.Resolution, &metadata.IsFavorite, &metadata.Rating)

	if err != nil {
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(metadata.Path); err != nil {
		return nil
	}

	metadata.Tags = c.getTags(metadata.ID)
	return &metadata
}

// GetPrevious returns the wallpaper before the currently viewed one in history
func (c *WallpaperCache) GetPrevious() *WallpaperMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Get the currently viewed wallpaper
	currentViewID := ""
	c.db.QueryRow(`SELECT current_wallpaper_id FROM view_state WHERE id = 1`).Scan(&currentViewID)

	var wallpaperID string
	var err error

	if currentViewID == "" {
		// No current view set, return the second most recent from history
		err = c.db.QueryRow(`
			SELECT wallpaper_id
			FROM usage_history
			GROUP BY wallpaper_id
			ORDER BY MAX(used_at) DESC
			LIMIT 1 OFFSET 1
		`).Scan(&wallpaperID)
	} else {
		// Find the wallpaper that comes before the current view in history
		err = c.db.QueryRow(`
			SELECT wallpaper_id
			FROM usage_history
			WHERE wallpaper_id IN (
				SELECT wallpaper_id
				FROM usage_history
				GROUP BY wallpaper_id
				HAVING MAX(used_at) < (
					SELECT MAX(used_at)
					FROM usage_history
					WHERE wallpaper_id = ?
				)
			)
			GROUP BY wallpaper_id
			ORDER BY MAX(used_at) DESC
			LIMIT 1
		`, currentViewID).Scan(&wallpaperID)
	}

	if err != nil {
		return nil
	}

	// Get the wallpaper metadata
	var metadata WallpaperMetadata
	err = c.db.QueryRow(`
		SELECT id, path, original_url, hash, size, downloaded_at, last_used, use_count,
		       categories, purities, COALESCE(resolution, ''), is_favorite, rating
		FROM wallpapers
		WHERE id = ?
	`, wallpaperID).Scan(&metadata.ID, &metadata.Path, &metadata.OriginalURL, &metadata.Hash,
		&metadata.Size, &metadata.DownloadedAt, &metadata.LastUsed, &metadata.UseCount,
		&metadata.Categories, &metadata.Purities, &metadata.Resolution, &metadata.IsFavorite, &metadata.Rating)
	if err != nil {
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(metadata.Path); err != nil {
		return nil
	}

	metadata.Tags = c.getTags(metadata.ID)
	return &metadata
}

// GetByID returns a wallpaper by its ID
func (c *WallpaperCache) GetByID(id string) *WallpaperMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var metadata WallpaperMetadata
	err := c.db.QueryRow(`
		SELECT id, path, original_url, hash, size, downloaded_at, last_used, use_count,
		       categories, purities, COALESCE(resolution, ''), is_favorite, rating
		FROM wallpapers
		WHERE id = ?
	`, id).Scan(&metadata.ID, &metadata.Path, &metadata.OriginalURL, &metadata.Hash,
		&metadata.Size, &metadata.DownloadedAt, &metadata.LastUsed, &metadata.UseCount,
		&metadata.Categories, &metadata.Purities, &metadata.Resolution, &metadata.IsFavorite, &metadata.Rating)
	if err != nil {
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(metadata.Path); err != nil {
		return nil
	}

	metadata.Tags = c.getTags(metadata.ID)
	return &metadata
}

// GetCurrent returns the most recently used wallpaper
func (c *WallpaperCache) GetCurrent() *WallpaperMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Get the most recent UNIQUE wallpaper ID from usage history
	// Group by wallpaper_id to get distinct wallpapers, ordered by their most recent usage
	var wallpaperID string
	err := c.db.QueryRow(`
		SELECT wallpaper_id
		FROM usage_history
		GROUP BY wallpaper_id
		ORDER BY MAX(used_at) DESC
		LIMIT 1
	`).Scan(&wallpaperID)
	if err != nil {
		return nil
	}

	// Get the wallpaper metadata
	var metadata WallpaperMetadata
	err = c.db.QueryRow(`
		SELECT id, path, original_url, hash, size, downloaded_at, last_used, use_count,
		       categories, purities, COALESCE(resolution, ''), is_favorite, rating
		FROM wallpapers
		WHERE id = ?
	`, wallpaperID).Scan(&metadata.ID, &metadata.Path, &metadata.OriginalURL, &metadata.Hash,
		&metadata.Size, &metadata.DownloadedAt, &metadata.LastUsed, &metadata.UseCount,
		&metadata.Categories, &metadata.Purities, &metadata.Resolution, &metadata.IsFavorite, &metadata.Rating)
	if err != nil {
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(metadata.Path); err != nil {
		return nil
	}

	metadata.Tags = c.getTags(metadata.ID)
	return &metadata
}

// getTags retrieves tags for a wallpaper
func (c *WallpaperCache) getTags(wallpaperID string) []string {
	rows, err := c.db.Query(`SELECT tag FROM wallpaper_tags WHERE wallpaper_id = ?`, wallpaperID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err == nil {
			tags = append(tags, tag)
		}
	}
	return tags
}

// FindDuplicate finds a wallpaper with the same hash
func (c *WallpaperCache) FindDuplicate(hash string) *WallpaperMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var metadata WallpaperMetadata
	err := c.db.QueryRow(`
		SELECT id, path, original_url, hash, size, downloaded_at, last_used, use_count,
		       categories, purities, COALESCE(resolution, ''), is_favorite, rating
		FROM wallpapers
		WHERE hash = ?
		LIMIT 1
	`, hash).Scan(&metadata.ID, &metadata.Path, &metadata.OriginalURL, &metadata.Hash,
		&metadata.Size, &metadata.DownloadedAt, &metadata.LastUsed, &metadata.UseCount,
		&metadata.Categories, &metadata.Purities, &metadata.Resolution, &metadata.IsFavorite, &metadata.Rating)
	if err != nil {
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(metadata.Path); err != nil {
		return nil
	}

	metadata.Tags = c.getTags(metadata.ID)
	return &metadata
}

// GetStatistics returns statistics about the cache
func (c *WallpaperCache) GetStatistics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var totalCount, validCount int
	var totalSize int64
	var oldestDownload, newestDownload time.Time

	c.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(size), 0) FROM wallpapers`).Scan(&totalCount, &totalSize)

	// Count valid wallpapers (files that exist)
	rows, err := c.db.Query(`SELECT path FROM wallpapers`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var path string
			if rows.Scan(&path) == nil {
				if _, err := os.Stat(path); err == nil {
					validCount++
				}
			}
		}
	}

	c.db.QueryRow(`SELECT MIN(downloaded_at), MAX(downloaded_at) FROM wallpapers`).Scan(&oldestDownload, &newestDownload)

	stats := map[string]interface{}{
		"total_wallpapers":   totalCount,
		"valid_wallpapers":   validCount,
		"invalid_wallpapers": totalCount - validCount,
		"total_size_mb":      float64(totalSize) / 1024 / 1024,
		"oldest_download":    oldestDownload,
		"newest_download":    newestDownload,
	}

	// Get current and previous wallpaper IDs
	var currentID, previousID string
	rows, err = c.db.Query(`
		SELECT DISTINCT wallpaper_id
		FROM usage_history
		ORDER BY used_at DESC
		LIMIT 2
	`)
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			rows.Scan(&currentID)
			stats["current_wallpaper"] = currentID
		}
		if rows.Next() {
			rows.Scan(&previousID)
			stats["previous_wallpaper"] = previousID
		}
	}

	// Get favorite count
	var favoriteCount int
	c.db.QueryRow(`SELECT COUNT(*) FROM wallpapers WHERE is_favorite = 1`).Scan(&favoriteCount)
	stats["favorite_count"] = favoriteCount

	// Get average rating
	var avgRating float64
	c.db.QueryRow(`SELECT COALESCE(AVG(rating), 0) FROM wallpapers WHERE rating > 0`).Scan(&avgRating)
	stats["average_rating"] = avgRating

	// Get top 5 most used wallpapers
	type MostUsed struct {
		ID       string
		Path     string
		UseCount int
	}
	mostUsed := []MostUsed{}
	rows, err = c.db.Query(`
		SELECT id, path, use_count
		FROM wallpapers
		ORDER BY use_count DESC
		LIMIT 5
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var mu MostUsed
			if rows.Scan(&mu.ID, &mu.Path, &mu.UseCount) == nil {
				mostUsed = append(mostUsed, mu)
			}
		}
	}
	stats["most_used"] = mostUsed

	// Get top 10 most common tags
	type TagCount struct {
		Tag   string
		Count int
	}
	topTags := []TagCount{}
	rows, err = c.db.Query(`
		SELECT tag, COUNT(*) as count
		FROM wallpaper_tags
		GROUP BY tag
		ORDER BY count DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tc TagCount
			if rows.Scan(&tc.Tag, &tc.Count) == nil {
				topTags = append(topTags, tc)
			}
		}
	}
	stats["top_tags"] = topTags

	// Get resolution distribution
	type ResolutionCount struct {
		Resolution string
		Count      int
	}
	resolutions := []ResolutionCount{}
	rows, err = c.db.Query(`
		SELECT COALESCE(resolution, 'unknown'), COUNT(*) as count
		FROM wallpapers
		GROUP BY resolution
		ORDER BY count DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rc ResolutionCount
			if rows.Scan(&rc.Resolution, &rc.Count) == nil {
				resolutions = append(resolutions, rc)
			}
		}
	}
	stats["resolutions"] = resolutions

	// Get usage activity (last 7 days, 30 days)
	var weekCount, monthCount int
	weekAgo := time.Now().AddDate(0, 0, -7)
	monthAgo := time.Now().AddDate(0, -1, 0)
	c.db.QueryRow(`SELECT COUNT(DISTINCT wallpaper_id) FROM usage_history WHERE used_at > ?`, weekAgo).Scan(&weekCount)
	c.db.QueryRow(`SELECT COUNT(DISTINCT wallpaper_id) FROM usage_history WHERE used_at > ?`, monthAgo).Scan(&monthCount)
	stats["unique_wallpapers_last_week"] = weekCount
	stats["unique_wallpapers_last_month"] = monthCount

	// Total usage history entries
	var historyCount int
	c.db.QueryRow(`SELECT COUNT(*) FROM usage_history`).Scan(&historyCount)
	stats["total_history_entries"] = historyCount

	return stats
}

// GetHistory returns wallpapers ordered by most recent usage
func (c *WallpaperCache) GetHistory(limit int) []*WallpaperMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 {
		limit = 50 // Default limit
	}

	rows, err := c.db.Query(`
		SELECT DISTINCT w.id, w.path, w.original_url, w.hash, w.size, w.downloaded_at, w.last_used, w.use_count,
		       w.categories, w.purities, COALESCE(w.resolution, ''), w.is_favorite, w.rating
		FROM wallpapers w
		JOIN usage_history uh ON w.id = uh.wallpaper_id
		GROUP BY w.id
		ORDER BY MAX(uh.used_at) DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()

	return c.scanWallpapers(rows)
}

// GetOldWallpapers returns wallpapers older than the specified duration
func (c *WallpaperCache) GetOldWallpapers(olderThan time.Duration) []*WallpaperMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cutoff := time.Now().Add(-olderThan)

	rows, err := c.db.Query(`
		SELECT id, path, original_url, hash, size, downloaded_at, last_used, use_count,
		       categories, purities, COALESCE(resolution, ''), is_favorite, rating
		FROM wallpapers
		WHERE last_used < ?
		ORDER BY last_used ASC
	`, cutoff)
	if err != nil {
		return nil
	}
	defer rows.Close()

	return c.scanWallpapers(rows)
}

// GetUnusedWallpapers returns wallpapers that have been used once or less
func (c *WallpaperCache) GetUnusedWallpapers() []*WallpaperMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rows, err := c.db.Query(`
		SELECT id, path, original_url, hash, size, downloaded_at, last_used, use_count,
		       categories, purities, COALESCE(resolution, ''), is_favorite, rating
		FROM wallpapers
		WHERE use_count <= 1
		ORDER BY downloaded_at ASC
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	return c.scanWallpapers(rows)
}

// scanWallpapers is a helper to scan multiple wallpapers from rows
func (c *WallpaperCache) scanWallpapers(rows *sql.Rows) []*WallpaperMetadata {
	var wallpapers []*WallpaperMetadata

	for rows.Next() {
		var metadata WallpaperMetadata
		err := rows.Scan(&metadata.ID, &metadata.Path, &metadata.OriginalURL, &metadata.Hash,
			&metadata.Size, &metadata.DownloadedAt, &metadata.LastUsed, &metadata.UseCount,
			&metadata.Categories, &metadata.Purities, &metadata.Resolution, &metadata.IsFavorite, &metadata.Rating)
		if err != nil {
			continue
		}

		// Check if file exists
		if _, err := os.Stat(metadata.Path); err != nil {
			continue
		}

		metadata.Tags = c.getTags(metadata.ID)
		wallpapers = append(wallpapers, &metadata)
	}

	return wallpapers
}

// RemoveWallpaper removes a wallpaper from the cache and deletes the file
func (c *WallpaperCache) RemoveWallpaper(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get the path first
	var path string
	err := c.db.QueryRow(`SELECT path FROM wallpapers WHERE id = ?`, id).Scan(&path)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("wallpaper not found in cache: %s", id)
		}
		return fmt.Errorf("failed to query wallpaper: %w", err)
	}

	// Remove file
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		slog.Warn("Failed to remove wallpaper file", "path", path, "error", err)
		return fmt.Errorf("failed to remove wallpaper file: %w", err)
	}

	// Remove from database (CASCADE will handle tags and history)
	_, err = c.db.Exec(`DELETE FROM wallpapers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete wallpaper from database: %w", err)
	}

	return nil
}

// CleanupInvalidEntries removes entries for files that no longer exist
func (c *WallpaperCache) CleanupInvalidEntries() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	rows, err := c.db.Query(`SELECT id, path FROM wallpapers`)
	if err != nil {
		return fmt.Errorf("failed to query wallpapers: %w", err)
	}
	defer rows.Close()

	var toRemove []string
	for rows.Next() {
		var id, path string
		if rows.Scan(&id, &path) == nil {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				toRemove = append(toRemove, id)
			}
		}
	}

	if len(toRemove) == 0 {
		return nil
	}

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, id := range toRemove {
		_, err := tx.Exec(`DELETE FROM wallpapers WHERE id = ?`, id)
		if err != nil {
			slog.Warn("Failed to delete invalid entry", "id", id, "error", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("Cleaned up invalid cache entries", "count", len(toRemove))
	return nil
}

// ToggleFavorite toggles the favorite status of a wallpaper
func (c *WallpaperCache) ToggleFavorite(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	result, err := c.db.Exec(`
		UPDATE wallpapers
		SET is_favorite = NOT is_favorite
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("failed to toggle favorite: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("wallpaper not found in cache: %s", id)
	}

	var isFavorite bool
	c.db.QueryRow(`SELECT is_favorite FROM wallpapers WHERE id = ?`, id).Scan(&isFavorite)

	return nil
}

// SetRating sets the rating for a wallpaper
func (c *WallpaperCache) SetRating(id string, rating int) error {
	if rating < constants.MinRating || rating > constants.MaxRating {
		return fmt.Errorf("rating must be between %d and %d", constants.MinRating, constants.MaxRating)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	result, err := c.db.Exec(`
		UPDATE wallpapers
		SET rating = ?
		WHERE id = ?
	`, rating, id)
	if err != nil {
		return fmt.Errorf("failed to set rating: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("wallpaper not found in cache: %s", id)
	}

	slog.Info("Set wallpaper rating", "id", id, "rating", rating)
	return nil
}

// AddTags adds tags to a wallpaper
func (c *WallpaperCache) AddTags(id string, tags []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if wallpaper exists
	var exists bool
	err := c.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM wallpapers WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check wallpaper existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("wallpaper not found in cache: %s", id)
	}

	existingTags := c.getTags(id)

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, tag := range tags {
		if !slices.Contains(existingTags, tag) {
			_, err := tx.Exec(`INSERT INTO wallpaper_tags (wallpaper_id, tag) VALUES (?, ?)`, id, tag)
			if err != nil {
				return fmt.Errorf("failed to add tag: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("Added tags to wallpaper", "id", id, "tags", tags)
	return nil
}

// RemoveTags removes tags from a wallpaper
func (c *WallpaperCache) RemoveTags(id string, tags []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if wallpaper exists
	var exists bool
	err := c.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM wallpapers WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check wallpaper existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("wallpaper not found in cache: %s", id)
	}

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, tag := range tags {
		_, err := tx.Exec(`DELETE FROM wallpaper_tags WHERE wallpaper_id = ? AND tag = ?`, id, tag)
		if err != nil {
			return fmt.Errorf("failed to remove tag: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("Removed tags from wallpaper", "id", id, "tags", tags)
	return nil
}

// GetFavorites returns all favorite wallpapers sorted by rating and last used
func (c *WallpaperCache) GetFavorites() []*WallpaperMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rows, err := c.db.Query(`
		SELECT id, path, original_url, hash, size, downloaded_at, last_used, use_count,
		       categories, purities, COALESCE(resolution, ''), is_favorite, rating
		FROM wallpapers
		WHERE is_favorite = 1
		ORDER BY rating DESC, last_used DESC
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	return c.scanWallpapers(rows)
}

// GetByRating returns wallpapers with at least the specified rating
func (c *WallpaperCache) GetByRating(minRating int) []*WallpaperMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rows, err := c.db.Query(`
		SELECT id, path, original_url, hash, size, downloaded_at, last_used, use_count,
		       categories, purities, COALESCE(resolution, ''), is_favorite, rating
		FROM wallpapers
		WHERE rating >= ?
		ORDER BY rating DESC, last_used DESC
	`, minRating)
	if err != nil {
		return nil
	}
	defer rows.Close()

	return c.scanWallpapers(rows)
}

// GetByTags returns wallpapers that have all the specified tags
func (c *WallpaperCache) GetByTags(tags []string) []*WallpaperMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(tags) == 0 {
		return nil
	}

	// Build query to find wallpapers with all specified tags
	query := `
		SELECT w.id, w.path, w.original_url, w.hash, w.size, w.downloaded_at, w.last_used,
		       w.use_count, w.categories, w.purities, COALESCE(w.resolution, ''), w.is_favorite, w.rating
		FROM wallpapers w
		WHERE (
			SELECT COUNT(DISTINCT tag)
			FROM wallpaper_tags
			WHERE wallpaper_id = w.id AND tag IN (?` + strings.Repeat(",?", len(tags)-1) + `)
		) = ?
		ORDER BY w.last_used DESC
	`

	args := make([]interface{}, len(tags)+1)
	for i, tag := range tags {
		args[i] = tag
	}
	args[len(tags)] = len(tags)

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	return c.scanWallpapers(rows)
}

// GetRandomFavorite returns a random favorite wallpaper
func (c *WallpaperCache) GetRandomFavorite() *WallpaperMetadata {
	favorites := c.GetFavorites()
	if len(favorites) == 0 {
		return nil
	}

	// Return a random favorite
	return favorites[rand.IntN(len(favorites))]
}

// EnforceCacheLimits removes least recently used wallpapers if cache exceeds limits
func (c *WallpaperCache) EnforceCacheLimits() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var totalCount int
	var totalSize int64
	c.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(size), 0) FROM wallpapers`).Scan(&totalCount, &totalSize)

	// Check if we're within limits
	if totalCount <= constants.MaxCacheSize && totalSize <= int64(constants.MaxCacheSizeMB)*1024*1024 {
		return nil
	}

	// Calculate targets (90% of max)
	targetCount := constants.MaxCacheSize * 90 / 100
	targetSize := int64(constants.MaxCacheSizeMB) * 1024 * 1024 * 90 / 100

	// Get wallpapers to remove (oldest, non-favorite first)
	rows, err := c.db.Query(`
		SELECT id, path, size
		FROM wallpapers
		WHERE is_favorite = 0
		ORDER BY last_used ASC
	`)
	if err != nil {
		return fmt.Errorf("failed to query wallpapers for cleanup: %w", err)
	}
	defer rows.Close()

	var removed int
	currentSize := totalSize
	currentCount := totalCount

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for rows.Next() && (currentCount > targetCount || currentSize > targetSize) {
		var id, path string
		var size int64
		if rows.Scan(&id, &path, &size) != nil {
			continue
		}

		// Remove file
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			slog.Warn("Failed to remove wallpaper during cache cleanup", "path", path, "error", err)
		}

		// Remove from database
		_, err := tx.Exec(`DELETE FROM wallpapers WHERE id = ?`, id)
		if err != nil {
			slog.Warn("Failed to delete wallpaper from database", "id", id, "error", err)
			continue
		}

		currentSize -= size
		currentCount--
		removed++
	}

	if removed > 0 {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit cleanup transaction: %w", err)
		}
		slog.Info("Enforced cache limits", "removed", removed, "remaining", currentCount)
	}

	return nil
}

// GetUsageHistory returns the usage history for a wallpaper
func (c *WallpaperCache) GetUsageHistory(id string, limit int) ([]time.Time, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	query := `SELECT used_at FROM usage_history WHERE wallpaper_id = ? ORDER BY used_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := c.db.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage history: %w", err)
	}
	defer rows.Close()

	var timestamps []time.Time
	for rows.Next() {
		var t time.Time
		if rows.Scan(&t) == nil {
			timestamps = append(timestamps, t)
		}
	}

	return timestamps, nil
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
