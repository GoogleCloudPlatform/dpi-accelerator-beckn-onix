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

func TestEncrypterProviderSuccess(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		config map[string]string
	}{
		{
			name:   "Valid empty config",
			ctx:    context.Background(),
			config: map[string]string{},
		},
		{
			name: "Valid config with algorithm",
			ctx:  context.Background(),
			config: map[string]string{
				"algorithm": "AES",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create provider and encrypter.
			provider := encrypterProvider{}
			encrypter, cleanup, err := provider.New(tt.ctx, tt.config)
			if err != nil {
				t.Fatalf("EncrypterProvider.New() error = %v", err)
			}
			if encrypter == nil {
				t.Fatal("EncrypterProvider.New() returned nil encrypter")
			}
			defer func() {
				if cleanup != nil {
					if err := cleanup(); err != nil {
						t.Errorf("Cleanup() error = %v", err)
					}
				}
			}()

		})
	}
}
