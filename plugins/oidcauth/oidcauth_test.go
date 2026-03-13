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

package oidcauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/api/idtoken"
)

func TestNew_MissingAudience(t *testing.T) {
	_, err := New(context.Background(), map[string]string{})
	if err == nil || err.Error() != "audience is required" {
		t.Fatalf("expected 'audience is required' error, got %v", err)
	}
}

func TestMiddleware(t *testing.T) {
	// Mock idtoken.Validate
	originalValidate := idtokenValidate
	defer func() { idtokenValidate = originalValidate }()

	tests := []struct {
		name              string
		authHeader        string
		idToken           string
		audience          string
		validationErr     error
		wantStatus        int
		wantHandlerCalled bool
		wantPayloadSub    string
	}{
		{
			name:              "missing header",
			audience:          "my-audience",
			wantStatus:        http.StatusUnauthorized,
			wantHandlerCalled: false,
		},
		{
			name:              "invalid header format",
			authHeader:        "InvalidFormat token123",
			audience:          "my-audience",
			wantStatus:        http.StatusUnauthorized,
			wantHandlerCalled: false,
		},
		{
			name:              "valid token",
			authHeader:        "Bearer valid-token",
			idToken:           "valid-token",
			audience:          "my-audience",
			wantStatus:        http.StatusOK,
			wantHandlerCalled: true,
			wantPayloadSub:    "user123",
		},
		{
			name:              "validation error",
			authHeader:        "Bearer invalid-token",
			idToken:           "invalid-token",
			audience:          "my-audience",
			validationErr:     errors.New("validation failed"),
			wantStatus:        http.StatusUnauthorized,
			wantHandlerCalled: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			idtokenValidate = func(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error) {
				if tc.validationErr != nil {
					return nil, tc.validationErr
				}
				if idToken == tc.idToken && audience == tc.audience {
					return &idtoken.Payload{
						Claims: map[string]any{"sub": tc.wantPayloadSub},
					}, nil
				}
				return nil, errors.New("validation failed in mock")
			}

			middleware, err := New(context.Background(), map[string]string{"audience": tc.audience})
			if err != nil {
				t.Fatalf("failed to create middleware: %v", err)
			}

			handlerCalled := false
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				payload, ok := FromContext(r.Context())
				if !ok {
					t.Fatal("expected payload in context")
				}
				if tc.wantPayloadSub != "" && payload.Claims["sub"] != tc.wantPayloadSub {
					t.Errorf("expected sub=%s, got %v", tc.wantPayloadSub, payload.Claims["sub"])
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "http://example.com/foo", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if handlerCalled != tc.wantHandlerCalled {
				t.Errorf("got handlerCalled=%t, want %t", handlerCalled, tc.wantHandlerCalled)
			}
			if w.Result().StatusCode != tc.wantStatus {
				t.Errorf("got status %d, want %d", w.Result().StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestValidateConfig_Nil(t *testing.T) {
	err := validateConfig(nil)
	if err == nil || err.Error() != "config cannot be nil" {
		t.Fatalf("expected 'config cannot be nil' error, got %v", err)
	}
}

func TestMiddleware_AuthHeaderCases(t *testing.T) {
	// Mock idtoken.Validate to return success
	originalValidate := idtokenValidate
	defer func() { idtokenValidate = originalValidate }()

	idtokenValidate = func(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error) {
		return &idtoken.Payload{}, nil
	}

	middleware, err := New(context.Background(), map[string]string{"audience": "my-audience"})
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	tests := []struct {
		name              string
		authHeader        string
		wantStatus        int
		wantHandlerCalled bool
	}{
		{
			name:              "uppercase bearer",
			authHeader:        "BEARER token123",
			wantStatus:        http.StatusOK,
			wantHandlerCalled: true,
		},
		{
			name:              "basic auth",
			authHeader:        "Basic token123",
			wantStatus:        http.StatusUnauthorized,
			wantHandlerCalled: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handlerCalled := false
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				if _, ok := FromContext(r.Context()); !ok {
					t.Error("expected payload in context")
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "http://example.com/foo", nil)
			req.Header.Set("Authorization", tc.authHeader)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Result().StatusCode != tc.wantStatus {
				t.Errorf("got status %d, want %d", w.Result().StatusCode, tc.wantStatus)
			}
			if handlerCalled != tc.wantHandlerCalled {
				t.Errorf("got handlerCalled=%t, want %t", handlerCalled, tc.wantHandlerCalled)
			}
		})
	}
}
