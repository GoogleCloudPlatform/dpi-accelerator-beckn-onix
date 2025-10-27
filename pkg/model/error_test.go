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

package model

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestErrorType_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		et       ErrorType
		expected string
		wantErr  bool
	}{
		{"AuthError", ErrorTypeAuthError, `"AUTH_ERROR"`, false},
		{"ValidationError", ErrorTypeValidationError, `"VALIDATION_ERROR"`, false},
		{"NotFoundError", ErrorTypeNotFoundError, `"NOT_FOUND"`, false},
		{"ConflictError", ErrorTypeConflictError, `"CONFLICT_ERROR"`, false},
		{"InternalError", ErrorTypeInternalError, `"INTERNAL_ERROR"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.et)
			if (err != nil) != tt.wantErr {
				t.Errorf("ErrorType.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.expected {
				t.Errorf("ErrorType.MarshalJSON() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestErrorType_UnmarshalJSON_Success(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected ErrorType
	}{
		{"AuthError", `"AUTH_ERROR"`, ErrorTypeAuthError},
		{"ValidationError", `"VALIDATION_ERROR"`, ErrorTypeValidationError},
		{"NotFoundError", `"NOT_FOUND"`, ErrorTypeNotFoundError},
		{"ConflictError", `"CONFLICT_ERROR"`, ErrorTypeConflictError},
		{"InternalError", `"INTERNAL_ERROR"`, ErrorTypeInternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var et ErrorType
			if err := json.Unmarshal([]byte(tt.jsonData), &et); err != nil {
				t.Fatalf("UnmarshalJSON(%q) error = %v, wantErr nil", tt.jsonData, err)
			}
			if et != tt.expected {
				t.Errorf("UnmarshalJSON(%q) got = %v, want %v", tt.jsonData, et, tt.expected)
			}
		})
	}
}

func TestErrorType_UnmarshalJSON_Error(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		expectedError string
	}{
		{"InvalidErrorType", `"INVALID_TYPE"`, "invalid ErrorType: INVALID_TYPE"},
		{"NonString", `123`, "json: cannot unmarshal number into Go value of type string"},
		{"MalformedJSON", `"UNCLOSED`, "unexpected end of JSON input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var et ErrorType
			err := json.Unmarshal([]byte(tt.jsonData), &et)
			if err == nil {
				t.Fatalf("UnmarshalJSON(%q) error = nil, want error containing %q", tt.jsonData, tt.expectedError)
			}
			if err.Error() != tt.expectedError {
				t.Errorf("UnmarshalJSON(%q) error = %q, want error %q", tt.jsonData, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestErrorCode_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		ec       ErrorCode
		expected string
		wantErr  bool
	}{
		{"MissingAuthHeader", ErrorCodeMissingAuthHeader, `"AUTH_ERROR_CODE_MISSING_HEADER"`, false},
		{"InvalidJSON", ErrorCodeInvalidJSON, `"VALIDATION_ERROR_INVALID_JSON"`, false},
		{"SubscriptionNotFound", ErrorCodeSubscriptionNotFound, `"SUBSCRIPTION_NOT_FOUND"`, false},
		{"DuplicateRequest", ErrorCodeDuplicateRequest, `"DUPLICATE_REQUEST"`, false},
		{"InternalServerError", ErrorCodeInternalServerError, `"INTERNAL_SERVER_ERROR"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.ec)
			if (err != nil) != tt.wantErr {
				t.Errorf("ErrorCode.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.expected {
				t.Errorf("ErrorCode.MarshalJSON() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestErrorCode_UnmarshalJSON_Success(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected ErrorCode
	}{
		{"MissingAuthHeader", `"AUTH_ERROR_CODE_MISSING_HEADER"`, ErrorCodeMissingAuthHeader},
		{"InvalidJSON", `"VALIDATION_ERROR_INVALID_JSON"`, ErrorCodeInvalidJSON},
		{"SubscriptionNotFound", `"SUBSCRIPTION_NOT_FOUND"`, ErrorCodeSubscriptionNotFound},
		{"DuplicateRequest", `"DUPLICATE_REQUEST"`, ErrorCodeDuplicateRequest},
		{"InternalServerError", `"INTERNAL_SERVER_ERROR"`, ErrorCodeInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ec ErrorCode
			if err := json.Unmarshal([]byte(tt.jsonData), &ec); err != nil {
				t.Fatalf("UnmarshalJSON(%q) error = %v, wantErr nil", tt.jsonData, err)
			}
			if ec != tt.expected {
				t.Errorf("UnmarshalJSON(%q) got = %v, want %v", tt.jsonData, ec, tt.expected)
			}
		})
	}
}

func TestErrorCode_UnmarshalJSON_Error(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		expectedError string
	}{
		{"InvalidErrorCode", `"INVALID_CODE"`, "invalid ErrorCode: INVALID_CODE"},
		{"NonString", `true`, "json: cannot unmarshal bool into Go value of type string"},
		{"MalformedJSON", `"UNCLOSED`, "unexpected end of JSON input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ec ErrorCode
			err := json.Unmarshal([]byte(tt.jsonData), &ec)
			if err == nil {
				t.Fatalf("UnmarshalJSON(%q) error = nil, want error containing %q", tt.jsonData, tt.expectedError)
			}
			if err.Error() != tt.expectedError {
				t.Errorf("UnmarshalJSON(%q) error = %q, want error %q", tt.jsonData, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestAuthError_Error(t *testing.T) {
	authErr := &AuthError{
		StatusCode:   http.StatusUnauthorized,
		ErrorType:    ErrorTypeAuthError,
		ErrorCode:    ErrorCodeInvalidSignature,
		Message:      "Signature is invalid",
		SubscriberID: "test-subscriber",
	}
	expected := "AuthError (HTTP 401): Type=AUTH_ERROR, Code=AUTH_ERROR_CODE_INVALID_SIGNATURE, Message=Signature is invalid, SubscriberID=test-subscriber"
	if got := authErr.Error(); got != expected {
		t.Errorf("AuthError.Error() = %q, want %q", got, expected)
	}
}

func TestNewAuthError(t *testing.T) {
	statusCode := http.StatusForbidden
	errType := ErrorTypeAuthError
	errCode := ErrorCodeIDMismatch
	errMsg := "ID mismatch"
	subscriberID := "test-sub"

	authErr := NewAuthError(statusCode, errType, errCode, errMsg, subscriberID)

	if authErr.StatusCode != statusCode {
		t.Errorf("NewAuthError() StatusCode = %d, want %d", authErr.StatusCode, statusCode)
	}
	if authErr.ErrorType != errType {
		t.Errorf("NewAuthError() ErrorType = %s, want %s", authErr.ErrorType, errType)
	}
	if authErr.ErrorCode != errCode {
		t.Errorf("NewAuthError() ErrorCode = %s, want %s", authErr.ErrorCode, errCode)
	}
	if authErr.Message != errMsg {
		t.Errorf("NewAuthError() Message = %q, want %q", authErr.Message, errMsg)
	}
	if authErr.SubscriberID != subscriberID {
		t.Errorf("NewAuthError() SubscriberID = %q, want %q", authErr.SubscriberID, subscriberID)
	}
}
