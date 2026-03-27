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

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestNewSuccess(t *testing.T) {
	ctx := context.Background()
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	testCases := []struct {
		name   string
		config map[string]string
	}{
		{
			name: "success",
			config: map[string]string{
				"addr": s.Addr(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			if cache == nil {
				t.Errorf("expected non-nil cache")
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
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	testCases := []struct {
		name   string
		config map[string]string
		key    string
		value  string
	}{
		{
			name: "success",
			config: map[string]string{
				"addr": s.Addr(),
			},
			key:   "testKey",
			value: "testValue",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s.Set(tc.key, tc.value)

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
		})
	}
}

func TestGetError(t *testing.T) {
	ctx := context.Background()
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	testCases := []struct {
		name        string
		config      map[string]string
		key         string
		expectedErr error
	}{
		{
			name: "get error",
			config: map[string]string{
				"addr": s.Addr(),
			},
			key:         "testKey",
			expectedErr: redis.Nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			_, err = cache.Get(ctx, tc.key)
			if err == nil || err != tc.expectedErr {
				t.Errorf("expected error: %v, got: %v", tc.expectedErr, err)
			}
		})
	}
}

func TestSetSuccess(t *testing.T) {
	ctx := context.Background()
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

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
				"addr": s.Addr(),
			},
			key:   "testKey",
			value: "testValue",
			ttl:   time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			err = cache.Set(ctx, tc.key, tc.value, tc.ttl)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			got, err := s.Get(tc.key)
			if err != nil {
				t.Errorf("failed to get key from miniredis: %v", err)
			}
			if got != tc.value {
				t.Errorf("value not set correctly in miniredis: expected %s, got %s", tc.value, got)
			}
			if s.TTL(tc.key) == 0 {
				t.Errorf("TTL not set correctly in miniredis")
			}
		})
	}
}

func TestSetError(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name  string
		key   string
		value string
		ttl   time.Duration
	}{
		{
			name:  "set error",
			key:   "testKey",
			value: "testValue",
			ttl:   time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := miniredis.Run()
			if err != nil {
				t.Fatalf("failed to start miniredis: %v", err)
			}

			config := map[string]string{
				"addr": s.Addr(),
			}
			cache, _, err := New(ctx, config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			s.Close()
			err = cache.Set(ctx, tc.key, tc.value, tc.ttl)
			if err == nil {
				t.Errorf("expected error but got nil")
			}
		})
	}
}

func TestDeleteSuccess(t *testing.T) {
	ctx := context.Background()
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	testCases := []struct {
		name   string
		config map[string]string
		key    string
	}{
		{
			name: "success",
			config: map[string]string{
				"addr": s.Addr(),
			},
			key: "testKey",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s.Set(tc.key, "some_value")

			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			err = cache.Delete(ctx, tc.key)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if s.Exists(tc.key) {
				t.Errorf("key %s should have been deleted but it exists", tc.key)
			}
		})
	}
}

func TestDeleteError(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name string
		key  string
	}{
		{
			name: "delete error",
			key:  "testKey",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := miniredis.Run()
			if err != nil {
				t.Fatalf("failed to start miniredis: %v", err)
			}

			config := map[string]string{
				"addr": s.Addr(),
			}
			cache, _, err := New(ctx, config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			s.Close()
			err = cache.Delete(ctx, tc.key)
			if err == nil {
				t.Errorf("expected error but got nil")
			}
		})
	}
}

func TestClearSuccess(t *testing.T) {
	ctx := context.Background()
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	testCases := []struct {
		name   string
		config map[string]string
	}{
		{
			name: "success",
			config: map[string]string{
				"addr": s.Addr(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s.Set("key1", "val1")
			s.Set("key2", "val2")

			cache, _, err := New(ctx, tc.config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			err = cache.Clear(ctx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(s.Keys()) > 0 {
				t.Errorf("expected cache to be cleared, but keys still exist: %v", s.Keys())
			}
		})
	}
}

func TestClearError(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name string
	}{
		{
			name: "clear error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := miniredis.Run()
			if err != nil {
				t.Fatalf("failed to start miniredis: %v", err)
			}

			config := map[string]string{
				"addr": s.Addr(),
			}
			cache, _, err := New(ctx, config)
			if err != nil {
				t.Fatalf("failed to create cache: %v", err)
			}

			s.Close()
			err = cache.Clear(ctx)
			if err == nil {
				t.Errorf("expected error but got nil")
			}
		})
	}
}

func TestCloseSuccess(t *testing.T) {
	ctx := context.Background()
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	testCases := []struct {
		name   string
		config map[string]string
	}{
		{
			name: "success",
			config: map[string]string{
				"addr": s.Addr(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
		})
	}
}

func TestCloseError(t *testing.T) {
	ctx := context.Background()
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	testCases := []struct {
		name        string
		config      map[string]string
		expectedErr error
	}{{
		name: "close error",
		config: map[string]string{
			"addr": s.Addr(),
		},
		expectedErr: errors.New("close failed"),
	},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
		})
	}
}
