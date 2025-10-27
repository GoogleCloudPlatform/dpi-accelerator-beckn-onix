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

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// mockGatewayAuthValidator is a mock implementation of gatewayAuthValidator.
type mockGatewayAuthValidator struct {
	validateErr *model.AuthError
}

func (m *mockGatewayAuthValidator) Validate(ctx context.Context, body []byte, authHeader string) *model.AuthError {
	return m.validateErr
}

// mockTaskQueuer is a mock implementation of taskQueuer.
type mockTaskQueuer struct {
	queueTxnTask *model.AsyncTask
	queueTxnErr  error
}

func (m *mockTaskQueuer) QueueTxn(ctx context.Context, reqCtx *model.Context, msg []byte, h http.Header) (*model.AsyncTask, error) {
	return m.queueTxnTask, m.queueTxnErr
}

// failingReader is an io.Reader that always returns an error.
type failingReader struct{}

func (f *failingReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("mock read error")
}

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

// TestNewGatewayHandler_Success tests successful creation of GatewayHandler.
func TestNewGatewayHandler_Success(t *testing.T) {
	mockAuth := &mockGatewayAuthValidator{}
	mockQueuer := &mockTaskQueuer{}

	handler, err := NewGatewayHandler(mockAuth, mockQueuer)
	if err != nil {
		t.Fatalf("NewGatewayHandler() error = %v, wantErr nil", err)
	}
	if handler == nil {
		t.Fatalf("NewGatewayHandler() expected handler, got nil")
	}
	if handler.authValidator != mockAuth {
		t.Errorf("NewGatewayHandler() authValidator not set correctly")
	}
	if handler.taskQueuer != mockQueuer {
		t.Errorf("NewGatewayHandler() taskQueuer not set correctly")
	}
}

// TestNewGatewayHandler_Error tests error cases for NewGatewayHandler.
func TestNewGatewayHandler_Error(t *testing.T) {
	tests := []struct {
		name       string
		auth       gatewayAuthValidator
		queuer     taskQueuer
		wantErrMsg string
	}{
		{
			name:       "nil authValidator",
			auth:       nil,
			queuer:     &mockTaskQueuer{},
			wantErrMsg: "authValidator dependency is nil",
		},
		{
			name:       "nil taskQueuer",
			auth:       &mockGatewayAuthValidator{},
			queuer:     nil,
			wantErrMsg: "taskQueuer dependency is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGatewayHandler(tt.auth, tt.queuer)
			if err == nil || err.Error() != tt.wantErrMsg {
				t.Errorf("NewGatewayHandler() error = %v, wantErrorMsg %q", err, tt.wantErrMsg)
			}
		})
	}
}

// TestWriteGatewayError tests the writeGatewayError helper function.
func TestWriteGatewayError(t *testing.T) {
	rr := httptest.NewRecorder()
	writeGatewayError(rr, http.StatusBadRequest, string(model.ErrorCodeBadRequest), "Test error message.")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("writeGatewayError() status code = %v, want %v", rr.Code, http.StatusBadRequest)
	}
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("writeGatewayError() Content-Type = %q, want %q", rr.Header().Get("Content-Type"), "application/json")
	}

	var errResp model.TxnResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("writeGatewayError() body is not valid JSON: %v. Body: %s", err, rr.Body.String())
	}

	if errResp.Message.Ack.Status != model.StatusNACK {
		t.Errorf("writeGatewayError() Ack Status = %q, want %q", errResp.Message.Ack.Status, model.StatusNACK)
	}
	if errResp.Message.Error == nil {
		t.Fatal("writeGatewayError() Error field is nil, want non-nil")
	}
	if errResp.Message.Error.Code != model.ErrorCodeBadRequest {
		t.Errorf("writeGatewayError() Error Code = %q, want %q", errResp.Message.Error.Code, model.ErrorCodeBadRequest)
	}
	if errResp.Message.Error.Message != "Test error message." {
		t.Errorf("writeGatewayError() Error Message = %q, want %q", errResp.Message.Error.Message, "Test error message.")
	}
}

// TestServeHttp_Success tests a successful request flow.
func TestServeHttp_Success(t *testing.T) {
	mockAuth := &mockGatewayAuthValidator{} // No error
	mockQueuer := &mockTaskQueuer{
		queueTxnTask: &model.AsyncTask{Type: model.AsyncTaskTypeProxy},
	}
	handler, _ := NewGatewayHandler(mockAuth, mockQueuer)

	reqBody := `{"context":{"action":"search"},"message":{}}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(reqBody))
	req.Header.Set(model.AuthHeaderSubscriber, "test-auth-header")
	rr := httptest.NewRecorder()

	handler.ServeHttp(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ServeHttp() status code = %v, want %v. Body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp model.TxnResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}

	if resp.Message.Ack.Status != model.StatusACK {
		t.Errorf("Response Ack Status = %q, want %q", resp.Message.Ack.Status, model.StatusACK)
	}
	if resp.Message.Error != nil {
		t.Errorf("Response Error is not nil: %+v", resp.Message.Error)
	}
}

// TestServeHttp_ReadBodyError tests when reading the request body fails.
func TestServeHttp_ReadBodyError(t *testing.T) {
	mockAuth := &mockGatewayAuthValidator{}
	mockQueuer := &mockTaskQueuer{}
	handler, _ := NewGatewayHandler(mockAuth, mockQueuer)

	req := httptest.NewRequest(http.MethodPost, "/test", &failingReader{})
	rr := httptest.NewRecorder()

	handler.ServeHttp(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ServeHttp() status code = %v, want %v", rr.Code, http.StatusInternalServerError)
	}
	var errResp model.TxnResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &errResp)

	if errResp.Message.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("Error Code = %q, want %q", errResp.Message.Error.Code, "INTERNAL_ERROR")
	}
}

// TestServeHttp_AuthValidationError tests when authentication fails.
func TestServeHttp_AuthValidationError(t *testing.T) {
	authErr := model.NewAuthError(http.StatusUnauthorized, model.ErrorTypeAuthError, model.ErrorCodeInvalidSignature, "Invalid signature.", "test-sub")
	mockAuth := &mockGatewayAuthValidator{validateErr: authErr}
	mockQueuer := &mockTaskQueuer{}
	handler, _ := NewGatewayHandler(mockAuth, mockQueuer)

	reqBody := `{"context":{"action":"search"},"message":{}}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(reqBody))
	req.Header.Set(model.AuthHeaderSubscriber, "invalid-auth-header")
	rr := httptest.NewRecorder()

	handler.ServeHttp(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("ServeHttp() status code = %v, want %v", rr.Code, http.StatusUnauthorized)
	}
	var errResp model.TxnResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to unmarshal error response body: %v. Body: %s", err, rr.Body.String())
	}
	if errResp.Message.Error.Code != model.ErrorCodeInvalidSignature {
		t.Errorf("Error Code = %q, want %q", errResp.Message.Error.Code, model.ErrorCodeInvalidSignature)
	}
	if !strings.Contains(errResp.Message.Error.Message, "Invalid signature.") {
		t.Errorf("Error Message = %q, want to contain %q", errResp.Message.Error.Message, "Invalid signature.")
	}
}

// TestServeHttp_UnmarshalBodyError tests when unmarshalling the request body fails.
func TestServeHttp_UnmarshalBodyError(t *testing.T) {
	mockAuth := &mockGatewayAuthValidator{}
	mockQueuer := &mockTaskQueuer{}
	handler, _ := NewGatewayHandler(mockAuth, mockQueuer)

	reqBody := `{"context":{"action":"search"},"message":` // Invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(reqBody))
	req.Header.Set(model.AuthHeaderSubscriber, "test-auth-header")
	rr := httptest.NewRecorder()

	handler.ServeHttp(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("ServeHttp() status code = %v, want %v", rr.Code, http.StatusBadRequest)
	}
	var errResp model.TxnResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &errResp)
	if errResp.Message.Error.Code != "INVALID_JSON" {
		t.Errorf("Error Code = %q, want %q", errResp.Message.Error.Code, "INVALID_JSON")
	}
}

// TestServeHttp_QueueTaskError tests when queuing a task fails.
func TestServeHttp_QueueTaskError(t *testing.T) {
	mockAuth := &mockGatewayAuthValidator{}
	mockQueuer := &mockTaskQueuer{
		queueTxnErr: errors.New("queue is full"),
	}
	handler, _ := NewGatewayHandler(mockAuth, mockQueuer)

	reqBody := `{"context":{"action":"search"},"message":{}}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(reqBody))
	req.Header.Set(model.AuthHeaderSubscriber, "test-auth-header")
	rr := httptest.NewRecorder()

	handler.ServeHttp(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ServeHttp() status code = %v, want %v", rr.Code, http.StatusInternalServerError)
	}
	var errResp model.TxnResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &errResp)
	if errResp.Message.Error.Code != "QUEUEING_FAILED" {
		t.Errorf("Error Code = %q, want %q", errResp.Message.Error.Code, "QUEUEING_FAILED")
	}
}

// TestServeHttp_EncodeResponseError tests when encoding the successful response fails.
func TestServeHttp_EncodeResponseError(t *testing.T) {
	mockAuth := &mockGatewayAuthValidator{}
	mockQueuer := &mockTaskQueuer{
		queueTxnTask: &model.AsyncTask{Type: model.AsyncTaskTypeProxy},
	}
	handler, _ := NewGatewayHandler(mockAuth, mockQueuer)

	reqBody := `{"context":{"action":"search"},"message":{}}`
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(reqBody))
	req.Header.Set(model.AuthHeaderSubscriber, "test-auth-header")

	// Use a response writer that will fail during encoding
	rr := &failingResponseWriter{ResponseRecorder: *httptest.NewRecorder()}

	handler.ServeHttp(rr, req)

	// The status code should still be set before the body write fails.
	if rr.Code != http.StatusOK {
		t.Errorf("ServeHttp() with failing writer status code = %v, want %v", rr.Code, http.StatusOK)
	}
	// No body can be asserted as the write failed. The error would be logged.
}
