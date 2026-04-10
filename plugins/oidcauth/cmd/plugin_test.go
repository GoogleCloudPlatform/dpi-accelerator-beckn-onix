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

func TestProviderNew_Error(t *testing.T) {
	provider := &oidcProvider{}
	// Missing audience and issuers
	cfg := map[string]string{}

	_, err := provider.New(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		wantAud  string
		wantIsss []string
		wantSAs  []string
	}{
		{
			name: "trims and filters issuers and SAs",
			input: map[string]string{
				"allowed_audience": "aud1",
				"allowed_issuers":  " iss1 , , iss2 ",
				"allowed_sas":      "  , sa1@google.com , sa2@google.com ,  ",
			},
			wantAud:  "aud1",
			wantIsss: []string{"iss1", "iss2"},
			wantSAs:  []string{"sa1@google.com", "sa2@google.com"},
		},
		{
			name: "missing keys result in empty slices",
			input: map[string]string{
				"allowed_audience": "aud1",
			},
			wantAud: "aud1",
		},
		{
			name: "explicitly empty values result in empty slices",
			input: map[string]string{
				"allowed_audience": "aud1",
				"allowed_issuers":  "",
				"allowed_sas":      " ",
			},
			wantAud: "aud1",
		},
		{
			name: "single value without spaces",
			input: map[string]string{
				"allowed_audience": "aud1",
				"allowed_issuers":  "iss1",
				"allowed_sas":      "sa1@google.com",
			},
			wantAud:  "aud1",
			wantIsss: []string{"iss1"},
			wantSAs:  []string{"sa1@google.com"},
		},
		{
			name: "multiple values without spaces",
			input: map[string]string{
				"allowed_audience": "aud1",
				"allowed_issuers":  "iss1,iss2",
				"allowed_sas":      "sa1,sa2",
			},
			wantAud:  "aud1",
			wantIsss: []string{"iss1", "iss2"},
			wantSAs:  []string{"sa1", "sa2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := config(tc.input)
			if got.AllowedAudience != tc.wantAud {
				t.Errorf("AllowedAudience = %q; want %q", got.AllowedAudience, tc.wantAud)
			}
			if !slicesEqual(got.AllowedIssuers, tc.wantIsss) {
				t.Errorf("AllowedIssuers = %v; want %v", got.AllowedIssuers, tc.wantIsss)
			}
			if !slicesEqual(got.AllowedSAs, tc.wantSAs) {
				t.Errorf("AllowedSAs = %v; want %v", got.AllowedSAs, tc.wantSAs)
			}
		})
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestMain(t *testing.T) {
	main()
}
