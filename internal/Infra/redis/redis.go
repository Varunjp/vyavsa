package redisDB

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func NewRedis(addr string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   db,
	})
}

func PingRedis(ctx context.Context, client *redis.Client) error {
	return client.Ping(ctx).Err()
}
