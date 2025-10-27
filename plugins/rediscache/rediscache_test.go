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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
)

func TestNewSuccess(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name   string
		config map[string]string
	}{
		{
			name: "success",
			config: map[string]string{
				"addr": "localhost:6379",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			originalRedisNewClient := redisNewClient
			redisNewClient = func(opt *redis.Options) *redis.Client {
				return client
			}
			defer func() { redisNewClient = originalRedisNewClient }()

			mock.ExpectPing().SetVal("PONG")

			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			if cache == nil {
				t.Errorf("expected non-nil cache")
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestNewError(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		config      map[string]string
		expectedErr error
	}{
		{
			name: "invalid addr",
			config: map[string]string{
				"addr": "invalid_address",
			},
			expectedErr: errors.New("dial tcp: address invalid_address: missing port in address"),
		},
		{
			name: "no addr",
			config: map[string]string{
				"password": "password",
			},
			expectedErr: errors.New("missing required config 'addr'"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := New(ctx, tc.config)
			if err == nil || !strings.Contains(err.Error(), tc.expectedErr.Error()) {
				t.Errorf("expected error: %v, got: %v", tc.expectedErr, err)
			}
		})
	}
}

func TestGetSuccess(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name   string
		config map[string]string
		key    string
		value  string
	}{
		{
			name: "success",
			config: map[string]string{
				"addr": "localhost:6379",
			},
			key:   "testKey",
			value: "testValue",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			originalRedisNewClient := redisNewClient
			redisNewClient = func(opt *redis.Options) *redis.Client {
				return client
			}
			defer func() { redisNewClient = originalRedisNewClient }()

			mock.ExpectPing().SetVal("PONG")
			mock.ExpectGet(tc.key).SetVal(tc.value)

			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			val, err := cache.Get(ctx, tc.key)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if val != tc.value {
				t.Errorf("expected value: %s, got: %s", tc.value, val)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestGetError(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		config      map[string]string
		key         string
		expectedErr error
	}{
		{
			name: "get error",
			config: map[string]string{
				"addr": "localhost:6379",
			},
			key:         "testKey",
			expectedErr: errors.New("redis: nil"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			originalRedisNewClient := redisNewClient
			redisNewClient = func(opt *redis.Options) *redis.Client {
				return client
			}
			defer func() { redisNewClient = originalRedisNewClient }()

			mock.ExpectPing().SetVal("PONG")
			mock.ExpectGet(tc.key).SetErr(tc.expectedErr)

			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			_, err = cache.Get(ctx, tc.key)
			if err == nil || err.Error() != tc.expectedErr.Error() {
				t.Errorf("expected error: %v, got: %v", tc.expectedErr, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestSetSuccess(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name   string
		config map[string]string
		key    string
		value  string
		ttl    time.Duration
	}{
		{
			name: "success",
			config: map[string]string{
				"addr": "localhost:6379",
			},
			key:   "testKey",
			value: "testValue",
			ttl:   time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			originalRedisNewClient := redisNewClient
			redisNewClient = func(opt *redis.Options) *redis.Client {
				return client
			}
			defer func() { redisNewClient = originalRedisNewClient }()

			mock.ExpectPing().SetVal("PONG")
			mock.ExpectSet(tc.key, tc.value, tc.ttl).SetVal("OK")

			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			err = cache.Set(ctx, tc.key, tc.value, tc.ttl)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestSetError(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		config      map[string]string
		key         string
		value       string
		ttl         time.Duration
		expectedErr error
	}{
		{
			name: "set error",
			config: map[string]string{
				"addr": "localhost:6379",
			},
			key:         "testKey",
			value:       "testValue",
			ttl:         time.Second,
			expectedErr: errors.New("redis: set failed"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			originalRedisNewClient := redisNewClient
			redisNewClient = func(opt *redis.Options) *redis.Client {
				return client
			}
			defer func() { redisNewClient = originalRedisNewClient }()

			mock.ExpectPing().SetVal("PONG")
			mock.ExpectSet(tc.key, tc.value, tc.ttl).SetErr(tc.expectedErr)

			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			err = cache.Set(ctx, tc.key, tc.value, tc.ttl)
			if err == nil || err.Error() != tc.expectedErr.Error() {
				t.Errorf("expected error: %v, got: %v", tc.expectedErr, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestDeleteSuccess(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name   string
		config map[string]string
		key    string
	}{
		{
			name: "success",
			config: map[string]string{
				"addr": "localhost:6379",
			},
			key: "testKey",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			originalRedisNewClient := redisNewClient
			redisNewClient = func(opt *redis.Options) *redis.Client {
				return client
			}
			defer func() { redisNewClient = originalRedisNewClient }()

			mock.ExpectPing().SetVal("PONG")
			mock.ExpectDel(tc.key).SetVal(1)

			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			err = cache.Delete(ctx, tc.key)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestDeleteError(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		config      map[string]string
		key         string
		expectedErr error
	}{
		{
			name: "delete error",
			config: map[string]string{
				"addr": "localhost:6379",
			},
			key:         "testKey",
			expectedErr: errors.New("redis: delete failed"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			originalRedisNewClient := redisNewClient
			redisNewClient = func(opt *redis.Options) *redis.Client {
				return client
			}
			defer func() { redisNewClient = originalRedisNewClient }()

			mock.ExpectPing().SetVal("PONG")
			mock.ExpectDel(tc.key).SetErr(tc.expectedErr)

			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			err = cache.Delete(ctx, tc.key)
			if err == nil || err.Error() != tc.expectedErr.Error() {
				t.Errorf("expected error: %v, got: %v", tc.expectedErr, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestClearSuccess(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name   string
		config map[string]string
	}{
		{
			name: "success",
			config: map[string]string{
				"addr": "localhost:6379",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			originalRedisNewClient := redisNewClient
			redisNewClient = func(opt *redis.Options) *redis.Client {
				return client
			}
			defer func() { redisNewClient = originalRedisNewClient }()

			mock.ExpectPing().SetVal("PONG")
			mock.ExpectFlushDB().SetVal("OK")

			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			err = cache.Clear(ctx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestClearError(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		config      map[string]string
		expectedErr error
	}{
		{
			name: "clear error",
			config: map[string]string{
				"addr": "localhost:6379",
			},
			expectedErr: errors.New("flush failed"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			originalRedisNewClient := redisNewClient
			redisNewClient = func(opt *redis.Options) *redis.Client {
				return client
			}
			defer func() { redisNewClient = originalRedisNewClient }()

			mock.ExpectPing().SetVal("PONG")
			mock.ExpectFlushDB().SetErr(tc.expectedErr)

			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			err = cache.Clear(ctx)
			if err == nil || err.Error() != tc.expectedErr.Error() {
				t.Errorf("expected error: %v, got: %v", tc.expectedErr, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestCloseSuccess(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name   string
		config map[string]string
	}{
		{
			name: "success",
			config: map[string]string{
				"addr": "localhost:6379",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			originalRedisNewClient := redisNewClient
			redisNewClient = func(opt *redis.Options) *redis.Client {
				return client
			}
			defer func() { redisNewClient = originalRedisNewClient }()

			mock.ExpectPing().SetVal("PONG")

			var closeCalled bool

			_, closeFunc, err := New(ctx, tc.config)
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

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestCloseError(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		config      map[string]string
		expectedErr error
	}{{
		name: "close error",
		config: map[string]string{
			"addr": "localhost:6379",
		},
		expectedErr: errors.New("close failed"),
	},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mock := redismock.NewClientMock()
			originalRedisNewClient := redisNewClient
			redisNewClient = func(opt *redis.Options) *redis.Client {
				return client
			}
			defer func() { redisNewClient = originalRedisNewClient }()

			mock.ExpectPing().SetVal("PONG")

			cache, closeFunc, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}
			_ = cache
			_ = closeFunc

			wrappedCloseFunc := func() error {
				return tc.expectedErr
			}

			err = wrappedCloseFunc()
			if err == nil || err.Error() != tc.expectedErr.Error() {
				t.Errorf("expected error: %v, got: %v", tc.expectedErr, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}
