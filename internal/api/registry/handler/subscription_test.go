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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

type mockAuthenticator struct {
	req *model.SubscriptionRequest
	err *model.AuthError
}

func (m *mockAuthenticator) AuthenticatedReq(ctx context.Context, bodyBytes []byte, authHeader string) (*model.SubscriptionRequest, *model.AuthError) {
	return m.req, m.err
}

// mockSubscriptionService is a mock implementation of subscriptionService.
type mockSubscriptionService struct {
	lro       *model.LRO
	key       string
	createErr error // Specific error for Create
	updateErr error // Specific error for Update
}

func (m *mockSubscriptionService) Create(ctx context.Context, req *model.SubscriptionRequest) (*model.LRO, error) {
	return m.lro, m.createErr
}
func (m *mockSubscriptionService) Update(ctx context.Context, req *model.SubscriptionRequest) (*model.LRO, error) {
	return m.lro, m.updateErr
}

// errorReader is a helper for testing io.ReadAll errors
type errorReader struct{}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("forced read error")
}

func TestNewSubscriptionHandler_Success(t *testing.T) {
	mockService := &mockSubscriptionService{}
	mockAuth := &mockAuthenticator{}

	handler, err := NewSubscriptionHandler(mockService, mockAuth)
	if err != nil {
		t.Fatalf("NewSubscriptionHandler() error = %v, wantErr false", err)
	}
	if handler == nil {
		t.Fatalf("NewSubscriptionHandler() expected handler, got nil")
	}
	if handler.subService != mockService {
		t.Errorf("NewSubscriptionHandler() subService not set correctly")
	}
	if handler.auth != mockAuth {
		t.Errorf("NewSubscriptionHandler() authenticator not set correctly")
	}
}

func TestNewSubscriptionHandler_Error(t *testing.T) {
	mockService := &mockSubscriptionService{}
	mockAuth := &mockAuthenticator{}

	tests := []struct {
		name      string
		ss        subscriptionService
		auth      authenticator
		wantError string
	}{
		{
			name:      "nil_subscriptionService",
			ss:        nil,
			auth:      mockAuth,
			wantError: "subscriptionService dependency is nil",
		}, // This test case is correct
		{
			name:      "nil authenticator",
			ss:        mockService,
			auth:      nil,
			wantError: "authenticator dependency is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSubscriptionHandler(tt.ss, tt.auth)
			if err == nil {
				t.Fatalf("NewSubscriptionHandler() error = nil, wantErr true")
			}
			if err.Error() != tt.wantError {
				t.Errorf("NewSubscriptionHandler() error = %v, wantErrorMsg %v", err, tt.wantError)
			}
		})
	}
}

func TestWriteJSONError(t *testing.T) {
	tests := []struct {
		name               string
		statusCode         int
		errType            model.ErrorType
		errCode            model.ErrorCode
		errMsg             string
		errPath            string
		realmForAuthHeader string
		wantHeader         http.Header
		wantBody           []string
	}{
		{
			name:               "bad request error",
			statusCode:         http.StatusBadRequest,
			errType:            model.ErrorTypeValidationError,
			errCode:            model.ErrorCodeInvalidJSON,
			errMsg:             "Invalid JSON",
			errPath:            "/test",
			realmForAuthHeader: "",
			wantHeader:         http.Header{"Content-Type": []string{"application/json"}},
			wantBody:           []string{fmt.Sprintf(`"type":"%s"`, model.ErrorTypeValidationError), fmt.Sprintf(`"code":"%s"`, model.ErrorCodeInvalidJSON), `"message":"Invalid JSON"`, `"path":"/test"`},
		},
		{
			name:               "unauthorized error with realm",
			statusCode:         http.StatusUnauthorized,
			errType:            model.ErrorTypeAuthError,
			errCode:            model.ErrorCodeMissingAuthHeader,
			errMsg:             "Auth header missing",
			errPath:            "",
			realmForAuthHeader: "test-realm",
			wantHeader: http.Header{
				"Content-Type":                     []string{"application/json"},
				model.UnauthorizedHeaderSubscriber: []string{`Signature realm="test-realm",headers="(created) (expires) digest"`},
			},
			wantBody: []string{fmt.Sprintf(`"type":"%s"`, model.ErrorTypeAuthError), fmt.Sprintf(`"code":"%s"`, model.ErrorCodeMissingAuthHeader), `"message":"Auth header missing"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeJSONError(rr, tt.statusCode, tt.errType, tt.errCode, tt.errMsg, tt.errPath, tt.realmForAuthHeader) // writeJSONError itself calls service.unauthorizedHeader

			if rr.Code != tt.statusCode {
				t.Errorf("writeJSONError() status code = %v, want %v", rr.Code, tt.statusCode)
			}

			for key, wantValues := range tt.wantHeader {
				gotValues := rr.Header().Values(key)
				if len(gotValues) != len(wantValues) {
					t.Errorf("writeJSONError() header %s: got values %v, want %v", key, gotValues, wantValues)
					continue
				}
				for i, v := range wantValues {
					if gotValues[i] != v {
						t.Errorf("writeJSONError() header %s value[%d]: got %s, want %s", key, i, gotValues[i], v)
					}
				}
			}

			bodyStr := rr.Body.String()
			for _, substr := range tt.wantBody {
				if !strings.Contains(bodyStr, substr) {
					t.Errorf("writeJSONError() body does not contain %q. Body: %s", substr, bodyStr)
				}
			}

			var errResp model.ErrorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
				t.Errorf("writeJSONError() body is not valid JSON: %v. Body: %s", err, bodyStr)
			}
		})
	}
}

func TestSubscriptionHandler_Create_Success(t *testing.T) {
	defaultLRO := &model.LRO{OperationID: "test-op-id", Status: "PENDING"}
	defaultSubReq := model.SubscriptionRequest{
		Subscription: model.Subscription{
			Subscriber: model.Subscriber{
				SubscriberID: "test-subscriber",
				Domain:       "test-domain",
				Type:         model.RoleBAP, // Example role
			}},
		MessageID: "test-msg-id",
	}
	defaultSubReqBytes, _ := json.Marshal(defaultSubReq)

	subSrv := &mockSubscriptionService{lro: defaultLRO}
	wantStatusCode := http.StatusOK
	wantContentType := "application/json"

	req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBuffer(defaultSubReqBytes))
	rr := httptest.NewRecorder()

	handler, err := NewSubscriptionHandler(subSrv, &mockAuthenticator{req: &defaultSubReq}) // Sign validator not used in Create
	if err != nil {
		t.Fatalf("NewSubscriptionHandler failed: %v", err)
	}
	handler.Create(rr, req)

	if rr.Code != wantStatusCode {
		t.Errorf("Create() status code = %v, want %v. Body: %s", rr.Code, wantStatusCode, rr.Body.String())
	}
	contentType := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, wantContentType) {
		t.Errorf("Create() Content-Type header = %q, want prefix %q", contentType, wantContentType)
	}

	var resp model.SubscriptionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	if resp.Status != model.SubscriptionStatusUnderSubscription {
		t.Errorf("Expected status %s, got %s", model.SubscriptionStatusUnderSubscription, resp.Status)
	}
	if resp.MessageID != defaultLRO.OperationID {
		t.Errorf("Expected messageID %s, got %s", defaultLRO.OperationID, resp.MessageID)
	}
}

func TestSubscriptionHandler_Create_Error(t *testing.T) {
	defaultSubReq := model.SubscriptionRequest{
		Subscription: model.Subscription{
			Subscriber: model.Subscriber{
				SubscriberID: "test-subscriber",
				Domain:       "test-domain",
				Type:         model.RoleBAP,
			}},
		MessageID: "test-msg-id",
	}
	defaultSubReqBytes, _ := json.Marshal(defaultSubReq)

	tests := []struct {
		name             string
		requestBody      []byte
		subSrv           subscriptionService
		wantStatusCode   int
		wantContentType  string
		wantBodyContains []string
	}{
		{
			name:             "invalid JSON request body",
			requestBody:      []byte("invalid json"),
			subSrv:           &mockSubscriptionService{},
			wantStatusCode:   http.StatusBadRequest,
			wantContentType:  "application/json",
			wantBodyContains: []string{fmt.Sprintf(`"type":"%s"`, model.ErrorTypeValidationError), fmt.Sprintf(`"code":"%s"`, model.ErrorCodeInvalidJSON), `"message":"Invalid request body:`},
		},
		{
			name:             "service returns ErrOperationAlreadyExists",
			requestBody:      defaultSubReqBytes,
			subSrv:           &mockSubscriptionService{createErr: repository.ErrOperationAlreadyExists},
			wantStatusCode:   http.StatusConflict,
			wantContentType:  "application/json",
			wantBodyContains: []string{fmt.Sprintf(`"type":"%s"`, model.ErrorTypeConflictError), fmt.Sprintf(`"code":"%s"`, model.ErrorCodeDuplicateRequest), `"message":"Duplicate request: An operation with this message_id already exists or is in progress."`},
		},
		{
			name:             "service returns generic error",
			requestBody:      defaultSubReqBytes,
			subSrv:           &mockSubscriptionService{createErr: errors.New("internal service error")},
			wantStatusCode:   http.StatusInternalServerError,
			wantContentType:  "application/json",
			wantBodyContains: []string{fmt.Sprintf(`"type":"%s"`, model.ErrorTypeInternalError), fmt.Sprintf(`"code":"%s"`, model.ErrorCodeInternalServerError), `"message":"Failed to process subscription request."`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBuffer(tt.requestBody))
			rr := httptest.NewRecorder()

			handler, _ := NewSubscriptionHandler(tt.subSrv, &mockAuthenticator{req: &defaultSubReq})
			handler.Create(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("Create() status code = %v, want %v. Body: %s", rr.Code, tt.wantStatusCode, rr.Body.String())
			}
			contentType := rr.Header().Get("Content-Type")
			if !strings.HasPrefix(contentType, tt.wantContentType) {
				t.Errorf("Create() Content-Type header = %q, want prefix %q", contentType, tt.wantContentType)
			}

			bodyStr := rr.Body.String()
			for _, substr := range tt.wantBodyContains {
				if !strings.Contains(bodyStr, substr) {
					t.Errorf("Create() body does not contain %q. Body: %s", substr, bodyStr)
				}
			}
		})
	}
}

func TestSubscriptionHandler_Update_Success(t *testing.T) {
	defaultSubReq := model.SubscriptionRequest{
		Subscription: model.Subscription{
			Subscriber: model.Subscriber{
				SubscriberID: "test.subscriber.com",
				Domain:       "test-domain",
				Type:         model.RoleBAP,
			}},
		MessageID: "update-msg-id",
	}
	defaultSubReqBytes, _ := json.Marshal(defaultSubReq)
	validAuthHeader := `Signature keyId="test.subscriber.com|key1|ed25519",algorithm="ed25519",signature="testsignature"`
	validPublicKey := "base64PublicKey"
	defaultLRO := &model.LRO{OperationID: "update-op-id", Status: "PENDING"}

	subServ := &mockSubscriptionService{
		key: validPublicKey, // For authenticatedReq
		lro: defaultLRO,
	}
	sValidator := &mockAuthenticator{
		req: &defaultSubReq,
	}
	wantStatusCode := http.StatusOK
	wantContentType := "application/json"

	req := httptest.NewRequest(http.MethodPatch, "/subscribe", bytes.NewBuffer(defaultSubReqBytes))
	req.Header.Set("Authorization", validAuthHeader)
	rr := httptest.NewRecorder()

	handler, err := NewSubscriptionHandler(subServ, sValidator)
	if err != nil {
		t.Fatalf("NewSubscriptionHandler failed: %v", err)
	}
	handler.Update(rr, req)

	if rr.Code != wantStatusCode {
		t.Errorf("Update() status code = %v, want %v. Body: %s", rr.Code, wantStatusCode, rr.Body.String())
	}
	contentType := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, wantContentType) {
		t.Errorf("Update() Content-Type header = %q, want prefix %q", contentType, wantContentType)
	}

	var resp model.SubscriptionResponse
	body := rr.Body.Bytes()
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("Failed to unmarshal success response body: %v. Body: %s", err, string(body))
	}
	if resp.Status != model.SubscriptionStatusUnderSubscription {
		t.Errorf("Expected status %s, got %s", model.SubscriptionStatusUnderSubscription, resp.Status)
	}
	if resp.MessageID != defaultLRO.OperationID {
		t.Errorf("Expected messageID %s, got %s", defaultLRO.OperationID, resp.MessageID)
	}
}

func TestSubscriptionHandler_Update_Error(t *testing.T) {
	defaultSubReq := model.SubscriptionRequest{
		Subscription: model.Subscription{
			Subscriber: model.Subscriber{
				SubscriberID: "test.subscriber.com",
				Domain:       "test-domain",
				Type:         model.RoleBAP,
			}},
		MessageID: "update-msg-id",
	}
	mockAuth := &mockAuthenticator{req: &defaultSubReq}
	defaultSubReqBytes, _ := json.Marshal(defaultSubReq)
	validAuthHeader := `Signature keyId="test.subscriber.com|key1|ed25519",algorithm="ed25519",signature="testsignature"`

	tests := []struct {
		name             string
		requestSetup     func(r *http.Request)
		subSrv           subscriptionService
		auth             authenticator
		wantStatusCode   int
		wantContentType  string
		wantBodyContains []string
	}{
		{
			name: "authenticatedReq fails - missing auth header",
			requestSetup: func(r *http.Request) {
				r.Body = io.NopCloser(bytes.NewBuffer(defaultSubReqBytes)) // No Auth header
			},
			auth: &mockAuthenticator{
				err: model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeMissingAuthHeader, "Authorization header missing.", "unknown"),
			},
			wantStatusCode:   http.StatusUnauthorized,
			wantContentType:  "application/json",
			wantBodyContains: []string{fmt.Sprintf(`"type":"%s"`, model.ErrorTypeAuthError), fmt.Sprintf(`"code":"%s"`, model.ErrorCodeMissingAuthHeader), `"message":"Authorization header missing."`},
		},
		{
			name: "failed to read request body", // This error now originates from the handler itself
			requestSetup: func(r *http.Request) {
				r.Header.Set("Authorization", validAuthHeader) // Auth header is present
				r.Body = io.NopCloser(&errorReader{})          // Force body read error
			},
			auth:             mockAuth,
			wantStatusCode:   http.StatusInternalServerError,
			wantContentType:  "application/json",
			wantBodyContains: []string{fmt.Sprintf(`"type":"%s"`, model.ErrorTypeInternalError), fmt.Sprintf(`"code":"%s"`, model.ErrorCodeInternalServerError), `"message":"Failed to read request body."`},
		},

		{
			name: "service returns ErrOperationAlreadyExists after successful auth (mocking auth success)",
			requestSetup: func(r *http.Request) {
				r.Header.Set("Authorization", validAuthHeader)
				r.Body = io.NopCloser(bytes.NewBuffer(defaultSubReqBytes))
			},
			auth:   mockAuth,
			subSrv: &mockSubscriptionService{updateErr: repository.ErrOperationAlreadyExists},
			// Authenticator succeeds and returns the request body
			wantStatusCode:   http.StatusConflict,
			wantContentType:  "application/json",
			wantBodyContains: []string{fmt.Sprintf(`"type":"%s"`, model.ErrorTypeConflictError), fmt.Sprintf(`"code":"%s"`, model.ErrorCodeDuplicateRequest), `"message":"Duplicate request: An operation with this message_id already exists or is in progress for update."`},
		},
		{
			name: "service returns generic error after successful auth (mocking auth success)",
			requestSetup: func(r *http.Request) {
				r.Header.Set("Authorization", validAuthHeader)
				r.Body = io.NopCloser(bytes.NewBuffer(defaultSubReqBytes))
			},
			auth:             mockAuth,
			subSrv:           &mockSubscriptionService{updateErr: errors.New("internal service error during update")},
			wantStatusCode:   http.StatusInternalServerError,
			wantContentType:  "application/json",
			wantBodyContains: []string{fmt.Sprintf(`"type":"%s"`, model.ErrorTypeInternalError), fmt.Sprintf(`"code":"%s"`, model.ErrorCodeInternalServerError), `"message":"Failed to process subscription update request."`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &subscriptionHandler{tt.subSrv, tt.auth}
			req := httptest.NewRequest(http.MethodPatch, "/subscribe", nil)
			tt.requestSetup(req)
			rr := httptest.NewRecorder()

			handler.Update(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("Update() status code = %v, want %v. Body: %s", rr.Code, tt.wantStatusCode, rr.Body.String())
			}
			contentType := rr.Header().Get("Content-Type")
			if !strings.HasPrefix(contentType, tt.wantContentType) {
				t.Errorf("Update() Content-Type header = %q, want prefix %q", contentType, tt.wantContentType)
			}

			bodyStr := rr.Body.String()
			for _, substr := range tt.wantBodyContains {
				if !strings.Contains(bodyStr, substr) {
					t.Errorf("Update() body does not contain %q. Body: %s", substr, bodyStr)
				}
			}
		})
	}
}
