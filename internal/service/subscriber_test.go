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
	"strings"
	"testing"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	becknmodel "github.com/beckn/beckn-onix/pkg/model"
)

// mockRegistryClient is a mock for registryClient.
type mockRegistryClient struct {
	createSubResp *model.SubscriptionResponse
	createSubErr  error
	updateSubResp *model.SubscriptionResponse
	updateSubErr  error
	getOpResp     *model.LRO
	getOpErr      error
}

func (m *mockRegistryClient) CreateSubscription(ctx context.Context, req *model.SubscriptionRequest) (*model.SubscriptionResponse, error) {
	return m.createSubResp, m.createSubErr
}
func (m *mockRegistryClient) UpdateSubscription(ctx context.Context, req *model.SubscriptionRequest, authHeader string) (*model.SubscriptionResponse, error) {
	return m.updateSubResp, m.updateSubErr
}
func (m *mockRegistryClient) GetOperation(ctx context.Context, operationID string) (*model.LRO, error) {
	return m.getOpResp, m.getOpErr
}

// mockOnSubscribeEventPublisher is a mock for onSubscribeEventPublisher.
type mockOnSubscribeEventPublisher struct {
	publishErr error
	eventID    string
}

func (m *mockOnSubscribeEventPublisher) PublishOnSubscribeRecievedEvent(ctx context.Context, lroID string) (string, error) {
	return m.eventID, m.publishErr
}

// mockKeyManager is a mock for keyManager.
type mockKeyManager struct {
	keysetToReturn      *becknmodel.Keyset
	keysetErr           error
	generateKeysetErr   error
	insertKeysetErr     error
	deleteKeysetErr     error
	lookupNPKeysSigning string
	lookupNPKeysEncr    string
	lookupNPKeysErr     error
}

func (m *mockKeyManager) Keyset(ctx context.Context, keyID string) (*becknmodel.Keyset, error) {
	return m.keysetToReturn, m.keysetErr
}
func (m *mockKeyManager) GenerateKeyset() (*becknmodel.Keyset, error) {
	if m.generateKeysetErr != nil {
		return nil, m.generateKeysetErr
	}
	return &becknmodel.Keyset{UniqueKeyID: "generated-key", SigningPublic: "gen-sign-pub", EncrPublic: "gen-encr-pub", EncrPrivate: "gen-encr-priv"}, nil
}
func (m *mockKeyManager) InsertKeyset(ctx context.Context, keyID string, keyset *becknmodel.Keyset) error {
	return m.insertKeysetErr
}
func (m *mockKeyManager) DeleteKeyset(ctx context.Context, keyID string) error {
	return m.deleteKeysetErr
}
func (m *mockKeyManager) LookupNPKeys(ctx context.Context, subscriberID, uniqueKeyID string) (signingPublicKey string, encrPublicKey string, err error) {
	return m.lookupNPKeysSigning, m.lookupNPKeysEncr, m.lookupNPKeysErr
}

// mockDecrypter is a mock for decrypter.
type mockDecrypter struct {
	decryptedData string
	decryptErr    error
}

func (m *mockDecrypter) Decrypt(ctx context.Context, data string, privateKeyBase64, publicKeyBase64 string) (string, error) {
	return m.decryptedData, m.decryptErr
}

// mockAuthGen is a mock for authGen.
type mockAuthGen struct {
	authHeader string
	err        error
}

func (m *mockAuthGen) AuthHeader(ctx context.Context, body []byte, keyID string) (string, error) {
	return m.authHeader, m.err
}

func TestNewSubscriberService_Success(t *testing.T) {
	_, err := NewSubscriberService(
		&mockRegistryClient{},
		&mockKeyManager{},
		&mockDecrypter{},
		&mockOnSubscribeEventPublisher{},
		&mockAuthGen{},
		"reg-id", "reg-key-id",
	)
	if err != nil {
		t.Fatalf("NewSubscriberService() unexpected error: %v", err)
	}
}

func TestNewSubscriberService_Error(t *testing.T) {
	tests := []struct {
		name       string
		registry   registryClient
		keyMgr     keyManager
		dec        decrypter
		evPub      onSubscribeEventPublisher
		authGen    authGen
		regID      string
		regKeyID   string
		wantErrMsg string
	}{
		{"nil registry", nil, &mockKeyManager{}, &mockDecrypter{}, &mockOnSubscribeEventPublisher{}, &mockAuthGen{}, "reg-id", "reg-key-id", "registryClient cannot be nil"},
		{"nil keyMgr", &mockRegistryClient{}, nil, &mockDecrypter{}, &mockOnSubscribeEventPublisher{}, &mockAuthGen{}, "reg-id", "reg-key-id", "keyManager cannot be nil"},
		{"nil decrypter", &mockRegistryClient{}, &mockKeyManager{}, nil, &mockOnSubscribeEventPublisher{}, &mockAuthGen{}, "reg-id", "reg-key-id", "decrypter cannot be nil"},
		{"nil evPub", &mockRegistryClient{}, &mockKeyManager{}, &mockDecrypter{}, nil, &mockAuthGen{}, "reg-id", "reg-key-id", "eventPublisher (onSubscribeEventPublisher) cannot be nil"},
		{"nil authGen", &mockRegistryClient{}, &mockKeyManager{}, &mockDecrypter{}, &mockOnSubscribeEventPublisher{}, nil, "reg-id", "reg-key-id", "authGen cannot be nil"},
		{"empty regID", &mockRegistryClient{}, &mockKeyManager{}, &mockDecrypter{}, &mockOnSubscribeEventPublisher{}, &mockAuthGen{}, "", "reg-key-id", "regID cannot be empty"},
		{"empty regKeyID", &mockRegistryClient{}, &mockKeyManager{}, &mockDecrypter{}, &mockOnSubscribeEventPublisher{}, &mockAuthGen{}, "reg-id", "", "regKeyID cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSubscriberService(tt.registry, tt.keyMgr, tt.dec, tt.evPub, tt.authGen, tt.regID, tt.regKeyID)
			if err == nil || err.Error() != tt.wantErrMsg {
				t.Errorf("NewSubscriberService() error = %v, want %q", err, tt.wantErrMsg)
			}
		})
	}
}

func TestSubscriberService_CreateSubscription_Success(t *testing.T) {
	ctx := context.Background()
	req := &model.NpSubscriptionRequest{
		Subscriber: model.Subscriber{
			SubscriberID: "sub1", Domain: "test.com", Type: model.RoleBAP,
		},
	}
	mockReg := &mockRegistryClient{createSubResp: &model.SubscriptionResponse{MessageID: "some-msg-id", Status: "ACK"}}
	mockKM := &mockKeyManager{} // Will generate new keyset
	svc, _ := NewSubscriberService(mockReg, mockKM, &mockDecrypter{}, &mockOnSubscribeEventPublisher{}, &mockAuthGen{}, "reg-id", "reg-key-id")

	msgID, err := svc.CreateSubscription(ctx, req)
	if err != nil {
		t.Fatalf("CreateSubscription() unexpected error: %v", err)
	}
	if msgID != "some-msg-id" {
		t.Errorf("CreateSubscription() got msgID %q, want %q", msgID, "some-msg-id")
	}

	req.MessageID = ""
	msgID, err = svc.CreateSubscription(ctx, req)
	if err != nil {
		t.Fatalf("CreateSubscription() unexpected error: %v", err)
	}
	if msgID != "some-msg-id" {
		t.Errorf("CreateSubscription() got msgID %q, want %q", msgID, "some-msg-id")
	}
}

func TestSubscriberService_CreateSubscription_Error(t *testing.T) {
	ctx := context.Background()
	baseReq := &model.NpSubscriptionRequest{
		Subscriber: model.Subscriber{SubscriberID: "sub1", Domain: "test.com", Type: model.RoleBAP},
		MessageID:  "msg1",
	}

	tests := []struct {
		name    string
		req     *model.NpSubscriptionRequest
		mockReg *mockRegistryClient
		mockKM  *mockKeyManager
		wantErr error
	}{
		{
			name:    "validation error - missing subscriber ID",
			req:     &model.NpSubscriptionRequest{Subscriber: model.Subscriber{Domain: "test.com", Type: model.RoleBAP}},
			wantErr: ErrMissingSubscriberID,
		},
		{
			name:    "validation error - missing domain",
			req:     &model.NpSubscriptionRequest{Subscriber: model.Subscriber{SubscriberID: "sub1", Type: model.RoleBAP}},
			wantErr: ErrMissingDomain,
		},
		{
			name:    "validation error - missing type",
			req:     &model.NpSubscriptionRequest{Subscriber: model.Subscriber{SubscriberID: "sub1", Domain: "test.com"}},
			wantErr: ErrMissingType,
		},
		{
			name: "keyManager.Keyset error (fetching existing)",
			req: &model.NpSubscriptionRequest{
				Subscriber: model.Subscriber{
					SubscriberID: "sub1",
					Domain:       "test.com",
					Type:         model.RoleBAP,
				},
				KeyID: "existing-key",
			},
			mockKM:  &mockKeyManager{keysetErr: errors.New("keyset fetch failed")},
			wantErr: fmt.Errorf("%w: %s", ErrKeyFetchFailed, "keyset fetch failed"),
		},
		{
			name:    "keyManager.GenerateKeyset error (generating new)",
			req:     baseReq,
			mockKM:  &mockKeyManager{generateKeysetErr: errors.New("keyset gen failed")},
			wantErr: fmt.Errorf("%w: %v", ErrKeyGenerationFailed, errors.New("keyset gen failed")),
		},
		{
			name:    "keyManager.InsertKeyset error",
			req:     baseReq,
			mockKM:  &mockKeyManager{insertKeysetErr: errors.New("keyset insert failed")},
			wantErr: fmt.Errorf("%w: %s", ErrKeyStoreFailed, "keyset insert failed"),
		},
		{
			name:    "registry.CreateSubscription error",
			req:     baseReq,
			mockReg: &mockRegistryClient{createSubErr: errors.New("registry create failed")},
			mockKM:  &mockKeyManager{},
			wantErr: fmt.Errorf("%w: %v", ErrRegistryOperationFailed, errors.New("registry create failed")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := NewSubscriberService(
				tt.mockReg, tt.mockKM, &mockDecrypter{}, &mockOnSubscribeEventPublisher{}, &mockAuthGen{}, "reg-id", "reg-key-id",
			)
			if svc.registry == nil { // Default to a working mock if not provided
				svc.registry = &mockRegistryClient{}
			}
			if svc.keyMgr == nil { // Default to a working mock if not provided
				svc.keyMgr = &mockKeyManager{}
			}

			_, err := svc.CreateSubscription(ctx, tt.req)
			if err == nil {
				t.Fatalf("CreateSubscription() error = nil, want %v", tt.wantErr)
			}
			if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
				t.Errorf("CreateSubscription() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestSubscriberService_UpdateSubscription_Success(t *testing.T) {
	ctx := context.Background()
	req := &model.NpSubscriptionRequest{
		Subscriber: model.Subscriber{
			SubscriberID: "sub1", Domain: "test.com", Type: model.RoleBAP,
		},
	}
	mockReg := &mockRegistryClient{updateSubResp: &model.SubscriptionResponse{MessageID: "some-msg-id", Status: "ACK"}}
	mockKM := &mockKeyManager{}
	mockAuth := &mockAuthGen{authHeader: "test-auth-header"}
	svc, _ := NewSubscriberService(mockReg, mockKM, &mockDecrypter{}, &mockOnSubscribeEventPublisher{}, mockAuth, "reg-id", "reg-key-id")

	msgID, err := svc.UpdateSubscription(ctx, req)
	if err != nil {
		t.Fatalf("UpdateSubscription() unexpected error: %v", err)
	}
	if msgID != "some-msg-id" {
		t.Errorf("UpdateSubscription() got msgID %q, want %q", msgID, "some-msg-id")
	}
}

func TestSubscriberService_UpdateSubscription_Error(t *testing.T) {
	ctx := context.Background()
	baseReq := &model.NpSubscriptionRequest{
		Subscriber: model.Subscriber{SubscriberID: "sub1", Domain: "test.com", Type: model.RoleBAP},
		MessageID:  "msg1",
	}

	tests := []struct {
		name     string
		req      *model.NpSubscriptionRequest
		mockReg  *mockRegistryClient
		mockKM   *mockKeyManager
		mockAuth *mockAuthGen
		wantErr  error
	}{
		{
			name:    "validation error",
			req:     &model.NpSubscriptionRequest{Subscriber: model.Subscriber{SubscriberID: "sub1"}},
			wantErr: ErrMissingDomain,
		},
		{
			name:    "keySet error",
			req:     baseReq,
			mockKM:  &mockKeyManager{generateKeysetErr: errors.New("gen failed")},
			wantErr: fmt.Errorf("%w: %v", ErrKeyGenerationFailed, errors.New("gen failed")),
		},
		{
			name:    "InsertKeyset error",
			req:     baseReq,
			mockKM:  &mockKeyManager{insertKeysetErr: errors.New("insert failed")},
			wantErr: fmt.Errorf("%w: %v", ErrKeyStoreFailed, errors.New("insert failed")),
		},
		{
			name:     "authHeader generation error",
			req:      baseReq,
			mockKM:   &mockKeyManager{},
			mockAuth: &mockAuthGen{err: errors.New("auth gen failed")},
			wantErr:  fmt.Errorf("%w: %v", ErrKeyGenerationFailed, errors.New("auth gen failed")),
		},
		{
			name:     "registry.UpdateSubscription error",
			req:      baseReq,
			mockReg:  &mockRegistryClient{updateSubErr: errors.New("registry update failed")},
			mockKM:   &mockKeyManager{},
			mockAuth: &mockAuthGen{},
			wantErr:  fmt.Errorf("%w: %v", ErrRegistryOperationFailed, errors.New("registry update failed")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := NewSubscriberService(
				tt.mockReg, tt.mockKM, &mockDecrypter{}, &mockOnSubscribeEventPublisher{}, tt.mockAuth, "reg-id", "reg-key-id",
			)
			if svc.registry == nil {
				svc.registry = &mockRegistryClient{}
			}
			if svc.keyMgr == nil {
				svc.keyMgr = &mockKeyManager{}
			}
			if svc.authGen == nil {
				svc.authGen = &mockAuthGen{}
			}

			_, err := svc.UpdateSubscription(ctx, tt.req)
			if err == nil {
				t.Fatalf("UpdateSubscription() error = nil, want %v", tt.wantErr)
			}
			if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
				t.Errorf("UpdateSubscription() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestSubscriberService_UpdateStatus_Success(t *testing.T) {
	ctx := context.Background()
	opID := "op1"
	mockReg := &mockRegistryClient{getOpResp: &model.LRO{Status: model.LROStatusApproved}}
	mockKM := &mockKeyManager{keysetToReturn: &becknmodel.Keyset{SubscriberID: "sub1"}}
	svc, _ := NewSubscriberService(mockReg, mockKM, &mockDecrypter{}, &mockOnSubscribeEventPublisher{}, &mockAuthGen{}, "reg-id", "reg-key-id")

	status, err := svc.UpdateStatus(ctx, opID)
	if err != nil {
		t.Fatalf("UpdateStatus() unexpected error: %v", err)
	}
	if status != model.LROStatusApproved {
		t.Errorf("UpdateStatus() got status %q, want %q", status, model.LROStatusApproved)
	}
}

func TestSubscriberService_UpdateStatus_Error(t *testing.T) {
	ctx := context.Background()
	opID := "op1"

	tests := []struct {
		name    string
		opID    string
		mockReg *mockRegistryClient
		mockKM  *mockKeyManager
		wantErr error
	}{
		{
			name:    "missing operation ID",
			opID:    "",
			wantErr: ErrMissingOperationID,
		},
		{
			name:    "GetOperation fails",
			opID:    opID,
			mockReg: &mockRegistryClient{getOpErr: errors.New("get op failed")},
			wantErr: fmt.Errorf("%w: %v", ErrLRONotFound, errors.New("get op failed")),
		},
		{
			name:    "LRO not found",
			opID:    opID,
			mockReg: &mockRegistryClient{getOpResp: nil},
			wantErr: ErrLRONotFound,
		},
		{
			name:    "LRO not approved",
			opID:    opID,
			mockReg: &mockRegistryClient{getOpResp: &model.LRO{Status: model.LROStatusPending}},
			wantErr: ErrLRONotApproved,
		},
		{
			name:    "Keyset fetch fails",
			opID:    opID,
			mockReg: &mockRegistryClient{getOpResp: &model.LRO{Status: model.LROStatusApproved}},
			mockKM:  &mockKeyManager{keysetErr: errors.New("keyset fetch failed")},
			wantErr: fmt.Errorf("%w: %v", ErrKeyFetchFailed, errors.New("keyset fetch failed")),
		},
		{
			name:    "Keyset insert fails",
			opID:    opID,
			mockReg: &mockRegistryClient{getOpResp: &model.LRO{Status: model.LROStatusApproved}},
			mockKM:  &mockKeyManager{keysetToReturn: &becknmodel.Keyset{SubscriberID: "sub1"}, insertKeysetErr: errors.New("keyset insert failed")},
			wantErr: fmt.Errorf("%w: %v", ErrKeyStoreFailed, errors.New("keyset insert failed")),
		},
		{
			name:    "Keyset delete fails (should not return error)",
			opID:    opID,
			mockReg: &mockRegistryClient{getOpResp: &model.LRO{Status: model.LROStatusApproved}},
			mockKM:  &mockKeyManager{keysetToReturn: &becknmodel.Keyset{SubscriberID: "sub1"}, deleteKeysetErr: errors.New("delete failed")},
			wantErr: nil, // The error is logged, not returned.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := NewSubscriberService(
				tt.mockReg, tt.mockKM, &mockDecrypter{}, &mockOnSubscribeEventPublisher{}, &mockAuthGen{}, "reg-id", "reg-key-id",
			)
			if svc.registry == nil {
				svc.registry = &mockRegistryClient{}
			}
			if svc.keyMgr == nil {
				svc.keyMgr = &mockKeyManager{}
			}

			_, err := svc.UpdateStatus(ctx, tt.opID)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("UpdateStatus() error = nil, want %v", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("UpdateStatus() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("UpdateStatus() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSubscriberService_OnSubscribe_Success(t *testing.T) {
	ctx := context.Background()
	req := &model.OnSubscribeRequest{MessageID: "msg1", Challenge: "encrypted-challenge"}
	mockKM := &mockKeyManager{
		keysetToReturn:   &becknmodel.Keyset{EncrPrivate: "np-private-key"},
		lookupNPKeysEncr: "reg-public-key",
	}
	mockDec := &mockDecrypter{decryptedData: "decrypted-answer"}
	mockEvPub := &mockOnSubscribeEventPublisher{eventID: "event1"}
	svc, _ := NewSubscriberService(&mockRegistryClient{}, mockKM, mockDec, mockEvPub, &mockAuthGen{}, "reg-id", "reg-key-id")

	resp, err := svc.OnSubscribe(ctx, req)
	if err != nil {
		t.Fatalf("OnSubscribe() unexpected error: %v", err)
	}
	if resp.Answer != "decrypted-answer" {
		t.Errorf("OnSubscribe() got answer %q, want %q", resp.Answer, "decrypted-answer")
	}
}

func TestSubscriberService_OnSubscribe_Error(t *testing.T) {
	ctx := context.Background()
	baseReq := &model.OnSubscribeRequest{MessageID: "msg1", Challenge: "encrypted-challenge"}

	tests := []struct {
		name       string
		req        *model.OnSubscribeRequest
		mockKM     *mockKeyManager
		mockDec    *mockDecrypter
		mockEvPub  *mockOnSubscribeEventPublisher
		wantErrMsg string
	}{
		{
			name:       "missing MessageID",
			req:        &model.OnSubscribeRequest{Challenge: "challenge"},
			wantErrMsg: "message_id is required",
		},
		{
			name:       "missing Challenge",
			req:        &model.OnSubscribeRequest{MessageID: "msg1"},
			wantErrMsg: "challenge is required",
		},
		{
			name:       "Keyset fetch fails",
			req:        baseReq,
			mockKM:     &mockKeyManager{keysetErr: errors.New("keyset fetch failed")},
			wantErrMsg: "failed to retrieve keys for message_id msg1: keyset fetch failed",
		},
		{
			name:       "Missing private key in keyset",
			req:        baseReq,
			mockKM:     &mockKeyManager{keysetToReturn: &becknmodel.Keyset{EncrPrivate: ""}},
			wantErrMsg: "encryption private key not found for message_id msg1",
		},
		{
			name:       "Registry key lookup fails",
			req:        baseReq,
			mockKM:     &mockKeyManager{keysetToReturn: &becknmodel.Keyset{EncrPrivate: "priv"}, lookupNPKeysErr: errors.New("lookup failed")},
			wantErrMsg: "failed to lookup registry keys for message_id msg1: lookup failed",
		},
		{
			name:       "Missing registry public key",
			req:        baseReq,
			mockKM:     &mockKeyManager{keysetToReturn: &becknmodel.Keyset{EncrPrivate: "priv"}, lookupNPKeysEncr: ""},
			wantErrMsg: "registry public key not found for message_id msg1",
		},
		{
			name:       "Decryption fails",
			req:        baseReq,
			mockKM:     &mockKeyManager{keysetToReturn: &becknmodel.Keyset{EncrPrivate: "priv"}, lookupNPKeysEncr: "pub"},
			mockDec:    &mockDecrypter{decryptErr: errors.New("decrypt failed")},
			wantErrMsg: "failed to decrypt challenge for message_id msg1: decrypt failed",
		},
		{
			name:       "Event publish fails (should not return error)",
			req:        baseReq,
			mockKM:     &mockKeyManager{keysetToReturn: &becknmodel.Keyset{EncrPrivate: "priv"}, lookupNPKeysEncr: "pub"},
			mockDec:    &mockDecrypter{decryptedData: "ok"},
			mockEvPub:  &mockOnSubscribeEventPublisher{publishErr: errors.New("publish failed")},
			wantErrMsg: "", // No error returned to caller
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := NewSubscriberService(
				&mockRegistryClient{}, tt.mockKM, tt.mockDec, tt.mockEvPub, &mockAuthGen{}, "reg-id", "reg-key-id",
			)
			if svc.keyMgr == nil {
				svc.keyMgr = &mockKeyManager{}
			}
			if svc.dec == nil {
				svc.dec = &mockDecrypter{}
			}
			if svc.evPub == nil {
				svc.evPub = &mockOnSubscribeEventPublisher{}
			}

			_, err := svc.OnSubscribe(ctx, tt.req)
			if tt.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("OnSubscribe() error = nil, want %q", tt.wantErrMsg)
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("OnSubscribe() error = %v, want error containing %q", err, tt.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("OnSubscribe() unexpected error: %v", err)
				}
			}
		})
	}
}
