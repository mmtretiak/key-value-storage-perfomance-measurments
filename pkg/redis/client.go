package redis

import (
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

func NewClusterClient(connectionURL string) (*client, error) {
	opt, err := redis.ParseClusterURL(connectionURL)
	if err != nil {
		return nil, err
	}

	cl := redis.NewClusterClient(opt)

	return &client{cl}, nil
}

func NewClient(connectionURL string) (*client, error) {
	opt, err := redis.ParseURL(connectionURL)
	if err != nil {
		return nil, err
	}

	cl := redis.NewClient(opt)

	return &client{cl}, nil
}

type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Keys(ctx context.Context, pattern string) *redis.StringSliceCmd
	FlushAll(ctx context.Context) *redis.StatusCmd
}

type client struct {
	client RedisClient
}

func (c *client) Set(ctx context.Context, key, value string) error {
	return c.client.Set(ctx, key, value, 0).Err()
}

func (c *client) Get(ctx context.Context, key string) (string, error) {
	res, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}

	return res, nil
}

func (c *client) FlushAll(ctx context.Context) error {
	return c.client.FlushAll(ctx).Err()
}

func (c *client) DeleteByPrefix(ctx context.Context, prefix string) error {
	keys, err := c.client.Keys(ctx, prefix+"*").Result()
	if err != nil {
		return err
	}

	for _, key := range keys {
		if err := c.client.Del(ctx, key).Err(); err != nil {
			return err
		}
	}

	return nil
}
