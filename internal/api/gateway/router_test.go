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

package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// mockGatewayHandler is a mock implementation of the gatewayHandler interface.
type mockGatewayHandler struct {
	serveHttpCalled bool
}

func (m *mockGatewayHandler) ServeHttp(w http.ResponseWriter, r *http.Request) {
	m.serveHttpCalled = true
	w.WriteHeader(http.StatusOK)
}

func TestNewRouter(t *testing.T) {
	gh := &mockGatewayHandler{}
	router := NewRouter(gh)

	if router == nil {
		t.Fatal("NewRouter() returned nil, expected a chi.Mux router")
	}
}

func TestRouter_Middleware_Recoverer(t *testing.T) {
	gh := &mockGatewayHandler{}
	router := NewRouter(gh)

	// Add a temporary route that panics
	router.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rr := httptest.NewRecorder()

	// The Recoverer middleware should catch the panic and return a 500
	// without crashing the server.
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}
}

func TestRouter_Routes(t *testing.T) {
	gh := &mockGatewayHandler{}
	router := NewRouter(gh)

	tests := []struct {
		name            string
		method          string
		path            string
		expectedStatus  int
		expectedBody    string
		expectedHeaders http.Header
		handlerCheck    func(t *testing.T, h *mockGatewayHandler)
	}{
		{
			name:            "HealthCheck",
			method:          http.MethodGet,
			path:            "/health",
			expectedStatus:  http.StatusOK,
			expectedBody:    `{"status":"ok"}`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
			handlerCheck: func(t *testing.T, h *mockGatewayHandler) {
				if h.serveHttpCalled {
					t.Error("ServeHttp was called for /health, but should not have been")
				}
			},
		},
		{
			name:           "Search",
			method:         http.MethodPost,
			path:           "/search",
			expectedStatus: http.StatusOK,
			handlerCheck: func(t *testing.T, h *mockGatewayHandler) {
				if !h.serveHttpCalled {
					t.Error("ServeHttp was not called for /search")
				}
			},
		},
		{
			name:           "OnSearch",
			method:         http.MethodPost,
			path:           "/on_search",
			expectedStatus: http.StatusOK,
			handlerCheck: func(t *testing.T, h *mockGatewayHandler) {
				if !h.serveHttpCalled {
					t.Error("ServeHttp was not called for /on_search")
				}
			},
		},
		{
			name:           "NotFound",
			method:         http.MethodGet,
			path:           "/not_a_real_path",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "404 page not found\n",
			handlerCheck: func(t *testing.T, h *mockGatewayHandler) {
				if h.serveHttpCalled {
					t.Error("ServeHttp was called for a non-existent path, but should not have been")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset mock handler state for each test
			gh.serveHttpCalled = false

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
			tc.handlerCheck(t, gh)
		})
	}
}
