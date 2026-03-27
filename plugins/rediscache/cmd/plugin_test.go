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

package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/beckn/beckn-onix/pkg/plugin/definition"
	"github.com/redis/go-redis/v9"
)

// redisNewClient is a package-level variable for redis.NewClient.
var redisNewClient = redis.NewClient

// testCache mimics definition.Cache for testing purposes.
type testCache struct {
	client *redis.Client
}

func (c *testCache) GetClient() *redis.Client {
	return c.client
}

func (c *testCache) SetClient(client *redis.Client) {
	c.client = client
}

func (c *testCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *testCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *testCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *testCache) Clear(ctx context.Context) error {
	return nil
}

// MockProvider implements the cache.Provider interface for testing.
type MockProvider struct {
	MockCache      definition.Cache
	MockCloseFn    func() error
	MockError      error
	MockCloseError error
}

func (m *MockProvider) New(ctx context.Context, config map[string]string) (definition.Cache, func() error, error) {
	closeFunc := func() error {
		return m.MockCloseError
	}
	return m.MockCache, closeFunc, m.MockError
}

var mockRedisCacheNew = func(ctx context.Context, config map[string]string) (definition.Cache, func() error, error) {
	return nil, nil, errors.New("not implemented")
}

func TestCacheProviderNewErrorFromRedisCacheNew(t *testing.T) {
	ctx := context.Background()
	config := map[string]string{
		"addr": "invalid_address",
	}
	expectedError := errors.New("failed to connect to redis")

	originalRedisCacheNew := mockRedisCacheNew
	mockRedisCacheNew = func(_ context.Context, _ map[string]string) (definition.Cache, func() error, error) {
		return nil, nil, expectedError
	}
	defer func() { mockRedisCacheNew = originalRedisCacheNew }()

	provider := cacheProvider{}
	c, _, err := provider.New(ctx, config)

	if err == nil {
		t.Errorf("Expected an error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), expectedError.Error()) {
		t.Errorf("Expected error containing '%v', got '%v'", expectedError, err)
	}
	if c != nil {
		t.Errorf("Expected nil cache, got %v", c)
	}
}

func TestNewMockedRedisSuccess(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	c := &testCache{client: client}

	err = c.GetClient().Ping(ctx).Err()
	if err != nil {
		t.Errorf("Ping error: %v", err)
	}

	err = c.GetClient().Set(ctx, "key", "value", 0*time.Second).Err()
	if err != nil {
		t.Errorf("Set error: %v", err)
	}

	val, err := c.GetClient().Get(ctx, "key").Result()
	if err != nil {
		t.Errorf("Get error: %v", err)
	}
	if val != "value" {
		t.Errorf("Expected value 'value', got '%v'", val)
	}

	err = c.GetClient().Del(ctx, "key").Err()
	if err != nil {
		t.Errorf("Del error: %v", err)
	}

	_, err = c.GetClient().Get(ctx, "nonexistent_key").Result()
	if err != redis.Nil {
		t.Errorf("Expected redis.Nil error, got %v", err)
	}
}

func TestNewMockedRedisError(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	c := &testCache{client: client}
	s.Close()

	err = c.GetClient().Set(ctx, "errKey", "errValue", 0*time.Second).Err()
	if err == nil {
		t.Errorf("Expected an error when setting key on closed redis connection but got nil")
	}
}

func TestNewCloseSuccess(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	config := map[string]string{
		"addr": s.Addr(),
	}

	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	mockCache := &testCache{client: client}
	mockProvider := &MockProvider{
		MockCache:      mockCache,
		MockCloseFn:    func() error { return nil },
		MockError:      nil,
		MockCloseError: nil,
	}

	provider := mockProvider

	var closeCalled bool

	_, closeFunc, err := provider.New(ctx, config)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	wrappedCloseFunc := func() error {
		closeCalled = true
		return closeFunc()
	}

	err = wrappedCloseFunc()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !closeCalled {
		t.Errorf("Close() was not called")
	}
}

func TestNewCloseError(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	config := map[string]string{
		"addr": s.Addr(),
	}

	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	mockCache := &testCache{client: client}
	mockProvider := &MockProvider{
		MockCache:      mockCache,
		MockCloseFn:    func() error { return nil },
		MockError:      nil,
		MockCloseError: errors.New("close failed"),
	}

	provider := mockProvider

	_, closeFunc, err := provider.New(ctx, config)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	wrappedCloseFunc := func() error {
		return closeFunc()
	}

	err = wrappedCloseFunc()
	if err == nil || err.Error() != "close failed" {
		t.Errorf("expected error: close failed, got: %v", err)
	}
}
