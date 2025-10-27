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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/dpi-accelerator-beckn-onix/internal/event"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"github.com/google/go-cmp/cmp"
)

// failingResponseWriter is a custom http.ResponseWriter that fails on Write,
// used to test JSON encoding error paths.
type failingResponseWriter struct {
	httptest.ResponseRecorder
}

// Write implements the io.Writer interface and always returns an error to
// simulate a failure during response body writing.
func (w *failingResponseWriter) Write(b []byte) (int, error) {
	return 0, errors.New("mock write error")
}

// mockSubscriberService is a mock implementation of subscriberService.
type mockSubscriberService struct {
	createSubOpID   string
	createSubErr    error
	updateSubLroID  string
	updateSubErr    error
	statusToReturn  model.LROStatus
	updateStatusErr error
	onSubscribeResp *model.OnSubscribeResponse
	onSubscribeErr  error
}

func (m *mockSubscriberService) CreateSubscription(ctx context.Context, req *model.NpSubscriptionRequest) (string, error) {
	return m.createSubOpID, m.createSubErr
}

func (m *mockSubscriberService) UpdateSubscription(ctx context.Context, req *model.NpSubscriptionRequest) (string, error) {
	return m.updateSubLroID, m.updateSubErr
}

func (m *mockSubscriberService) UpdateStatus(ctx context.Context, opID string) (model.LROStatus, error) {
	return m.statusToReturn, m.updateStatusErr
}

func (m *mockSubscriberService) OnSubscribe(ctx context.Context, req *model.OnSubscribeRequest) (*model.OnSubscribeResponse, error) {
	return m.onSubscribeResp, m.onSubscribeErr
}

// TestNewSubscriberHandler_Success tests successful creation of SubscriberHandler.
func TestNewSubscriberHandler_Success(t *testing.T) {
	mockSrv := &mockSubscriberService{}
	handler, err := NewSubscriberHandler(mockSrv)
	if err != nil {
		t.Fatalf("NewSubscriberHandler() error = %v, wantErr false", err)
	}
	if handler == nil {
		t.Fatalf("NewSubscriberHandler() expected handler, got nil")
	}
	if handler.srv != mockSrv {
		t.Errorf("NewSubscriberHandler() srv not set correctly")
	}
}

// TestNewSubscriberHandler_Error tests error cases for NewSubscriberHandler.
func TestNewSubscriberHandler_Error(t *testing.T) {
	_, err := NewSubscriberHandler(nil)
	if err == nil {
		t.Fatalf("NewSubscriberHandler(nil) error = nil, wantErr true")
	}
	expectedErrorMsg := "SubscriberService dependency is nil"
	if err.Error() != expectedErrorMsg {
		t.Errorf("NewSubscriberHandler(nil) error = %v, wantErrorMsg %v", err, expectedErrorMsg)
	}
}

// TestWriteSubscriberJSONError tests the writeSubscriberJSONError helper function.
func TestWriteSubscriberJSONError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		errType        model.ErrorType
		errCode        model.ErrorCode
		errMsg         string
		wantHeader     http.Header
		wantBodySubstr []string // Check for substrings in the body
	}{
		{
			name:           "bad request error",
			statusCode:     http.StatusBadRequest,
			errType:        model.ErrorTypeValidationError,
			errCode:        model.ErrorCodeInvalidJSON,
			errMsg:         "Invalid JSON provided",
			wantHeader:     http.Header{"Content-Type": []string{"application/json"}},
			wantBodySubstr: []string{`"type":"VALIDATION_ERROR"`, `"code":"VALIDATION_ERROR_INVALID_JSON"`, `"message":"Invalid JSON provided"`},
		},
		{
			name:           "internal server error",
			statusCode:     http.StatusInternalServerError,
			errType:        model.ErrorTypeInternalError,
			errCode:        model.ErrorCodeInternalServerError,
			errMsg:         "Internal error",
			wantHeader:     http.Header{"Content-Type": []string{"application/json"}},
			wantBodySubstr: []string{`"type":"INTERNAL_ERROR"`, `"code":"INTERNAL_SERVER_ERROR"`, `"message":"Internal error"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeSubscriberJSONError(rr, tt.statusCode, tt.errType, tt.errCode, tt.errMsg)

			if rr.Code != tt.statusCode {
				t.Errorf("writeSubscriberJSONError() status code = %v, want %v", rr.Code, tt.statusCode)
			}

			for key, wantValues := range tt.wantHeader {
				gotValues := rr.Header().Values(key)
				if diff := cmp.Diff(wantValues, gotValues); diff != "" {
					t.Errorf("writeSubscriberJSONError() header %s mismatch (-want +got):\n%s", key, diff)
				}
			}

			bodyStr := rr.Body.String()
			for _, substr := range tt.wantBodySubstr {
				if !strings.Contains(bodyStr, substr) {
					t.Errorf("writeSubscriberJSONError() body does not contain %q. Body: %s", substr, bodyStr)
				}
			}

			var errResp model.ErrorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
				t.Errorf("writeSubscriberJSONError() body is not valid JSON: %v. Body: %s", err, bodyStr)
			}
			if errResp.Error.Type != tt.errType {
				t.Errorf("writeSubscriberJSONError() Error.Type got = %s, want %s", errResp.Error.Type, tt.errType)
			}
			if errResp.Error.Code != tt.errCode {
				t.Errorf("writeSubscriberJSONError() Error.Code got = %s, want %s", errResp.Error.Code, tt.errCode)
			}
			if errResp.Error.Message != tt.errMsg {
				t.Errorf("writeSubscriberJSONError() Error.Message got = %s, want %s", errResp.Error.Message, tt.errMsg)
			}
		})
	}

	t.Run("json encode error", func(t *testing.T) {
		// This test is to ensure the slog.Error is called, but we can't easily inspect logs without a custom logger.
		// We can at least execute the path for coverage.
		fw := &failingResponseWriter{ResponseRecorder: *httptest.NewRecorder()}
		writeSubscriberJSONError(fw, http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeInvalidJSON, "some error")

		// The status code is written before the body write fails.
		if fw.Code != http.StatusBadRequest {
			t.Errorf("writeSubscriberJSONError() with failing writer status code = %v, want %v", fw.Code, http.StatusBadRequest)
		}
	})
}

// TestSubscriberHandler_CreateSubscription_Success tests successful creation.
func TestSubscriberHandler_CreateSubscription_Success(t *testing.T) {
	mockSrv := &mockSubscriberService{createSubOpID: "op-123"}
	handler, _ := NewSubscriberHandler(mockSrv)

	reqBody := &model.NpSubscriptionRequest{
		Subscriber: model.Subscriber{SubscriberID: "test-sub"},
	}
	reqBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBuffer(reqBytes))
	rr := httptest.NewRecorder()

	handler.CreateSubscription(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("CreateSubscription() status code = %v, want %v. Body: %s", rr.Code, http.StatusAccepted, rr.Body.String())
	}

	var gotOpID string
	if err := json.Unmarshal(rr.Body.Bytes(), &gotOpID); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	if gotOpID != "op-123" {
		t.Errorf("CreateSubscription() got operation ID %q, want %q", gotOpID, "op-123")
	}
}

// TestSubscriberHandler_CreateSubscription_Error tests error cases.
func TestSubscriberHandler_CreateSubscription_Error(t *testing.T) {
	tests := []struct {
		name             string
		requestBody      []byte
		mockServiceSetup func(*mockSubscriberService)
		wantStatusCode   int
		wantErrorCode    model.ErrorCode
		wantErrorMessage string
	}{
		{
			name:             "invalid JSON request body",
			requestBody:      []byte("{not-json"),
			mockServiceSetup: func(ms *mockSubscriberService) {},
			wantStatusCode:   http.StatusBadRequest,
			wantErrorCode:    model.ErrorCodeInvalidJSON,
			wantErrorMessage: "Invalid request body",
		},
		{
			name:        "service returns error",
			requestBody: []byte(`{"subscriber_id":"test"}`),
			mockServiceSetup: func(ms *mockSubscriberService) {
				ms.createSubErr = errors.New("service layer error")
			},
			wantStatusCode:   http.StatusBadRequest,
			wantErrorCode:    model.ErrorCodeBadRequest,
			wantErrorMessage: "service layer error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &mockSubscriberService{}
			tt.mockServiceSetup(mockSrv)
			handler, _ := NewSubscriberHandler(mockSrv)

			req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBuffer(tt.requestBody))
			rr := httptest.NewRecorder()

			handler.CreateSubscription(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("CreateSubscription() status code = %v, want %v. Body: %s", rr.Code, tt.wantStatusCode, rr.Body.String())
			}

			var gotErrorResp model.ErrorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &gotErrorResp); err != nil {
				t.Fatalf("Failed to unmarshal error response: %v. Body: %s", err, rr.Body.String())
			}

			if gotErrorResp.Error.Code != tt.wantErrorCode {
				t.Errorf("CreateSubscription() Error.Code = %s, want %s", gotErrorResp.Error.Code, tt.wantErrorCode)
			}
			if !strings.Contains(gotErrorResp.Error.Message, tt.wantErrorMessage) {
				t.Errorf("CreateSubscription() Error.Message = %q, want to contain %q", gotErrorResp.Error.Message, tt.wantErrorMessage)
			}
		})
	}
}

// TestSubscriberHandler_CreateSubscription_EncodeError tests the JSON encoding failure path.
func TestSubscriberHandler_CreateSubscription_EncodeError(t *testing.T) {
	mockSrv := &mockSubscriberService{createSubOpID: "op-123"}
	handler, _ := NewSubscriberHandler(mockSrv)

	reqBody := &model.NpSubscriptionRequest{
		Subscriber: model.Subscriber{SubscriberID: "test-sub"},
	}
	reqBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBuffer(reqBytes))

	// Use a response writer that will fail during encoding to test the error logging path.
	rr := &failingResponseWriter{ResponseRecorder: *httptest.NewRecorder()}

	handler.CreateSubscription(rr, req)

	// The status code should still be set before the body write fails.
	if rr.Code != http.StatusAccepted {
		t.Errorf("CreateSubscription() with failing writer status code = %v, want %v", rr.Code, http.StatusAccepted)
	}
}

// TestSubscriberHandler_UpdateSubscription_EncodeError tests the JSON encoding failure path.
func TestSubscriberHandler_UpdateSubscription_EncodeError(t *testing.T) {
	mockSrv := &mockSubscriberService{updateSubLroID: "op-456"}
	handler, _ := NewSubscriberHandler(mockSrv)

	reqBody := &model.NpSubscriptionRequest{
		Subscriber: model.Subscriber{SubscriberID: "test-sub-update"},
	}
	reqBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, "/subscribe/test-sub-update", bytes.NewBuffer(reqBytes))

	rr := &failingResponseWriter{ResponseRecorder: *httptest.NewRecorder()}

	handler.UpdateSubscription(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("UpdateSubscription() with failing writer status code = %v, want %v", rr.Code, http.StatusAccepted)
	}
}

// TestSubscriberHandler_OnSubscribe_EncodeError tests the JSON encoding failure path.
func TestSubscriberHandler_OnSubscribe_EncodeError(t *testing.T) {
	wantResp := &model.OnSubscribeResponse{Answer: "decrypted-challenge"}
	mockSrv := &mockSubscriberService{onSubscribeResp: wantResp}
	handler, _ := NewSubscriberHandler(mockSrv)

	reqBody := &model.OnSubscribeRequest{
		MessageID: "msg-123",
		Challenge: "encrypted-challenge",
	}
	reqBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/on_subscribe", bytes.NewBuffer(reqBytes))

	rr := &failingResponseWriter{ResponseRecorder: *httptest.NewRecorder()}

	handler.OnSubscribe(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("OnSubscribe() with failing writer status code = %v, want %v", rr.Code, http.StatusOK)
	}
}

// TestSubscriberHandler_UpdateSubscription_Success tests successful update.
func TestSubscriberHandler_UpdateSubscription_Success(t *testing.T) {
	mockSrv := &mockSubscriberService{updateSubLroID: "op-456"}
	handler, _ := NewSubscriberHandler(mockSrv)

	reqBody := &model.NpSubscriptionRequest{
		Subscriber: model.Subscriber{SubscriberID: "test-sub-update"},
	}
	reqBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPatch, "/subscribe/test-sub-update", bytes.NewBuffer(reqBytes))
	rr := httptest.NewRecorder()

	handler.UpdateSubscription(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("UpdateSubscription() status code = %v, want %v. Body: %s", rr.Code, http.StatusAccepted, rr.Body.String())
	}

	var gotLroID string
	if err := json.Unmarshal(rr.Body.Bytes(), &gotLroID); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	if gotLroID != "op-456" {
		t.Errorf("UpdateSubscription() got LRO ID %q, want %q", gotLroID, "op-456")
	}
}

// TestSubscriberHandler_UpdateSubscription_Error tests error cases.
func TestSubscriberHandler_UpdateSubscription_Error(t *testing.T) {
	tests := []struct {
		name             string
		requestBody      []byte
		mockServiceSetup func(*mockSubscriberService)
		wantStatusCode   int
		wantErrorCode    model.ErrorCode
		wantErrorMessage string
	}{
		{
			name:             "invalid JSON request body",
			requestBody:      []byte("{not-json"),
			mockServiceSetup: func(ms *mockSubscriberService) {},
			wantStatusCode:   http.StatusBadRequest,
			wantErrorCode:    model.ErrorCodeInvalidJSON,
			wantErrorMessage: "Invalid request body",
		},
		{
			name:        "service returns error",
			requestBody: []byte(`{"subscriber_id":"test"}`),
			mockServiceSetup: func(ms *mockSubscriberService) {
				ms.updateSubErr = errors.New("update failed")
			},
			wantStatusCode:   http.StatusBadRequest,
			wantErrorCode:    model.ErrorCodeBadRequest,
			wantErrorMessage: "update failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &mockSubscriberService{}
			tt.mockServiceSetup(mockSrv)
			handler, _ := NewSubscriberHandler(mockSrv)

			req := httptest.NewRequest(http.MethodPatch, "/subscribe/some-id", bytes.NewBuffer(tt.requestBody))
			rr := httptest.NewRecorder()

			handler.UpdateSubscription(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("UpdateSubscription() status code = %v, want %v. Body: %s", rr.Code, tt.wantStatusCode, rr.Body.String())
			}

			var gotErrorResp model.ErrorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &gotErrorResp); err != nil {
				t.Fatalf("Failed to unmarshal error response: %v. Body: %s", err, rr.Body.String())
			}

			if gotErrorResp.Error.Code != tt.wantErrorCode {
				t.Errorf("UpdateSubscription() Error.Code = %s, want %s", gotErrorResp.Error.Code, tt.wantErrorCode)
			}
			if !strings.Contains(gotErrorResp.Error.Message, tt.wantErrorMessage) {
				t.Errorf("UpdateSubscription() Error.Message = %q, want to contain %q", gotErrorResp.Error.Message, tt.wantErrorMessage)
			}
		})
	}
}

// TestSubscriberHandler_StatusUpdate_Success tests successful status update.
func TestSubscriberHandler_StatusUpdate_Success(t *testing.T) {
	mockSrv := &mockSubscriberService{statusToReturn: model.LROStatusApproved}
	handler, _ := NewSubscriberHandler(mockSrv)

	reqBody := &event.OnSubscribeRecievedEvent{
		OperationID: "op-789",
	}
	reqBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/statusUpdate", bytes.NewBuffer(reqBytes))
	rr := httptest.NewRecorder()

	handler.StatusUpdate(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("StatusUpdate() status code = %v, want %v. Body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

// TestSubscriberHandler_StatusUpdate_Error tests error cases.
func TestSubscriberHandler_StatusUpdate_Error(t *testing.T) {
	tests := []struct {
		name             string
		requestBody      []byte
		mockServiceSetup func(*mockSubscriberService)
		wantStatusCode   int
		wantErrorCode    model.ErrorCode
		wantErrorMessage string
	}{
		{
			name:             "invalid JSON request body",
			requestBody:      []byte("{not-json"),
			mockServiceSetup: func(ms *mockSubscriberService) {},
			wantStatusCode:   http.StatusBadRequest,
			wantErrorCode:    model.ErrorCodeInvalidJSON,
			wantErrorMessage: "Invalid request body",
		},
		{
			name:        "service returns error",
			requestBody: []byte(`{"operation_id":"op-789"}`),
			mockServiceSetup: func(ms *mockSubscriberService) {
				ms.updateStatusErr = errors.New("status update failed")
			},
			wantStatusCode:   http.StatusBadRequest,
			wantErrorCode:    model.ErrorCodeBadRequest,
			wantErrorMessage: "status update failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &mockSubscriberService{}
			tt.mockServiceSetup(mockSrv)
			handler, _ := NewSubscriberHandler(mockSrv)

			req := httptest.NewRequest(http.MethodPost, "/statusUpdate", bytes.NewBuffer(tt.requestBody))
			rr := httptest.NewRecorder()

			handler.StatusUpdate(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("StatusUpdate() status code = %v, want %v. Body: %s", rr.Code, tt.wantStatusCode, rr.Body.String())
			}

			var gotErrorResp model.ErrorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &gotErrorResp); err != nil {
				t.Fatalf("Failed to unmarshal error response: %v. Body: %s", err, rr.Body.String())
			}

			if gotErrorResp.Error.Code != tt.wantErrorCode {
				t.Errorf("StatusUpdate() Error.Code = %s, want %s", gotErrorResp.Error.Code, tt.wantErrorCode)
			}
			if !strings.Contains(gotErrorResp.Error.Message, tt.wantErrorMessage) {
				t.Errorf("StatusUpdate() Error.Message = %q, want to contain %q", gotErrorResp.Error.Message, tt.wantErrorMessage)
			}
		})
	}
}

// TestSubscriberHandler_OnSubscribe_Success tests successful on_subscribe handling.
func TestSubscriberHandler_OnSubscribe_Success(t *testing.T) {
	wantResp := &model.OnSubscribeResponse{Answer: "decrypted-challenge"}
	mockSrv := &mockSubscriberService{onSubscribeResp: wantResp}
	handler, _ := NewSubscriberHandler(mockSrv)

	reqBody := &model.OnSubscribeRequest{
		MessageID: "msg-123",
		Challenge: "encrypted-challenge",
	}
	reqBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/on_subscribe", bytes.NewBuffer(reqBytes))
	rr := httptest.NewRecorder()

	handler.OnSubscribe(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("OnSubscribe() status code = %v, want %v. Body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var gotResp model.OnSubscribeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &gotResp); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	if diff := cmp.Diff(wantResp, &gotResp); diff != "" {
		t.Errorf("OnSubscribe() response mismatch (-want +got):\n%s", diff)
	}
}

// TestSubscriberHandler_OnSubscribe_Error tests error cases.
func TestSubscriberHandler_OnSubscribe_Error(t *testing.T) {
	tests := []struct {
		name             string
		requestBody      []byte
		mockServiceSetup func(*mockSubscriberService)
		wantStatusCode   int
		wantErrorCode    model.ErrorCode
		wantErrorMessage string
	}{
		{
			name:             "invalid JSON request body",
			requestBody:      []byte("{not-json"),
			mockServiceSetup: func(ms *mockSubscriberService) {},
			wantStatusCode:   http.StatusBadRequest,
			wantErrorCode:    model.ErrorCodeInvalidJSON,
			wantErrorMessage: "Invalid request body",
		},
		{
			name:        "service returns error",
			requestBody: []byte(`{"message_id":"msg-123"}`),
			mockServiceSetup: func(ms *mockSubscriberService) {
				ms.onSubscribeErr = errors.New("on_subscribe processing failed")
			},
			wantStatusCode:   http.StatusInternalServerError,
			wantErrorCode:    model.ErrorCodeInternalServerError,
			wantErrorMessage: "Failed to process on_subscribe: on_subscribe processing failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &mockSubscriberService{}
			tt.mockServiceSetup(mockSrv)
			handler, _ := NewSubscriberHandler(mockSrv)

			req := httptest.NewRequest(http.MethodPost, "/on_subscribe", bytes.NewBuffer(tt.requestBody))
			rr := httptest.NewRecorder()

			handler.OnSubscribe(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("OnSubscribe() status code = %v, want %v. Body: %s", rr.Code, tt.wantStatusCode, rr.Body.String())
			}

			var gotErrorResp model.ErrorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &gotErrorResp); err != nil {
				t.Fatalf("Failed to unmarshal error response: %v. Body: %s", err, rr.Body.String())
			}

			if gotErrorResp.Error.Code != tt.wantErrorCode {
				t.Errorf("OnSubscribe() Error.Code = %s, want %s", gotErrorResp.Error.Code, tt.wantErrorCode)
			}
			if !strings.Contains(gotErrorResp.Error.Message, tt.wantErrorMessage) {
				t.Errorf("OnSubscribe() Error.Message = %q, want to contain %q", gotErrorResp.Error.Message, tt.wantErrorMessage)
			}
		})
	}
}
