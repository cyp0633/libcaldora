package recurrence

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRecurrenceCache_BasicOperations(t *testing.T) {
	cache := NewRecurrenceCache(CacheConfig{
		TTL:             5 * time.Minute,
		MaxEntries:      100,
		CleanupInterval: 1 * time.Minute,
	})
	defer cache.Close()

	// Test basic get/set
	masterStart := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	masterEnd := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	recInfo := RecurrenceInfo{
		RRULE: "FREQ=DAILY;COUNT=5",
	}

	// Cache miss first
	result, found := cache.Get("test", masterStart, masterEnd, recInfo, rangeStart, rangeEnd)
	if found {
		t.Error("Expected cache miss, got hit")
	}
	if result != nil {
		t.Error("Expected nil result on cache miss")
	}

	// Set value
	cache.Set("test", masterStart, masterEnd, recInfo, rangeStart, rangeEnd, true)

	// Cache hit
	result, found = cache.Get("test", masterStart, masterEnd, recInfo, rangeStart, rangeEnd)
	if !found {
		t.Error("Expected cache hit, got miss")
	}
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}
}

func TestRecurrenceCache_TTLExpiration(t *testing.T) {
	cache := NewRecurrenceCache(CacheConfig{
		TTL:             100 * time.Millisecond, // Very short TTL for testing
		MaxEntries:      100,
		CleanupInterval: 50 * time.Millisecond,
	})
	defer cache.Close()

	masterStart := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	masterEnd := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	recInfo := RecurrenceInfo{
		RRULE: "FREQ=DAILY;COUNT=5",
	}

	// Set value
	cache.Set("test", masterStart, masterEnd, recInfo, rangeStart, rangeEnd, true)

	// Should be found immediately
	result, found := cache.Get("test", masterStart, masterEnd, recInfo, rangeStart, rangeEnd)
	if !found || result != true {
		t.Error("Expected cache hit immediately after set")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired now
	_, found = cache.Get("test", masterStart, masterEnd, recInfo, rangeStart, rangeEnd)
	if found {
		t.Error("Expected cache miss after TTL expiration")
	}
}

func TestRecurrenceCache_DifferentKeys(t *testing.T) {
	cache := NewRecurrenceCache(DefaultCacheConfig)
	defer cache.Close()

	masterStart := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	masterEnd := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	recInfo1 := RecurrenceInfo{RRULE: "FREQ=DAILY;COUNT=5"}
	recInfo2 := RecurrenceInfo{RRULE: "FREQ=WEEKLY;COUNT=5"}

	// Set different values for different recurrence rules
	cache.Set("test", masterStart, masterEnd, recInfo1, rangeStart, rangeEnd, true)
	cache.Set("test", masterStart, masterEnd, recInfo2, rangeStart, rangeEnd, false)

	// Verify both are cached separately
	result1, found1 := cache.Get("test", masterStart, masterEnd, recInfo1, rangeStart, rangeEnd)
	result2, found2 := cache.Get("test", masterStart, masterEnd, recInfo2, rangeStart, rangeEnd)

	if !found1 || result1 != true {
		t.Error("Expected first cache entry to be true")
	}
	if !found2 || result2 != false {
		t.Error("Expected second cache entry to be false")
	}
}

func TestRecurrenceCache_Stats(t *testing.T) {
	cache := NewRecurrenceCache(DefaultCacheConfig)
	defer cache.Close()

	// Initial stats
	stats := cache.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 initial entries, got %d", stats.TotalEntries)
	}

	// Add some entries
	masterStart := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	masterEnd := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		recInfo := RecurrenceInfo{RRULE: "FREQ=DAILY;COUNT=" + string(rune('0'+i))}
		cache.Set("test", masterStart, masterEnd, recInfo, rangeStart, rangeEnd, true)
	}

	stats = cache.Stats()
	if stats.TotalEntries != 5 {
		t.Errorf("Expected 5 entries, got %d", stats.TotalEntries)
	}
	if stats.ActiveEntries != 5 {
		t.Errorf("Expected 5 active entries, got %d", stats.ActiveEntries)
	}
}

// Test cache size limits and LRU eviction
func TestRecurrenceCache_MaxEntriesEviction(t *testing.T) {
	cache := NewRecurrenceCache(CacheConfig{
		TTL:             5 * time.Minute,
		MaxEntries:      3, // Small limit for testing
		CleanupInterval: 1 * time.Minute,
	})
	defer cache.Close()

	masterStart := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	masterEnd := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Add entries up to limit
	for i := 0; i < 3; i++ {
		recInfo := RecurrenceInfo{RRULE: fmt.Sprintf("FREQ=DAILY;COUNT=%d", i+1)}
		cache.Set("test", masterStart, masterEnd, recInfo, rangeStart, rangeEnd, true)
	}

	// Verify all entries are present
	stats := cache.Stats()
	if stats.TotalEntries != 3 {
		t.Errorf("Expected 3 entries, got %d", stats.TotalEntries)
	}

	// Add one more entry, should trigger eviction
	recInfo4 := RecurrenceInfo{RRULE: "FREQ=WEEKLY;COUNT=1"}
	cache.Set("test", masterStart, masterEnd, recInfo4, rangeStart, rangeEnd, false)

	// Should still have max entries
	stats = cache.Stats()
	if stats.TotalEntries != 3 {
		t.Errorf("Expected 3 entries after eviction, got %d", stats.TotalEntries)
	}

	// Newest entry should be present
	result, found := cache.Get("test", masterStart, masterEnd, recInfo4, rangeStart, rangeEnd)
	if !found || result != false {
		t.Error("Expected newest entry to be present after eviction")
	}

	// Oldest entry should be evicted (LRU)
	recInfo1 := RecurrenceInfo{RRULE: "FREQ=DAILY;COUNT=1"}
	_, found = cache.Get("test", masterStart, masterEnd, recInfo1, rangeStart, rangeEnd)
	if found {
		t.Error("Expected oldest entry to be evicted")
	}
}

// Test concurrent access to cache
func TestRecurrenceCache_ConcurrentAccess(t *testing.T) {
	cache := NewRecurrenceCache(CacheConfig{
		TTL:             5 * time.Minute,
		MaxEntries:      100,
		CleanupInterval: 1 * time.Minute,
	})
	defer cache.Close()

	const numGoroutines = 10
	const operationsPerGoroutine = 100

	masterStart := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	masterEnd := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup

	// Run concurrent read/write operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				recInfo := RecurrenceInfo{
					RRULE: fmt.Sprintf("FREQ=DAILY;COUNT=%d", goroutineID*operationsPerGoroutine+j),
				}

				// Mix of reads and writes
				if j%2 == 0 {
					// Write operation
					cache.Set("test", masterStart, masterEnd, recInfo, rangeStart, rangeEnd, true)
				} else {
					// Read operation
					cache.Get("test", masterStart, masterEnd, recInfo, rangeStart, rangeEnd)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is still functional after concurrent access
	testRecInfo := RecurrenceInfo{RRULE: "FREQ=DAILY;COUNT=999"}
	cache.Set("test", masterStart, masterEnd, testRecInfo, rangeStart, rangeEnd, true)
	result, found := cache.Get("test", masterStart, masterEnd, testRecInfo, rangeStart, rangeEnd)

	if !found || result != true {
		t.Error("Cache should still be functional after concurrent access")
	}
}

// Test cache key generation with different parameter combinations
func TestRecurrenceCache_KeyGeneration(t *testing.T) {
	cache := NewRecurrenceCache(DefaultCacheConfig)
	defer cache.Close()

	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	testCases := []struct {
		name         string
		eventID      string
		masterStart  time.Time
		masterEnd    time.Time
		recInfo      RecurrenceInfo
		rangeStart   time.Time
		rangeEnd     time.Time
		expectedDiff bool // Should this generate a different key than the base case?
	}{
		{
			name:         "Base case",
			eventID:      "event1",
			masterStart:  baseTime,
			masterEnd:    baseTime.Add(time.Hour),
			recInfo:      RecurrenceInfo{RRULE: "FREQ=DAILY;COUNT=5"},
			rangeStart:   baseTime.Add(-time.Hour),
			rangeEnd:     baseTime.Add(2 * time.Hour),
			expectedDiff: false,
		},
		{
			name:         "Different event ID",
			eventID:      "event2",
			masterStart:  baseTime,
			masterEnd:    baseTime.Add(time.Hour),
			recInfo:      RecurrenceInfo{RRULE: "FREQ=DAILY;COUNT=5"},
			rangeStart:   baseTime.Add(-time.Hour),
			rangeEnd:     baseTime.Add(2 * time.Hour),
			expectedDiff: true,
		},
		{
			name:         "Different master start",
			eventID:      "event1",
			masterStart:  baseTime.Add(time.Minute),
			masterEnd:    baseTime.Add(time.Hour),
			recInfo:      RecurrenceInfo{RRULE: "FREQ=DAILY;COUNT=5"},
			rangeStart:   baseTime.Add(-time.Hour),
			rangeEnd:     baseTime.Add(2 * time.Hour),
			expectedDiff: true,
		},
		{
			name:         "Different RRULE",
			eventID:      "event1",
			masterStart:  baseTime,
			masterEnd:    baseTime.Add(time.Hour),
			recInfo:      RecurrenceInfo{RRULE: "FREQ=WEEKLY;COUNT=5"},
			rangeStart:   baseTime.Add(-time.Hour),
			rangeEnd:     baseTime.Add(2 * time.Hour),
			expectedDiff: true,
		},
		{
			name:         "Different range start",
			eventID:      "event1",
			masterStart:  baseTime,
			masterEnd:    baseTime.Add(time.Hour),
			recInfo:      RecurrenceInfo{RRULE: "FREQ=DAILY;COUNT=5"},
			rangeStart:   baseTime.Add(-2 * time.Hour),
			rangeEnd:     baseTime.Add(2 * time.Hour),
			expectedDiff: true,
		},
		{
			name:        "With EXDATE",
			eventID:     "event1",
			masterStart: baseTime,
			masterEnd:   baseTime.Add(time.Hour),
			recInfo: RecurrenceInfo{
				RRULE:  "FREQ=DAILY;COUNT=5",
				EXDATE: []time.Time{baseTime.Add(24 * time.Hour)},
			},
			rangeStart:   baseTime.Add(-time.Hour),
			rangeEnd:     baseTime.Add(2 * time.Hour),
			expectedDiff: true,
		},
		{
			name:        "With RDATE",
			eventID:     "event1",
			masterStart: baseTime,
			masterEnd:   baseTime.Add(time.Hour),
			recInfo: RecurrenceInfo{
				RRULE: "FREQ=DAILY;COUNT=5",
				RDATE: []time.Time{baseTime.Add(7 * 24 * time.Hour)},
			},
			rangeStart:   baseTime.Add(-time.Hour),
			rangeEnd:     baseTime.Add(2 * time.Hour),
			expectedDiff: true,
		},
	}

	// Set base case
	baseCase := testCases[0]
	cache.Set(baseCase.eventID, baseCase.masterStart, baseCase.masterEnd,
		baseCase.recInfo, baseCase.rangeStart, baseCase.rangeEnd, true)

	for _, tc := range testCases[1:] {
		t.Run(tc.name, func(t *testing.T) {
			// Set the test case
			cache.Set(tc.eventID, tc.masterStart, tc.masterEnd,
				tc.recInfo, tc.rangeStart, tc.rangeEnd, false)

			// Check if base case is still there
			result, found := cache.Get(baseCase.eventID, baseCase.masterStart, baseCase.masterEnd,
				baseCase.recInfo, baseCase.rangeStart, baseCase.rangeEnd)

			if tc.expectedDiff {
				// Should not affect base case
				if !found || result != true {
					t.Errorf("Test case '%s' should generate different key but affected base case", tc.name)
				}
			}

			// Check test case is stored correctly
			result, found = cache.Get(tc.eventID, tc.masterStart, tc.masterEnd,
				tc.recInfo, tc.rangeStart, tc.rangeEnd)
			if !found || result != false {
				t.Errorf("Test case '%s' was not stored correctly", tc.name)
			}
		})
	}
}

// Test cache behavior with extreme values
func TestRecurrenceCache_ExtremeValues(t *testing.T) {
	cache := NewRecurrenceCache(DefaultCacheConfig)
	defer cache.Close()

	// Test with very old dates
	oldDate := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	recInfo := RecurrenceInfo{RRULE: "FREQ=YEARLY;COUNT=1"}

	cache.Set("old", oldDate, oldDate.Add(time.Hour), recInfo,
		oldDate.Add(-time.Hour), oldDate.Add(2*time.Hour), true)

	result, found := cache.Get("old", oldDate, oldDate.Add(time.Hour), recInfo,
		oldDate.Add(-time.Hour), oldDate.Add(2*time.Hour))
	if !found || result != true {
		t.Error("Cache should handle very old dates")
	}

	// Test with very future dates
	futureDate := time.Date(2200, 12, 31, 23, 59, 59, 0, time.UTC)
	cache.Set("future", futureDate, futureDate.Add(time.Hour), recInfo,
		futureDate.Add(-time.Hour), futureDate.Add(2*time.Hour), false)

	result, found = cache.Get("future", futureDate, futureDate.Add(time.Hour), recInfo,
		futureDate.Add(-time.Hour), futureDate.Add(2*time.Hour))
	if !found || result != false {
		t.Error("Cache should handle very future dates")
	}

	// Test with very long RRULE
	longRRULE := "FREQ=DAILY;BYDAY=MO,TU,WE,TH,FR,SA,SU;BYMONTH=1,2,3,4,5,6,7,8,9,10,11,12;COUNT=1000"
	longRecInfo := RecurrenceInfo{RRULE: longRRULE}

	now := time.Now()
	cache.Set("long", now, now.Add(time.Hour), longRecInfo,
		now.Add(-time.Hour), now.Add(2*time.Hour), true)

	result, found = cache.Get("long", now, now.Add(time.Hour), longRecInfo,
		now.Add(-time.Hour), now.Add(2*time.Hour))
	if !found || result != true {
		t.Error("Cache should handle very long RRULE strings")
	}
}

// Test cleanup behavior in detail
func TestRecurrenceCache_DetailedCleanup(t *testing.T) {
	cache := NewRecurrenceCache(CacheConfig{
		TTL:             200 * time.Millisecond,
		MaxEntries:      10,
		CleanupInterval: 100 * time.Millisecond,
	})
	defer cache.Close()

	masterStart := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	masterEnd := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Add multiple entries
	for i := 0; i < 5; i++ {
		recInfo := RecurrenceInfo{RRULE: fmt.Sprintf("FREQ=DAILY;COUNT=%d", i+1)}
		cache.Set("test", masterStart, masterEnd, recInfo, rangeStart, rangeEnd, true)
	}

	// Verify all entries are present
	stats := cache.Stats()
	if stats.TotalEntries != 5 {
		t.Errorf("Expected 5 entries, got %d", stats.TotalEntries)
	}

	// Wait for first cleanup cycle
	time.Sleep(150 * time.Millisecond)

	// Entries should still be there (TTL not expired yet)
	stats = cache.Stats()
	if stats.TotalEntries != 5 {
		t.Errorf("Expected 5 entries after first cleanup, got %d", stats.TotalEntries)
	}

	// Wait for TTL expiration
	time.Sleep(100 * time.Millisecond)

	// Wait for next cleanup cycle
	time.Sleep(150 * time.Millisecond)

	// Now entries should be cleaned up
	stats = cache.Stats()
	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 entries after TTL expiration and cleanup, got %d", stats.TotalEntries)
	}
}

// Test cache performance under load
func TestRecurrenceCache_PerformanceUnderLoad(t *testing.T) {
	cache := NewRecurrenceCache(CacheConfig{
		TTL:             5 * time.Minute,
		MaxEntries:      1000,
		CleanupInterval: 1 * time.Minute,
	})
	defer cache.Close()

	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// Measure time for many operations
	start := time.Now()

	const numOperations = 1000
	hitCount := 0

	for i := 0; i < numOperations; i++ {
		recInfo := RecurrenceInfo{RRULE: fmt.Sprintf("FREQ=DAILY;COUNT=%d", i%100+1)}

		// Try to get first (should be miss)
		_, found := cache.Get("test", baseTime, baseTime.Add(time.Hour), recInfo,
			baseTime.Add(-time.Hour), baseTime.Add(2*time.Hour))

		if found {
			hitCount++
		}

		// Set the value
		cache.Set("test", baseTime, baseTime.Add(time.Hour), recInfo,
			baseTime.Add(-time.Hour), baseTime.Add(2*time.Hour), true)

		// Get again (should be hit)
		_, found = cache.Get("test", baseTime, baseTime.Add(time.Hour), recInfo,
			baseTime.Add(-time.Hour), baseTime.Add(2*time.Hour))

		if found {
			hitCount++
		}
	}

	duration := time.Since(start)

	// Should have reasonable performance (less than 1ms per operation)
	avgTimePerOp := duration / (numOperations * 2) // 2 operations per iteration
	if avgTimePerOp > time.Millisecond {
		t.Errorf("Performance too slow: %v per operation", avgTimePerOp)
	}

	// Should have good hit rate on second access
	expectedHits := numOperations // One hit per iteration on second get
	if hitCount < expectedHits {
		t.Errorf("Expected at least %d hits, got %d", expectedHits, hitCount)
	}

	t.Logf("Performance test: %d operations in %v (avg: %v per op)",
		numOperations*2, duration, avgTimePerOp)
	t.Logf("Hit rate: %d/%d hits", hitCount, numOperations*2)
}

// Test edge cases with empty and nil values
func TestRecurrenceCache_EdgeCasesEmptyValues(t *testing.T) {
	cache := NewRecurrenceCache(DefaultCacheConfig)
	defer cache.Close()

	masterStart := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	masterEnd := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Test with empty recurrence info
	emptyRecInfo := RecurrenceInfo{}
	cache.Set("", masterStart, masterEnd, emptyRecInfo, rangeStart, rangeEnd, true)
	result, found := cache.Get("", masterStart, masterEnd, emptyRecInfo, rangeStart, rangeEnd)
	if !found || result != true {
		t.Error("Cache should handle empty recurrence info and empty event ID")
	}

	// Test with zero times
	zeroTime := time.Time{}
	cache.Set("zero", zeroTime, zeroTime, emptyRecInfo, zeroTime, zeroTime, false)
	result, found = cache.Get("zero", zeroTime, zeroTime, emptyRecInfo, zeroTime, zeroTime)
	if !found || result != false {
		t.Error("Cache should handle zero time values")
	}

	// Test with same start and end times
	cache.Set("same", masterStart, masterStart, emptyRecInfo, masterStart, masterStart, true)
	result, found = cache.Get("same", masterStart, masterStart, emptyRecInfo, masterStart, masterStart)
	if !found || result != true {
		t.Error("Cache should handle same start and end times")
	}
}

func TestEngineWithCache_LogicalCorrectness(t *testing.T) {
	// Create engines with and without cache
	engineWithCache := NewEngine()
	engineWithoutCache := NewEngineWithoutCache()
	defer engineWithCache.Close()

	// Test data
	masterStart := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	masterEnd := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	testCases := []struct {
		name       string
		recurrence RecurrenceInfo
		expectTrue bool
	}{
		{
			name: "Daily recurrence with occurrences in range",
			recurrence: RecurrenceInfo{
				RRULE: "FREQ=DAILY;COUNT=5",
			},
			expectTrue: true,
		},
		{
			name: "Weekly recurrence ended before master event",
			recurrence: RecurrenceInfo{
				RRULE: "FREQ=WEEKLY;UNTIL=20231201T000000Z",
			},
			expectTrue: true, // Master event is still valid even if UNTIL is before it
		},
		{
			name: "Master event with EXDATE exclusion",
			recurrence: RecurrenceInfo{
				EXDATE: []time.Time{masterStart},
			},
			expectTrue: false,
		},
		{
			name: "RDATE within range",
			recurrence: RecurrenceInfo{
				RDATE: []time.Time{
					time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				},
			},
			expectTrue: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test without cache
			resultWithoutCache, errWithoutCache := engineWithoutCache.HasOccurrenceInRange(
				masterStart, masterEnd, tc.recurrence, rangeStart, rangeEnd)

			// Test with cache (first call - cache miss)
			resultWithCache1, errWithCache1 := engineWithCache.HasOccurrenceInRange(
				masterStart, masterEnd, tc.recurrence, rangeStart, rangeEnd)

			// Test with cache (second call - cache hit)
			resultWithCache2, errWithCache2 := engineWithCache.HasOccurrenceInRange(
				masterStart, masterEnd, tc.recurrence, rangeStart, rangeEnd)

			// All results should be identical
			if errWithoutCache != nil {
				t.Errorf("Error without cache: %v", errWithoutCache)
			}
			if errWithCache1 != nil {
				t.Errorf("Error with cache (first call): %v", errWithCache1)
			}
			if errWithCache2 != nil {
				t.Errorf("Error with cache (second call): %v", errWithCache2)
			}

			if resultWithoutCache != tc.expectTrue {
				t.Errorf("Without cache: expected %v, got %v", tc.expectTrue, resultWithoutCache)
			}
			if resultWithCache1 != tc.expectTrue {
				t.Errorf("With cache (first): expected %v, got %v", tc.expectTrue, resultWithCache1)
			}
			if resultWithCache2 != tc.expectTrue {
				t.Errorf("With cache (second): expected %v, got %v", tc.expectTrue, resultWithCache2)
			}

			// All should match
			if resultWithoutCache != resultWithCache1 || resultWithCache1 != resultWithCache2 {
				t.Errorf("Results don't match: without_cache=%v, with_cache_1=%v, with_cache_2=%v",
					resultWithoutCache, resultWithCache1, resultWithCache2)
			}
		})
	}
}

func TestEngineConfiguration_LogicalCorrectness(t *testing.T) {
	// Test that different configurations produce logically correct results
	configs := []struct {
		name   string
		config EngineConfig
	}{
		{"Default", DefaultEngineConfig},
		{"HighPerformance", HighPerformanceConfig},
		{"LowMemory", LowMemoryConfig},
		{"DisabledCache", DisabledCacheConfig},
	}

	masterStart := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	masterEnd := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 3, 31, 23, 59, 59, 0, time.UTC) // Large range

	recurrence := RecurrenceInfo{
		RRULE: "FREQ=WEEKLY;BYDAY=MO,WE,FR;COUNT=20",
	}

	var results []bool

	for _, config := range configs {
		t.Run(config.name, func(t *testing.T) {
			engine := NewEngineWithConfig(config.config)
			if engine.cache != nil {
				defer engine.Close()
			}

			result, err := engine.HasOccurrenceInRange(masterStart, masterEnd, recurrence, rangeStart, rangeEnd)
			if err != nil {
				t.Errorf("Error with %s config: %v", config.name, err)
			}

			results = append(results, result)
		})
	}

	// All configurations should produce the same result
	if len(results) > 1 {
		for i := 1; i < len(results); i++ {
			if results[0] != results[i] {
				t.Errorf("Configuration %d produced different result: %v vs %v", i, results[0], results[i])
			}
		}
	}
}
