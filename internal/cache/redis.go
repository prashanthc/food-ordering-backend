package cache

import (
	"context"
	"encoding/json"
	"time"

	"food-ordering/internal/config"
	"food-ordering/internal/resilience"

	"github.com/redis/go-redis/v9"
)

const redisTimeout = 2 * time.Second

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
	_, cbErr := resilience.RedisCB.Execute(func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, redisTimeout)
		defer cancel()
		return nil, c.rdb.Set(ctx, key, data, ttl).Err()
	})
	return cbErr
}

func (c *Client) Get(ctx context.Context, key string, dest interface{}) error {
	result, cbErr := resilience.RedisCB.Execute(func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, redisTimeout)
		defer cancel()
		return c.rdb.Get(ctx, key).Bytes()
	})
	if cbErr != nil {
		return cbErr
	}
	return json.Unmarshal(result.([]byte), dest)
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	_, cbErr := resilience.RedisCB.Execute(func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, redisTimeout)
		defer cancel()
		return nil, c.rdb.Del(ctx, keys...).Err()
	})
	return cbErr
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) RDB() *redis.Client {
	return c.rdb
}
