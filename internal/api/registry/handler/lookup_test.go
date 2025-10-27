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

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// mockLookupService is a mock implementation of the lookupService interface for testing.
type mockLookupService struct {
	subscriptions []model.Subscription
	err           error
}

func (m *mockLookupService) Lookup(ctx context.Context, filter *model.Subscription) ([]model.Subscription, error) {
	return m.subscriptions, m.err
}

// TestNewLookupHandlerSuccess tests the successful creation of a new LookupHandler.
func TestNewLookupHandlerSuccess(t *testing.T) {
	mockSvc := &mockLookupService{}
	handler := NewLookupHandler(mockSvc)
	if handler == nil {
		t.Fatal("NewLookupHandler() returned nil, want non-nil handler")
	}
	if handler.lhService == nil {
		t.Error("NewLookupHandler() returned handler with nil service, want non-nil service")
	}
}

// TestLookupHandlerLookupSuccess covers successful lookup scenarios.
func TestLookupHandlerLookupSuccess(t *testing.T) {
	expectedSubscriptions := []model.Subscription{
		{
			Subscriber: model.Subscriber{
				SubscriberID: "test-sub-1",
				URL:          "http://example.com/api/1",
				Type:         model.RoleBAP,
				Domain:       "test.domain.com",
			},
			KeyID: "key1",
		},
		{
			Subscriber: model.Subscriber{
				SubscriberID: "test-sub-2",
				URL:          "http://example.org/api/2",
				Type:         model.RoleBPP,
				Domain:       "another.domain.net",
			},
			KeyID: "key2",
		},
	}

	tests := []struct {
		name           string
		requestBody    *model.Subscription
		mockService    *mockLookupService
		expectedStatus int
		expectedBody   []model.Subscription
	}{
		{
			name:        "SuccessfulLookupNoFilter",
			requestBody: &model.Subscription{},
			mockService: &mockLookupService{
				subscriptions: expectedSubscriptions,
				err:           nil,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   expectedSubscriptions,
		},
		{
			name: "SuccessfulLookupWithFilter",
			requestBody: &model.Subscription{
				Subscriber: model.Subscriber{
					SubscriberID: "test-sub-1",
				},
			},
			mockService: &mockLookupService{
				subscriptions: []model.Subscription{expectedSubscriptions[0]},
				err:           nil,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []model.Subscription{expectedSubscriptions[0]},
		},
		{
			name: "SuccessfulLookupNoResults",
			requestBody: &model.Subscription{
				Subscriber: model.Subscriber{
					SubscriberID: "non-existent-subscriber",
				},
			},
			mockService: &mockLookupService{
				subscriptions: []model.Subscription{},
				err:           nil,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []model.Subscription{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqBodyJSON, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body for test case %s: %v", tc.name, err)
			}
			req := httptest.NewRequest(http.MethodPost, "/lookup", bytes.NewReader(reqBodyJSON))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler := NewLookupHandler(tc.mockService)

			router := chi.NewRouter()
			router.Post("/lookup", handler.Lookup)
			router.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("handler.Lookup returned wrong status code: got %v want %v. Body: %s", rr.Code, tc.expectedStatus, rr.Body.String())
			}

			var gotSubscriptions []model.Subscription
			if err := json.Unmarshal(rr.Body.Bytes(), &gotSubscriptions); err != nil {
				t.Fatalf("Failed to unmarshal response body for test case %s: %v. Body: %s", tc.name, err, rr.Body.String())
			}

			if diff := cmp.Diff(tc.expectedBody, gotSubscriptions,
				cmpopts.IgnoreFields(model.Subscription{}, "ValidFrom", "ValidUntil", "Created", "Updated"),
			); diff != "" {
				t.Errorf("handler.Lookup returned unexpected body (-want +got):\n%s", diff)
			}
		})
	}
}

// TestLookupHandlerLookupError covers scenarios where the lookup fails.
func TestLookupHandlerLookupError(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    io.Reader
		mockService    *mockLookupService
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "InvalidRequestBody",
			requestBody:    bytes.NewBufferString(`{"invalid json`),
			mockService:    &mockLookupService{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body\n",
		},
		{
			name: "ServiceLookupError",
			requestBody: func() io.Reader {
				reqBody, err := json.Marshal(&model.Subscription{
					Subscriber: model.Subscriber{SubscriberID: "error-id"},
				})
				if err != nil {
					t.Fatalf("Failed to marshal request body in test setup: %v", err)
				}
				return bytes.NewReader(reqBody)
			}(),
			mockService: &mockLookupService{
				subscriptions: nil,
				err:           errors.New("database connection lost"),
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to lookup subscriptions\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/lookup", tc.requestBody)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler := NewLookupHandler(tc.mockService)

			router := chi.NewRouter()
			router.Post("/lookup", handler.Lookup)
			router.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("handler.Lookup returned wrong status code: got %v want %v. Body: %s", rr.Code, tc.expectedStatus, rr.Body.String())
			}
			if diff := cmp.Diff(tc.expectedBody, rr.Body.String()); diff != "" {
				t.Errorf("handler.Lookup returned unexpected body (-want +got):\n%s", diff)
			}
		})
	}
}

// ErrorWriter is an http.ResponseWriter that can be configured to return an error on Write.
type ErrorWriter struct {
	HeaderMap  http.Header
	WriteErr   error
	StatusCode int
}

func NewErrorWriter(err error) *ErrorWriter {
	return &ErrorWriter{
		WriteErr:  err,
		HeaderMap: make(http.Header),
	}
}

// Header implements http.ResponseWriter.
func (ew *ErrorWriter) Header() http.Header {
	return ew.HeaderMap
}

// Write always returns the configured WriteErr.
func (ew *ErrorWriter) Write(b []byte) (int, error) {
	return 0, ew.WriteErr
}

// WriteHeader implements http.ResponseWriter.
func (ew *ErrorWriter) WriteHeader(statusCode int) {
	ew.StatusCode = statusCode
}

func TestLookupHandlerLookupErrorEncodeFailure(t *testing.T) {
	mockSvc := &mockLookupService{
		subscriptions: []model.Subscription{{
			Subscriber: model.Subscriber{SubscriberID: "test-encode-failure"},
		}},
		err: nil,
	}

	reqBody, err := json.Marshal(&model.Subscription{})
	if err != nil {
		t.Fatalf("Failed to marshal request body in test setup: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/lookup", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	expectedWriteErr := errors.New("simulated write error to client connection")
	ew := NewErrorWriter(expectedWriteErr)

	handler := NewLookupHandler(mockSvc)

	router := chi.NewRouter()
	router.Post("/lookup", handler.Lookup)
	router.ServeHTTP(ew, req)

	if ew.StatusCode != http.StatusInternalServerError {
		t.Errorf("handler.Lookup WriteHeader status code = %d, want %d", ew.StatusCode, http.StatusInternalServerError)
	}
}
