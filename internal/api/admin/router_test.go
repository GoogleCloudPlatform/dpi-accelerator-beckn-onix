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

package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type mockAdminHandler struct {
	handleSubscriptionActionCalled bool
}

func (m *mockAdminHandler) HandleSubscriptionAction(w http.ResponseWriter, r *http.Request) {
	m.handleSubscriptionActionCalled = true
	w.WriteHeader(http.StatusOK)
}

func TestRouter_Routes(t *testing.T) {
	h := &mockAdminHandler{}

	router := NewRouter(h)

	tests := []struct {
		name            string
		method          string
		path            string
		expectedStatus  int
		expectedBody    string
		expectedHeaders http.Header
		handlerCheck    func(t *testing.T)
	}{
		{
			name:            "HealthCheck",
			method:          http.MethodGet,
			path:            "/health",
			expectedStatus:  http.StatusOK,
			expectedBody:    `{"status":"ok"}`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
			handlerCheck:    func(t *testing.T) { /* No specific handler mock to check */ },
		},
		{
			name:           "SubscriptionAction",
			method:         http.MethodPost,
			path:           "/operations/action",
			expectedStatus: http.StatusOK,
			handlerCheck: func(t *testing.T) {
				if !h.handleSubscriptionActionCalled {
					t.Error("SubscriptionActionHandler.SubscriptionAction was not called")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

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
			tc.handlerCheck(t)
		})
	}
}
