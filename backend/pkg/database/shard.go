package database

import "github.com/google/uuid"

// NumShards is the number of DB shards.  Change only during planned migrations.
const NumShards = 4

// Shard derives a deterministic shard index [0, NumShards) from a UUID.
// Uses the first byte of the UUID so the distribution is uniform.
//
// Tables to shard:   wallet_balances, wallet_transactions, orders
// Tables NOT to shard: admin, feature_flags, config, geo_scores, route_metrics
func Shard(id uuid.UUID) int {
	return int(id[0]) % NumShards
}

// ShardKey returns a human-readable shard name.
func ShardKey(id uuid.UUID) string {
	keys := []string{"shard_0", "shard_1", "shard_2", "shard_3"}
	return keys[Shard(id)]
}
