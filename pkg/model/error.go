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
	"fmt"
)

// ErrorType defines the category of the error.
type ErrorType string

// Defines the valid BecknErrorType values.
const (
	// ErrorTypeAuthError indicates an error related to authentication or authorization.
	ErrorTypeAuthError ErrorType = "AUTH_ERROR"
	// ErrorTypeValidationError indicates an error due to invalid input data.
	ErrorTypeValidationError ErrorType = "VALIDATION_ERROR"
	// ErrorTypeNotFoundError indicates that a requested resource was not found.
	ErrorTypeNotFoundError ErrorType = "NOT_FOUND"
	// ErrorTypeConflictError indicates a conflict, such as a duplicate request or resource.
	ErrorTypeConflictError ErrorType = "CONFLICT_ERROR" // For duplicate requests
	// ErrorTypeInternalError indicates a general server-side error.
	ErrorTypeInternalError ErrorType = "INTERNAL_ERROR" // For general server errors
)

var validErrorTypes = map[ErrorType]bool{
	ErrorTypeAuthError:       true,
	ErrorTypeValidationError: true,
	ErrorTypeNotFoundError:   true,
	ErrorTypeConflictError:   true,
	ErrorTypeInternalError:   true,
}

// MarshalJSON implements the json.Marshaler interface for ErrorType.
func (et ErrorType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(et))
}

// UnmarshalJSON implements the json.Unmarshaler interface for ErrorType.
func (et *ErrorType) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*et = ErrorType(str)
	if !validErrorTypes[*et] {
		return fmt.Errorf("invalid ErrorType: %s", str)
	}
	return nil
}

// ErrorCode defines the specific error code.
type ErrorCode string

// Defines the valid BecknErrorCode values.
const (
	// Auth Errors
	// ErrorCodeMissingAuthHeader indicates that the required Authorization header is missing from the request.
	ErrorCodeMissingAuthHeader ErrorCode = "AUTH_ERROR_CODE_MISSING_HEADER"
	// ErrorCodeInvalidAuthHeader indicates that the Authorization header is present but malformed or invalid.
	ErrorCodeInvalidAuthHeader ErrorCode = "AUTH_ERROR_CODE_INVALID_HEADER"
	// ErrorCodeIDMismatch indicates a mismatch between an identifier in the auth header and an identifier in the request body.
	ErrorCodeIDMismatch ErrorCode = "AUTH_ERROR_CODE_ID_MISMATCH"
	// ErrorCodeKeyUnavailable indicates that the necessary cryptographic key for an operation (e.g., signature validation) is unavailable.
	ErrorCodeKeyUnavailable ErrorCode = "AUTH_ERROR_CODE_KEY_UNAVAILABLE"
	// ErrorCodeInvalidSignature indicates that the request signature is invalid.
	ErrorCodeInvalidSignature ErrorCode = "AUTH_ERROR_CODE_INVALID_SIGNATURE"
	// Validation Errors
	// ErrorCodeInvalidJSON indicates that the request body contains malformed or invalid JSON.
	ErrorCodeInvalidJSON ErrorCode = "VALIDATION_ERROR_INVALID_JSON"
	// ErrorCodeBadRequest indicates a general validation error with the request.
	ErrorCodeBadRequest ErrorCode = "VALIDATION_ERROR_BAD_REQUEST" // General validation
	// Not Found Errors
	// ErrorCodeSubscriptionNotFound indicates that a specific subscription was not found.
	ErrorCodeSubscriptionNotFound ErrorCode = "SUBSCRIPTION_NOT_FOUND"
	// Not Found Error
	ErrorCodeOperationNotFound ErrorCode = "OPERATION_NOT_FOUND"
	// Conflict Errors
	// ErrorCodeDuplicateRequest indicates that the request is a duplicate of a previous one, often identified by a message ID.
	ErrorCodeDuplicateRequest ErrorCode = "DUPLICATE_REQUEST"
	// Internal Errors
	// ErrorCodeInternalServerError indicates a generic, unexpected error on the server.
	ErrorCodeInternalServerError ErrorCode = "INTERNAL_SERVER_ERROR"

	// ErrorCodeTypeInvalidAction indicates that the action performed is invalid.
	ErrorCodeTypeInvalidAction ErrorCode = "INVALID_ACTION"
)

var validErrorCodes = map[ErrorCode]bool{
	ErrorCodeMissingAuthHeader:    true,
	ErrorCodeInvalidAuthHeader:    true,
	ErrorCodeIDMismatch:           true,
	ErrorCodeKeyUnavailable:       true,
	ErrorCodeInvalidSignature:     true,
	ErrorCodeInvalidJSON:          true,
	ErrorCodeBadRequest:           true,
	ErrorCodeSubscriptionNotFound: true,
	ErrorCodeDuplicateRequest:     true,
	ErrorCodeOperationNotFound:    true,
	ErrorCodeInternalServerError:  true,
	ErrorCodeTypeInvalidAction:    true,
}

// MarshalJSON implements the json.Marshaler interface for ErrorCode.
func (ec ErrorCode) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(ec))
}

// UnmarshalJSON implements the json.Unmarshaler interface for ErrorCode.
func (ec *ErrorCode) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*ec = ErrorCode(str)
	if !validErrorCodes[*ec] {
		return fmt.Errorf("invalid ErrorCode: %s", str)
	}
	return nil
}

// Error represents the standard error structure.
type Error struct {
	Type    ErrorType `json:"type,omitempty"`
	Code    ErrorCode `json:"code"`
	Path    string    `json:"path,omitempty"`
	Message string    `json:"message"`
}

// ErrorResponse wraps the Error.
type ErrorResponse struct {
	Error Error `json:"error"`
}

// AuthError represents a structured authentication or authorization error,
// typically returned by authentication services for handlers to process.
type AuthError struct {
	StatusCode   int
	ErrorType    ErrorType
	ErrorCode    ErrorCode
	Message      string
	SubscriberID string // Can be used for realm in WWW-Authenticate or specific error details
}

// Error makes AuthError satisfy the error interface.
func (e *AuthError) Error() string {
	return fmt.Sprintf("AuthError (HTTP %d): Type=%s, Code=%s, Message=%s, SubscriberID=%s", e.StatusCode, e.ErrorType, e.ErrorCode, e.Message, e.SubscriberID)
}

// NewAuthError is a helper to create AuthError instances.
func NewAuthError(statusCode int, errType ErrorType, errCode ErrorCode, errMsg string, subscriberID string) *AuthError {
	return &AuthError{
		StatusCode:   statusCode,
		ErrorType:    errType,
		ErrorCode:    errCode,
		Message:      errMsg,
		SubscriberID: subscriberID,
	}
}
