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

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

func testRetryConfig() NPClientConfig {
	return NPClientConfig{
		Timeout: 1 * time.Second,
	}
}

func TestHttpNPClient_OnSubscribe_Success(t *testing.T) {
	expectedResponse := &model.OnSubscribeResponse{Answer: "correct_answer"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != onSubscribePath {
			t.Errorf("expected path %q, got %q", onSubscribePath, r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected method %q, got %q", http.MethodPost, r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(expectedResponse); err != nil {
			t.Fatalf("Failed to write mock response: %v", err)
		}
	}))
	defer server.Close()

	client := NewNPClient(testRetryConfig())
	request := &model.OnSubscribeRequest{Challenge: "test_challenge"}

	resp, err := client.OnSubscribe(context.Background(), server.URL, request)

	if err != nil {
		t.Fatalf("OnSubscribe() returned an unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("OnSubscribe() response is nil, want non-nil")
	}
	if resp.Answer != expectedResponse.Answer {
		t.Errorf("response Answer = %q, want %q", resp.Answer, expectedResponse.Answer)
	}
}

func TestHttpNPClient_OnSubscribe_Error(t *testing.T) {
	validRequest := &model.OnSubscribeRequest{Challenge: "test_challenge"}
	testCases := []struct {
		name        string
		callbackURL string // Can be overridden for non-server tests
		handler     http.HandlerFunc
		useServer   bool
		request     *model.OnSubscribeRequest
		ctx         context.Context
		wantErrMsg  string
	}{
		{
			name:        "should fail on malformed callback URL",
			useServer:   false,
			callbackURL: "://invalid-url", // Malformed URL
			request:     validRequest,
			ctx:         context.Background(),
			wantErrMsg:  "failed to create HTTP request",
		},
		{
			name:      "should fail when server returns 500",
			useServer: true,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError) // Consistent failure
			},
			request:    validRequest,
			ctx:        context.Background(),
			wantErrMsg: "NP callback failed with status 500",
		},
		{
			name:      "should fail immediately when server returns 400",
			useServer: true,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest) // Non-retryable client error
			},
			request:    validRequest,
			ctx:        context.Background(),
			wantErrMsg: "NP callback failed with status 400",
		},
		{
			name:      "should fail when response body is not valid JSON",
			useServer: true,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if _, err := io.WriteString(w, `{"answer": "missing_quote}`); err != nil {
					t.Fatalf("Failed to write mock response: %v", err)
				}
			},
			request:    validRequest,
			ctx:        context.Background(),
			wantErrMsg: "failed to decode NP response",
		},
		{
			name:      "should fail when context times out",
			useServer: true,
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(50 * time.Millisecond) // Sleep longer than the context timeout
				w.WriteHeader(http.StatusOK)
			},
			request: validRequest,
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
				_ = cancel // To satisfy linter, though timeout will trigger it.
				return ctx
			}(),
			wantErrMsg: "context deadline exceeded",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			serverURL := tc.callbackURL
			if tc.useServer {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				serverURL = server.URL
			}

			client := NewNPClient(testRetryConfig())
			resp, err := client.OnSubscribe(tc.ctx, serverURL, tc.request)

			if err == nil {
				t.Fatalf("OnSubscribe() expected an error, but got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErrMsg) {
				t.Errorf("OnSubscribe() error = %q, want error containing %q", err.Error(), tc.wantErrMsg)
			}
			if resp != nil {
				t.Errorf("OnSubscribe() response should be nil on error, but got %+v", resp)
			}
		})
	}
}

func TestHttpNPClient_OnSubscribe_MarshalError(t *testing.T) {
	client := NewNPClient(testRetryConfig())
	request := &model.OnSubscribeRequest{Challenge: "test_challenge"}
	wantErrMsg := "failed to marshal request"

	oldMarshal := jsonMarshal
	jsonMarshal = func(v any) ([]byte, error) {
		return nil, fmt.Errorf("simulated marshal error")
	}
	defer func() { jsonMarshal = oldMarshal }() // Restore original function

	resp, err := client.OnSubscribe(context.Background(), "http://dummyurl", request)

	if err == nil {
		t.Fatalf("OnSubscribe() expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), wantErrMsg) {
		t.Errorf("OnSubscribe() error = %q, want error containing %q", err.Error(), wantErrMsg)
	}
	if resp != nil {
		t.Errorf("OnSubscribe() response should be nil on error, but got %+v", resp)
	}
}
