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

package subscriber

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// mockSubscriberHandler is a mock implementation of the subscriberHandler interface.
type mockSubscriberHandler struct {
	createSubscriptionCalled bool
	updateSubscriptionCalled bool
	statusUpdateCalled       bool
	onSubscribeCalled        bool
}

func (m *mockSubscriberHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	m.createSubscriptionCalled = true
	w.WriteHeader(http.StatusOK)
}

func (m *mockSubscriberHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	m.updateSubscriptionCalled = true
	w.WriteHeader(http.StatusOK)
}

func (m *mockSubscriberHandler) StatusUpdate(w http.ResponseWriter, r *http.Request) {
	m.statusUpdateCalled = true
	w.WriteHeader(http.StatusOK)
}

func (m *mockSubscriberHandler) OnSubscribe(w http.ResponseWriter, r *http.Request) {
	m.onSubscribeCalled = true
	w.WriteHeader(http.StatusOK)
}

func TestRouter_Routes(t *testing.T) {
	h := &mockSubscriberHandler{}
	router := NewRouter(h)

	tests := []struct {
		name            string
		method          string
		path            string
		expectedStatus  int
		expectedBody    string
		expectedHeaders http.Header
		handlerCheck    func(t *testing.T, h *mockSubscriberHandler)
	}{
		{
			name:            "HealthCheck",
			method:          http.MethodGet,
			path:            "/health",
			expectedStatus:  http.StatusOK,
			expectedBody:    `{"status":"ok","service":"subscriber"}`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
			handlerCheck:    func(t *testing.T, h *mockSubscriberHandler) { /* No specific handler mock to check */ },
		},
		{
			name:           "CreateSubscription",
			method:         http.MethodPost,
			path:           "/subscribe",
			expectedStatus: http.StatusOK,
			handlerCheck: func(t *testing.T, h *mockSubscriberHandler) {
				if !h.createSubscriptionCalled {
					t.Error("CreateSubscription was not called")
				}
			},
		},
		{
			name:           "UpdateSubscription",
			method:         http.MethodPatch,
			path:           "/subscribe",
			expectedStatus: http.StatusOK,
			handlerCheck: func(t *testing.T, h *mockSubscriberHandler) {
				if !h.updateSubscriptionCalled {
					t.Error("UpdateSubscription was not called")
				}
			},
		},
		{
			name:           "StatusUpdate",
			method:         http.MethodPost,
			path:           "/updateStatus",
			expectedStatus: http.StatusOK,
			handlerCheck: func(t *testing.T, h *mockSubscriberHandler) {
				if !h.statusUpdateCalled {
					t.Error("StatusUpdate was not called")
				}
			},
		},
		{
			name:           "OnSubscribe at root",
			method:         http.MethodPost,
			path:           "/on_subscribe",
			expectedStatus: http.StatusOK,
			handlerCheck: func(t *testing.T, h *mockSubscriberHandler) {
				if !h.onSubscribeCalled {
					t.Error("OnSubscribe was not called")
				}
			},
		},
		{
			name:           "OnSubscribe with prefix",
			method:         http.MethodPost,
			path:           "/v1/on_subscribe",
			expectedStatus: http.StatusOK,
			handlerCheck: func(t *testing.T, h *mockSubscriberHandler) {
				if !h.onSubscribeCalled {
					t.Error("OnSubscribe was not called")
				}
			},
		},
		{
			name:           "NotFound for wildcard POST",
			method:         http.MethodPost,
			path:           "/some/other/path",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "404 page not found\n",
			handlerCheck: func(t *testing.T, h *mockSubscriberHandler) {
				if h.onSubscribeCalled {
					t.Error("OnSubscribe was called but should not have been")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset mock handler state for each test
			h.createSubscriptionCalled = false
			h.updateSubscriptionCalled = false
			h.statusUpdateCalled = false
			h.onSubscribeCalled = false

			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectedStatus)
			}

			if tc.expectedBody != "" {
				if diff := cmp.Diff(tc.expectedBody, rr.Body.String()); diff != "" {
					t.Errorf("handler returned unexpected body (-want +got):\n%s", diff)
				}
			}

			if tc.expectedHeaders != nil {
				for key, wantValues := range tc.expectedHeaders {
					if diff := cmp.Diff(wantValues, rr.Header().Values(key)); diff != "" {
						t.Errorf("handler returned unexpected header %q (-want +got):\n%s", key, diff)
					}
				}
			}
			tc.handlerCheck(t, h)
		})
	}
}
