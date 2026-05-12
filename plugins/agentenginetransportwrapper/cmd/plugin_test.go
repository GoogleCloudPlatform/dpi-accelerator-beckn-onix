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
	"strings"
	"testing"
)

// TestAgentEngineProviderSuccess verifies the provider's success path:
// a valid context + empty config returns a non-nil wrapper and a nil
// error, with no leaked closer.
//
// The underlying agentenginetransportwrapper.New eagerly builds an OAuth2
// access-token source via Application Default Credentials, so this test
// requires a developer environment where ADC resolves (e.g. via
// `gcloud auth application-default login`). On a CI runner without ADC,
// this test will skip — the failure paths in TestAgentEngineProviderFailure
// still cover provider's error-wrapping behaviour there.
func TestAgentEngineProviderSuccess(t *testing.T) {
	wrapper, cleanup, err := Provider.New(t.Context(), map[string]any{})
	if err != nil {
		t.Skipf("skipping: ADC unavailable in this environment: %v", err)
	}
	if wrapper == nil {
		t.Fatal("Provider.New(ctx, {}) wrapper = nil, want non-nil")
	}
	if cleanup != nil {
		cleanup()
	}
}

func TestAgentEngineProviderFailure(t *testing.T) {
	provider := Provider

	cases := []struct {
		name    string
		config  map[string]any
		wantErr string
	}{
		{
			name:    "serviceAccount wrong type",
			config:  map[string]any{"serviceAccount": 42},
			wantErr: "serviceAccount",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := provider.New(t.Context(), tt.config)
			if err == nil {
				t.Fatalf("provider.New(ctx, %v) error = nil, want error containing %q", tt.config, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("provider.New(ctx, %v) error = %v, want error containing %q", tt.config, err, tt.wantErr)
			}
		})
	}
}

// TestMain covers the empty main function required for package main.
func TestMain(t *testing.T) {
	main()
}
