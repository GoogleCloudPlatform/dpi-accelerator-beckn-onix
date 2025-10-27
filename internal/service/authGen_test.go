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
	"strings"
	"testing"

	"github.com/beckn/beckn-onix/pkg/model"
)

// mockSigningKM is a mock implementation of the signingKM interface.
type mockSigningKM struct {
	keyset *model.Keyset
	err    error
}

func (m *mockSigningKM) Keyset(ctx context.Context, subscriberID string) (*model.Keyset, error) {
	return m.keyset, m.err
}

// mockSigner is a mock implementation of the signer interface.
type mockSigner struct {
	signature string
	err       error
}

func (m *mockSigner) Sign(ctx context.Context, body []byte, privateKey string, created, expires int64) (string, error) {
	return m.signature, m.err
}

func TestNewAuthGenService(t *testing.T) {
	tests := []struct {
		name       string
		keyManager signingKM
		signer     signer
		wantErr    string
	}{
		{
			name:       "success",
			keyManager: &mockSigningKM{},
			signer:     &mockSigner{},
			wantErr:    "",
		},
		{
			name:       "nil keyManager",
			keyManager: nil,
			signer:     &mockSigner{},
			wantErr:    "keyManager cannot be nil",
		},
		{
			name:       "nil signer",
			keyManager: &mockSigningKM{},
			signer:     nil,
			wantErr:    "signer cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAuthGenService(tt.keyManager, tt.signer)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("NewAuthGenService() error = %v, want %q", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("NewAuthGenService() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestAuthHeader(t *testing.T) {
	ctx := context.Background()
	subscriberID := "test.subscriber.com"
	body := []byte(`{"message":"hello"}`)
	validKeyset := &model.Keyset{
		UniqueKeyID:    "key-123",
		SigningPrivate: "private-key-data",
	}
	validSignature := "generated-signature"

	tests := []struct {
		name           string
		mockKM         *mockSigningKM
		mockSigner     *mockSigner
		subscriberID   string
		body           []byte
		wantHeaderPart []string
		wantErr        string
	}{
		{
			name:         "success",
			mockKM:       &mockSigningKM{keyset: validKeyset},
			mockSigner:   &mockSigner{signature: validSignature},
			subscriberID: subscriberID,
			body:         body,
			wantHeaderPart: []string{
				`keyId="test.subscriber.com|key-123|ed25519"`,
				`algorithm="ed25519"`,
				`headers="(created) (expires) digest"`,
				`signature="generated-signature"`,
			},
			wantErr: "",
		},
		{
			name:         "keyset fetch error",
			mockKM:       &mockSigningKM{err: errors.New("db connection failed")},
			mockSigner:   &mockSigner{},
			subscriberID: subscriberID,
			body:         body,
			wantErr:      "failed to get keyset for signing",
		},
		{
			name:         "signing error",
			mockKM:       &mockSigningKM{keyset: validKeyset},
			mockSigner:   &mockSigner{err: errors.New("crypto error")},
			subscriberID: subscriberID,
			body:         body,
			wantErr:      "failed to sign body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authGenService, _ := NewAuthGenService(tt.mockKM, tt.mockSigner)
			gotHeader, err := authGenService.AuthHeader(ctx, tt.body, tt.subscriberID)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("AuthHeader() error = %v, want error containing %q", err, tt.wantErr)
				}
				if gotHeader != "" {
					t.Errorf("AuthHeader() gotHeader = %q, want empty string on error", gotHeader)
				}
			} else {
				if err != nil {
					t.Errorf("AuthHeader() unexpected error = %v", err)
				}
				for _, part := range tt.wantHeaderPart {
					if !strings.Contains(gotHeader, part) {
						t.Errorf("AuthHeader() = %q, does not contain expected part %q", gotHeader, part)
					}
				}
			}
		})
	}
}
