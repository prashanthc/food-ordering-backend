package cache

import (
	"context"
	"encoding/json"
	"time"

	"food-ordering/internal/config"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

func NewClient(cfg *config.Config) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisURL,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	})
	return &Client{rdb: rdb}
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

func (c *Client) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) RDB() *redis.Client {
	return c.rdb
}
