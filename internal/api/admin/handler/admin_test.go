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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"
	"github.com/google/dpi-accelerator-beckn-onix/internal/service"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"github.com/google/go-cmp/cmp"
)

// mockAdminService is a mock implementation of adminService.
type mockAdminService struct {
	lro *model.LRO
	err error
}

func (m *mockAdminService) ApproveSubscription(ctx context.Context, req *model.OperationActionRequest) (*model.Subscription, *model.LRO, error) {
	return nil, m.lro, m.err
}

func (m *mockAdminService) RejectSubscription(ctx context.Context, req *model.OperationActionRequest) (*model.LRO, error) {
	return m.lro, m.err
}

// TestNewAdminHandler_Success tests successful creation of AdminHandler.
func TestNewAdminHandler_Success(t *testing.T) {
	mockSrv := &mockAdminService{}
	handler, err := NewAdminHandler(mockSrv)
	if err != nil {
		t.Fatalf("NewAdminHandler() error = %v, wantErr false", err)
	}
	if handler == nil {
		t.Fatalf("NewAdminHandler() expected handler, got nil")
	}
	if handler.srv != mockSrv {
		t.Errorf("NewAdminHandler() srv not set correctly")
	}
}

// TestNewAdminHandler_Error tests error cases for NewAdminHandler.
func TestNewAdminHandler_Error(t *testing.T) {
	_, err := NewAdminHandler(nil)
	if err == nil {
		t.Fatalf("NewAdminHandler(nil) error = nil, wantErr true")
	}
	expectedErrorMsg := "AdminLROService dependency is nil"
	if err.Error() != expectedErrorMsg {
		t.Errorf("NewAdminHandler(nil) error = %v, wantErrorMsg %v", err, expectedErrorMsg)
	}
}

// TestWriteAdminJSONError tests the writeAdminJSONError helper function.
func TestWriteAdminJSONError(t *testing.T) {
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
			name:           "not found error",
			statusCode:     http.StatusNotFound,
			errType:        model.ErrorTypeNotFoundError,
			errCode:        model.ErrorCodeOperationNotFound,
			errMsg:         "Operation not found",
			wantHeader:     http.Header{"Content-Type": []string{"application/json"}},
			wantBodySubstr: []string{`"type":"NOT_FOUND"`, `"code":"OPERATION_NOT_FOUND"`, `"message":"Operation not found"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeAdminJSONError(rr, tt.statusCode, tt.errType, tt.errCode, tt.errMsg)

			if rr.Code != tt.statusCode {
				t.Errorf("writeAdminJSONError() status code = %v, want %v", rr.Code, tt.statusCode)
			}

			for key, wantValues := range tt.wantHeader {
				gotValues := rr.Header().Values(key)
				if diff := cmp.Diff(wantValues, gotValues); diff != "" {
					t.Errorf("writeAdminJSONError() header %s mismatch (-want +got):\n%s", key, diff)
				}
			}

			bodyStr := rr.Body.String()
			for _, substr := range tt.wantBodySubstr {
				if !strings.Contains(bodyStr, substr) {
					t.Errorf("writeAdminJSONError() body does not contain %q. Body: %s", substr, bodyStr)
				}
			}

			// Verify it's valid JSON
			var errResp model.ErrorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
				t.Errorf("writeAdminJSONError() body is not valid JSON: %v. Body: %s", err, bodyStr)
			}
			if errResp.Error.Type != tt.errType {
				t.Errorf("writeAdminJSONError() Error.Type got = %s, want %s", errResp.Error.Type, tt.errType)
			}
			if errResp.Error.Code != tt.errCode {
				t.Errorf("writeAdminJSONError() Error.Code got = %s, want %s", errResp.Error.Code, tt.errCode)
			}
			if errResp.Error.Message != tt.errMsg {
				t.Errorf("writeAdminJSONError() Error.Message got = %s, want %s", errResp.Error.Message, tt.errMsg)
			}
		})
	}
}

// TestAdminHandler_HandleSubscriptionAction_Success tests successful actions.
func TestAdminHandler_HandleSubscriptionAction_Success(t *testing.T) {
	operationID := "test-op-123"
	approvedLRO := &model.LRO{OperationID: operationID, Status: model.LROStatusApproved, Type: model.OperationTypeCreateSubscription}
	rejectedLRO := &model.LRO{OperationID: operationID, Status: model.LROStatusRejected, Type: model.OperationTypeCreateSubscription, ErrorDataJSON: []byte(`{"reason":"admin rejected"}`)}

	tests := []struct {
		name             string
		actionRequest    model.OperationActionRequest
		mockServiceSetup func(*mockAdminService)
		wantStatusCode   int
		wantLROBody      *model.LRO
	}{
		{
			name: "approve subscription success",
			actionRequest: model.OperationActionRequest{
				OperationID: operationID,
				Action:      model.OperationActionApproveSubscription,
			},
			mockServiceSetup: func(ms *mockAdminService) {
				ms.lro = approvedLRO
			},
			wantStatusCode: http.StatusOK,
			wantLROBody:    approvedLRO,
		},
		{
			name: "reject subscription success",
			actionRequest: model.OperationActionRequest{
				OperationID: operationID,
				Action:      model.OperationActionRejectSubscription,
				Reason:      "Admin decision",
			},
			mockServiceSetup: func(ms *mockAdminService) {
				ms.lro = rejectedLRO
			},
			wantStatusCode: http.StatusOK,
			wantLROBody:    rejectedLRO,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &mockAdminService{}
			tt.mockServiceSetup(mockSrv)

			handler, _ := NewAdminHandler(mockSrv)

			actionReqBytes, _ := json.Marshal(tt.actionRequest)
			req := httptest.NewRequest(http.MethodPost, "/subscriptions/"+operationID+"/action", bytes.NewBuffer(actionReqBytes))
			rr := httptest.NewRecorder()
			handler.HandleSubscriptionAction(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("HandleSubscriptionAction() status code = %v, want %v. Body: %s", rr.Code, tt.wantStatusCode, rr.Body.String())
			}

			if tt.wantLROBody != nil {
				var gotLRO model.LRO
				if err := json.Unmarshal(rr.Body.Bytes(), &gotLRO); err != nil {
					t.Fatalf("Failed to unmarshal response LRO: %v. Body: %s", err, rr.Body.String())
				}
				if diff := cmp.Diff(tt.wantLROBody, &gotLRO); diff != "" {
					t.Errorf("HandleSubscriptionAction() LRO response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// TestAdminHandler_HandleSubscriptionAction_Error tests error handling.
func TestAdminHandler_HandleSubscriptionAction_Error(t *testing.T) {
	operationID := "test-op-err"

	tests := []struct {
		name             string
		requestBody      []byte
		mockServiceSetup func(*mockAdminService) // Setup for service-level errors
		wantStatusCode   int
		wantErrorType    model.ErrorType
		wantErrorCode    model.ErrorCode
		wantErrorMessage string
	}{
		{
			name:             "invalid JSON request body",
			requestBody:      []byte("this is not json"),
			mockServiceSetup: func(ms *mockAdminService) {},
			wantStatusCode:   http.StatusBadRequest,
			wantErrorType:    model.ErrorTypeValidationError,
			wantErrorCode:    model.ErrorCodeInvalidJSON,
			wantErrorMessage: "Invalid request body: invalid character 'h' in literal true (expecting 'r')", // Error message can be specific to JSON parser
		},
		{
			name: "missing reason for reject action",
			requestBody: func() []byte {
				ar := model.OperationActionRequest{OperationID: operationID, Action: model.OperationActionRejectSubscription}
				b, _ := json.Marshal(ar)
				return b
			}(),
			mockServiceSetup: func(ms *mockAdminService) {},
			wantStatusCode:   http.StatusBadRequest,
			wantErrorType:    model.ErrorTypeValidationError,
			wantErrorCode:    model.ErrorCodeTypeInvalidAction,
			wantErrorMessage: "Reason is required for REJECT action.",
		},
		{
			name: "invalid action specified",
			requestBody: func() []byte {
				ar := model.OperationActionRequest{OperationID: operationID, Action: "INVALID_ACTION_TYPE"}
				b, _ := json.Marshal(ar)
				return b
			}(),
			mockServiceSetup: func(ms *mockAdminService) {},
			wantStatusCode:   http.StatusBadRequest,
			wantErrorType:    model.ErrorTypeValidationError,
			wantErrorCode:    model.ErrorCodeTypeInvalidAction,
			wantErrorMessage: "Invalid action specified. Must be 'APPROVE_SUBSCRIPTION' or 'REJECT_SUBSCRIPTION'.",
		},
		{
			name: "service returns ErrOperationNotFound on approve",
			requestBody: func() []byte {
				ar := model.OperationActionRequest{OperationID: operationID, Action: model.OperationActionApproveSubscription}
				b, _ := json.Marshal(ar)
				return b
			}(),
			mockServiceSetup: func(ms *mockAdminService) {
				ms.err = repository.ErrOperationNotFound
			},
			wantStatusCode:   http.StatusNotFound,
			wantErrorType:    model.ErrorTypeNotFoundError,
			wantErrorCode:    model.ErrorCodeOperationNotFound,
			wantErrorMessage: fmt.Sprintf("Operation with id %s not found.", operationID),
		},
		{
			name: "service returns ErrOperationNotFound on reject",
			requestBody: func() []byte {
				ar := model.OperationActionRequest{OperationID: operationID, Action: model.OperationActionRejectSubscription, Reason: "test"}
				b, _ := json.Marshal(ar)
				return b
			}(),
			mockServiceSetup: func(ms *mockAdminService) {
				ms.err = repository.ErrOperationNotFound
			},
			wantStatusCode:   http.StatusNotFound,
			wantErrorType:    model.ErrorTypeNotFoundError,
			wantErrorCode:    model.ErrorCodeOperationNotFound,
			wantErrorMessage: fmt.Sprintf("Operation with id %s not found.", operationID),
		},
		{
			name: "service returns ErrLROAlreadyProcessed on approve",
			requestBody: func() []byte {
				ar := model.OperationActionRequest{OperationID: operationID, Action: model.OperationActionApproveSubscription}
				b, _ := json.Marshal(ar)
				return b
			}(),
			mockServiceSetup: func(ms *mockAdminService) {
				ms.err = service.ErrLROAlreadyProcessed
			},
			wantStatusCode:   http.StatusConflict,
			wantErrorType:    model.ErrorTypeConflictError,
			wantErrorCode:    model.ErrorCodeDuplicateRequest,
			wantErrorMessage: fmt.Sprintf("Operation %s has already been processed.", operationID),
		},
		{
			name: "service returns generic error on approve",
			requestBody: func() []byte {
				ar := model.OperationActionRequest{OperationID: operationID, Action: model.OperationActionApproveSubscription}
				b, _ := json.Marshal(ar)
				return b
			}(),
			mockServiceSetup: func(ms *mockAdminService) {
				ms.err = errors.New("internal service failure")
			},
			wantStatusCode:   http.StatusInternalServerError,
			wantErrorType:    model.ErrorTypeInternalError,
			wantErrorCode:    model.ErrorCodeInternalServerError,
			wantErrorMessage: "Failed to process subscription action due to an internal error.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSrv := &mockAdminService{}
			tt.mockServiceSetup(mockSrv)

			handler, _ := NewAdminHandler(mockSrv)

			req := httptest.NewRequest(http.MethodPost, "/subscriptions/"+operationID+"/action", bytes.NewBuffer(tt.requestBody))
			rr := httptest.NewRecorder()
			handler.HandleSubscriptionAction(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("HandleSubscriptionAction() status code = %v, want %v. Body: %s", rr.Code, tt.wantStatusCode, rr.Body.String())
			}

			var gotErrorResp model.ErrorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &gotErrorResp); err != nil {
				// For the "invalid JSON request body" case, the body might not be a valid ErrorResponse JSON.
				// We still check the status code and can check for a substring in the message if needed.
				if tt.wantErrorCode == model.ErrorCodeInvalidJSON {
					if !strings.Contains(rr.Body.String(), tt.wantErrorMessage) {
						t.Errorf("HandleSubscriptionAction() body for InvalidJSON did not contain %q. Body: %s", tt.wantErrorMessage, rr.Body.String())
					}
					return // Skip further JSON parsing checks for this specific case
				}
				t.Fatalf("Failed to unmarshal error response: %v. Body: %s", err, rr.Body.String())
			}

			if gotErrorResp.Error.Type != tt.wantErrorType {
				t.Errorf("HandleSubscriptionAction() Error.Type = %s, want %s", gotErrorResp.Error.Type, tt.wantErrorType)
			}
			if gotErrorResp.Error.Code != tt.wantErrorCode {
				t.Errorf("HandleSubscriptionAction() Error.Code = %s, want %s", gotErrorResp.Error.Code, tt.wantErrorCode)
			}
			if gotErrorResp.Error.Message != tt.wantErrorMessage {
				t.Errorf("HandleSubscriptionAction() Error.Message = %q, want %q", gotErrorResp.Error.Message, tt.wantErrorMessage)
			}
		})
	}
}
