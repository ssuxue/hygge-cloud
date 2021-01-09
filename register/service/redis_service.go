package service

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"sync"
	"time"
)

var ctx = context.Background()

type RedisClient struct {
	mutex  *sync.Mutex
	client *redis.Client
}

// Return redis client.(Connect redis with redis info.)
func NewRedisClient() *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "47.116.137.69:6379",
		Password: "suxue",
		DB:       0, // use default DB
	})
	return &RedisClient{client: rdb}
}

// Set redis data with key.
func (cli *RedisClient) Set(key, value string) error {
	cli.mutex.Lock()
	err := cli.client.Set(ctx, key, value, 0).Err()
	cli.mutex.Unlock()
	if err != nil {
		return err
	}
	return nil
}

// Get redis data with key.
func (cli *RedisClient) Get(key string) (string, error) {
	result, err := cli.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", errors.New(key + " does not exist")
		}
		return "", err
	}

	return result, nil
}

// Set the expiration.
func (cli *RedisClient) Expire(key string, expire time.Duration) {
	// cli.client.Exists(ctx, key)
	if _, err := cli.Get(key); err != nil {
		return
	}
	cli.client.Expire(ctx, key, expire)
}

// TODO This is not right.
func (cli RedisClient) Delete(key string) {
	cli.client.Do(ctx, "DEL", key)
}
