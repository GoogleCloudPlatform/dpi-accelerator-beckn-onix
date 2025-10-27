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
	"net/http"
	"strings"
	"testing"

	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// mockSubscriptionKeyProvider is a mock for subscriptionKeyProvider.
type mockSubscriptionKeyProvider struct {
	key string
	err error
}

func (m *mockSubscriptionKeyProvider) GetSigningPublicKey(ctx context.Context, subscriberID string, domain string, role model.Role, keyID string) (string, error) {
	return m.key, m.err
}

// mockSignValidator is a mock for signValidator.
type mockSignValidator struct {
	err error
}

func (m *mockSignValidator) Validate(ctx context.Context, body []byte, header string, publicKeyBase64 string) error {
	return m.err
}

// mockNPKeyProvider is a mock for npKeyProvider.
type mockNPKeyProvider struct {
	signingKey string
	encrKey    string
	err        error
}

func (m *mockNPKeyProvider) LookupNPKeys(ctx context.Context, subscriberID, uniqueKeyID string) (signingPublicKey string, encrPublicKey string, err error) {
	return m.signingKey, m.encrKey, m.err
}

func TestParseAuthHeader(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		want       *model.AuthHeader
		wantErr    string
	}{
		{
			name:       "valid header",
			authHeader: `Signature keyId="bpp.example.com|key-1|ed25519",algorithm="ed25519",created="1678886400",expires="1678886700",headers="(created) (expires) digest",signature="signature_value"`,
			want:       &model.AuthHeader{SubscriberID: "bpp.example.com", UniqueID: "key-1", Algorithm: "ed25519"},
			wantErr:    "",
		},
		{
			name:       "missing keyId parameter",
			authHeader: `Signature algorithm="ed25519",created="1678886400"`,
			want:       nil,
			wantErr:    "keyId parameter not found in Authorization header",
		},
		{
			name:       "malformed keyId (too few components)",
			authHeader: `Signature keyId="bpp.example.com|key-1",algorithm="ed25519"`,
			want:       nil,
			wantErr:    "keyId parameter has incorrect format",
		},
		{
			name:       "malformed keyId (too many components)",
			authHeader: `Signature keyId="bpp.example.com|key-1|ed25519|extra",algorithm="ed25519"`,
			want:       nil,
			wantErr:    "keyId parameter has incorrect format",
		},
		{
			name:       "empty header",
			authHeader: "",
			want:       nil,
			wantErr:    "keyId parameter not found in Authorization header", // parseAuthHeader itself doesn't check for empty, keySet does.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAuthHeader(tt.authHeader)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("parseAuthHeader() error = %v, want error containing %q", err, tt.wantErr)
				}
				if got != nil {
					t.Errorf("parseAuthHeader() got = %+v, want nil on error", got)
				}
			} else {
				if err != nil {
					t.Errorf("parseAuthHeader() unexpected error = %v", err)
				}
				if got == nil || *got != *tt.want {
					t.Errorf("parseAuthHeader() got = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestKeySet(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		authHeader string
		wantAuthH  *model.AuthHeader
		wantErr    *model.AuthError
	}{
		{
			name:       "valid header",
			authHeader: `Signature keyId="bpp.example.com|key-1|ed25519",algorithm="ed25519"`,
			wantAuthH:  &model.AuthHeader{SubscriberID: "bpp.example.com", UniqueID: "key-1", Algorithm: "ed25519"},
			wantErr:    nil,
		},
		{
			name:       "empty auth header",
			authHeader: "",
			wantAuthH:  nil,
			wantErr:    model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeMissingAuthHeader, "Authorization header missing.", "unknown"),
		},
		{
			name:       "malformed auth header",
			authHeader: `Signature keyId="malformed"`,
			wantAuthH:  nil,
			wantErr:    model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeInvalidAuthHeader, "Invalid Authorization header format: keyId parameter has incorrect format", "unknown"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAuthH, gotErr := keySet(ctx, tt.authHeader)
			if tt.wantErr != nil {
				if gotErr == nil || gotErr.StatusCode != tt.wantErr.StatusCode || !strings.Contains(gotErr.Message, tt.wantErr.Message) {
					t.Errorf("keySet() error = %v, want %v", gotErr, tt.wantErr)
				}
				if gotAuthH != nil {
					t.Errorf("keySet() gotAuthH = %+v, want nil on error", gotAuthH)
				}
			} else {
				if gotErr != nil {
					t.Errorf("keySet() unexpected error = %v", gotErr)
				}
				if gotAuthH == nil || *gotAuthH != *tt.wantAuthH {
					t.Errorf("keySet() gotAuthH = %+v, want %+v", gotAuthH, tt.wantAuthH)
				}
			}
		})
	}
}

func TestUnauthorizedHeader(t *testing.T) {
	realm := "test_realm"
	expected := `Signature realm="test_realm",headers="(created) (expires) digest"`
	got := UnauthorizedHeader(realm)

	if got != expected {
		t.Errorf("UnauthorizedHeader() = %q, want %q", got, expected)
	}
}

func TestNewAuthService(t *testing.T) {
	tests := []struct {
		name         string
		subService   subscriptionKeyProvider
		sigValidator signValidator
		wantErr      string
	}{
		{
			name:         "success",
			subService:   &mockSubscriptionKeyProvider{},
			sigValidator: &mockSignValidator{},
			wantErr:      "",
		},
		{
			name:         "nil subService",
			subService:   nil,
			sigValidator: &mockSignValidator{},
			wantErr:      "authSubscriptionService dependency is nil for AuthService",
		},
		{
			name:         "nil sigValidator",
			subService:   &mockSubscriptionKeyProvider{},
			sigValidator: nil,
			wantErr:      "signValidator dependency is nil for AuthService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAuthService(tt.subService, tt.sigValidator)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("NewAuthService() error = %v, want %q", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("NewAuthService() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestAuthenticatedReq(t *testing.T) {
	ctx := context.Background()
	validAuthHeader := `Signature keyId="test.com|key1|ed25519",algorithm="ed25519"`
	validBody := []byte(`{"subscriber_id":"test.com","domain":"test.domain","type":"BAP"}`)
	validPublicKey := "mock-public-key"

	tests := []struct {
		name       string
		body       []byte
		authHeader string
		mockSubSvc *mockSubscriptionKeyProvider
		mockSigVal *mockSignValidator
		wantSubReq *model.SubscriptionRequest
		wantErr    *model.AuthError
	}{
		{
			name:       "success",
			body:       validBody,
			authHeader: validAuthHeader,
			mockSubSvc: &mockSubscriptionKeyProvider{key: validPublicKey},
			mockSigVal: &mockSignValidator{},
			wantSubReq: &model.SubscriptionRequest{Subscription: model.Subscription{Subscriber: model.Subscriber{SubscriberID: "test.com", Domain: "test.domain", Type: "BAP"}}},
			wantErr:    nil,
		},
		{
			name:       "invalid auth header",
			body:       validBody,
			authHeader: `Signature keyId="malformed"`,
			mockSubSvc: &mockSubscriptionKeyProvider{},
			mockSigVal: &mockSignValidator{},
			wantSubReq: nil,
			wantErr:    model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeInvalidAuthHeader, "Invalid Authorization header format", "unknown"),
		},
		{
			name:       "invalid request body JSON",
			body:       []byte(`{"subscriber_id":"test.com","domain":"test.domain","type":"BAP"`), // Malformed JSON
			authHeader: validAuthHeader,
			mockSubSvc: &mockSubscriptionKeyProvider{},
			mockSigVal: &mockSignValidator{},
			wantSubReq: nil,
			wantErr:    model.NewAuthError(http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeInvalidJSON, "Invalid request body", ""),
		},
		{
			name:       "subscriber ID mismatch",
			body:       []byte(`{"subscriber_id":"wrong.com","domain":"test.domain","type":"BAP"}`),
			authHeader: validAuthHeader,
			mockSubSvc: &mockSubscriptionKeyProvider{},
			mockSigVal: &mockSignValidator{},
			wantSubReq: nil,
			wantErr:    model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeIDMismatch, "Subscriber ID in auth header and body do not match.", "test.com"),
		},
		{
			name:       "get signing public key not found",
			body:       validBody,
			authHeader: validAuthHeader,
			mockSubSvc: &mockSubscriptionKeyProvider{err: repository.ErrSubscriberKeyNotFound},
			mockSigVal: &mockSignValidator{},
			wantSubReq: nil,
			wantErr:    model.NewAuthError(http.StatusNotFound, model.ErrorTypeNotFoundError, model.ErrorCodeSubscriptionNotFound, "Signing key not found for the subscriber.", "test.com"),
		},
		{
			name:       "get signing public key generic error",
			body:       validBody,
			authHeader: validAuthHeader,
			mockSubSvc: &mockSubscriptionKeyProvider{err: errors.New("db error")},
			mockSigVal: &mockSignValidator{},
			wantSubReq: nil,
			wantErr:    model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeKeyUnavailable, "Could not retrieve key for signature validation.", "test.com"),
		},
		{
			name:       "signature validation fails",
			body:       validBody,
			authHeader: validAuthHeader,
			mockSubSvc: &mockSubscriptionKeyProvider{key: validPublicKey},
			mockSigVal: &mockSignValidator{err: errors.New("invalid signature")},
			wantSubReq: nil,
			wantErr:    model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeInvalidSignature, "Invalid request signature.", "test.com"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authService, _ := NewAuthService(tt.mockSubSvc, tt.mockSigVal)
			gotSubReq, gotErr := authService.AuthenticatedReq(ctx, tt.body, tt.authHeader)

			if tt.wantErr != nil {
				if gotErr == nil || gotErr.StatusCode != tt.wantErr.StatusCode || !strings.Contains(gotErr.Message, tt.wantErr.Message) {
					t.Errorf("AuthenticatedReq() error = %v, want %v", gotErr, tt.wantErr)
				}
				if gotSubReq != nil {
					t.Errorf("AuthenticatedReq() gotSubReq = %+v, want nil on error", gotSubReq)
				}
			} else {
				if gotErr != nil {
					t.Errorf("AuthenticatedReq() unexpected error = %v", gotErr)
				}
				if gotSubReq == nil || gotSubReq.SubscriberID != tt.wantSubReq.SubscriberID || gotSubReq.Domain != tt.wantSubReq.Domain || gotSubReq.Type != tt.wantSubReq.Type {
					t.Errorf("AuthenticatedReq() gotSubReq = %+v, want %+v", gotSubReq, tt.wantSubReq)
				}
			}
		})
	}
}

func TestNewTxnSignValidator(t *testing.T) {
	tests := []struct {
		name    string
		sv      signValidator
		km      npKeyProvider
		wantErr string
	}{
		{
			name:    "success",
			sv:      &mockSignValidator{},
			km:      &mockNPKeyProvider{},
			wantErr: "",
		},
		{
			name:    "nil signValidator",
			sv:      nil,
			km:      &mockNPKeyProvider{},
			wantErr: "signValidator dependency is nil",
		},
		{
			name:    "nil npKeyProvider",
			sv:      &mockSignValidator{},
			km:      nil,
			wantErr: "npKeyProvider dependency is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTxnSignValidator(tt.sv, tt.km)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("NewTxnSignValidator() error = %v, want %q", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("NewTxnSignValidator() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestTxnSignValidator_Validate(t *testing.T) {
	ctx := context.Background()
	validAuthHeader := `Signature keyId="test.com|key1|ed25519",algorithm="ed25519"`
	validBody := []byte(`{"message":"test"}`)
	validSigningKey := "mock-signing-key"

	tests := []struct {
		name       string
		body       []byte
		authHeader string
		mockSV     *mockSignValidator
		mockKM     *mockNPKeyProvider
		wantErr    *model.AuthError
	}{
		{
			name:       "success",
			body:       validBody,
			authHeader: validAuthHeader,
			mockSV:     &mockSignValidator{},
			mockKM:     &mockNPKeyProvider{signingKey: validSigningKey},
			wantErr:    nil,
		},
		{
			name:       "invalid auth header",
			body:       validBody,
			authHeader: `Signature keyId="malformed"`,
			mockSV:     &mockSignValidator{},
			mockKM:     &mockNPKeyProvider{},
			wantErr:    model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeInvalidAuthHeader, "Invalid Authorization header format", "unknown"),
		},
		{
			name:       "lookup NP keys fails",
			body:       validBody,
			authHeader: validAuthHeader,
			mockSV:     &mockSignValidator{},
			mockKM:     &mockNPKeyProvider{err: errors.New("key lookup error")},
			wantErr:    model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeKeyUnavailable, "Failed to retrieve signing key for validation.", "test.com"),
		},
		{
			name:       "signature validation fails",
			body:       validBody,
			authHeader: validAuthHeader,
			mockSV:     &mockSignValidator{err: errors.New("signature mismatch")},
			mockKM:     &mockNPKeyProvider{signingKey: validSigningKey},
			wantErr:    model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeInvalidSignature, "Invalid request signature.", "test.com"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, _ := NewTxnSignValidator(tt.mockSV, tt.mockKM)
			gotErr := validator.Validate(ctx, tt.body, tt.authHeader)

			if tt.wantErr != nil {
				if gotErr == nil || gotErr.StatusCode != tt.wantErr.StatusCode || !strings.Contains(gotErr.Message, tt.wantErr.Message) {
					t.Errorf("Validate() error = %v, want %v", gotErr, tt.wantErr)
				}
			} else {
				if gotErr != nil {
					t.Errorf("Validate() unexpected error = %v", gotErr)
				}
			}
		})
	}
}
