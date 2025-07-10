// pricing_data_shard_manager.go
// Go implementation for dynamic shard allocation for pricing data.
package sharding

import (
	"fmt"
	"math/rand"
	"time"
)

type ShardManager struct {
	shards []string
}

func NewShardManager(shards []string) *ShardManager {
	return &ShardManager{shards: shards}
}

func (sm *ShardManager) AllocateShard() string {
	// Simple random allocation with fallback to a default shard if none available.
	if len(sm.shards) == 0 {
		return "default_shard"
	}
	rand.Seed(time.Now().UnixNano())
	shard := sm.shards[rand.Intn(len(sm.shards))]
	fmt.Printf("Allocated shard: %s\n", shard)
	return shard
}
