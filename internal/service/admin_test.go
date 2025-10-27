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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"github.com/google/go-cmp/cmp"
)

// mockAdminEventPublisher is a mock implementation of adminEventPublisher.
type mockAdminEventPublisher struct {
	msgID string
	err   error
}

func (m *mockAdminEventPublisher) PublishSubscriptionRequestApprovedEvent(ctx context.Context, req *model.LRO) (string, error) {
	return m.msgID, m.err
}
func (m *mockAdminEventPublisher) PublishSubscriptionRequestRejectedEvent(ctx context.Context, req *model.LRO) (string, error) {
	return m.msgID, m.err
}

// mockRegRepo is a mock implementation of regRepo interface.
type mockRegRepo struct {
	getOperationErr             error
	updateOperationErr          error
	upsertSubscriptionAndLROErr error
	lroToReturn                 *model.LRO
	subToReturn                 *model.Subscription
	lookupSubsToReturn          []model.Subscription
	lookupErr                   error
	updatedLROToReturn          *model.LRO // For UpdateOperation and Upsert
}

func (m *mockRegRepo) GetOperation(ctx context.Context, operationID string) (*model.LRO, error) {
	return m.lroToReturn, m.getOperationErr
}

func (m *mockRegRepo) UpdateOperation(ctx context.Context, lro *model.LRO) (*model.LRO, error) {
	return m.updatedLROToReturn, m.updateOperationErr
}

func (m *mockRegRepo) UpsertSubscriptionAndLRO(ctx context.Context, sub *model.Subscription, lro *model.LRO) (*model.Subscription, *model.LRO, error) {
	return m.subToReturn, m.updatedLROToReturn, m.upsertSubscriptionAndLROErr
}

func (m *mockRegRepo) Lookup(ctx context.Context, sub *model.Subscription) ([]model.Subscription, error) {
	return m.lookupSubsToReturn, m.lookupErr
}

// mockChallengeSrv is a mock implementation of challengeSrv.
type mockChallengeSrv struct {
	challengeToReturn string
	newChallengeErr   error
	verifyResult      bool
}

func (m *mockChallengeSrv) NewChallenge() (string, error) {
	return m.challengeToReturn, m.newChallengeErr
}

func (m *mockChallengeSrv) Verify(challenge, answer string) bool {
	return m.verifyResult
}

// mockEncryptionSrv is a mock implementation of encrypter.
type mockEncryptionSrv struct {
	encryptedDataToReturn string
	encryptErr            error
}

func (m *mockEncryptionSrv) Encrypt(ctx context.Context, data string, npKey string) (string, error) {
	return m.encryptedDataToReturn, m.encryptErr
}

// mockNPClient is a mock implementation of npClient.
type mockNPClient struct {
	onSubscribeResponseToReturn *model.OnSubscribeResponse
	onSubscribeErr              error
}

func (m *mockNPClient) OnSubscribe(ctx context.Context, callbackURL string, request *model.OnSubscribeRequest) (*model.OnSubscribeResponse, error) {
	return m.onSubscribeResponseToReturn, m.onSubscribeErr
}

func TestNewAdminService_Success(t *testing.T) {
	cfg := &AdminConfig{OperationRetryMax: 3}
	_, err := NewAdminService(&mockRegRepo{}, &mockChallengeSrv{}, &mockEncryptionSrv{}, &mockNPClient{}, &mockAdminEventPublisher{}, cfg)
	if err != nil {
		t.Fatalf("NewAdminService() error = %v, wantErr nil", err)
	}
}

func TestNewAdminService_Error(t *testing.T) {
	validCfg := &AdminConfig{OperationRetryMax: 3}
	invalidCfg := &AdminConfig{OperationRetryMax: -3}

	tests := []struct {
		name      string
		regRepo   regRepo
		chSrv     challengeSrv
		encryptor encrypterSrv
		npClient  npClient
		cfg       *AdminConfig
		evPub     adminEventPublisher
		wantErr   string
	}{
		{"nil regRepo", nil, &mockChallengeSrv{}, &mockEncryptionSrv{}, &mockNPClient{}, validCfg, &mockAdminEventPublisher{}, "regRepo cannot be nil"},
		{"nil challengeService", &mockRegRepo{}, nil, &mockEncryptionSrv{}, &mockNPClient{}, validCfg, &mockAdminEventPublisher{}, "challengeService cannot be nil"},
		{"nil encryptor", &mockRegRepo{}, &mockChallengeSrv{}, nil, &mockNPClient{}, validCfg, &mockAdminEventPublisher{}, "encryptor cannot be nil"},
		{"nil npClient", &mockRegRepo{}, &mockChallengeSrv{}, &mockEncryptionSrv{}, nil, validCfg, &mockAdminEventPublisher{}, "npClient cannot be nil"},
		{"nil eventPublisher", &mockRegRepo{}, &mockChallengeSrv{}, &mockEncryptionSrv{}, &mockNPClient{}, validCfg, nil, "eventPublisher cannot be nil"},
		{"nil AdminConfig", &mockRegRepo{}, &mockChallengeSrv{}, &mockEncryptionSrv{}, &mockNPClient{}, nil, &mockAdminEventPublisher{}, "AdminConfig cannot be nil"},
		{"invalid AdminConfig", &mockRegRepo{}, &mockChallengeSrv{}, &mockEncryptionSrv{}, &mockNPClient{}, invalidCfg, &mockAdminEventPublisher{}, "AdminConfig.OperationRetryMax cannot be zero or negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAdminService(tt.regRepo, tt.chSrv, tt.encryptor, tt.npClient, tt.evPub, tt.cfg)
			if err == nil {
				t.Fatalf("NewAdminService() error = nil, wantErr %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("NewAdminService() error = %q, wantErr %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestAdminService_ApproveSubscription_Success(t *testing.T) {
	ctx := context.Background()
	opID := "test-op-approve-success"
	now := time.Now()
	subReq := &model.SubscriptionRequest{
		Subscription: model.Subscription{
			Subscriber: model.Subscriber{
				SubscriberID: "sub1",
				URL:          "http://np.com",
				Type:         model.RoleBAP,
				Domain:       "retail",
			},
			KeyID:         "key1",
			EncrPublicKey: "np-encr-pub-key",
		},
		MessageID: opID,
	}
	subReqJSON, _ := json.Marshal(subReq)

	initialLRO := &model.LRO{
		OperationID: opID,
		Type:        model.OperationTypeCreateSubscription,
		Status:      model.LROStatusPending,
		RequestJSON: subReqJSON,
		CreatedAt:   now,
		UpdatedAt:   now,
		RetryCount:  0,
	}

	approvedSub := &model.Subscription{
		Subscriber:    subReq.Subscriber,
		KeyID:         subReq.KeyID,
		EncrPublicKey: subReq.EncrPublicKey,
		Status:        model.SubscriptionStatusSubscribed, // Status updated
	}

	approvedLRO := &model.LRO{
		OperationID: opID,
		Type:        model.OperationTypeCreateSubscription,
		Status:      model.LROStatusApproved, // Status updated
		RequestJSON: subReqJSON,
		CreatedAt:   now,
		UpdatedAt:   now, // Updated time
		RetryCount:  0,
	}

	mockRepo := &mockRegRepo{
		lroToReturn:        initialLRO,
		subToReturn:        approvedSub,
		updatedLROToReturn: approvedLRO,
	}
	mockChSrv := &mockChallengeSrv{challengeToReturn: "challenge123", verifyResult: true}
	mockEnc := &mockEncryptionSrv{encryptedDataToReturn: "encryptedChallenge"}
	mockNpCli := &mockNPClient{onSubscribeResponseToReturn: &model.OnSubscribeResponse{Answer: "challenge123"}}
	cfg := &AdminConfig{OperationRetryMax: 3}

	service, _ := NewAdminService(mockRepo, mockChSrv, mockEnc, mockNpCli, &mockAdminEventPublisher{}, cfg)

	req := &model.OperationActionRequest{OperationID: opID}
	gotSub, gotLRO, err := service.ApproveSubscription(ctx, req)
	if err != nil {
		t.Fatalf("ApproveSubscription() error = %v, wantErr nil", err)
	}

	if diff := cmp.Diff(approvedSub, gotSub); diff != "" {
		t.Errorf("ApproveSubscription() subscription mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(approvedLRO, gotLRO); diff != "" {
		t.Errorf("ApproveSubscription() LRO mismatch (-want +got):\n%s", diff)
	}
}

func TestAdminService_ApproveSubscription_EventPublishError(t *testing.T) {
	ctx := context.Background()
	opID := "test-op-approve-event-error"
	now := time.Now()
	subReq := &model.SubscriptionRequest{
		Subscription: model.Subscription{
			Subscriber: model.Subscriber{
				SubscriberID: "sub1",
				URL:          "http://np.com",
				Type:         model.RoleBAP,
				Domain:       "retail",
			},
			KeyID:            "key1",
			EncrPublicKey:    "np-encr-pub-key",
			SigningPublicKey: "np-signing-pub-key",
		},
		MessageID: opID,
	}
	subReqJSON, _ := json.Marshal(subReq)

	initialLRO := &model.LRO{OperationID: opID, Type: model.OperationTypeCreateSubscription, Status: model.LROStatusPending, RequestJSON: subReqJSON, CreatedAt: now, UpdatedAt: now}
	approvedLRO := &model.LRO{OperationID: opID, Type: model.OperationTypeCreateSubscription, Status: model.LROStatusApproved, RequestJSON: subReqJSON, CreatedAt: now, UpdatedAt: now}
	approvedSub := &model.Subscription{Subscriber: subReq.Subscriber, Status: model.SubscriptionStatusSubscribed}

	mockRepo := &mockRegRepo{lroToReturn: initialLRO, subToReturn: approvedSub, updatedLROToReturn: approvedLRO}
	mockChSrv := &mockChallengeSrv{challengeToReturn: "challenge123", verifyResult: true}
	mockEnc := &mockEncryptionSrv{encryptedDataToReturn: "encryptedChallenge"}
	mockNpCli := &mockNPClient{onSubscribeResponseToReturn: &model.OnSubscribeResponse{Answer: "challenge123"}}
	mockEvPub := &mockAdminEventPublisher{err: errors.New("event publish failed")} // Simulate event publish error
	cfg := &AdminConfig{OperationRetryMax: 3}

	service, _ := NewAdminService(mockRepo, mockChSrv, mockEnc, mockNpCli, mockEvPub, cfg)

	req := &model.OperationActionRequest{OperationID: opID}
	_, _, err := service.ApproveSubscription(ctx, req)
	if err != nil {
		t.Fatalf("ApproveSubscription() unexpected error: %v", err)
	}
	// The error is logged, not returned, so we just ensure the function completes without panicking.
	// In a real scenario, you might use a test logger to assert the log message.
}

func TestAdminService_ApproveSubscription_Error(t *testing.T) {
	ctx := context.Background()
	opID := "test-op-approve-error"
	now := time.Now()
	validSubReq := &model.SubscriptionRequest{
		Subscription: model.Subscription{
			Subscriber:    model.Subscriber{SubscriberID: "sub1", URL: "http://np.com", Type: model.RoleBAP, Domain: "retail"},
			KeyID:         "key1",
			EncrPublicKey: "np-encr-pub-key",
		},
		MessageID: opID,
	}
	validSubReqJSON, _ := json.Marshal(validSubReq)

	baseLRO := func() *model.LRO { // Helper to get a fresh LRO for each test
		return &model.LRO{
			OperationID: opID, Type: model.OperationTypeCreateSubscription, Status: model.LROStatusPending,
			RequestJSON: validSubReqJSON, CreatedAt: now, UpdatedAt: now, RetryCount: 0,
		}
	}

	tests := []struct {
		name               string
		operationID        string
		mockRepoSetup      func(*mockRegRepo)
		mockChallengeSetup func(*mockChallengeSrv)
		mockEncrypterSetup func(*mockEncryptionSrv)
		mockNPClientSetup  func(*mockNPClient)
		adminCfg           *AdminConfig
		wantErrMsgContains string
		wantLROStatus      model.LROStatus // Expected LRO status after error handling
	}{
		{
			name:        "LRO not found",
			operationID: "nonexistent-op",
			mockRepoSetup: func(m *mockRegRepo) {
				m.getOperationErr = repository.ErrOperationNotFound
			},
			wantErrMsgContains: "failed to get LRO: operation not found",
		},
		{
			name:        "Max retries exceeded",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				lro := baseLRO()
				lro.RetryCount = 5 // Exceeds default max
				m.lroToReturn = lro
			},
			adminCfg:           &AdminConfig{OperationRetryMax: 3},
			wantErrMsgContains: "max retries exceeded for operation",
		},
		{
			name:        "Invalid LRO type",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				lro := baseLRO()
				lro.Type = "INVALID_TYPE"
				m.lroToReturn = lro
			},
			wantErrMsgContains: "invalid operation type: INVALID_TYPE",
		},
		{
			name:        "LRO already approved",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				lro := baseLRO()
				lro.Status = model.LROStatusApproved
				m.lroToReturn = lro
			},
			wantErrMsgContains: fmt.Sprintf("%s: operation %s has status %s", ErrLROAlreadyProcessed, opID, model.LROStatusApproved),
		},
		{
			name:        "LRO already rejected",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				lro := baseLRO()
				lro.Status = model.LROStatusRejected
				m.lroToReturn = lro
			},
			wantErrMsgContains: fmt.Sprintf("%s: operation %s has status %s", ErrLROAlreadyProcessed, opID, model.LROStatusRejected),
		},
		{
			name:        "Failed to unmarshal LRO request JSON",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				lro := baseLRO()
				lro.RequestJSON = []byte("invalid json")
				m.lroToReturn = lro
			},
			wantErrMsgContains: "failed to unmarshal LRO request JSON",
			wantLROStatus:      model.LROStatusRejected, // Should be rejected after this failure
		},
		{
			name:        "Callback URL missing in subscription request",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				lro := baseLRO()
				badSubReq := *validSubReq // copy
				badSubReq.URL = ""
				badSubReqJSON, _ := json.Marshal(badSubReq)
				lro.RequestJSON = badSubReqJSON
				m.lroToReturn = lro
			},
			wantErrMsgContains: "callback URL missing in subscription request",
			wantLROStatus:      model.LROStatusRejected,
		},
		{
			name:        "Encryption public key missing in subscription request",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				lro := baseLRO()
				badSubReq := *validSubReq // copy
				badSubReq.EncrPublicKey = ""
				badSubReqJSON, _ := json.Marshal(badSubReq)
				lro.RequestJSON = badSubReqJSON
				m.lroToReturn = lro
			},
			wantErrMsgContains: "encryption public key missing",
			wantLROStatus:      model.LROStatusRejected,
		},
		{
			name:        "Failed to generate challenge",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				m.lroToReturn = baseLRO()
			},
			mockChallengeSetup: func(m *mockChallengeSrv) {
				m.newChallengeErr = errors.New("challenge gen error")
			},
			wantErrMsgContains: "failed to generate challenge",
			wantLROStatus:      model.LROStatusFailure,
		},
		{
			name:        "Failed to encrypt challenge",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				m.lroToReturn = baseLRO()
			},
			mockChallengeSetup: func(m *mockChallengeSrv) { m.challengeToReturn = "chal1" },
			mockEncrypterSetup: func(m *mockEncryptionSrv) {
				m.encryptErr = errors.New("encrypt error")
			},
			wantErrMsgContains: "failed to encrypt challenge",
			wantLROStatus:      model.LROStatusFailure,
		},
		{
			name:        "NP /on_subscribe callback failed",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				m.lroToReturn = baseLRO()
			},
			mockChallengeSetup: func(m *mockChallengeSrv) { m.challengeToReturn = "chal1" },
			mockEncrypterSetup: func(m *mockEncryptionSrv) { m.encryptedDataToReturn = "encrChal" },
			mockNPClientSetup: func(m *mockNPClient) {
				m.onSubscribeErr = errors.New("np client error")
			},
			wantErrMsgContains: "network Participant /on_subscribe callback failed",
			wantLROStatus:      model.LROStatusFailure,
		},
		{
			name:        "Challenge verification failed",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				m.lroToReturn = baseLRO()
			},
			mockChallengeSetup: func(m *mockChallengeSrv) {
				m.challengeToReturn = "chal1"
				m.verifyResult = false // Verification fails
			},
			mockEncrypterSetup: func(m *mockEncryptionSrv) { m.encryptedDataToReturn = "encrChal" },
			mockNPClientSetup: func(m *mockNPClient) {
				m.onSubscribeResponseToReturn = &model.OnSubscribeResponse{Answer: "wrongAnswer"}
			},
			wantErrMsgContains: "challenge verification failed",
			wantLROStatus:      model.LROStatusFailure,
		},
		{
			name:        "Failed to upsert subscription and LRO on final approval",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				m.lroToReturn = baseLRO()
				m.upsertSubscriptionAndLROErr = errors.New("db upsert failed")
			},
			mockChallengeSetup: func(m *mockChallengeSrv) {
				m.challengeToReturn = "chal1"
				m.verifyResult = true
			},
			mockEncrypterSetup: func(m *mockEncryptionSrv) { m.encryptedDataToReturn = "encrChal" },
			mockNPClientSetup: func(m *mockNPClient) {
				m.onSubscribeResponseToReturn = &model.OnSubscribeResponse{Answer: "chal1"}
			},
			wantErrMsgContains: "db upsert failed",
		},
		{
			name:        "Failed to update LRO with error status after original failure (critical error)",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				m.lroToReturn = baseLRO()
				m.updateOperationErr = errors.New("critical: failed to update LRO status")
			},
			mockChallengeSetup: func(m *mockChallengeSrv) {
				m.newChallengeErr = errors.New("original challenge gen error") // Original error
			},
			wantErrMsgContains: "failed to generate challenge: original challenge gen error",
		},
		{
			name:        "Lookup fails",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				m.lroToReturn = baseLRO()
				m.lookupErr = errors.New("db lookup error")
			},
			wantErrMsgContains: "lookup failed: db lookup error",
			// LRO status is not updated by the service in this case, as the error happens before updateLROError is called.
			// The original LRO is returned, so its status remains PENDING.
			wantLROStatus: model.LROStatusPending,
		},
		{
			name:        "Subscription already exists for CREATE operation",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				lro := baseLRO()
				lro.Type = model.OperationTypeCreateSubscription // Ensure it's a CREATE operation
				m.lroToReturn = lro
				m.lookupSubsToReturn = []model.Subscription{{Subscriber: model.Subscriber{SubscriberID: "sub1"}}} // Simulate existing subscription
			},
			wantErrMsgContains: "subscription already exists: subscriber_id 'sub1'",
			wantLROStatus:      model.LROStatusFailure, // Should be failed and LRO updated
		},
		{
			name:        "Subscription does not exist for UPDATE operation",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				lro := baseLRO()
				lro.Type = model.OperationTypeUpdateSubscription // Ensure it's an UPDATE operation
				m.lroToReturn = lro
				m.lookupSubsToReturn = []model.Subscription{} // Simulate no existing subscription
			},
			wantErrMsgContains: "subscription does not exists: subscriber_id 'sub1'",
			wantLROStatus:      model.LROStatusFailure, // Should be failed and LRO updated
		},
		{
			name:        "Lookup returns error",
			operationID: opID,
			mockRepoSetup: func(m *mockRegRepo) {
				m.lroToReturn = baseLRO()
				m.lookupErr = errors.New("simulated lookup error")
			},
			wantErrMsgContains: "lookup failed: simulated lookup error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRegRepo{}
			mockChSrv := &mockChallengeSrv{}
			mockEnc := &mockEncryptionSrv{}
			mockNpCli := &mockNPClient{}

			if tt.mockRepoSetup != nil {
				tt.mockRepoSetup(mockRepo)
			}
			if tt.mockChallengeSetup != nil {
				tt.mockChallengeSetup(mockChSrv)
			}
			if tt.mockEncrypterSetup != nil {
				tt.mockEncrypterSetup(mockEnc)
			}
			if tt.mockNPClientSetup != nil {
				tt.mockNPClientSetup(mockNpCli)
			}

			cfg := tt.adminCfg
			if cfg == nil {
				cfg = &AdminConfig{OperationRetryMax: 3}
			}

			service, _ := NewAdminService(mockRepo, mockChSrv, mockEnc, mockNpCli, &mockAdminEventPublisher{}, cfg)

			req := &model.OperationActionRequest{OperationID: tt.operationID}
			_, lroAfterError, err := service.ApproveSubscription(ctx, req)

			if err == nil {
				t.Fatalf("ApproveSubscription() error = nil, want error containing %q", tt.wantErrMsgContains)
			}
			if !strings.Contains(err.Error(), tt.wantErrMsgContains) {
				t.Errorf("ApproveSubscription() error = %q, want error containing %q", err.Error(), tt.wantErrMsgContains)
			}

			if tt.wantLROStatus != "" && lroAfterError != nil {
				if lroAfterError.Status != tt.wantLROStatus {
					t.Errorf("ApproveSubscription() LRO status after error = %s, want %s", lroAfterError.Status, tt.wantLROStatus)
				}
				if len(lroAfterError.ErrorDataJSON) == 0 {
					if tt.name != "LRO not found" && tt.name != "Max retries exceeded" && tt.name != "Invalid LRO type" && !strings.Contains(tt.name, "already processed") && !strings.Contains(tt.name, "Failed to upsert") && !strings.Contains(tt.name, "critical error") {
						t.Errorf("ApproveSubscription() LRO ErrorDataJSON is empty after error, expected it to be populated. LRO: %+v", lroAfterError)
					}
				}
			}
		})
	}
}

func TestAdminService_RejectSubscription_Success(t *testing.T) {
	ctx := context.Background()
	opID := "test-op-reject-success"
	reason := "Admin rejected"
	now := time.Now()
	subReq := &model.SubscriptionRequest{MessageID: opID}
	subReqJSON, _ := json.Marshal(subReq)

	initialLRO := &model.LRO{
		OperationID: opID, Type: model.OperationTypeCreateSubscription, Status: model.LROStatusPending,
		RequestJSON: subReqJSON, CreatedAt: now, UpdatedAt: now, RetryCount: 0,
	}

	rejectedLRO := &model.LRO{
		OperationID: opID,
		Type:        model.OperationTypeCreateSubscription,
		Status:      model.LROStatusRejected,
		RequestJSON: subReqJSON,
		ErrorDataJSON: func() []byte {
			b, _ := json.Marshal(map[string]string{"reason": reason})
			return b
		}(),
		CreatedAt:  now,
		UpdatedAt:  now,
		RetryCount: 0,
	}

	mockRepo := &mockRegRepo{
		lroToReturn:        initialLRO,
		updatedLROToReturn: rejectedLRO,
	}
	cfg := &AdminConfig{OperationRetryMax: 3}
	service, _ := NewAdminService(mockRepo, &mockChallengeSrv{}, &mockEncryptionSrv{}, &mockNPClient{}, &mockAdminEventPublisher{}, cfg)

	req := &model.OperationActionRequest{OperationID: opID, Reason: reason}
	gotLRO, err := service.RejectSubscription(ctx, req)
	if err != nil {
		t.Fatalf("RejectSubscription() error = %v, wantErr nil", err)
	}
	if diff := cmp.Diff(rejectedLRO, gotLRO); diff != "" {
		t.Errorf("RejectSubscription() LRO mismatch (-want +got):\n%s", diff)
	}
}

func TestAdminService_RejectSubscription_EventPublishError(t *testing.T) {
	ctx := context.Background()
	opID := "test-op-reject-event-error"
	reason := "Admin rejected"
	now := time.Now()
	subReqJSON, _ := json.Marshal(&model.SubscriptionRequest{MessageID: opID})

	initialLRO := &model.LRO{OperationID: opID, Type: model.OperationTypeCreateSubscription, Status: model.LROStatusPending, RequestJSON: subReqJSON, CreatedAt: now, UpdatedAt: now}
	rejectedLRO := &model.LRO{OperationID: opID, Type: model.OperationTypeCreateSubscription, Status: model.LROStatusRejected, RequestJSON: subReqJSON, CreatedAt: now, UpdatedAt: now}

	mockRepo := &mockRegRepo{lroToReturn: initialLRO, updatedLROToReturn: rejectedLRO}
	mockEvPub := &mockAdminEventPublisher{err: errors.New("event publish failed")} // Simulate event publish error
	cfg := &AdminConfig{OperationRetryMax: 3}

	service, _ := NewAdminService(mockRepo, &mockChallengeSrv{}, &mockEncryptionSrv{}, &mockNPClient{}, mockEvPub, cfg)

	req := &model.OperationActionRequest{OperationID: opID, Reason: reason}
	_, err := service.RejectSubscription(ctx, req)
	if err != nil {
		t.Fatalf("RejectSubscription() unexpected error: %v", err)
	}
	// The error is logged, not returned, so we just ensure the function completes without panicking.
}

func TestAdminService_RejectSubscription_Error(t *testing.T) {
	ctx := context.Background()
	opID := "test-op-reject-error"
	reason := "Admin rejected"
	now := time.Now()
	subReqJSON, _ := json.Marshal(&model.SubscriptionRequest{MessageID: opID})

	baseLRO := func() *model.LRO {
		return &model.LRO{
			OperationID: opID, Type: model.OperationTypeCreateSubscription, Status: model.LROStatusPending,
			RequestJSON: subReqJSON, CreatedAt: now, UpdatedAt: now, RetryCount: 0,
		}
	}

	tests := []struct {
		name               string
		mockRepoSetup      func(*mockRegRepo)
		adminCfg           *AdminConfig
		wantErrMsgContains string
		req                *model.OperationActionRequest
	}{
		{
			name: "LRO not found",
			mockRepoSetup: func(m *mockRegRepo) {
				m.getOperationErr = repository.ErrOperationNotFound
			},
			wantErrMsgContains: "failed to get LRO: operation not found",
			req:                &model.OperationActionRequest{OperationID: "nonexistent-op", Reason: reason},
		},
		{
			name: "Max retries exceeded (though not directly applicable to reject, lro() check is common)",
			mockRepoSetup: func(m *mockRegRepo) {
				lro := baseLRO()
				lro.RetryCount = 5
				m.lroToReturn = lro
			},
			adminCfg:           &AdminConfig{OperationRetryMax: 3},
			wantErrMsgContains: "max retries exceeded for operation",
			req:                &model.OperationActionRequest{OperationID: opID, Reason: reason},
		},
		{
			name: "Failed to update LRO during rejection",
			mockRepoSetup: func(m *mockRegRepo) {
				m.lroToReturn = baseLRO()
				m.updateOperationErr = errors.New("db update failed for reject")
			},
			wantErrMsgContains: "failed to update LRO error: db update failed for reject",
			req:                &model.OperationActionRequest{OperationID: opID, Reason: reason},
		},
		{
			name: "nil OperationActionRequest",
			mockRepoSetup: func(m *mockRegRepo) {
				m.lroToReturn = baseLRO()
			},
			req:                nil, // Explicitly test nil request
			wantErrMsgContains: "OperationActionRequest cannot be nil",
		},
		{
			name: "empty OperationID",
			mockRepoSetup: func(m *mockRegRepo) {
				m.lroToReturn = baseLRO()
			},
			wantErrMsgContains: "OperationID cannot be empty",
			req:                &model.OperationActionRequest{OperationID: "", Reason: reason},
		},
		{
			name: "empty Reason",
			mockRepoSetup: func(m *mockRegRepo) {
				m.lroToReturn = baseLRO()
			},
			wantErrMsgContains: "reason cannot be empty",
			req:                &model.OperationActionRequest{OperationID: opID, Reason: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRegRepo{}
			if tt.mockRepoSetup != nil {
				tt.mockRepoSetup(mockRepo)
			}

			cfg := tt.adminCfg
			if cfg == nil {
				cfg = &AdminConfig{OperationRetryMax: 3}
			}

			service, _ := NewAdminService(mockRepo, &mockChallengeSrv{}, &mockEncryptionSrv{}, &mockNPClient{}, &mockAdminEventPublisher{}, cfg)

			_, err := service.RejectSubscription(ctx, tt.req)

			if err == nil {
				t.Fatalf("RejectSubscription() error = nil, want error containing %q", tt.wantErrMsgContains)
			}
			if !strings.Contains(err.Error(), tt.wantErrMsgContains) {
				t.Errorf("RejectSubscription() error = %q, want error containing %q", err.Error(), tt.wantErrMsgContains)
			}
		})
	}
}
