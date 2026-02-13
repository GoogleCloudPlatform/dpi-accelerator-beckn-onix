// Copyright 2026 Google LLC
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
	"testing"
)

func TestDecrypterProviderSuccess(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		config map[string]string
	}{
		{
			name:   "Valid context with empty config",
			ctx:    context.Background(),
			config: map[string]string{},
		},
		{
			name:   "Valid context with non-empty config",
			ctx:    context.Background(),
			config: map[string]string{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := decrypterProvider{}
			decrypter, cleanup, err := provider.New(tt.ctx, tt.config)

			// Check error.
			if err != nil {
				t.Errorf("New() error = %v, want no error", err)
			}

			// Check decrypter.
			if decrypter == nil {
				t.Error("New() decrypter is nil, want non-nil")
			}

			// Test cleanup function if it exists.
			if cleanup != nil {
				if err := cleanup(); err != nil {
					t.Errorf("cleanup() error = %v", err)
				}
			}
		})
	}
}
