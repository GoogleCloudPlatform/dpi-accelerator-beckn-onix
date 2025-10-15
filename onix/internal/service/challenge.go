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
	"fmt"
	"crypto/rand"
	"encoding/hex"
)

type challengeService struct{}

// NewChallengeService creates a new ChallengeService.
func NewChallengeService() *challengeService {
	return &challengeService{}
}

// NewChallenge generates a new random challenge string.
// The challenge is a 32-character hex-encoded string.
func (s *challengeService) NewChallenge() (string, error) {
	bytes := make([]byte, 16) // 16 bytes = 32 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes for challenge: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// Verify checks if the provided answer matches the original challenge.
func (s *challengeService) Verify(challenge, answer string) bool {
	return challenge == answer
}
