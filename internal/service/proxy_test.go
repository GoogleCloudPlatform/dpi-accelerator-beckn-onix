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
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"github.com/hashicorp/go-retryablehttp"
)

// mockHttpClient is a mock implementation of the httpClient interface.
type mockHttpClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"message":{"ack":{"status":"ACK"}}}`))}, nil
}

// Helper to create a dummy AsyncTask
func newTestAsyncTask(targetURL string, body []byte, headers http.Header) *model.AsyncTask {
	u, _ := url.Parse(targetURL)
	return &model.AsyncTask{
		Type:    model.AsyncTaskTypeProxy,
		Target:  u,
		Body:    body,
		Headers: headers,
		Context: model.Context{Action: "search"},
	}
}

// Helper to create a mock HTTP response
func newMockHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestNewProxyTaskProcessor(t *testing.T) {
	mockAuth := &mockAuthGen{}

	tests := []struct {
		name     string
		auth     authGen
		keyID    string
		retryCfg RetryConfig
		wantErr  string
		check    func(*testing.T, *proxyTaskProcessor, RetryConfig)
	}{
		{
			name:     "success",
			auth:     mockAuth,
			keyID:    "test-key-id",
			retryCfg: RetryConfig{RetryMax: 1, RetryWaitMin: 1 * time.Millisecond, RetryWaitMax: 1 * time.Millisecond},
			wantErr:  "",
		},
		{
			name:     "nil authGen",
			auth:     nil,
			keyID:    "test-key-id",
			retryCfg: RetryConfig{},
			wantErr:  "authGen cannot be nil",
		},
		{
			name:     "empty keyID",
			auth:     mockAuth,
			keyID:    "",
			retryCfg: RetryConfig{},
			wantErr:  "keyID cannot be empty",
		},
		{
			name:  "full client configuration",
			auth:  mockAuth,
			keyID: "test-key-id",
			retryCfg: RetryConfig{
				RetryMax:            5,
				RetryWaitMin:        100 * time.Millisecond,
				RetryWaitMax:        5 * time.Second,
				Timeout:             15 * time.Second,
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 10,
				MaxConnsPerHost:     20,
				IdleConnTimeout:     2 * time.Minute,
			},
			wantErr: "",
			check:   checkClientConfig,
		},
		{
			name:     "zero client configuration uses defaults",
			auth:     mockAuth,
			keyID:    "test-key-id",
			retryCfg: RetryConfig{},
			wantErr:  "",
			check:    checkClientConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProxyTaskProcessor(tt.auth, tt.keyID, tt.retryCfg)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("NewProxyTaskProcessor() error = %v, want %q", err, tt.wantErr)
				}
				if p != nil {
					t.Errorf("NewProxyTaskProcessor() got %v, want nil on error", p)
				}
			} else {
				if err != nil {
					t.Errorf("NewProxyTaskProcessor() unexpected error = %v", err)
				}
				if p == nil {
					t.Fatal("NewProxyTaskProcessor() returned nil, want non-nil")
				}
				if p.auth != tt.auth {
					t.Errorf("NewProxyTaskProcessor() auth not set correctly")
				}
				if p.keyID != tt.keyID {
					t.Errorf("NewProxyTaskProcessor() keyID not set correctly")
				}
				if p.client == nil {
					t.Errorf("NewProxyTaskProcessor() httpClient not initialized")
				}
				if tt.check != nil {
					tt.check(t, p, tt.retryCfg)
				}
			}
		})
	}
}

func checkClientConfig(t *testing.T, p *proxyTaskProcessor, retryCfg RetryConfig) {
	t.Helper()
	// Type assert the interface back to a concrete client to inspect its fields.
	client, ok := p.client.(*http.Client)
	if !ok {
		t.Fatalf("processor.client is not of type *http.Client, but %T", p.client)
	}
	rt, ok := client.Transport.(*retryablehttp.RoundTripper)
	if !ok {
		t.Fatalf("client transport is not a *retryablehttp.RoundTripper, but %T", client.Transport)
	}
	transport, ok := rt.Client.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("underlying transport is not an *http.Transport, but %T", rt.Client.HTTPClient.Transport)
	}

	if retryCfg.MaxIdleConns > 0 && transport.MaxIdleConns != retryCfg.MaxIdleConns {
		t.Errorf("Transport MaxIdleConns = %d, want %d", transport.MaxIdleConns, retryCfg.MaxIdleConns)
	}
	if retryCfg.MaxIdleConnsPerHost > 0 && transport.MaxIdleConnsPerHost != retryCfg.MaxIdleConnsPerHost {
		t.Errorf("Transport MaxIdleConnsPerHost = %d, want %d", transport.MaxIdleConnsPerHost, retryCfg.MaxIdleConnsPerHost)
	}
	if retryCfg.MaxConnsPerHost > 0 && transport.MaxConnsPerHost != retryCfg.MaxConnsPerHost {
		t.Errorf("Transport MaxConnsPerHost = %d, want %d", transport.MaxConnsPerHost, retryCfg.MaxConnsPerHost)
	}
	if retryCfg.IdleConnTimeout > 0 && transport.IdleConnTimeout != retryCfg.IdleConnTimeout {
		t.Errorf("Transport IdleConnTimeout = %v, want %v", transport.IdleConnTimeout, retryCfg.IdleConnTimeout)
	}
}

func TestProxyTaskProcessor_validateTask(t *testing.T) {
	p := &proxyTaskProcessor{} // No need for full initialization for this method
	validTask := newTestAsyncTask("http://example.com", []byte(`{}`), make(http.Header))

	tests := []struct {
		name    string
		task    *model.AsyncTask
		wantErr string
	}{
		{name: "valid task", task: validTask, wantErr: ""},
		{name: "nil task", task: nil, wantErr: "async task cannot be nil"},
		{name: "nil target", task: &model.AsyncTask{Headers: make(http.Header)}, wantErr: "async task target URL cannot be nil"},
		{name: "nil headers", task: &model.AsyncTask{Target: &url.URL{}}, wantErr: "async task headers cannot be nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.validateTask(context.Background(), tt.task)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("validateTask() error = %v, want %q", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("validateTask() unexpected error = %v", err)
			}
		})
	}
}

func TestProxyTaskProcessor_httpReq(t *testing.T) {
	ctx := context.Background()
	mockAuth := &mockAuthGen{authHeader: "Signature test-auth"}
	p := &proxyTaskProcessor{auth: mockAuth, keyID: "test-key"}

	tests := []struct {
		name            string
		task            *model.AsyncTask
		authGenErr      error
		wantAuthHeader  string
		wantContentType string
		wantErr         string
	}{
		{
			name:            "success - auth header generated",
			task:            newTestAsyncTask("http://example.com/search", []byte(`{"key":"value"}`), make(http.Header)),
			wantAuthHeader:  "Signature test-auth",
			wantContentType: "application/json",
			wantErr:         "",
		},
		{
			name:            "success - auth header already present",
			task:            newTestAsyncTask("http://example.com/search", []byte(`{}`), http.Header{model.AuthHeaderGateway: []string{"Existing-Auth"}}),
			wantAuthHeader:  "Existing-Auth",    // Auth header is already present, so it's not generated.
			wantContentType: "application/json", // Body is not empty, so Content-Type is set.
			wantErr:         "",
		},
		{
			name:            "success - Content-Type already present",
			task:            newTestAsyncTask("http://example.com/search", []byte(`{"key":"value"}`), http.Header{"Content-Type": []string{"application/xml"}}),
			wantAuthHeader:  "Signature test-auth", // Auth header is generated.
			wantContentType: "application/xml",     // Content-Type is already present, so it's not forced to application/json.
			wantErr:         "",
		},
		{
			name:           "authGen returns error",
			task:           newTestAsyncTask("http://example.com/search", []byte(`{}`), make(http.Header)),
			authGenErr:     errors.New("auth error"),
			wantAuthHeader: "",
			wantErr:        "failed to generate auth header: auth error",
		},
		{
			name:            "nil task body, no Content-Type set",
			task:            newTestAsyncTask("http://example.com/search", nil, make(http.Header)),
			wantAuthHeader:  "Signature test-auth",
			wantContentType: "",
			wantErr:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.auth.(*mockAuthGen).err = tt.authGenErr // Set mock error
			req, err := p.httpReq(ctx, tt.task)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("httpReq() error = %v, want error containing %q", err, tt.wantErr)
				}
				if req != nil {
					t.Errorf("httpReq() got request %v, want nil on error", req)
				}
			} else {
				if err != nil {
					t.Errorf("httpReq() unexpected error = %v", err)
				}
				if req == nil {
					t.Fatal("httpReq() returned nil request")
				}
				if gotAuth := req.Header.Get(model.AuthHeaderGateway); gotAuth != tt.wantAuthHeader {
					t.Errorf("httpReq() AuthHeaderGateway = %q, want %q", gotAuth, tt.wantAuthHeader)
				}
				if gotCT := req.Header.Get("Content-Type"); gotCT != tt.wantContentType {
					t.Errorf("httpReq() Content-Type = %q, want %q", gotCT, tt.wantContentType)
				}
			}
		})
	}
}

func TestProxyTaskProcessor_proxy(t *testing.T) {
	ctx := context.Background()
	p := &proxyTaskProcessor{} // Will set client mock per test
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://example.com/test", nil)

	tests := []struct {
		name       string
		mockClient func(*mockHttpClient)
		wantErr    string
	}{
		{
			name: "success - 200 OK with ACK",
			mockClient: func(m *mockHttpClient) {
				m.doFunc = func(r *http.Request) (*http.Response, error) {
					return newMockHTTPResponse(http.StatusOK, `{"message":{"ack":{"status":"ACK"}}}`), nil
				}
			},
			wantErr: "",
		},
		{
			name: "HTTP request fails (network error)",
			mockClient: func(m *mockHttpClient) {
				m.doFunc = func(r *http.Request) (*http.Response, error) {
					return nil, errors.New("network unreachable")
				}
			},
			wantErr: "HTTP request to http://example.com/test failed: network unreachable",
		},
		{
			name: "non-200 status code",
			mockClient: func(m *mockHttpClient) {
				m.doFunc = func(r *http.Request) (*http.Response, error) {
					return newMockHTTPResponse(http.StatusBadRequest, `{"error":"bad request"}`), nil
				}
			},
			wantErr: "unexpected status code 400 from http://example.com/test. Body: {\"error\":\"bad request\"}",
		},
		{
			name: "failed to read response body",
			mockClient: func(m *mockHttpClient) {
				m.doFunc = func(r *http.Request) (*http.Response, error) {
					// Simulate a body that errors on read
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(errorReader{})}, nil
				}
			},
			wantErr: "failed to read response body from http://example.com/test: mock read error",
		},
		{
			name: "failed to unmarshal response body (invalid JSON)",
			mockClient: func(m *mockHttpClient) {
				m.doFunc = func(r *http.Request) (*http.Response, error) {
					return newMockHTTPResponse(http.StatusOK, `invalid json`), nil // Truly invalid JSON
				}
			},
			wantErr: "failed to unmarshal response body from http://example.com/test into model.TxnResponse: invalid character 'i' looking for beginning of value. Body: invalid json",
		},
		{
			name: "response status is NACK",
			mockClient: func(m *mockHttpClient) {
				m.doFunc = func(r *http.Request) (*http.Response, error) {
					return newMockHTTPResponse(http.StatusOK, `{"message":{"ack":{"status":"NACK"}}}`), nil
				}
			},
			wantErr: "response status is not ACK",
		},
		{
			name: "response status is NACK with error details",
			mockClient: func(m *mockHttpClient) {
				m.doFunc = func(r *http.Request) (*http.Response, error) {
					return newMockHTTPResponse(http.StatusOK, `{"message":{"ack":{"status":"NACK"},"error":{"code":"INTERNAL_SERVER_ERROR","message":"some error"}}}`), nil
				}
			},
			wantErr: "response status is NACK from http://example.com/test: Code=INTERNAL_SERVER_ERROR, Message=some error",
		},
		{
			name: "non-200 status code with body read error",
			mockClient: func(m *mockHttpClient) {
				m.doFunc = func(r *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(errorReader{}),
					}, nil
				}
			},
			wantErr: "unexpected status code 500 from http://example.com/test. Body: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockHttpClient{}
			tt.mockClient(mockClient)
			p.client = mockClient
			err := p.proxy(ctx, req)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("proxy() error = %v, want error containing %q", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("proxy() unexpected error = %v", err)
			}
		})
	}
}

// errorReader is an io.Reader that always returns an error.
type errorReader struct{}

func (er errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("mock read error")
}

func TestProxyTaskProcessor_Process(t *testing.T) {
	ctx := context.Background()
	validTask := newTestAsyncTask("http://example.com/process", []byte(`{"data":"test"}`), make(http.Header))
	validTask.Headers.Set(model.AuthHeaderGateway, "Auth test") // Pre-set auth header for simplicity

	tests := []struct {
		name       string
		task       *model.AsyncTask
		mockClient func(*mockHttpClient)
		mockAuth   *mockAuthGen
		wantErr    string
	}{
		{
			name: "success - end to end",
			task: validTask,
			mockClient: func(m *mockHttpClient) {
				m.doFunc = func(r *http.Request) (*http.Response, error) {
					return newMockHTTPResponse(http.StatusOK, `{"message":{"ack":{"status":"ACK"}}}`), nil
				}
			},
			mockAuth: &mockAuthGen{authHeader: "Signature test-auth"},
			wantErr:  "",
		},
		{
			name: "validation error",
			task: nil, // This will cause validation to fail
			mockClient: func(m *mockHttpClient) {
				m.doFunc = func(r *http.Request) (*http.Response, error) { return nil, nil }
			},
			mockAuth: &mockAuthGen{},
			wantErr:  "async task cannot be nil",
		},
		{
			name: "httpReq fails (authGen error)",
			task: newTestAsyncTask("http://example.com/process", []byte(`{}`), make(http.Header)), // No auth header, so it will try to generate
			mockClient: func(m *mockHttpClient) {
				m.doFunc = func(r *http.Request) (*http.Response, error) { return nil, nil }
			},
			mockAuth: &mockAuthGen{err: errors.New("auth gen failed")},
			wantErr:  "failed to generate auth header: auth gen failed",
		},
		{
			name: "proxy fails (non-200 status)",
			task: validTask,
			mockClient: func(m *mockHttpClient) {
				m.doFunc = func(r *http.Request) (*http.Response, error) {
					return newMockHTTPResponse(http.StatusInternalServerError, `{"error":"server error"}`), nil
				}
			},
			mockAuth: &mockAuthGen{},
			wantErr:  "unexpected status code 500 from http://example.com/process. Body: {\"error\":\"server error\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockHttpClient{}
			if tt.mockClient != nil {
				tt.mockClient(mockClient)
			}

			p := &proxyTaskProcessor{
				client: mockClient,
				auth:   tt.mockAuth,
				keyID:  "test-key-id",
			}

			err := p.Process(ctx, tt.task)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Process() error = %v, want error containing %q", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("Process() unexpected error = %v", err)
			}
		})
	}
}
