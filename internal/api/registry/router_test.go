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

package registry

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
)

// mockSubscriptionHandler is a mock implementation of the subscriptionHandler interface.
type mockSubscriptionHandler struct {
	createCalled bool
	updateCalled bool
}

func (m *mockSubscriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	m.createCalled = true
	w.WriteHeader(http.StatusOK)
}

func (m *mockSubscriptionHandler) Update(w http.ResponseWriter, r *http.Request) {
	m.updateCalled = true
	w.WriteHeader(http.StatusOK)
}

// mockLookupHandler is a mock implementation of the lookupHandler interface.
type mockLookupHandler struct {
	lookupCalled bool
}

func (m *mockLookupHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	m.lookupCalled = true
	w.WriteHeader(http.StatusOK)
}

// mockLROHandler is a mock implementation of the lroHandler interface.
type mockLROHandler struct {
	getCalled   bool
	operationID string
}

func (m *mockLROHandler) Get(w http.ResponseWriter, r *http.Request) {
	m.getCalled = true
	m.operationID = chi.URLParam(r, "operation_id")
	w.WriteHeader(http.StatusOK)
}

func TestNewRouter_Initialization(t *testing.T) {
	sh := &mockSubscriptionHandler{}
	lh := &mockLookupHandler{}
	lroh := &mockLROHandler{}

	router := NewRouter(sh, lh, lroh)

	if router == nil {
		t.Fatal("New() returned nil, expected a chi.Mux router")
	}
}

func TestRouter_Middleware_Recoverer(t *testing.T) {
	sh := &mockSubscriptionHandler{}
	lh := &mockLookupHandler{}
	lroh := &mockLROHandler{}
	router := NewRouter(sh, lh, lroh)

	// Add a temporary route that panics
	router.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rr := httptest.NewRecorder()

	// The Recoverer middleware should catch the panic and return a 500
	// without crashing the server.
	// We don't need a defer panic check here because the middleware handles it.
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}
}

func TestRouter_Routes(t *testing.T) {
	sh := &mockSubscriptionHandler{}
	lh := &mockLookupHandler{}
	lroh := &mockLROHandler{}

	router := NewRouter(sh, lh, lroh)

	tests := []struct {
		name            string
		method          string
		path            string
		expectedStatus  int
		expectedBody    string // Optional, for routes with fixed bodies like /health
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
			name:           "SubscribeCreate",
			method:         http.MethodPost,
			path:           "/subscribe",
			expectedStatus: http.StatusOK,
			handlerCheck: func(t *testing.T) {
				if !sh.createCalled {
					t.Error("subscriptionHandler.Create was not called")
				}
			},
		},
		{
			name:           "SubscribeUpdate",
			method:         http.MethodPatch,
			path:           "/subscribe",
			expectedStatus: http.StatusOK,
			handlerCheck: func(t *testing.T) {
				if !sh.updateCalled {
					t.Error("subscriptionHandler.Update was not called")
				}
			},
		},
		{
			name:           "Lookup",
			method:         http.MethodPost,
			path:           "/lookup",
			expectedStatus: http.StatusOK,
			handlerCheck: func(t *testing.T) {
				if !lh.lookupCalled {
					t.Error("lookupHandler.Lookup was not called")
				}
			},
		},
		{
			name:           "GetLRO",
			method:         http.MethodGet,
			path:           "/operations/op123",
			expectedStatus: http.StatusOK,
			handlerCheck: func(t *testing.T) {
				if !lroh.getCalled {
					t.Error("lroHandler.Get was not called")
				}
				if lroh.operationID != "op123" {
					t.Errorf("lroHandler.Get received wrong operation_id: got %q, want %q", lroh.operationID, "op123")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset mock states for each test
			sh.createCalled, sh.updateCalled = false, false
			lh.lookupCalled = false
			lroh.getCalled, lroh.operationID = false, ""

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
