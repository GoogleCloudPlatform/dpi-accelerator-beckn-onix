// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rediscache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisNewClient is a package-level variable for redis.NewClient.
var redisNewClient = redis.NewClient

// cache implements the Cache interface using Redis.
type cache struct {
	client *redis.Client
}

// New creates a new RedisCache instance and returns a close function.
func New(ctx context.Context, config map[string]string) (*cache, func() error, error) {
	addr, ok := config["addr"]
	if !ok {
		return nil, nil, fmt.Errorf("missing required config 'addr'")
	}

	password, ok := config["password"]
	if !ok {
		password = ""
	}

	client := redisNewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		//DB: 0 (default) is used for caching simplicity and isolation.
		DB: 0,
	})

	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	cache := &cache{client: client}

	closeFunc := func() error {
		return cache.close()
	}

	return cache, closeFunc, nil
}

// GetClient is a getter method to get the redis client.
func (c *cache) GetClient() *redis.Client {
	return c.client
}

// SetClient is a setter method to set the redis client.
func (c *cache) SetClient(client *redis.Client) {
	c.client = client
}

// Get retrieves a value from Redis.
func (c *cache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Set stores a value in Redis with a TTL.
func (c *cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Delete removes a value from Redis.
func (c *cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Clear removes all values from Redis.
func (c *cache) Clear(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}

// close closes the Redis client.
func (c *cache) close() error {
	return c.client.Close()
}
