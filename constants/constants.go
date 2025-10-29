// Package constants defines application constants
package constants

// Time range constants for wallhaven search
const (
	Range1Day   = "1d"
	Range3Days  = "3d"
	Range1Week  = "1w"
	Range1Month = "1M"
	Range3Month = "3M"
	Range6Month = "6M"
	Range1Year  = "1y"
)

// Valid time ranges
var ValidRanges = []string{
	Range1Day, Range3Days, Range1Week, Range1Month,
	Range3Month, Range6Month, Range1Year,
}

// Sort order constants
const (
	SortRelevance  = "relevance"
	SortRandom     = "random"
	SortDateAdded  = "date_added"
	SortViews      = "views"
	SortFavorites  = "favorites"
	SortToplist    = "toplist"
)

// Valid sort orders
var ValidSorts = []string{
	SortRelevance, SortRandom, SortDateAdded,
	SortViews, SortFavorites, SortToplist,
}

// Order constants
const (
	OrderAsc  = "asc"
	OrderDesc = "desc"
)

// Valid orders
var ValidOrders = []string{OrderAsc, OrderDesc}

// Cleanup mode constants
const (
	CleanupModeUnused  = "unused"
	CleanupModeOld     = "old"
	CleanupModeInvalid = "invalid"
)

// Valid cleanup modes
var ValidCleanupModes = []string{
	CleanupModeUnused, CleanupModeOld, CleanupModeInvalid,
}

// Default values
const (
	DefaultRange          = Range1Year
	DefaultPurity         = "110" // SFW + Sketchy
	DefaultCategories     = "010" // Anime only
	DefaultSort           = SortToplist
	DefaultOrder          = OrderDesc
	DefaultMaxPages       = 5
	DefaultAtLeast        = "2560x1440"
	DefaultCleanupOlderThan = "30d"
)

// Default ratios
var DefaultRatios = []string{"16x9", "16x10"}

// Application constants
const (
	AppName     = "wallhaven_dl"
	AppVersion  = "2.0.0"
	UserAgent   = "wallhaven_dl/2.0"
	CacheDir    = ".cache"
	MetadataFile = "metadata.json"
)

// HTTP constants
const (
	MaxRetries        = 3
	RequestTimeout    = 30 // seconds
	MaxIdleConns      = 10
	MaxIdleConnsPerHost = 2
	IdleConnTimeout   = 30 // seconds
	RetryDelaySeconds = 1
)

// Cache constants
const (
	MaxHistorySize   = 100
	MaxCacheSize     = 1000 // Maximum number of wallpapers in cache
	MaxCacheSizeMB   = 5000 // Maximum cache size in megabytes (5GB)
	MinRating        = 1
	MaxRating        = 5
)

// File permission constants
const (
	DirPermissions  = 0o755
	FilePermissions = 0o644
)