package wallhaven

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWallpaperCache(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, ".cache")

	cache, err := NewWallpaperCache(cacheDir)
	if err != nil {
		t.Fatalf("NewWallpaperCache() error = %v", err)
	}
	defer cache.Close()

	if cache == nil {
		t.Fatal("Expected cache to be non-nil")
	}

	// Verify database was created
	dbPath := filepath.Join(cacheDir, "wallpapers.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Expected database file to exist")
	}

	// Check initial state
	stats := cache.GetStatistics()
	if stats["total_wallpapers"].(int) != 0 {
		t.Errorf("Expected empty cache, got %d wallpapers", stats["total_wallpapers"])
	}
}

func TestWallpaperCache_AddWallpaper(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, ".cache")

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.jpg")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	cache, err := NewWallpaperCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	wallpaper := &Wallpaper{
		Path: "https://example.com/test.jpg",
	}

	err = cache.AddWallpaper(wallpaper, testFile, "010", "110")
	if err != nil {
		t.Fatalf("AddWallpaper() error = %v", err)
	}

	// Verify wallpaper was added
	stats := cache.GetStatistics()
	if stats["total_wallpapers"].(int) != 1 {
		t.Errorf("Expected 1 wallpaper, got %d", stats["total_wallpapers"])
	}

	// Verify we can retrieve it
	current := cache.GetCurrent()
	if current == nil {
		t.Fatal("Expected to find current wallpaper")
	}

	if current.Path != testFile {
		t.Errorf("Expected path %s, got %s", testFile, current.Path)
	}

	if current.Categories != "010" {
		t.Errorf("Expected categories '010', got '%s'", current.Categories)
	}

	if current.Purities != "110" {
		t.Errorf("Expected purities '110', got '%s'", current.Purities)
	}
}

func TestWallpaperCache_MarkAsUsed(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, ".cache")

	testFile := filepath.Join(tmpDir, "test.jpg")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	cache, err := NewWallpaperCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	wallpaper := &Wallpaper{
		Path: "https://example.com/test.jpg",
	}

	err = cache.AddWallpaper(wallpaper, testFile, "010", "110")
	if err != nil {
		t.Fatal(err)
	}

	id := GenerateID(wallpaper.Path)

	// Get initial state
	current := cache.GetCurrent()
	if current == nil {
		t.Fatal("Expected to find current wallpaper")
	}
	initialUseCount := current.UseCount
	initialLastUsed := current.LastUsed

	time.Sleep(10 * time.Millisecond) // Ensure time difference

	err = cache.MarkAsUsed(id)
	if err != nil {
		t.Fatalf("MarkAsUsed() error = %v", err)
	}

	// Get updated state
	current = cache.GetCurrent()
	if current == nil {
		t.Fatal("Expected to find current wallpaper after update")
	}

	if current.UseCount != initialUseCount+1 {
		t.Errorf("Expected use count %d, got %d", initialUseCount+1, current.UseCount)
	}

	if !current.LastUsed.After(initialLastUsed) {
		t.Error("Expected LastUsed to be updated")
	}

	// Verify usage history was recorded
	history, err := cache.GetUsageHistory(id, 10)
	if err != nil {
		t.Fatalf("GetUsageHistory() error = %v", err)
	}

	// Should have 2 entries: initial add + mark as used
	if len(history) != 2 {
		t.Errorf("Expected 2 usage history entries, got %d", len(history))
	}
}

func TestWallpaperCache_Favorites(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, ".cache")

	testFile := filepath.Join(tmpDir, "test.jpg")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	cache, err := NewWallpaperCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	wallpaper := &Wallpaper{
		Path: "https://example.com/test.jpg",
	}

	err = cache.AddWallpaper(wallpaper, testFile, "010", "110")
	if err != nil {
		t.Fatal(err)
	}

	id := GenerateID(wallpaper.Path)

	// Initially not a favorite
	favorites := cache.GetFavorites()
	if len(favorites) != 0 {
		t.Error("Expected no favorites initially")
	}

	// Toggle to favorite
	err = cache.ToggleFavorite(id)
	if err != nil {
		t.Fatalf("ToggleFavorite() error = %v", err)
	}

	// Should now be a favorite
	favorites = cache.GetFavorites()
	if len(favorites) != 1 {
		t.Errorf("Expected 1 favorite, got %d", len(favorites))
	}

	// Toggle again to remove from favorites
	err = cache.ToggleFavorite(id)
	if err != nil {
		t.Fatalf("ToggleFavorite() error = %v", err)
	}

	favorites = cache.GetFavorites()
	if len(favorites) != 0 {
		t.Error("Expected no favorites after toggling off")
	}
}

func TestWallpaperCache_Rating(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, ".cache")

	testFile := filepath.Join(tmpDir, "test.jpg")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	cache, err := NewWallpaperCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	wallpaper := &Wallpaper{
		Path: "https://example.com/test.jpg",
	}

	err = cache.AddWallpaper(wallpaper, testFile, "010", "110")
	if err != nil {
		t.Fatal(err)
	}

	id := GenerateID(wallpaper.Path)

	// Set rating
	err = cache.SetRating(id, 4)
	if err != nil {
		t.Fatalf("SetRating() error = %v", err)
	}

	// Verify rating was set
	current := cache.GetCurrent()
	if current == nil {
		t.Fatal("Expected to find current wallpaper")
	}

	if current.Rating != 4 {
		t.Errorf("Expected rating 4, got %d", current.Rating)
	}

	// Test GetByRating
	rated := cache.GetByRating(3)
	if len(rated) != 1 {
		t.Errorf("Expected 1 wallpaper with rating >= 3, got %d", len(rated))
	}

	rated = cache.GetByRating(5)
	if len(rated) != 0 {
		t.Errorf("Expected 0 wallpapers with rating >= 5, got %d", len(rated))
	}
}

func TestWallpaperCache_Tags(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, ".cache")

	testFile := filepath.Join(tmpDir, "test.jpg")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	cache, err := NewWallpaperCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	wallpaper := &Wallpaper{
		Path: "https://example.com/test.jpg",
	}

	err = cache.AddWallpaper(wallpaper, testFile, "010", "110")
	if err != nil {
		t.Fatal(err)
	}

	id := GenerateID(wallpaper.Path)

	// Add tags
	tags := []string{"nature", "mountains"}
	err = cache.AddTags(id, tags)
	if err != nil {
		t.Fatalf("AddTags() error = %v", err)
	}

	// Verify tags were added
	current := cache.GetCurrent()
	if current == nil {
		t.Fatal("Expected to find current wallpaper")
	}

	if len(current.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(current.Tags))
	}

	// Test GetByTags
	tagged := cache.GetByTags([]string{"nature"})
	if len(tagged) != 1 {
		t.Errorf("Expected 1 wallpaper with tag 'nature', got %d", len(tagged))
	}

	// Remove one tag
	err = cache.RemoveTags(id, []string{"mountains"})
	if err != nil {
		t.Fatalf("RemoveTags() error = %v", err)
	}

	current = cache.GetCurrent()
	if len(current.Tags) != 1 {
		t.Errorf("Expected 1 tag after removal, got %d", len(current.Tags))
	}
}

func TestGenerateID(t *testing.T) {
	url1 := "https://example.com/test1.jpg"
	url2 := "https://example.com/test2.jpg"

	id1 := GenerateID(url1)
	id2 := GenerateID(url2)

	if id1 == id2 {
		t.Error("Expected different IDs for different URLs")
	}

	if len(id1) != 16 {
		t.Errorf("Expected ID length 16, got %d", len(id1))
	}

	// Same URL should generate same ID
	id1Again := GenerateID(url1)
	if id1 != id1Again {
		t.Error("Expected same ID for same URL")
	}
}
