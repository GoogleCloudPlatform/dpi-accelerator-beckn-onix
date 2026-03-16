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
	"testing"
)

func TestProvider_New(t *testing.T) {
	ctx := t.Context()
	testCases := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name:   "success with empty config",
			config: map[string]any{},
		},
		{
			name: "success with valid override",
			config: map[string]any{
				"audience_override": "https://api.example.com",
			},
		},
		{
			name: "error with invalid config type",
			config: map[string]any{
				"audience_override": 12345, // Should trigger error in library
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wrapper, cleanup, err := Provider.New(ctx, tc.config)

			if tc.wantErr {
				if err == nil {
					t.Errorf("Provider.New(%v) got nil error, want error", tc.config)
				}
				return
			}

			if err != nil {
				t.Fatalf("Provider.New(%v) failed unexpectedly: %v", tc.config, err)
			}

			if wrapper == nil {
				t.Error("Provider.New() returned nil wrapper, want non-nil")
			}

			if cleanup != nil {
				cleanup() // Execute the cleanup function to cover that branch
			}
		})
	}
}

// TestMain covers the empty main function required for package main.
func TestMain(t *testing.T) {
	main()
}
