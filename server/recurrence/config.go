package recurrence

import (
	"time"
)

// EngineConfig holds configuration options for the recurrence engine
type EngineConfig struct {
	// Cache configuration
	CacheEnabled bool
	CacheConfig  CacheConfig

	// Performance tuning
	MaxExpansionOccurrences int           // Maximum occurrences to check in HasOccurrenceInRange
	LargeRangeThreshold     time.Duration // Threshold for "large" time ranges that get limited expansion
	LargeRangeLimit         time.Duration // Limit for expansion when range exceeds threshold
}

// DefaultEngineConfig provides sensible defaults for production use
var DefaultEngineConfig = EngineConfig{
	CacheEnabled: true,
	CacheConfig:  DefaultCacheConfig,

	MaxExpansionOccurrences: 100,
	LargeRangeThreshold:     90 * 24 * time.Hour, // 90 days
	LargeRangeLimit:         90 * 24 * time.Hour, // Limit to 90 days expansion
}

// HighPerformanceConfig is optimized for high-traffic scenarios
var HighPerformanceConfig = EngineConfig{
	CacheEnabled: true,
	CacheConfig: CacheConfig{
		TTL:             30 * time.Minute, // Longer cache TTL
		MaxEntries:      5000,             // More cache entries
		CleanupInterval: 10 * time.Minute, // Less frequent cleanup
	},

	MaxExpansionOccurrences: 50,                  // Fewer occurrences checked for speed
	LargeRangeThreshold:     30 * 24 * time.Hour, // Shorter threshold
	LargeRangeLimit:         30 * 24 * time.Hour, // Shorter limit
}

// LowMemoryConfig is optimized for memory-constrained environments
var LowMemoryConfig = EngineConfig{
	CacheEnabled: true,
	CacheConfig: CacheConfig{
		TTL:             5 * time.Minute, // Shorter cache TTL
		MaxEntries:      100,             // Fewer cache entries
		CleanupInterval: 2 * time.Minute, // More frequent cleanup
	},

	MaxExpansionOccurrences: 200,                  // More thorough checking
	LargeRangeThreshold:     180 * 24 * time.Hour, // Longer threshold
	LargeRangeLimit:         180 * 24 * time.Hour, // Longer limit
}

// DisabledCacheConfig turns off caching entirely
var DisabledCacheConfig = EngineConfig{
	CacheEnabled: false,
	CacheConfig:  CacheConfig{}, // Not used

	MaxExpansionOccurrences: 1000,                 // More thorough without cache
	LargeRangeThreshold:     365 * 24 * time.Hour, // Very long threshold
	LargeRangeLimit:         365 * 24 * time.Hour, // Very long limit
}

// NewEngineWithConfig creates a new recurrence engine with custom configuration
func NewEngineWithConfig(config EngineConfig) *Engine {
	var cache *RecurrenceCache
	if config.CacheEnabled {
		cache = NewRecurrenceCache(config.CacheConfig)
	}

	return &Engine{
		cache:  cache,
		config: config,
	}
}
