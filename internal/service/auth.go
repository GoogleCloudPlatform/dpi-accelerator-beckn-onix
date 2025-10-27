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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// subscriptionKeyProvider defines the subset of subscriptionService needed by auth logic.
type subscriptionKeyProvider interface {
	GetSigningPublicKey(ctx context.Context, subscriberID string, domain string, role model.Role, keyID string) (string, error)
}

// signValidator defines the interface for validating request signatures.
type signValidator interface {
	Validate(ctx context.Context, body []byte, header string, publicKeyBase64 string) error
}

// parseAuthHeader extracts subscriber_id and unique_key_id from the Authorization header.
// Example keyId format: "{subscriber_id}|{unique_key_id}|{algorithm}"
func parseAuthHeader(authHeader string) (*model.AuthHeader, error) {
	// Example: Signature keyId="bpp.example.com|key-1|ed25519",algorithm="ed25519",...
	keyIDPart := ""
	// Look for keyId="<value>"
	const keyIdPrefix = `keyId="`
	startIndex := strings.Index(authHeader, keyIdPrefix)
	if startIndex != -1 {
		startIndex += len(keyIdPrefix)
		endIndex := strings.Index(authHeader[startIndex:], `"`)
		if endIndex != -1 {
			keyIDPart = strings.TrimSpace(authHeader[startIndex : startIndex+endIndex])
		}
	}

	if keyIDPart == "" {
		return nil, fmt.Errorf("keyId parameter not found in Authorization header")
	}

	keyIDComponents := strings.Split(keyIDPart, "|")
	if len(keyIDComponents) != 3 {
		return nil, fmt.Errorf("keyId parameter has incorrect format, expected 3 components separated by '|', got %d for '%s'", len(keyIDComponents), keyIDPart)
	}

	return &model.AuthHeader{
		SubscriberID: strings.TrimSpace(keyIDComponents[0]),
		UniqueID:     strings.TrimSpace(keyIDComponents[1]),
		Algorithm:    strings.TrimSpace(keyIDComponents[2]),
	}, nil
}

// UnauthorizedHeader creates the WWW-Authenticate header string.
func UnauthorizedHeader(realm string) string {
	return fmt.Sprintf("Signature realm=\"%s\",headers=\"(created) (expires) digest\"", realm)
}

// keySet extracts and parses the keyId from the Authorization header.
func keySet(ctx context.Context, authHeader string) (*model.AuthHeader, *model.AuthError) {
	if authHeader == "" {
		slog.ErrorContext(ctx, "parseAuthHeader: Authorization header missing")
		return nil, model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeMissingAuthHeader, "Authorization header missing.", "unknown")
	}

	parsedKeyID, err := parseAuthHeader(authHeader)
	if err != nil {
		slog.ErrorContext(ctx, "parseAuthHeader: Failed to parse keyId from Authorization header", "error", err, "header", authHeader)
		return nil, model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeInvalidAuthHeader, "Invalid Authorization header format: "+err.Error(), "unknown")
	}
	return parsedKeyID, nil
}

// subscriptionAuth handles request authentication.
type subscriptionAuth struct {
	subService   subscriptionKeyProvider
	sigValidator signValidator
}

// NewAuthService creates a new AuthService.
func NewAuthService(subService subscriptionKeyProvider, sigValidator signValidator) (*subscriptionAuth, error) {
	if subService == nil {
		return nil, errors.New("authSubscriptionService dependency is nil for AuthService")
	}
	if sigValidator == nil {
		return nil, errors.New("signValidator dependency is nil for AuthService")
	}
	return &subscriptionAuth{subService: subService, sigValidator: sigValidator}, nil
}

// AuthenticatedReq handles authorization, signature validation, and request body parsing.
// It returns the parsed SubscriptionRequest or an AuthError if authentication/parsing fails.
func (s *subscriptionAuth) AuthenticatedReq(ctx context.Context, body []byte, authHeader string) (*model.SubscriptionRequest, *model.AuthError) {
	slog.DebugContext(ctx, "processAuthenticatedRequest: Processing authentication", "authorization_header_present", authHeader != "")

	// 1. Parse Auth Header
	ah, authErr := keySet(ctx, authHeader)
	if authErr != nil {
		return nil, authErr
	}

	var subReq model.SubscriptionRequest
	if err := json.NewDecoder(bytes.NewBuffer(body)).Decode(&subReq); err != nil {
		slog.ErrorContext(ctx, "decodeRequestBody: Failed to decode request body", "error", err)
		return nil, model.NewAuthError(http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeInvalidJSON, "Invalid request body: "+err.Error(), "")
	}
	// 3. Validate Subscriber ID Match
	if subReq.SubscriberID != ah.SubscriberID {
		slog.ErrorContext(ctx, "validateSubscriberIDMatch: SubscriberID in auth header does not match SubscriberID in body", "header_subscriber_id", ah.SubscriberID, "body_subscriber_id", subReq.SubscriberID)
		return nil, model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeIDMismatch, "Subscriber ID in auth header and body do not match.", ah.SubscriberID)
	}

	// 4. Fetch Signing Public Key
	publicKey, err := s.subService.GetSigningPublicKey(ctx, ah.SubscriberID, subReq.Domain, subReq.Type, ah.UniqueID)
	if err != nil {
		slog.ErrorContext(ctx, "fetchSigningPublicKey: Failed to fetch public key for signature validation", "error", err, "subscriber_id", ah.SubscriberID)
		return nil, handleGetSigningKeyError(err, ah.SubscriberID)
	}

	// 5. Validate Signature
	if err := s.sigValidator.Validate(ctx, body, authHeader, publicKey); err != nil {
		slog.ErrorContext(ctx, "validateSignature: Signature validation failed", "error", err)
		return nil, model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeInvalidSignature, "Invalid request signature.", ah.SubscriberID) // SubscriberID might not be available here if keyID parsing failed earlier, but it's available in the main method. Let's pass it.
	}

	slog.DebugContext(ctx, "processAuthenticatedRequest: Signature validated successfully", "subscriber_id", ah.SubscriberID)
	return &subReq, nil
}

func handleGetSigningKeyError(err error, subscriberID string) *model.AuthError {
	if errors.Is(err, repository.ErrSubscriberKeyNotFound) {
		return model.NewAuthError(http.StatusNotFound, model.ErrorTypeNotFoundError, model.ErrorCodeSubscriptionNotFound, "Signing key not found for the subscriber.", subscriberID)
	}
	return model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeKeyUnavailable, "Could not retrieve key for signature validation.", subscriberID)
}

// npKeyProvider defines the interface for retrieving signing public keys.
type npKeyProvider interface {
	LookupNPKeys(ctx context.Context, subscriberID, uniqueKeyID string) (signingPublicKey string, encrPublicKey string, err error)
}

type txnSignValidator struct {
	sv signValidator
	km npKeyProvider
}

// NewTxnSignValidator initializes and returns a new validate sign step.
func NewTxnSignValidator(sv signValidator, km npKeyProvider) (*txnSignValidator, error) {
	if sv == nil {
		slog.Error("NewTxnSignValidator: signValidator dependency is nil")
		return nil, errors.New("signValidator dependency is nil")
	}
	if km == nil {
		slog.Error("NewTxnSignValidator: npKeyProvider dependency is nil")
		return nil, errors.New("npKeyProvider dependency is nil")
	}
	return &txnSignValidator{sv: sv, km: km}, nil
}

func (s *txnSignValidator) Validate(ctx context.Context, body []byte, authHeader string) *model.AuthError {
	ah, authErr := keySet(ctx, authHeader)
	if authErr != nil {
		return authErr
	}

	slog.DebugContext(ctx, "txnSignValidator.Validate: Auth header parsed", "subscriber_id", ah.SubscriberID, "key_id", ah.UniqueID)

	key, _, err := s.km.LookupNPKeys(ctx, ah.SubscriberID, ah.UniqueID)
	if err != nil {
		slog.ErrorContext(ctx, "txnSignValidator.Validate: Failed to get signing public key from npKeyProvider", "error", err, "subscriber_id", ah.SubscriberID, "key_id", ah.UniqueID)
		return model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeKeyUnavailable, "Failed to retrieve signing key for validation.", ah.SubscriberID)
	}

	if err := s.sv.Validate(ctx, body, authHeader, key); err != nil {
		slog.ErrorContext(ctx, "txnSignValidator.Validate: Signature validation failed", "error", err, "subscriber_id", ah.SubscriberID)
		return model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeInvalidSignature, "Invalid request signature.", ah.SubscriberID)
	}

	slog.DebugContext(ctx, "txnSignValidator.Validate: Signature validated successfully", "subscriber_id", ah.SubscriberID)
	return nil
}
