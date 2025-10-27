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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/beckn/beckn-onix/pkg/model"
)

// signingKM defines the interface for retrieving signing keys.
// Reused from proxy.go context.
type signingKM interface {
	Keyset(ctx context.Context, subscriberID string) (*model.Keyset, error)
}

// signer defines the interface for signing request bodies.
// Reused from proxy.go context.
type signer interface {
	Sign(ctx context.Context, body []byte, privateKey string, created, expires int64) (string, error)
}

type authGenService struct {
	keyManager signingKM
	signer     signer
}

// NewAuthGenService creates a new authGenService.
func NewAuthGenService(keyManager signingKM, signer signer) (*authGenService, error) {
	if keyManager == nil {
		slog.Error("NewAuthGenService: keyManager cannot be nil")
		return nil, errors.New("keyManager cannot be nil")
	}
	if signer == nil {
		slog.Error("NewAuthGenService: signer cannot be nil")
		return nil, errors.New("signer cannot be nil")
	}
	return &authGenService{
		keyManager: keyManager,
		signer:     signer,
	}, nil
}

// AuthHeader signs the provided body using the specified subscriber's key
// and generates the Authorization header value.
func (s *authGenService) AuthHeader(ctx context.Context, body []byte, subscriberID string) (string, error) {
	keySet, err := s.keyManager.Keyset(ctx, subscriberID)
	if err != nil {
		slog.ErrorContext(ctx, "AuthGenService: Failed to get keyset for signing", "error", err, "subscriber_id", subscriberID)
		return "", fmt.Errorf("failed to get keyset for signing for subscriber %s: %w", subscriberID, err)
	}

	createdAt := time.Now().Unix()
	expires := time.Now().Add(5 * time.Minute).Unix()

	signature, err := s.signer.Sign(ctx, body, keySet.SigningPrivate, createdAt, expires)
	if err != nil {
		slog.ErrorContext(ctx, "AuthGenService: Failed to sign body", "error", err)
		return "", fmt.Errorf("failed to sign body: %w", err)
	}
	return fmt.Sprintf(
		`Signature keyId="%s|%s|ed25519",algorithm="ed25519",created="%d",expires="%d",headers="(created) (expires) digest",signature="%s"`,
		subscriberID, keySet.UniqueKeyID, createdAt, expires, signature), nil
}
