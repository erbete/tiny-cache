package tinycache

import (
	"crypto/sha1"
	"sync"
	"time"
)

type (
	shard struct {
		sync.RWMutex
		items         map[string]*payload
		sweepInterval time.Duration
	}

	payload struct {
		data    string
		expires int
	}

	cache []*shard
)

func NewCache(shardSize int, sweepInterval string) (cache, error) {
	interval, err := time.ParseDuration(sweepInterval)
	if err != nil {
		return nil, err
	}

	cache := make([]*shard, shardSize)
	for i := 0; i < shardSize; i++ {
		shard := &shard{
			RWMutex:       sync.RWMutex{},
			items:         make(map[string]*payload),
			sweepInterval: interval,
		}

		cache[i] = shard
		go shard.sweep()
	}

	return cache, nil
}

func (c cache) getShard(key string) *shard {
	// calculate shard index for key
	checksum := sha1.Sum([]byte(key))
	hash := int(checksum[13])<<8 | int(checksum[17])
	shardIndex := hash % len(c)

	return c[shardIndex]
}

// add data to the cache
func (c cache) Add(key, data, expiration string) error {
	exp, err := time.ParseDuration(expiration)
	if err != nil {
		return err
	}

	shard := c.getShard(key)
	shard.Lock()
	defer shard.Unlock()

	shard.items[key] = &payload{
		data,
		int(time.Now().Add(exp).UnixNano()),
	}

	return nil
}

// get data from the cache
func (c cache) Get(key string) (string, error) {
	shard := c.getShard(key)
	shard.RLock()
	defer shard.RUnlock()

	payload, ok := shard.items[key]
	if !ok {
		return "", &ErrorKeyNotExist{key}
	}

	if isExpired(payload.expires) {
		return "", nil
	}

	return payload.data, nil
}

// delete data from the cache
func (c cache) Delete(key string) {
	shard := c.getShard(key)
	shard.Lock()
	defer shard.Unlock()

	delete(shard.items, key)
}

// check if a key is in the cache
func (c cache) Contains(key string) bool {
	shard := c.getShard(key)
	shard.RLock()
	defer shard.RUnlock()

	payload, ok := shard.items[key]
	if !ok {
		return false
	}

	if payload.data == "" {
		return false
	}

	if isExpired(payload.expires) {
		return false
	}

	return true
}

// empty the cache of all data
func (c cache) Flush() {
	for _, shard := range c {
		shard.Lock()
		shard.items = make(map[string]*payload)
		shard.Unlock()
	}
}

// get a list of all the keys
func (c cache) Keys() []string {
	keys := make([]string, 0)
	mutex := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(c))

	for _, shrd := range c {
		go func(s *shard) {
			s.RLock()

			for key, payload := range s.items {
				mutex.Lock()
				if !isExpired(payload.expires) {
					keys = append(keys, key)
				}
				mutex.Unlock()
			}

			s.RUnlock()
			wg.Done()
		}(shrd)
	}

	wg.Wait()

	return keys
}

// sweep deletes any expired payload found inside a shard
func (s *shard) sweep() {
	ticker := time.NewTicker(s.sweepInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.Lock()
		for key, payload := range s.items {
			if isExpired(payload.expires) {
				delete(s.items, key)
			}
		}
		s.Unlock()
	}
}

// check if a payload is expired
func isExpired(expires int) bool {
	return int(time.Now().UnixNano()) >= expires
}
