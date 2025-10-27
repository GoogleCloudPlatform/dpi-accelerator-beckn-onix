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

package service

import (
	"testing"
)

func TestNewChallengeService(t *testing.T) {
	s := NewChallengeService()
	if s == nil {
		t.Error("NewChallengeService() returned nil, want non-nil")
	}
}

func TestChallengeService_NewChallenge(t *testing.T) {
	s := NewChallengeService()
	challenge, err := s.NewChallenge()
	if err != nil {
		t.Fatalf("NewChallenge() error = %v, wantErr nil", err)
	}
	if challenge == "" {
		t.Error("NewChallenge() returned empty challenge, want non-empty")
	}
	// Check length (32 hex characters for 16 bytes)
	if len(challenge) != 32 {
		t.Errorf("NewChallenge() challenge length = %d, want 32", len(challenge))
	}

	// Check if it's a valid hex string (basic check, not exhaustive)
	for _, r := range challenge {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			t.Errorf("NewChallenge() challenge contains non-hex character: %c", r)
			break
		}
	}

	// Generate another challenge to ensure they are different (highly probable for random data)
	challenge2, err2 := s.NewChallenge()
	if err2 != nil {
		t.Fatalf("NewChallenge() [second call] error = %v, wantErr nil", err2)
	}
	if challenge == challenge2 {
		t.Error("NewChallenge() generated two identical challenges, want different")
	}
}

func TestChallengeService_Verify(t *testing.T) {
	s := NewChallengeService()
	tests := []struct {
		name      string
		challenge string
		answer    string
		want      bool
	}{
		{
			name:      "matching challenge and answer",
			challenge: "abcdef1234567890",
			answer:    "abcdef1234567890",
			want:      true,
		},
		{
			name:      "non-matching challenge and answer",
			challenge: "abcdef1234567890",
			answer:    "0987654321fedcba",
			want:      false,
		},
		{
			name:      "empty challenge and answer",
			challenge: "",
			answer:    "",
			want:      true,
		},
		{
			name:      "empty challenge, non-empty answer",
			challenge: "",
			answer:    "someanswer",
			want:      false,
		},
		{
			name:      "non-empty challenge, empty answer",
			challenge: "somechallenge",
			answer:    "",
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Verify(tt.challenge, tt.answer); got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}
