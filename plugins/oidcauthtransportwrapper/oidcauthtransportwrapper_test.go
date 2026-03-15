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

package oidcauthtransportwrapper

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"google.golang.org/api/idtoken"
	"golang.org/x/oauth2"
)

type mockTokenSource struct {
	token string
	err   error
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &oauth2.Token{AccessToken: m.token}, nil
}

type mockRoundTripper struct {
	lastReq *http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.lastReq = req
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func TestNew(t *testing.T) {
	ctx := t.Context()
	testCases := []struct {
		name    string
		config  map[string]any
		wantErr string
	}{
		{
			name: "success with override",
			config: map[string]any{
				"audience_override": "https://custom.aud",
			},
		},
		{
			name:   "success with nil config",
			config: nil,
		},
		{
			name: "error invalid type",
			config: map[string]any{
				"audience_override": 123,
			},
			wantErr: "oidc: config 'audience_override' must be a string, but got int",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wrapper, cleanup, err := New(ctx, tc.config)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("New(%v) error got %v, want substring %q", tc.config, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("New(%v) failed unexpectedly: %v", tc.config, err)
			}

			// Call cleanup if provided to cover the returned func() {} line.
			if cleanup != nil {
				cleanup()
			}

			if tc.config != nil && tc.config["audience_override"] != nil {
				want := tc.config["audience_override"].(string)
				if got := wrapper.audienceOverride; got != want {
					t.Errorf("wrapper.audienceOverride got %q, want %q", got, want)
				}
			}
		})
	}
}

func TestOIDCTransport_RoundTrip(t *testing.T) {
	original := idtokenNewTokenSource
	defer func() { idtokenNewTokenSource = original }()

	expectedToken := "fake-google-oidc-token"
	idtokenNewTokenSource = func(ctx context.Context, audience string, opts ...idtoken.ClientOption) (oauth2.TokenSource, error) {
		return &mockTokenSource{token: expectedToken}, nil
	}

	mockBase := &mockRoundTripper{}
	transport := &oidcTransport{
		base:         mockBase,
		tokenSources: make(map[string]oauth2.TokenSource),
		ctx:          t.Context(),
	}

	req, err := http.NewRequest("GET", "https://api.example.com/search", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() failed: %v", err)
	}

	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip() failed: %v", err)
	}

	got := mockBase.lastReq.Header.Get("Authorization")
	want := "Bearer " + expectedToken
	if got != want {
		t.Errorf("RoundTrip() Authorization header got %q, want %q", got, want)
	}
}

func TestOIDCTransport_Errors(t *testing.T) {
	original := idtokenNewTokenSource
	defer func() { idtokenNewTokenSource = original }()

	testCases := []struct {
		name          string
		factoryErr    error
		tokenErr      error
		expectedError string
	}{
		{
			name:          "factory error",
			factoryErr:    fmt.Errorf("failed to create source"),
			expectedError: "oidc: failed to get token source",
		},
		{
			name:          "token fetch error",
			tokenErr:      fmt.Errorf("failed to fetch token"),
			expectedError: "oidc: failed to fetch token",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idtokenNewTokenSource = func(ctx context.Context, aud string, opts ...idtoken.ClientOption) (oauth2.TokenSource, error) {
				if tc.factoryErr != nil {
					return nil, tc.factoryErr
				}
				return &mockTokenSource{err: tc.tokenErr}, nil
			}

			transport := &oidcTransport{
				base:         &mockRoundTripper{},
				tokenSources: make(map[string]oauth2.TokenSource),
				ctx:          t.Context(),
			}

			req, _ := http.NewRequest("GET", "https://api.example.com", nil)
			_, err := transport.RoundTrip(req)
			if err == nil || !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("%s: RoundTrip() error got %v, want substring %q", tc.name, err, tc.expectedError)
			}
		})
	}
}

func TestOIDCTransport_AudienceLogic(t *testing.T) {
	original := idtokenNewTokenSource
	defer func() { idtokenNewTokenSource = original }()

	var capturedAudience string
	idtokenNewTokenSource = func(ctx context.Context, audience string, opts ...idtoken.ClientOption) (oauth2.TokenSource, error) {
		capturedAudience = audience
		return &mockTokenSource{token: "token"}, nil
	}

	testCases := []struct {
		name             string
		audienceOverride string
		requestURL       string
		expectedAudience string
	}{
		{
			name:             "use override",
			audienceOverride: "https://custom.audience",
			requestURL:       "https://api.example.com",
			expectedAudience: "https://custom.audience",
		},
		{
			name:             "use dynamic URL fallback",
			audienceOverride: "",
			requestURL:       "https://api.example.com/v1/search",
			expectedAudience: "https://api.example.com",
		},
		{
			name:             "fallback to https when scheme is missing",
			audienceOverride: "",
			requestURL:       "//api.example.com/v1/search",
			expectedAudience: "https://api.example.com",
		},
		{
			name:             "preserve existing http scheme",
			audienceOverride: "",
			requestURL:       "http://api.example.com/v1/search",
			expectedAudience: "http://api.example.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			capturedAudience = ""
			transport := &oidcTransport{
				base:             &mockRoundTripper{},
				audienceOverride: tc.audienceOverride,
				tokenSources:     make(map[string]oauth2.TokenSource),
				ctx:              t.Context(),
			}

			req, err := http.NewRequest("GET", tc.requestURL, nil)
			if err != nil {
				t.Fatalf("http.NewRequest(%q) failed: %v", tc.requestURL, err)
			}

			if _, err := transport.RoundTrip(req); err != nil {
				t.Fatalf("RoundTrip() failed: %v", err)
			}

			if got, want := capturedAudience, tc.expectedAudience; got != want {
				t.Errorf("capturedAudience got %q, want %q", got, want)
			}
		})
	}
}

func TestOIDCTransport_ConcurrencyAndCaching(t *testing.T) {
	original := idtokenNewTokenSource
	defer func() { idtokenNewTokenSource = original }()

	callCount := 0
	started := make(chan struct{})
	finish := make(chan struct{})

	// Mock factory that signals when it starts and waits to finish.
	idtokenNewTokenSource = func(ctx context.Context, audience string, opts ...idtoken.ClientOption) (oauth2.TokenSource, error) {
		callCount++
		started <- struct{}{} // Signal that we are in the slow path.
		<-finish              // Wait to simulate a slow I/O operation.
		return &mockTokenSource{token: "cached-token"}, nil
	}

	// Initialize with nil map to test defensive initialization coverage.
	transport := &oidcTransport{
		base:         &mockRoundTripper{},
		tokenSources: nil,
		ctx:          t.Context(),
	}

	// Trigger two concurrent requests for the same audience to hit the double-check return.
	var wg sync.WaitGroup
	wg.Add(2)
	req, _ := http.NewRequest("GET", "https://api.com", nil)

	// Launch first request.
	go func() {
		defer wg.Done()
		if _, err := transport.RoundTrip(req); err != nil {
			t.Errorf("First RoundTrip() failed: %v", err)
		}
	}()

	<-started // Wait for first one to start the slow path.

	// Launch second request for the same audience.
	go func() {
		defer wg.Done()
		if _, err := transport.RoundTrip(req); err != nil {
			t.Errorf("Second RoundTrip() failed: %v", err)
		}
	}()

	<-started // Wait for second one to also reach the slow path.

	// Release both. One will update the map, the second will hit the double-check return.
	close(finish)
	wg.Wait()

	// KILL MUTANT: Call a third time sequentially.
	// This will hit the FIRST "return ts, nil" (Fast Path) and should NOT increment callCount.
	if _, err := transport.RoundTrip(req); err != nil {
		t.Errorf("Third RoundTrip() failed: %v", err)
	}

	if got, want := callCount, 2; got != want {
		t.Errorf("idtokenNewTokenSource call count got %d, want %d (token source should be cached and reused)", got, want)
	}
}

func TestOIDCWrapper_Wrap(t *testing.T) {
	ctx := t.Context()
	wrapper, _, _ := New(ctx, map[string]any{"audience_override": "aud"})

	mockBase := &mockRoundTripper{}
	transport := wrapper.Wrap(mockBase)

	if transport == nil {
		t.Errorf("Wrap() returned nil")
	}

	// Verify it returned the correct concrete type.
	if _, ok := transport.(*oidcTransport); !ok {
		t.Errorf("Wrap() returned type %T, want *oidcTransport", transport)
	}
}
