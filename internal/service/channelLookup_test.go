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
	"sync"
	"testing"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// mockLookupClient is a mock for the lookupClient interface.
type mockLookupClient struct {
	subscriptions []model.Subscription
	err           error
}

func (m *mockLookupClient) Lookup(ctx context.Context, request *model.Subscription) ([]model.Subscription, error) {
	return m.subscriptions, m.err
}

// mockTaskQueuer is a mock for the taskQueuer interface.
type mockTaskQueuer struct {
	err          error
	callCount    int
	QueueTxnFunc func(ctx context.Context, reqCtx *model.Context, msg []byte, h http.Header) (*model.AsyncTask, error)
}

func (m *mockTaskQueuer) QueueTxn(ctx context.Context, reqCtx *model.Context, msg []byte, h http.Header) (*model.AsyncTask, error) {
	if m.QueueTxnFunc != nil {
		return m.QueueTxnFunc(ctx, reqCtx, msg, h)
	}
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	// Return a dummy task, as the processor doesn't use the return value.
	return &model.AsyncTask{}, nil
}

func TestNewChannelLookupProcessor(t *testing.T) {
	tests := []struct {
		name           string
		registryClient lookupClient
		authGen        authGen
		taskQueuer     taskQueuer
		subID          string
		maxProxyTasks  int
		wantErr        string
	}{
		{
			name:           "success",
			registryClient: &mockLookupClient{},
			authGen:        &mockAuthGen{},
			taskQueuer:     &mockTaskQueuer{},
			subID:          "test-id",
			maxProxyTasks:  10,
			wantErr:        "",
		},
		{
			name:           "nil registryClient",
			registryClient: nil,
			authGen:        &mockAuthGen{},
			taskQueuer:     &mockTaskQueuer{},
			subID:          "test-id",
			maxProxyTasks:  10,
			wantErr:        "registryClient cannot be nil",
		},
		{
			name:           "nil authGen",
			registryClient: &mockLookupClient{},
			authGen:        nil,
			taskQueuer:     &mockTaskQueuer{},
			subID:          "test-id",
			maxProxyTasks:  10,
			wantErr:        "authGen cannot be nil",
		},
		{
			name:           "nil taskQueuer",
			registryClient: &mockLookupClient{},
			authGen:        &mockAuthGen{},
			taskQueuer:     nil,
			subID:          "test-id",
			maxProxyTasks:  10,
			wantErr:        "taskQueuer cannot be nil",
		},
		{
			name:           "empty subID",
			registryClient: &mockLookupClient{},
			authGen:        &mockAuthGen{},
			taskQueuer:     &mockTaskQueuer{},
			subID:          "",
			maxProxyTasks:  10,
			wantErr:        "subID cannot be empty",
		},
		{
			name:           "zero maxProxyTasks (valid)",
			registryClient: &mockLookupClient{},
			authGen:        &mockAuthGen{},
			taskQueuer:     &mockTaskQueuer{},
			subID:          "test-id",
			maxProxyTasks:  0,
			wantErr:        "",
		},
		{
			name:           "negative maxProxyTasks (valid, defaults to 0)",
			registryClient: &mockLookupClient{},
			authGen:        &mockAuthGen{},
			taskQueuer:     &mockTaskQueuer{},
			subID:          "test-id",
			maxProxyTasks:  -5,
			wantErr:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewChannelLookupProcessor(tt.registryClient, tt.authGen, tt.taskQueuer, tt.subID, tt.maxProxyTasks)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("NewChannelLookupProcessor() error = %v, want %q", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("NewChannelLookupProcessor() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestChannelLookupProcessor_Process(t *testing.T) {
	ctx := context.Background()
	validTask := &model.AsyncTask{
		Type: model.AsyncTaskTypeLookup,
		Body: []byte(`{"context":{"domain":"test-domain"}}`),
		Context: model.Context{
			Domain: "test-domain",
			Action: "search",
		},
		Headers: http.Header{"X-Test": []string{"true"}},
	}
	validSubs := []model.Subscription{
		{Subscriber: model.Subscriber{SubscriberID: "sub1", URL: "http://sub1.com"}},
		{Subscriber: model.Subscriber{SubscriberID: "sub2", URL: "http://sub2.com"}},
	}

	tests := []struct {
		name           string
		task           *model.AsyncTask
		maxProxyTasks  int
		setupMocks     func(mockLookup *mockLookupClient, mockAuth *mockAuthGen, mockQueuer *mockTaskQueuer)
		wantErrMsg     string
		wantQueueCalls int
	}{
		{
			name:          "success - multiple subscribers found and queued",
			task:          validTask,
			maxProxyTasks: 10,
			setupMocks: func(mockLookup *mockLookupClient, mockAuth *mockAuthGen, mockQueuer *mockTaskQueuer) {
				mockLookup.subscriptions = validSubs
				mockAuth.authHeader = "test-auth-header"
			},
			wantErrMsg:     "",
			wantQueueCalls: 2,
		},
		{
			name:          "success - no subscribers found",
			task:          validTask,
			maxProxyTasks: 10,
			setupMocks: func(mockLookup *mockLookupClient, mockAuth *mockAuthGen, mockQueuer *mockTaskQueuer) {
				mockLookup.subscriptions = []model.Subscription{}
			},
			wantErrMsg:     "",
			wantQueueCalls: 0,
		},
		{
			name:          "success - maxProxyTasks limit is respected",
			task:          validTask,
			maxProxyTasks: 1,
			setupMocks: func(mockLookup *mockLookupClient, mockAuth *mockAuthGen, mockQueuer *mockTaskQueuer) {
				mockLookup.subscriptions = validSubs
				mockAuth.authHeader = "test-auth-header"
			},
			wantErrMsg:     "",
			wantQueueCalls: 1,
		},
		{
			name:          "success - subscriber with empty URL is skipped",
			task:          validTask,
			maxProxyTasks: 10,
			setupMocks: func(mockLookup *mockLookupClient, mockAuth *mockAuthGen, mockQueuer *mockTaskQueuer) {
				mockLookup.subscriptions = []model.Subscription{
					{Subscriber: model.Subscriber{SubscriberID: "sub1", URL: "http://sub1.com"}},
					{Subscriber: model.Subscriber{SubscriberID: "sub2-no-url", URL: ""}},
				}
				mockAuth.authHeader = "test-auth-header"
			},
			wantErrMsg:     "",
			wantQueueCalls: 1,
		},
		{
			name:       "error - nil task",
			task:       nil,
			wantErrMsg: "async task cannot be nil",
		},
		{
			name:       "error - invalid task type",
			task:       &model.AsyncTask{Type: model.AsyncTaskTypeProxy},
			wantErrMsg: "invalid task type for LookupTaskProcessor",
		},
		{
			name:       "error - empty task body",
			task:       &model.AsyncTask{Type: model.AsyncTaskTypeLookup, Body: []byte{}},
			wantErrMsg: "task body for lookup cannot be empty",
		},
		{
			name: "error - lookup fails",
			task: validTask,
			setupMocks: func(mockLookup *mockLookupClient, mockAuth *mockAuthGen, mockQueuer *mockTaskQueuer) {
				mockLookup.err = errors.New("db connection error")
			},
			wantErrMsg: "failed to lookup subscribers",
		},
		{
			name: "error - auth header generation fails",
			task: validTask,
			setupMocks: func(mockLookup *mockLookupClient, mockAuth *mockAuthGen, mockQueuer *mockTaskQueuer) {
				mockLookup.subscriptions = validSubs
				mockAuth.err = errors.New("signing error")
			},
			wantErrMsg: "failed to prepare signed headers",
		},
		{
			name: "error - task queuing fails on all subscribers",
			task: validTask,
			setupMocks: func(mockLookup *mockLookupClient, mockAuth *mockAuthGen, mockQueuer *mockTaskQueuer) {
				mockLookup.subscriptions = validSubs // 2 subscribers
				mockAuth.authHeader = "test-auth-header"
				mockQueuer.err = errors.New("queue is full")
			},
			wantErrMsg:     "failed to queue proxy task for subscriber",
			wantQueueCalls: 2, // It will be called for both subscribers
		},
		{
			name: "error - task queuing fails on second subscriber",
			task: validTask,
			setupMocks: func(mockLookup *mockLookupClient, mockAuth *mockAuthGen, mockQueuer *mockTaskQueuer) {
				mockLookup.subscriptions = validSubs
				mockAuth.authHeader = "test-auth-header"
				var callCount int
				var mu sync.Mutex
				mockQueuer.QueueTxnFunc = func(ctx context.Context, reqCtx *model.Context, msg []byte, h http.Header) (*model.AsyncTask, error) {
					mu.Lock()
					defer mu.Unlock()
					mockQueuer.callCount++
					callCount++
					if callCount == 2 {
						return nil, errors.New("queue full on second call")
					}
					return &model.AsyncTask{}, nil
				}
			},
			wantErrMsg:     "failed to queue proxy task for subscriber", // More generic check
			wantQueueCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLookup := &mockLookupClient{}
			mockAuth := &mockAuthGen{}
			mockQueuer := &mockTaskQueuer{}

			if tt.setupMocks != nil {
				tt.setupMocks(mockLookup, mockAuth, mockQueuer)
			}

			maxTasks := tt.maxProxyTasks
			if maxTasks == 0 {
				maxTasks = 10 // Default for tests not specifying it
			}

			processor, _ := NewChannelLookupProcessor(mockLookup, mockAuth, mockQueuer, "test-id", maxTasks)
			err := processor.Process(ctx, tt.task)

			if tt.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("Process() error = %v, want error containing %q", err, tt.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Process() unexpected error = %v", err)
				}
			}

			if mockQueuer.callCount != tt.wantQueueCalls {
				t.Errorf("Process() taskQueuer was called %d times, want %d", mockQueuer.callCount, tt.wantQueueCalls)
			}
		})
	}
}
