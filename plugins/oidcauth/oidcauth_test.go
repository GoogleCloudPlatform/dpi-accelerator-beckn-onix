// Copyright 2026 Google LLC
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
	"strings"
	"testing"

	"google.golang.org/api/idtoken"
)

func TestNew_MissingAudience(t *testing.T) {
	_, err := New(context.Background(), &Config{AllowedIssuers: []string{"test-issuer"}})
	if err == nil || err.Error() != "allowed audience is required" {
		t.Fatalf("expected 'allowed audience is required' error, got %v", err)
	}
}

func TestNew_MissingIssuers(t *testing.T) {
	_, err := New(context.Background(), &Config{AllowedAudience: "test-audience"})
	if err == nil || err.Error() != "allowed issuers are required" {
		t.Fatalf("expected 'allowed issuers are required' error, got %v", err)
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
		wantAllowedIssuer string
		wantAllowedSA     string
		tokenIssuer       string
		expectedBody      string
	}{
		{
			name:              "missing header",
			audience:          "my-audience",
			wantStatus:        http.StatusUnauthorized,
			wantHandlerCalled: false,
			expectedBody:      "Authorization header is required\n",
		},
		{
			name:              "invalid header format",
			authHeader:        "InvalidFormat token123",
			audience:          "my-audience",
			wantStatus:        http.StatusUnauthorized,
			wantHandlerCalled: false,
			expectedBody:      "Invalid Authorization header format\n",
		},
		{
			name:              "valid token (no sa or issuer restrictions)",
			authHeader:        "Bearer valid-token",
			idToken:           "valid-token",
			audience:          "my-audience",
			tokenIssuer:       "some-other-issuer",
			wantStatus:        http.StatusOK,
			wantHandlerCalled: true,
			wantPayloadSub:    "user123",
		},
		{
			name:              "valid google token matching sa",
			authHeader:        "Bearer valid-token-google",
			idToken:           "valid-token-google",
			audience:          "my-audience",
			tokenIssuer:       "https://accounts.google.com",
			wantStatus:        http.StatusOK,
			wantHandlerCalled: true,
			wantPayloadSub:    "user123",
			wantAllowedSA:     "test@abc.com",
		},
		{
			name:              "invalid google token mismatching sa",
			authHeader:        "Bearer invalid-token-google",
			idToken:           "invalid-token-google",
			audience:          "my-audience",
			tokenIssuer:       "accounts.google.com",
			wantStatus:        http.StatusUnauthorized,
			wantHandlerCalled: false,
			wantPayloadSub:    "user123",
			wantAllowedSA:     "invalid@abc.com",
			expectedBody:      "Unauthorized: invalid sa\n",
		},
		{
			name:              "valid custom token matching issuer",
			authHeader:        "Bearer valid-token-custom",
			idToken:           "valid-token-custom",
			audience:          "my-audience",
			tokenIssuer:       "my-custom-issuer",
			wantStatus:        http.StatusOK,
			wantHandlerCalled: true,
			wantPayloadSub:    "user123",
			wantAllowedIssuer: "my-custom-issuer",
		},
		{
			name:              "invalid custom token mismatching issuer",
			authHeader:        "Bearer invalid-token-custom",
			idToken:           "invalid-token-custom",
			audience:          "my-audience",
			tokenIssuer:       "wrong-custom-issuer",
			wantStatus:        http.StatusUnauthorized,
			wantHandlerCalled: false,
			wantPayloadSub:    "user123",
			wantAllowedIssuer: "my-custom-issuer",
			expectedBody:      "Unauthorized: invalid issuer\n",
		},
		{
			name:              "validation error",
			authHeader:        "Bearer invalid-token",
			idToken:           "invalid-token",
			audience:          "my-audience",
			tokenIssuer:       "any-issuer",
			validationErr:     errors.New("validation failed"),
			wantStatus:        http.StatusUnauthorized,
			wantHandlerCalled: false,
			expectedBody:      "Invalid token\n",
		},
		{
			name:              "google token missing email claim",
			authHeader:        "Bearer valid-token-google-no-email",
			idToken:           "valid-token-google-no-email",
			audience:          "my-audience",
			tokenIssuer:       "https://accounts.google.com",
			wantStatus:        http.StatusUnauthorized,
			wantHandlerCalled: false,
			wantAllowedSA:     "test@abc.com",
			expectedBody:      "Unauthorized: invalid sa\n",
		},
		{
			name:              "google token empty email claim",
			authHeader:        "Bearer valid-token-google-empty-email",
			idToken:           "valid-token-google-empty-email",
			audience:          "my-audience",
			tokenIssuer:       "https://accounts.google.com",
			wantStatus:        http.StatusUnauthorized,
			wantHandlerCalled: false,
			wantAllowedSA:     "test@abc.com",
			expectedBody:      "Unauthorized: invalid sa\n",
		},
		{
			name:              "google token unauthorized (empty allowedSAs)",
			authHeader:        "Bearer valid-token-google-no-config-sas",
			idToken:           "valid-token-google-no-config-sas",
			audience:          "my-audience",
			tokenIssuer:       "https://accounts.google.com",
			wantStatus:        http.StatusUnauthorized,
			wantHandlerCalled: false,
			wantAllowedSA:     "", // simulate no allowed SAs in config
			expectedBody:      "Unauthorized: invalid sa\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			idtokenValidate = func(ctx context.Context, idToken string, audience string) (*idtoken.Payload, error) {
				if tc.validationErr != nil {
					return nil, tc.validationErr
				}
				if idToken == tc.idToken && audience == tc.audience {
					payload := &idtoken.Payload{
						Issuer: tc.tokenIssuer,
						Claims: map[string]any{"sub": tc.wantPayloadSub},
					}

					if tc.wantAllowedSA != "" {
						if strings.Contains(tc.idToken, "no-email") {
							// don't set email
						} else if strings.Contains(tc.idToken, "empty-email") {
							payload.Claims["email"] = ""
						} else if tc.wantStatus == http.StatusUnauthorized && strings.Contains(tc.wantAllowedSA, "invalid") {
							payload.Claims["email"] = "wrong@abc.com"
						} else {
							payload.Claims["email"] = tc.wantAllowedSA
						}
					}

					return payload, nil
				}
				return nil, errors.New("validation failed in mock")
			}

			cfg := &Config{AllowedAudience: tc.audience}
			if tc.wantAllowedIssuer != "" {
				cfg.AllowedIssuers = []string{tc.wantAllowedIssuer} // Make config match what the test expects for valid custom issuers
			} else {
				cfg.AllowedIssuers = []string{"some-other-issuer"} // Fallback mandatory config setup
			}
			if tc.wantAllowedSA != "" {
				cfg.AllowedSAs = []string{"test@abc.com"} // All tests assume this is the configured authorized service account
			} else if tc.name == "google token unauthorized (empty allowedSAs)" {
				cfg.AllowedSAs = []string{}
			}
			middleware, err := New(context.Background(), cfg)
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
			if tc.expectedBody != "" && w.Body.String() != tc.expectedBody {
				t.Errorf("got body %q, want %q", w.Body.String(), tc.expectedBody)
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
		return &idtoken.Payload{Issuer: "test"}, nil
	}

	middleware, err := New(context.Background(), &Config{AllowedAudience: "my-audience", AllowedIssuers: []string{"test"}})
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

func TestFromContext(t *testing.T) {
	ctx := context.Background()
	if _, ok := FromContext(ctx); ok {
		t.Error("expected ok=false for empty context")
	}

	payload := &idtoken.Payload{Issuer: "test"}
	ctx = context.WithValue(ctx, oidcPayloadKey, payload)
	if p, ok := FromContext(ctx); !ok || p != payload {
		t.Errorf("expected payload %v, got %v (ok=%t)", payload, p, ok)
	}
}

func TestHelpers(t *testing.T) {
	t.Run("isGoogleIssuer", func(t *testing.T) {
		expect := map[string]bool{
			"https://accounts.google.com": true,
			"accounts.google.com":         true,
			"google.com":                  false,
			"other":                       false,
		}
		for iss, want := range expect {
			if got := isGoogleIssuer(iss); got != want {
				t.Errorf("isGoogleIssuer(%q) = %t, want %t", iss, got, want)
			}
		}
	})

	t.Run("isSAAuthorized", func(t *testing.T) {
		tests := []struct {
			claims  map[string]any
			allowed []string
			want    bool
		}{
			{map[string]any{"email": "a@b.com"}, []string{"a@b.com"}, true},
			{map[string]any{"email": "A@B.COM"}, []string{"a@b.com"}, true}, // case insensitive
			{map[string]any{"email": "a@b.com"}, []string{"b@c.com", "a@b.com"}, true},
			{map[string]any{"email": "x@y.com"}, []string{"a@b.com"}, false},
			{map[string]any{"email": 123}, []string{"a@b.com"}, false}, // wrong type
			{map[string]any{}, []string{"a@b.com"}, false},             // missing email
			{map[string]any{"email": ""}, []string{"a@b.com"}, false},  // empty email
			{map[string]any{"email": "a@b.com"}, []string{}, false},    // empty allowed
			{map[string]any{"email": "a@b.com"}, nil, false},           // nil allowed
		}
		for _, tc := range tests {
			if got := isSAAuthorized(context.Background(), tc.claims, tc.allowed); got != tc.want {
				t.Errorf("isSAAuthorized(%v, %v) = %t, want %t", tc.claims, tc.allowed, got, tc.want)
			}
		}
	})

	t.Run("isIssuerAuthorized", func(t *testing.T) {
		tests := []struct {
			issuer  string
			allowed []string
			want    bool
		}{
			{"iss1", []string{"iss1"}, true},
			{"iss1", []string{" iss1 "}, true}, // trims spaces
			{"iss1", []string{"iss2", "iss1"}, true},
			{"iss1", []string{"iss2"}, false},
			{"iss1", []string{}, false},
			{"iss1", nil, false},
		}
		for _, tc := range tests {
			if got := isIssuerAuthorized(context.Background(), tc.issuer, tc.allowed); got != tc.want {
				t.Errorf("isIssuerAuthorized(%q, %v) = %t, want %t", tc.issuer, tc.allowed, got, tc.want)
			}
		}
	})
}
