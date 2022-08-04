package tinycache

import (
	"hash/crc32"
	"log"
	"sync"
	"time"
)

var makeTable = crc32.MakeTable(crc32.IEEE)

type payload struct {
	data    string
	expires int64
}

type shard struct {
	sync.RWMutex
	items map[string]*payload
}

type cache struct {
	shards        []*shard
	shardCount    uint16
	sweepInterval time.Duration
}

func NewCache(shardCount uint16, sweepInterval string) cache {
	interval, err := time.ParseDuration(sweepInterval)
	if err != nil {
		log.Fatalf("init cache: %v", err)
	}

	shards := make([]*shard, shardCount)
	for i := 0; uint16(i) < shardCount; i++ {
		shard := &shard{
			RWMutex: sync.RWMutex{},
			items:   make(map[string]*payload),
		}

		shards[i] = shard
		go shard.sweep(interval)
	}

	return cache{
		shards:        shards,
		shardCount:    shardCount,
		sweepInterval: interval,
	}
}

func (c cache) getShard(key string) *shard {
	checksum := crc32.Checksum([]byte(key), makeTable)
	shardIndex := checksum % uint32(c.shardCount)

	return c.shards[shardIndex]
}

// add data to the cache
func (c cache) Add(key, data, expiration string) error {
	shard := c.getShard(key)
	shard.Lock()
	defer shard.Unlock()

	exp, err := time.ParseDuration(expiration)
	if err != nil {
		return err
	}

	shard.items[key] = &payload{
		data,
		time.Now().Add(exp).UnixNano(),
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
	for _, shard := range c.shards {
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
	wg.Add(len(c.shards))

	for _, shrd := range c.shards {
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
func (s *shard) sweep(interval time.Duration) {
	ticker := time.NewTicker(interval)
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
func isExpired(expires int64) bool {
	return time.Now().UnixNano() >= expires
}
