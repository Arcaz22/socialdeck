package config

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedis(cfg *Config) (*redis.Client, error) {
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("redis parse url: %w", err)
	}

	client := redis.NewClient(opt)

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return client, nil
}
