package tinycache

import (
	"strconv"
	"testing"
	"time"
)

const (
	tKey  = "test-key"
	tData = "some data for the cache"
)

func TestNewCache(t *testing.T) {
	_, err := NewCache(5, "10m")
	if err != nil {
		t.Error("initializing cache failed", err)
	}
}

func TestAdd(t *testing.T) {
	cache, _ := NewCache(2, "200ms")

	defer func() {
		if r := recover(); r != nil {
			t.Error("add to cache caused a panic")
		}
	}()
	cache.Add(tKey, tData, "100ms")
}

func TestGet(t *testing.T) {
	cache, _ := NewCache(2, "200ms")
	cache.Add(tKey, tData, "100ms")

	data, err := cache.Get(tKey)
	if err != nil {
		t.Error(err)
	}

	if data != tData {
		t.Errorf("expected \"%s\" to not equal \"%s\"", tData, data)
	}

	// wait for payload to expire (should not return any data)
	time.Sleep(101 * time.Millisecond)
	data, err = cache.Get(tKey)
	if err != nil {
		t.Error(err)
	}

	if data != "" {
		t.Errorf("expected no data to be associated with key \"%s\"", tKey)
	}
}

func TestDelete(t *testing.T) {
	cache, _ := NewCache(2, "200ms")

	cache.Add(tKey, tData, "100ms")
	cache.Delete(tKey)
	data, err := cache.Get(tKey)
	if _, ok := err.(*ErrorKeyNotExist); !ok {
		t.Error("unexpected error: ", err)
	}

	if err.(*ErrorKeyNotExist).Key != tKey {
		t.Errorf("expected key to equal \"%s\", but got \"%s\"", tKey, err.(*ErrorKeyNotExist).Key)
	}

	if data != "" {
		t.Errorf("expected data to be empty, but got \"%s\": ", data)
	}
}

func TestContains(t *testing.T) {
	cache, _ := NewCache(2, "200ms")
	cache.Add(tKey, tData, "100ms")
	if !cache.Contains(tKey) {
		t.Error("expected data not found")
	}

	if cache.Contains("i-dont-exist") {
		t.Errorf("key \"%s\" should not exist", "i-dont-exist")
	}

	// wait for payload to expire (should return false)
	time.Sleep(101 * time.Millisecond)
	if cache.Contains(tKey) {
		t.Errorf("expected no data to be associated with key \"%s\"", tKey)
	}
}

// TODO
func TestSweeper(t *testing.T) {
	cache, _ := NewCache(5, "100ms")
	const tKey1 = "test-key-1"
	const tKey2 = "test-key-2"
	const tKey3 = "dont-delete-me-plz"
	cache.Add(tKey1, tData, "100ms")
	cache.Add(tKey2, tData, "200ms")
	cache.Add(tKey3, tData, "10m")

	if !cache.Contains(tKey1) || !cache.Contains(tKey2) || !cache.Contains(tKey3) {
		t.Error("expected data not found")
	}

	time.Sleep(400 * time.Millisecond)
	foundtKey3 := false
	for _, shard := range cache {
		shard.RLock()
		if _, ok := shard.items[tKey1]; ok {
			t.Errorf("sweeper did not remove expired payload with key \"%s\"", tKey1)
		}

		if _, ok := shard.items[tKey2]; ok {
			t.Errorf("sweeper did not remove expired payload with key \"%s\"", tKey2)
		}

		if _, ok := shard.items[tKey3]; ok {
			foundtKey3 = true
		}
		shard.RUnlock()
	}

	if !foundtKey3 {
		t.Errorf("sweeper removed unexpired payload with key \"%s\"", tKey3)
	}
}

func TestFlush(t *testing.T) {
	cache, _ := NewCache(5, "10m")

	cache.Add("test-key-1", "some data", "5m")
	cache.Add("test-key-2", "some data", "5m")
	cache.Add("test-key-3", "some data", "5m")
	cache.Add("test-key-4", "some data", "5m")
	cache.Add("test-key-5", "some data", "5m")

	cache.Flush()

	for _, shard := range cache {
		if len(shard.items) > 0 {
			t.Error("flush did not remove data from the cache")
		}
	}
}

func TestKeys(t *testing.T) {
	cache, _ := NewCache(5, "50ms")

	const (
		tKey1 = "test-key-1"
		tKey2 = "test-key-2"
		tKey3 = "test-key-3"
		tKey4 = "test-key-4"
		tKey5 = "test-key-5"
		tKey6 = "i-should-not-be-here"
	)

	cache.Add(tKey1, "some data", "5m")
	cache.Add(tKey2, "some data", "5m")
	cache.Add(tKey3, "some data", "5m")
	cache.Add(tKey4, "some data", "5m")
	cache.Add(tKey5, "some data", "5m")
	cache.Add(tKey6, "some data", "10ms")

	time.Sleep(80 * time.Millisecond)
	keys := cache.Keys()
	if len(keys) != 5 {
		t.Error("not all keys found")
	}

	for _, key := range keys {
		if !cache.Contains(key) {
			t.Error("not all keys found")
		}
	}
}

func BenchmarkTestGet(b *testing.B) {
	cache, _ := NewCache(10, "5m")
	const tData = "test data"

	for i := 0; i < b.N; i++ {
		cache.Add(strconv.Itoa(i), tData, "10m")
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i))
	}
}

func BenchmarkTestAdd(b *testing.B) {
	cache, _ := NewCache(10, "5m")
	const tData = "test data"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Add(strconv.Itoa(i), tData, "10m")
	}
}
