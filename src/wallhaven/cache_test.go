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

	if cache == nil {
		t.Fatal("Expected cache to be non-nil")
	}

	if len(cache.wallpapers) != 0 {
		t.Errorf("Expected empty wallpapers map, got %d items", len(cache.wallpapers))
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

	wallpaper := &Wallpaper{
		Path: "https://example.com/test.jpg",
	}

	err = cache.AddWallpaper(wallpaper, testFile, "010", "110")
	if err != nil {
		t.Fatalf("AddWallpaper() error = %v", err)
	}

	if len(cache.wallpapers) != 1 {
		t.Errorf("Expected 1 wallpaper, got %d", len(cache.wallpapers))
	}

	id := GenerateID(wallpaper.Path)
	metadata, exists := cache.wallpapers[id]
	if !exists {
		t.Error("Wallpaper not found in cache")
	}

	if metadata.Path != testFile {
		t.Errorf("Expected path %s, got %s", testFile, metadata.Path)
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

	wallpaper := &Wallpaper{
		Path: "https://example.com/test.jpg",
	}

	err = cache.AddWallpaper(wallpaper, testFile, "010", "110")
	if err != nil {
		t.Fatal(err)
	}

	id := GenerateID(wallpaper.Path)
	initialUseCount := cache.wallpapers[id].UseCount
	initialLastUsed := cache.wallpapers[id].LastUsed

	time.Sleep(10 * time.Millisecond) // Ensure time difference

	err = cache.MarkAsUsed(id)
	if err != nil {
		t.Fatalf("MarkAsUsed() error = %v", err)
	}

	if cache.wallpapers[id].UseCount != initialUseCount+1 {
		t.Errorf("Expected use count %d, got %d", initialUseCount+1, cache.wallpapers[id].UseCount)
	}

	if !cache.wallpapers[id].LastUsed.After(initialLastUsed) {
		t.Error("Expected LastUsed to be updated")
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