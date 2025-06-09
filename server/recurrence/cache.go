package recurrence

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// CacheEntry represents a cached recurrence result
type CacheEntry struct {
	Result     interface{} // Can store bool for HasOccurrence or []time.Time for expansion
	ExpiresAt  time.Time
	AccessedAt time.Time
}

// RecurrenceCache provides caching for recurrence expansion and validation results
type RecurrenceCache struct {
	entries         map[string]*CacheEntry
	mutex           sync.RWMutex
	ttl             time.Duration
	maxEntries      int
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// CacheConfig holds configuration for the recurrence cache
type CacheConfig struct {
	TTL             time.Duration // How long entries stay valid
	MaxEntries      int           // Maximum number of entries before cleanup
	CleanupInterval time.Duration // How often to run cleanup
}

// DefaultCacheConfig provides sensible defaults for recurrence caching
var DefaultCacheConfig = CacheConfig{
	TTL:             15 * time.Minute, // Cache results for 15 minutes
	MaxEntries:      1000,             // Keep up to 1000 cached results
	CleanupInterval: 5 * time.Minute,  // Cleanup every 5 minutes
}

// NewRecurrenceCache creates a new recurrence cache with the given configuration
func NewRecurrenceCache(config CacheConfig) *RecurrenceCache {
	cache := &RecurrenceCache{
		entries:         make(map[string]*CacheEntry),
		ttl:             config.TTL,
		maxEntries:      config.MaxEntries,
		cleanupInterval: config.CleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// generateCacheKey creates a unique key for the cache based on input parameters
func (c *RecurrenceCache) generateCacheKey(operation string, masterStart, masterEnd time.Time, recInfo RecurrenceInfo, rangeStart, rangeEnd time.Time) string {
	// Create a hash of all relevant parameters
	hasher := sha256.New()

	// Include operation type
	hasher.Write([]byte(operation))

	// Include time parameters
	hasher.Write([]byte(masterStart.Format(time.RFC3339Nano)))
	hasher.Write([]byte(masterEnd.Format(time.RFC3339Nano)))
	hasher.Write([]byte(rangeStart.Format(time.RFC3339Nano)))
	hasher.Write([]byte(rangeEnd.Format(time.RFC3339Nano)))

	// Include recurrence info
	hasher.Write([]byte(recInfo.RRULE))

	// Include RDATE
	for _, rdate := range recInfo.RDATE {
		hasher.Write([]byte(rdate.Format(time.RFC3339Nano)))
	}

	// Include EXDATE
	for _, exdate := range recInfo.EXDATE {
		hasher.Write([]byte(exdate.Format(time.RFC3339Nano)))
	}

	// Include RecurrenceID if present
	if recInfo.RecurrenceID != nil {
		hasher.Write([]byte(recInfo.RecurrenceID.Format(time.RFC3339Nano)))
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// Get retrieves a cached result if it exists and hasn't expired
func (c *RecurrenceCache) Get(operation string, masterStart, masterEnd time.Time, recInfo RecurrenceInfo, rangeStart, rangeEnd time.Time) (interface{}, bool) {
	key := c.generateCacheKey(operation, masterStart, masterEnd, recInfo, rangeStart, rangeEnd)

	c.mutex.RLock()
	entry, exists := c.entries[key]
	c.mutex.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if entry has expired
	now := time.Now()
	if now.After(entry.ExpiresAt) {
		// Entry expired, remove it
		c.mutex.Lock()
		delete(c.entries, key)
		c.mutex.Unlock()
		return nil, false
	}

	// Update access time
	c.mutex.Lock()
	entry.AccessedAt = now
	c.mutex.Unlock()

	return entry.Result, true
}

// Set stores a result in the cache
func (c *RecurrenceCache) Set(operation string, masterStart, masterEnd time.Time, recInfo RecurrenceInfo, rangeStart, rangeEnd time.Time, result interface{}) {
	key := c.generateCacheKey(operation, masterStart, masterEnd, recInfo, rangeStart, rangeEnd)
	now := time.Now()

	entry := &CacheEntry{
		Result:     result,
		ExpiresAt:  now.Add(c.ttl),
		AccessedAt: now,
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries[key] = entry

	// If we're over the limit, trigger cleanup
	if len(c.entries) > c.maxEntries {
		c.cleanup()
	}
}

// cleanup removes expired entries and oldest entries if over limit
func (c *RecurrenceCache) cleanup() {
	now := time.Now()

	// Remove expired entries
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}

	// If still over limit, remove least recently accessed entries
	if len(c.entries) > c.maxEntries {
		// Create a slice of keys sorted by access time
		type keyAccess struct {
			key        string
			accessedAt time.Time
		}

		var keyAccessList []keyAccess
		for key, entry := range c.entries {
			keyAccessList = append(keyAccessList, keyAccess{
				key:        key,
				accessedAt: entry.AccessedAt,
			})
		}

		// Sort by access time (oldest first)
		for i := 0; i < len(keyAccessList)-1; i++ {
			for j := i + 1; j < len(keyAccessList); j++ {
				if keyAccessList[i].accessedAt.After(keyAccessList[j].accessedAt) {
					keyAccessList[i], keyAccessList[j] = keyAccessList[j], keyAccessList[i]
				}
			}
		}

		// Remove oldest entries to get under the limit
		entriesToRemove := len(c.entries) - c.maxEntries
		for i := 0; i < entriesToRemove && i < len(keyAccessList); i++ {
			delete(c.entries, keyAccessList[i].key)
		}
	}
}

// cleanupLoop runs periodic cleanup
func (c *RecurrenceCache) cleanupLoop() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mutex.Lock()
			c.cleanup()
			c.mutex.Unlock()
		case <-c.stopCleanup:
			return
		}
	}
}

// Close stops the cleanup goroutine and clears the cache
func (c *RecurrenceCache) Close() {
	close(c.stopCleanup)
	c.mutex.Lock()
	c.entries = make(map[string]*CacheEntry)
	c.mutex.Unlock()
}

// Stats returns cache statistics
func (c *RecurrenceCache) Stats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entryCount := len(c.entries)
	expiredCount := 0
	now := time.Now()

	for _, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			expiredCount++
		}
	}

	return CacheStats{
		TotalEntries:   entryCount,
		ExpiredEntries: expiredCount,
		ActiveEntries:  entryCount - expiredCount,
	}
}

// CacheStats provides information about cache performance
type CacheStats struct {
	TotalEntries   int
	ExpiredEntries int
	ActiveEntries  int
}
