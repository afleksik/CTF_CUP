package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"veladora/config"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client
var ctx = context.Background()

func InitRedis(cfg *config.Config) error {
	Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: 40,
	})

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return nil
}

func SetCache(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, data, expiration).Err()
}

func GetCache(key string, dest interface{}) error {
	data, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func DeleteCache(key string) error {
	return Client.Del(ctx, key).Err()
}

func SetSession(sessionID string, userID int, expiration time.Duration) error {
	return Client.Set(ctx, fmt.Sprintf("session:%s", sessionID), userID, expiration).Err()
}

func GetSession(sessionID string) (int, error) {
	val, err := Client.Get(ctx, fmt.Sprintf("session:%s", sessionID)).Int()
	if err != nil {
		return 0, err
	}
	return val, nil
}

func DeleteSession(sessionID string) error {
	return Client.Del(ctx, fmt.Sprintf("session:%s", sessionID)).Err()
}

func Close() {
	if Client != nil {
		Client.Close()
	}
}
