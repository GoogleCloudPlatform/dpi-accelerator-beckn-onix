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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"github.com/google/go-cmp/cmp"
)

// isNil checks if an interface value is nil.
// This is necessary because an interface containing a nil pointer or slice is not nil itself.
func isNil(i any) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// testRegistryClientConfig returns a default config for tests.
func testRegistryClientConfig(baseURL string) *RegistryClientConfig {
	return &RegistryClientConfig{
		Timeout: 1 * time.Second,
		BaseURL: baseURL,
	}
}

// TestNewRegistryClient tests the constructor for the registry client.
func TestNewRegistryClient(t *testing.T) {
	t.Run("success with no connection pooling config", func(t *testing.T) {
		cfg := &RegistryClientConfig{
			BaseURL: "http://localhost:8080",
			Timeout: 5 * time.Second,
		}
		client, err := NewRegistryClient(cfg)
		if err != nil {
			t.Fatalf("NewRegistryClient() error = %v, wantErr false", err)
		}
		if client == nil {
			t.Fatal("NewRegistryClient() returned nil client")
		}
		if client.baseURL != cfg.BaseURL {
			t.Errorf("client.baseURL = %q, want %q", client.baseURL, cfg.BaseURL)
		}
		if client.client.Timeout != cfg.Timeout {
			t.Errorf("client.client.Timeout = %v, want %v", client.client.Timeout, cfg.Timeout)
		}
	})

	t.Run("success with connection pooling config", func(t *testing.T) {
		cfg := &RegistryClientConfig{
			BaseURL:             "http://localhost:8080",
			Timeout:             5 * time.Second,
			MaxIdleConns:        50,
			MaxIdleConnsPerHost: 10,
			MaxConnsPerHost:     20,
			IdleConnTimeout:     90 * time.Second,
		}
		client, err := NewRegistryClient(cfg)
		if err != nil {
			t.Fatalf("NewRegistryClient() error = %v, wantErr false", err)
		}
		if client == nil {
			t.Fatal("NewRegistryClient() returned nil client")
		}

		transport, ok := client.client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("client.client.Transport is not an *http.Transport")
		}

		if transport.MaxIdleConns != cfg.MaxIdleConns {
			t.Errorf("transport.MaxIdleConns = %d, want %d", transport.MaxIdleConns, cfg.MaxIdleConns)
		}
		if transport.MaxIdleConnsPerHost != cfg.MaxIdleConnsPerHost {
			t.Errorf("transport.MaxIdleConnsPerHost = %d, want %d", transport.MaxIdleConnsPerHost, cfg.MaxIdleConnsPerHost)
		}
	})

	t.Run("default timeout", func(t *testing.T) {
		cfg := &RegistryClientConfig{
			BaseURL: "http://localhost:8080",
		}
		client, err := NewRegistryClient(cfg)
		if err != nil {
			t.Fatalf("NewRegistryClient() error = %v, wantErr false", err)
		}
		if client.client.Timeout != 10*time.Second {
			t.Errorf("client.client.Timeout = %v, want %v", client.client.Timeout, 10*time.Second)
		}
	})

	t.Run("error cases", func(t *testing.T) {
		testCases := []struct {
			name    string
			config  *RegistryClientConfig
			wantErr string
		}{
			{
				name:    "nil config",
				config:  nil,
				wantErr: "RegistryClientConfig cannot be nil",
			},
			{
				name:    "empty baseURL",
				config:  &RegistryClientConfig{},
				wantErr: "BaseURL cannot be empty in RegistryClientConfig",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := NewRegistryClient(tc.config)
				if err == nil {
					t.Fatal("NewRegistryClient() expected an error, but got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("NewRegistryClient() error = %q, want error containing %q", err.Error(), tc.wantErr)
				}
			})
		}
	})
}

func runErrorTests(t *testing.T, testName string, clientCall func(context.Context, *httpRegistryClient) (any, error), logAction string, hasBody bool) {
	t.Helper()

	testCases := []struct {
		name       string
		handler    http.HandlerFunc
		ctx        context.Context
		wantErrMsg string
		setup      func(cfg *RegistryClientConfig)
	}{
		{
			name: "server returns 500",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				if _, err := io.WriteString(w, "internal error"); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			},
			ctx:        context.Background(),
			wantErrMsg: fmt.Sprintf("registry %s failed with status 500: internal error", logAction),
		},
		{
			name: "response body is not valid JSON",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if _, err := io.WriteString(w, `{"key": "malformed"`); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			},
			ctx:        context.Background(),
			wantErrMsg: fmt.Sprintf("failed to unmarshal Registry %s response", logAction),
		},
		{
			name: "context times out",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(50 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
				_ = cancel
				return ctx
			}(),
			wantErrMsg: "context deadline exceeded",
		},
		{
			name: "network error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// This handler will not be reached
			},
			ctx: context.Background(),
			setup: func(cfg *RegistryClientConfig) {
				// Use an invalid URL to simulate a network error
				cfg.BaseURL = "http://unreachable-host:9999"
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s: %s", testName, tc.name), func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			cfg := testRegistryClientConfig(server.URL)
			if tc.setup != nil {
				tc.setup(cfg)
			}

			client, err := NewRegistryClient(cfg)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			resp, err := clientCall(tc.ctx, client)

			if err == nil {
				t.Fatalf("%s() expected an error, but got nil", testName)
			}
			if tc.wantErrMsg != "" && !strings.Contains(err.Error(), tc.wantErrMsg) {
				t.Errorf("%s() error = %q, want error containing %q", testName, err.Error(), tc.wantErrMsg)
			}
			if !isNil(resp) {
				t.Errorf("%s() response should be nil on error, but got %+v", testName, resp)
			}
		})
	}
}

func runMarshalErrorTest(t *testing.T, testName string, clientCall func(context.Context, *httpRegistryClient) (any, error), logAction string) {
	t.Helper()

	t.Run(fmt.Sprintf("%s: MarshalError", testName), func(t *testing.T) {
		client, _ := NewRegistryClient(testRegistryClientConfig("http://dummy-url"))
		wantErrMsg := fmt.Sprintf("failed to marshal %s request", logAction)
		simulatedErr := "simulated marshal error"

		// Monkey-patch json.Marshal
		oldMarshal := jsonMarshal
		jsonMarshal = func(v any) ([]byte, error) {
			return nil, errors.New(simulatedErr)
		}
		defer func() { jsonMarshal = oldMarshal }()

		resp, err := clientCall(context.Background(), client)

		if err == nil {
			t.Fatalf("%s() expected an error, but got nil", testName)
		}
		if !strings.Contains(err.Error(), wantErrMsg) || !strings.Contains(err.Error(), simulatedErr) {
			t.Errorf("%s() error = %q, want error containing %q and %q", testName, err.Error(), wantErrMsg, simulatedErr)
		}
		if !isNil(resp) {
			t.Errorf("%s() response should be nil on error, but got %+v", testName, resp)
		}
	})
}

// --- Lookup Tests ---

func TestHttpRegistryClient_Lookup_Success(t *testing.T) {
	expectedRequest := &model.Subscription{
		Subscriber: model.Subscriber{SubscriberID: "test-sub"},
	}
	expectedResponse := []model.Subscription{{
		Subscriber: model.Subscriber{SubscriberID: "test-sub"},
		Status:     "SUBSCRIBED",
	}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != lookupPath {
			t.Errorf("expected path %q, got %q", lookupPath, r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected method %q, got %q", http.MethodPost, r.Method)
		}

		var gotRequest model.Subscription
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if diff := cmp.Diff(expectedRequest, &gotRequest); diff != "" {
			t.Errorf("request body mismatch (-want +got):\n%s", diff)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(expectedResponse); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client, _ := NewRegistryClient(testRegistryClientConfig(server.URL))
	resp, err := client.Lookup(context.Background(), expectedRequest)

	if err != nil {
		t.Fatalf("Lookup() returned an unexpected error: %v", err)
	}
	if diff := cmp.Diff(expectedResponse, resp); diff != "" {
		t.Errorf("Lookup() response mismatch (-want +got):\n%s", diff)
	}
}

func TestHttpRegistryClient_Lookup_Error(t *testing.T) {
	runErrorTests(t, "Lookup",
		func(ctx context.Context, client *httpRegistryClient) (any, error) {
			return client.Lookup(ctx, &model.Subscription{Subscriber: model.Subscriber{SubscriberID: "test-sub"}})
		},
		"POST /lookup", true)
}

func TestHttpRegistryClient_Lookup_MarshalError(t *testing.T) {
	runMarshalErrorTest(t, "Lookup",
		func(ctx context.Context, client *httpRegistryClient) (any, error) {
			return client.Lookup(ctx, &model.Subscription{Subscriber: model.Subscriber{SubscriberID: "test-sub"}})
		},
		"POST /lookup")
}

// --- CreateSubscription Tests ---

func TestHttpRegistryClient_CreateSubscription_Success(t *testing.T) {
	expectedRequest := &model.SubscriptionRequest{
		Subscription: model.Subscription{Subscriber: model.Subscriber{SubscriberID: "new-sub"}},
	}
	expectedResponse := &model.SubscriptionResponse{MessageID: "msg-123"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != subscribePath {
			t.Errorf("expected path %q, got %q", subscribePath, r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected method %q, got %q", http.MethodPost, r.Method)
		}

		var gotRequest model.SubscriptionRequest
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if diff := cmp.Diff(expectedRequest, &gotRequest); diff != "" {
			t.Errorf("request body mismatch (-want +got):\n%s", diff)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(expectedResponse); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client, _ := NewRegistryClient(testRegistryClientConfig(server.URL))
	resp, err := client.CreateSubscription(context.Background(), expectedRequest)

	if err != nil {
		t.Fatalf("CreateSubscription() returned an unexpected error: %v", err)
	}
	if diff := cmp.Diff(expectedResponse, resp); diff != "" {
		t.Errorf("CreateSubscription() response mismatch (-want +got):\n%s", diff)
	}
}

func TestHttpRegistryClient_CreateSubscription_Error(t *testing.T) {
	runErrorTests(t, "CreateSubscription",
		func(ctx context.Context, client *httpRegistryClient) (any, error) {
			return client.CreateSubscription(ctx, &model.SubscriptionRequest{Subscription: model.Subscription{Subscriber: model.Subscriber{SubscriberID: "new-sub"}}})
		},
		"POST /subscribe", true)
}

func TestHttpRegistryClient_CreateSubscription_MarshalError(t *testing.T) {
	runMarshalErrorTest(t, "CreateSubscription",
		func(ctx context.Context, client *httpRegistryClient) (any, error) {
			return client.CreateSubscription(ctx, &model.SubscriptionRequest{Subscription: model.Subscription{Subscriber: model.Subscriber{SubscriberID: "new-sub"}}})
		},
		"POST /subscribe")
}

// --- UpdateSubscription Tests ---

func TestHttpRegistryClient_UpdateSubscription_Success(t *testing.T) {
	expectedRequest := &model.SubscriptionRequest{
		Subscription: model.Subscription{Subscriber: model.Subscriber{SubscriberID: "update-sub"}},
	}
	expectedResponse := &model.SubscriptionResponse{MessageID: "msg-456"}
	authHeader := "Bearer some-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != subscribePath {
			t.Errorf("expected path %q, got %q", subscribePath, r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("expected method %q, got %q", http.MethodPatch, r.Method)
		}
		if r.Header.Get(model.AuthHeaderSubscriber) != authHeader {
			t.Errorf("expected auth header %q, got %q", authHeader, r.Header.Get(model.AuthHeaderSubscriber))
		}

		var gotRequest model.SubscriptionRequest
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if diff := cmp.Diff(expectedRequest, &gotRequest); diff != "" {
			t.Errorf("request body mismatch (-want +got):\n%s", diff)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(expectedResponse); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client, _ := NewRegistryClient(testRegistryClientConfig(server.URL))
	resp, err := client.UpdateSubscription(context.Background(), expectedRequest, authHeader)

	if err != nil {
		t.Fatalf("UpdateSubscription() returned an unexpected error: %v", err)
	}
	if diff := cmp.Diff(expectedResponse, resp); diff != "" {
		t.Errorf("UpdateSubscription() response mismatch (-want +got):\n%s", diff)
	}
}

func TestHttpRegistryClient_UpdateSubscription_Error(t *testing.T) {
	runErrorTests(t, "UpdateSubscription",
		func(ctx context.Context, client *httpRegistryClient) (any, error) {
			return client.UpdateSubscription(ctx, &model.SubscriptionRequest{Subscription: model.Subscription{Subscriber: model.Subscriber{SubscriberID: "update-sub"}}}, "auth")
		},
		"PATCH /subscribe", true)
}

func TestHttpRegistryClient_UpdateSubscription_MarshalError(t *testing.T) {
	runMarshalErrorTest(t, "UpdateSubscription",
		func(ctx context.Context, client *httpRegistryClient) (any, error) {
			return client.UpdateSubscription(ctx, &model.SubscriptionRequest{Subscription: model.Subscription{Subscriber: model.Subscriber{SubscriberID: "update-sub"}}}, "auth")
		},
		"PATCH /subscribe")
}

// --- GetOperation Tests ---

func TestHttpRegistryClient_GetOperation_Success(t *testing.T) {
	operationID := "op-123"
	expectedResponse := &model.LRO{OperationID: operationID, Status: model.LROStatusApproved}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := fmt.Sprintf(operationsPathFmt, operationID)
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected method %q, got %q", http.MethodGet, r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(expectedResponse); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client, _ := NewRegistryClient(testRegistryClientConfig(server.URL))
	resp, err := client.GetOperation(context.Background(), operationID)

	if err != nil {
		t.Fatalf("GetOperation() returned an unexpected error: %v", err)
	}
	if diff := cmp.Diff(expectedResponse, resp); diff != "" {
		t.Errorf("GetOperation() response mismatch (-want +got):\n%s", diff)
	}
}

func TestHttpRegistryClient_GetOperation_Error(t *testing.T) {
	operationID := "op-err-123"
	logAction := fmt.Sprintf("GET /operations/%s", operationID)
	runErrorTests(t, "GetOperation", func(ctx context.Context, client *httpRegistryClient) (any, error) {
		return client.GetOperation(ctx, operationID)
	}, logAction, false)
}
