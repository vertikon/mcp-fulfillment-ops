package cache

import (
	"context"
	"testing"
	"time"
)

// mockCache is a mock cache implementation for testing L2/L3
type mockCache struct {
	data  map[string][]byte
	stats CacheStats
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string][]byte),
	}
}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, ok := m.data[key]
	if !ok {
		m.stats.Misses++
		return nil, ErrCacheMiss
	}
	m.stats.Hits++
	return val, nil
}

func (m *mockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.data[key] = value
	m.stats.Size++
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	if _, ok := m.data[key]; ok {
		delete(m.data, key)
		m.stats.Size--
	}
	return nil
}

func (m *mockCache) Clear(ctx context.Context) error {
	m.data = make(map[string][]byte)
	m.stats.Size = 0
	return nil
}

func (m *mockCache) Stats() CacheStats {
	return m.stats
}

func TestNewMultiLevelCache(t *testing.T) {
	tests := []struct {
		name   string
		l1Size int
		l2     Cache
		l3     Cache
	}{
		{
			name:   "L1 only",
			l1Size: 100,
			l2:     nil,
			l3:     nil,
		},
		{
			name:   "L1 and L2",
			l1Size: 100,
			l2:     newMockCache(),
			l3:     nil,
		},
		{
			name:   "L1, L2, and L3",
			l1Size: 100,
			l2:     newMockCache(),
			l3:     newMockCache(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewMultiLevelCache(tt.l1Size, tt.l2, tt.l3)
			if cache == nil {
				t.Fatal("NewMultiLevelCache returned nil")
			}
			if cache.l1 == nil {
				t.Error("L1 cache should not be nil")
			}
			if cache.l2 != tt.l2 {
				t.Error("L2 cache mismatch")
			}
			if cache.l3 != tt.l3 {
				t.Error("L3 cache mismatch")
			}
		})
	}
}

func TestMultiLevelCache_Get_Set_L1Only(t *testing.T) {
	cache := NewMultiLevelCache(100, nil, nil)
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set value
	if err := cache.Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get value
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("Get() = %v, want %v", got, value)
	}
}

func TestMultiLevelCache_Get_Miss(t *testing.T) {
	cache := NewMultiLevelCache(100, nil, nil)
	ctx := context.Background()

	_, err := cache.Get(ctx, "non-existent-key")
	if err != ErrCacheMiss {
		t.Errorf("Get() error = %v, want ErrCacheMiss", err)
	}
}

func TestMultiLevelCache_Get_L2Promotion(t *testing.T) {
	l2 := newMockCache()
	cache := NewMultiLevelCache(100, l2, nil)
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set in L2 only
	_ = l2.Set(ctx, key, value, time.Minute)

	// Get should promote from L2 to L1
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("Get() = %v, want %v", got, value)
	}

	// Verify promotion to L1
	l1Got, err := cache.l1.Get(key)
	if err != nil {
		t.Error("Value should be promoted to L1")
	}
	if string(l1Got) != string(value) {
		t.Errorf("L1 value = %v, want %v", l1Got, value)
	}
}

func TestMultiLevelCache_Get_L3Promotion(t *testing.T) {
	l2 := newMockCache()
	l3 := newMockCache()
	cache := NewMultiLevelCache(100, l2, l3)
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set in L3 only
	_ = l3.Set(ctx, key, value, 0)

	// Get should promote from L3 to L2 and L1
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("Get() = %v, want %v", got, value)
	}

	// Verify promotion to L1
	l1Got, err := cache.l1.Get(key)
	if err != nil {
		t.Error("Value should be promoted to L1")
	}
	if string(l1Got) != string(value) {
		t.Errorf("L1 value = %v, want %v", l1Got, value)
	}

	// Verify promotion to L2
	l2Got, err := l2.Get(ctx, key)
	if err != nil {
		t.Error("Value should be promoted to L2")
	}
	if string(l2Got) != string(value) {
		t.Errorf("L2 value = %v, want %v", l2Got, value)
	}
}

func TestMultiLevelCache_Delete(t *testing.T) {
	cache := NewMultiLevelCache(100, nil, nil)
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set value
	_ = cache.Set(ctx, key, value, time.Minute)

	// Delete value
	if err := cache.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Get should fail
	_, err := cache.Get(ctx, key)
	if err != ErrCacheMiss {
		t.Errorf("Get() error = %v, want ErrCacheMiss", err)
	}
}

func TestMultiLevelCache_Delete_MultiLevel(t *testing.T) {
	l2 := newMockCache()
	l3 := newMockCache()
	cache := NewMultiLevelCache(100, l2, l3)
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set in all levels
	_ = cache.Set(ctx, key, value, time.Minute)
	_ = l2.Set(ctx, key, value, time.Minute)
	_ = l3.Set(ctx, key, value, 0)

	// Delete should remove from all levels
	if err := cache.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion from all levels
	_, err := cache.Get(ctx, key)
	if err != ErrCacheMiss {
		t.Error("Value should be deleted from L1")
	}

	_, err = l2.Get(ctx, key)
	if err != ErrCacheMiss {
		t.Error("Value should be deleted from L2")
	}

	_, err = l3.Get(ctx, key)
	if err != ErrCacheMiss {
		t.Error("Value should be deleted from L3")
	}
}

func TestMultiLevelCache_Clear(t *testing.T) {
	cache := NewMultiLevelCache(100, nil, nil)
	ctx := context.Background()

	// Set multiple values
	_ = cache.Set(ctx, "key1", []byte("value1"), time.Minute)
	_ = cache.Set(ctx, "key2", []byte("value2"), time.Minute)

	// Clear cache
	if err := cache.Clear(ctx); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify all values are cleared
	_, err := cache.Get(ctx, "key1")
	if err != ErrCacheMiss {
		t.Error("key1 should be cleared")
	}

	_, err = cache.Get(ctx, "key2")
	if err != ErrCacheMiss {
		t.Error("key2 should be cleared")
	}
}

func TestMultiLevelCache_Stats(t *testing.T) {
	cache := NewMultiLevelCache(100, nil, nil)
	ctx := context.Background()

	// Set some values
	_ = cache.Set(ctx, "key1", []byte("value1"), time.Minute)
	_ = cache.Set(ctx, "key2", []byte("value2"), time.Minute)

	// Get some values (hits)
	_, _ = cache.Get(ctx, "key1")
	_, _ = cache.Get(ctx, "key2")

	// Get non-existent (miss)
	_, _ = cache.Get(ctx, "non-existent")

	stats := cache.Stats()
	if stats.Hits < 2 {
		t.Errorf("Expected at least 2 hits, got %d", stats.Hits)
	}
	if stats.Misses < 1 {
		t.Errorf("Expected at least 1 miss, got %d", stats.Misses)
	}
	if stats.Size < 2 {
		t.Errorf("Expected at least size 2, got %d", stats.Size)
	}
}

func TestMultiLevelCache_TTL(t *testing.T) {
	cache := NewMultiLevelCache(100, nil, nil)
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set with short TTL
	_ = cache.Set(ctx, key, value, 50*time.Millisecond)

	// Get immediately should succeed
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(got) != string(value) {
		t.Errorf("Get() = %v, want %v", got, value)
	}

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Get should fail
	_, err = cache.Get(ctx, key)
	if err != ErrCacheMiss {
		t.Errorf("Get() error = %v, want ErrCacheMiss", err)
	}
}

func TestMultiLevelCache_ConcurrentAccess(t *testing.T) {
	cache := NewMultiLevelCache(1000, nil, nil)
	ctx := context.Background()

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "key-" + string(rune(id))
			value := []byte("value-" + string(rune(id)))
			_ = cache.Set(ctx, key, value, time.Minute)
			done <- true
		}(i)
	}

	// Wait for writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "key-" + string(rune(id))
			_, err := cache.Get(ctx, key)
			if err != nil {
				t.Errorf("Failed to get key %d: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for reads
	for i := 0; i < 10; i++ {
		<-done
	}
}
