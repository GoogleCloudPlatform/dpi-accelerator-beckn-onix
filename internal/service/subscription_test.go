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
	"reflect"
	"testing"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/internal/event/mock"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"github.com/google/go-cmp/cmp"
)

// mockLROCreator is a mock implementation of lroCreator.
type mockLROCreator struct {
	lro *model.LRO
	err error
}

func (m *mockLROCreator) Create(ctx context.Context, lro *model.LRO) (*model.LRO, error) {
	return m.lro, m.err
}

// mockSubscriptionRepository is a mock implementation of subscriptionRepository.
type mockSubscriptionRepository struct {
	key           string
	err           error
	subscriptions []model.Subscription
}

func (m *mockSubscriptionRepository) Lookup(ctx context.Context, filter *model.Subscription) ([]model.Subscription, error) {
	return m.subscriptions, m.err
}

func (m *mockSubscriptionRepository) GetSubscriberSigningKey(ctx context.Context, subscriberID string, domain string, subType model.Role, keyID string) (string, error) {
	return m.key, m.err
}

func TestNewSubscriptionService_Success(t *testing.T) {
	mockLRO := &mockLROCreator{}
	mockRepo := &mockSubscriptionRepository{}
	service, _ := NewSubscriptionService(mockLRO, mockRepo, &mock.EventPublisher{})

	if service == nil {
		t.Fatal("NewSubscriptionService() returned nil")
	}
	if service.lroCreator != mockLRO {
		t.Errorf("NewSubscriptionService() lroCreator not set correctly")
	}
	if service.subscriptionRepository != mockRepo {
		t.Errorf("NewSubscriptionService() subscriptionRepository not set correctly")
	}
}

func TestNewSubscriptionService_Error(t *testing.T) {
	tests := []struct {
		name                   string
		lroCreator             lroCreator
		subscriptionRepository subscriptionRepository
		evPub                  subscriptionEventPublisher
		expectedErrorMsg       string
	}{
		{
			name:                   "nil lroCreator",
			lroCreator:             nil,
			subscriptionRepository: &mockSubscriptionRepository{},
			evPub:                  &mock.EventPublisher{},
			expectedErrorMsg:       "lroCreator cannot be nil",
		},
		{
			name:                   "nil subscriptionRepository",
			lroCreator:             &mockLROCreator{},
			subscriptionRepository: nil,
			evPub:                  &mock.EventPublisher{},
			expectedErrorMsg:       "subscriptionRepository cannot be nil",
		},
		{
			name:                   "nil eventPublisher",
			lroCreator:             &mockLROCreator{},
			subscriptionRepository: &mockSubscriptionRepository{},
			evPub:                  nil,
			expectedErrorMsg:       "eventPublisher cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSubscriptionService(tt.lroCreator, tt.subscriptionRepository, tt.evPub)
			if err == nil || err.Error() != tt.expectedErrorMsg {
				t.Errorf("NewSubscriptionService() error = %v, want error message %q", err, tt.expectedErrorMsg)
			}
		})
	}
}

func TestSubscriptionServiceLookupSuccess(t *testing.T) {
	baseTime := time.Date(2023, time.January, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		filter       *model.Subscription
		mockRepoSubs []model.Subscription
		mockRepoErr  error
		expectedSubs []model.Subscription
	}{
		{
			name:   "Successful Lookup with results",
			filter: &model.Subscription{Subscriber: model.Subscriber{SubscriberID: "test1"}},
			mockRepoSubs: []model.Subscription{
				{
					Subscriber: model.Subscriber{SubscriberID: "test1", URL: "http://example.com", Type: model.RoleBAP},
					KeyID:      "key1", ValidFrom: baseTime, ValidUntil: baseTime.Add(time.Hour),
				},
			},
			mockRepoErr: nil,
			expectedSubs: []model.Subscription{
				{
					Subscriber: model.Subscriber{SubscriberID: "test1", URL: "http://example.com", Type: model.RoleBAP},
					KeyID:      "key1", ValidFrom: baseTime, ValidUntil: baseTime.Add(time.Hour),
				},
			},
		},
		{
			name:         "Successful Lookup with no results",
			filter:       &model.Subscription{Subscriber: model.Subscriber{SubscriberID: "nonexistent"}},
			mockRepoSubs: []model.Subscription{},
			mockRepoErr:  nil,
			expectedSubs: []model.Subscription{},
		},
		{
			name:   "Nil filter returns all",
			filter: nil,
			mockRepoSubs: []model.Subscription{
				{
					Subscriber: model.Subscriber{SubscriberID: "all_subs", URL: "http://all.com"},
				},
			},
			mockRepoErr: nil,
			expectedSubs: []model.Subscription{
				{
					Subscriber: model.Subscriber{SubscriberID: "all_subs", URL: "http://all.com"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockSubscriptionRepository{
				subscriptions: tt.mockRepoSubs,
				err:           tt.mockRepoErr,
			}
			service, err := NewSubscriptionService(&mockLROCreator{}, mockRepo, &mock.EventPublisher{})
			if err != nil {
				t.Fatalf("NewSubscriptionService() failed: %v", err)
			}
			ctx := context.Background()

			gotSubs, err := service.Lookup(ctx, tt.filter)

			if err != nil {
				t.Errorf("Lookup() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(gotSubs, tt.expectedSubs) {
				t.Errorf("Lookup() got = %+v, want %+v", gotSubs, tt.expectedSubs)
			}
		})
	}
}

func TestSubscriptionServiceLookupError(t *testing.T) {
	repoErr := errors.New("database connection failed")

	tests := []struct {
		name          string
		filter        *model.Subscription
		mockRepoErr   error
		expectedError error
	}{
		{
			name:          "Repository returns error",
			filter:        &model.Subscription{Subscriber: model.Subscriber{SubscriberID: "error_case"}},
			mockRepoErr:   repoErr,
			expectedError: fmt.Errorf("failed to lookup subscriptions: %w", repoErr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockSubscriptionRepository{
				subscriptions: nil,
				err:           tt.mockRepoErr,
			}
			service, err := NewSubscriptionService(&mockLROCreator{}, mockRepo, &mock.EventPublisher{})
			if err != nil {
				t.Fatalf("NewSubscriptionService() failed: %v", err)
			}
			ctx := context.Background()

			_, err = service.Lookup(ctx, tt.filter)
			if err == nil {
				t.Errorf("Lookup() expected an error, got nil")
			}
			if err != nil && tt.expectedError != nil {
				if err.Error() != tt.expectedError.Error() {
					t.Errorf("Lookup() error message mismatch:\n  got:  '%v'\n  want: '%v'", err, tt.expectedError)
				}
				if !errors.Is(err, repoErr) {
					t.Errorf("Lookup() returned error does not wrap original repository error '%v'", repoErr)
				}
			}
		})
	}
}

func TestSubscriptionService_Create_Success(t *testing.T) {
	ctx := context.Background()
	defaultReq := &model.SubscriptionRequest{
		Subscription: model.Subscription{
			Subscriber: model.Subscriber{
				SubscriberID: "test-sub-id",
				Domain:       "test.com",
				Type:         model.RoleBAP,
			},
			Status: model.SubscriptionStatusInitiated,
		},
		MessageID: "test-msg-id",
	}
	reqBytes, _ := json.Marshal(defaultReq)
	defaultLROWithReqJSON := &model.LRO{OperationID: "test-msg-id", Status: model.LROStatusPending, Type: model.OperationTypeCreateSubscription, RequestJSON: reqBytes}

	req := defaultReq
	mockLRO := &mockLROCreator{lro: defaultLROWithReqJSON}
	wantLRO := defaultLROWithReqJSON

	service, _ := NewSubscriptionService(mockLRO, &mockSubscriptionRepository{}, &mock.EventPublisher{})
	gotLRO, err := service.Create(ctx, req)

	if err != nil {
		t.Fatalf("Create() error = %v, wantErr false", err)
	}
	if diff := cmp.Diff(wantLRO, gotLRO); diff != "" {
		t.Errorf("Create() LRO mismatch (-want +got):\n%s", diff)
	}
}

func TestSubscriptionService_Create_Error(t *testing.T) {
	ctx := context.Background()
	defaultReq := &model.SubscriptionRequest{
		Subscription: model.Subscription{
			Subscriber: model.Subscriber{
				SubscriberID: "test-sub-id",
				Domain:       "test.com",
				Type:         model.RoleBAP,
			},
			Status: model.SubscriptionStatusInitiated,
		},
		MessageID: "test-msg-id",
	}

	tests := []struct {
		name       string
		req        *model.SubscriptionRequest
		mockLRO    *mockLROCreator
		wantErrMsg string
	}{
		{
			name:       "lroCreator returns error",
			req:        defaultReq,
			mockLRO:    &mockLROCreator{err: errors.New("lro create failed")},
			wantErrMsg: fmt.Sprintf("failed to initiate LRO type %s: lro create failed", model.OperationTypeCreateSubscription),
		},
		{
			name:       "nil request passed to Create",
			mockLRO:    &mockLROCreator{},
			wantErrMsg: "subscription request cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _ := NewSubscriptionService(tt.mockLRO, &mockSubscriptionRepository{}, &mock.EventPublisher{})
			_, err := service.Create(ctx, tt.req)

			if err == nil {
				t.Fatalf("Create() error = nil, want error containing %q", tt.wantErrMsg)
			}
			if err.Error() != tt.wantErrMsg {
				t.Errorf("Create() error = %q, wantErrMsg %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

func TestSubscriptionService_Update_Success(t *testing.T) {
	ctx := context.Background()
	defaultReq := &model.SubscriptionRequest{
		Subscription: model.Subscription{Subscriber: model.Subscriber{SubscriberID: "update-sub-id"}, Status: model.SubscriptionStatusSubscribed},
		MessageID:    "update-msg-id",
	}
	reqBytes, _ := json.Marshal(defaultReq)
	defaultLROWithReqJSON := &model.LRO{OperationID: "update-msg-id", Status: model.LROStatusPending, Type: model.OperationTypeUpdateSubscription, RequestJSON: reqBytes}

	req := defaultReq
	mockLRO := &mockLROCreator{lro: defaultLROWithReqJSON}
	wantLRO := defaultLROWithReqJSON

	service, _ := NewSubscriptionService(mockLRO, &mockSubscriptionRepository{}, &mock.EventPublisher{})
	gotLRO, err := service.Update(ctx, req)

	if err != nil {
		t.Fatalf("Update() error = %v, wantErr false", err)
	}
	if diff := cmp.Diff(wantLRO, gotLRO); diff != "" {
		t.Errorf("Update() LRO mismatch (-want +got):\n%s", diff)
	}
}

func TestSubscriptionService_Update_Error(t *testing.T) {
	ctx := context.Background()
	defaultReq := &model.SubscriptionRequest{
		Subscription: model.Subscription{Subscriber: model.Subscriber{SubscriberID: "update-sub-id"}, Status: model.SubscriptionStatusSubscribed},
		MessageID:    "update-msg-id",
	}

	tests := []struct {
		name       string
		req        *model.SubscriptionRequest
		mockLRO    *mockLROCreator
		wantErrMsg string
	}{
		{
			name:       "lroCreator returns error on update",
			req:        defaultReq,
			mockLRO:    &mockLROCreator{err: errors.New("lro update failed")},
			wantErrMsg: fmt.Sprintf("failed to initiate LRO type %s: lro update failed", model.OperationTypeUpdateSubscription),
		},
		{
			name:       "nil request passed to Update",
			mockLRO:    &mockLROCreator{},
			wantErrMsg: "subscription request cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _ := NewSubscriptionService(tt.mockLRO, &mockSubscriptionRepository{}, &mock.EventPublisher{})
			_, err := service.Update(ctx, tt.req)

			if err == nil {
				t.Fatalf("Update() error = nil, want error containing %q", tt.wantErrMsg)
			}
			if err.Error() != tt.wantErrMsg {
				t.Errorf("Update() error = %q, wantErrMsg %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

func TestSubscriptionService_GetSigningPublicKey_Success(t *testing.T) {
	ctx := context.Background()
	wantKey := "test-public-key"
	mockRepo := &mockSubscriptionRepository{key: wantKey}

	service, _ := NewSubscriptionService(&mockLROCreator{}, mockRepo, &mock.EventPublisher{})
	gotKey, err := service.GetSigningPublicKey(ctx, "sub1", "domain1", model.RoleBAP, "key1")

	if err != nil {
		t.Fatalf("GetSigningPublicKey() error = %v, wantErr false", err)
	}
	if gotKey != wantKey {
		t.Errorf("GetSigningPublicKey() gotKey = %q, wantKey %q", gotKey, wantKey)
	}
}

func TestSubscriptionService_GetSigningPublicKey_Error(t *testing.T) {
	ctx := context.Background()
	wantErrMsg := "db error"
	mockRepo := &mockSubscriptionRepository{err: errors.New("db error")}

	t.Run("repository returns error", func(t *testing.T) {
		service, _ := NewSubscriptionService(&mockLROCreator{}, mockRepo, &mock.EventPublisher{})
		_, err := service.GetSigningPublicKey(ctx, "sub1", "domain1", model.RoleBAP, "key1")

		if err == nil {
			t.Fatalf("GetSigningPublicKey() error = nil, want error %q", wantErrMsg)
		}
		if err.Error() != wantErrMsg {
			t.Errorf("GetSigningPublicKey() error = %q, wantErrMsg %q", err.Error(), wantErrMsg)
		}
	})
}
