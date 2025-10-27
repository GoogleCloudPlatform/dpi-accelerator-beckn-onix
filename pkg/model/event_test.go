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
	"testing"
)

func TestEventType_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		et       EventType
		expected string
	}{
		{"NewSubscriptionRequest", EventTypeNewSubscriptionRequest, `"NEW_SUBSCRIPTION_REQUEST"`},
		{"UpdateSubscriptionRequest", EventTypeUpdateSubscriptionRequest, `"UPDATE_SUBSCRIPTION_REQUEST"`},
		{"SubscriptionRequestApproved", EventTypeSubscriptionRequestApproved, `"SUBSCRIPTION_REQUEST_APPROVED"`},
		{"SubscriptionRequestRejected", EventTypeSubscriptionRequestRejected, `"SUBSCRIPTION_REQUEST_REJECTED"`},
		{"OnSubscribeRecieved", EventTypeOnSubscribeRecieved, `"ON_SUBSCRIBE_RECIEVED"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.et)
			if err != nil {
				t.Fatalf("EventType.MarshalJSON() error = %v, wantErr nil", err)
			}
			if string(got) != tt.expected {
				t.Errorf("EventType.MarshalJSON() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestEventType_UnmarshalJSON_Success(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected EventType
	}{
		{"NewSubscriptionRequest", `"NEW_SUBSCRIPTION_REQUEST"`, EventTypeNewSubscriptionRequest},
		{"UpdateSubscriptionRequest", `"UPDATE_SUBSCRIPTION_REQUEST"`, EventTypeUpdateSubscriptionRequest},
		{"SubscriptionRequestApproved", `"SUBSCRIPTION_REQUEST_APPROVED"`, EventTypeSubscriptionRequestApproved},
		{"SubscriptionRequestRejected", `"SUBSCRIPTION_REQUEST_REJECTED"`, EventTypeSubscriptionRequestRejected},
		{"OnSubscribeRecieved", `"ON_SUBSCRIBE_RECIEVED"`, EventTypeOnSubscribeRecieved},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var et EventType
			if err := json.Unmarshal([]byte(tt.jsonData), &et); err != nil {
				t.Fatalf("UnmarshalJSON(%q) error = %v, wantErr nil", tt.jsonData, err)
			}
			if et != tt.expected {
				t.Errorf("UnmarshalJSON(%q) got = %v, want %v", tt.jsonData, et, tt.expected)
			}
		})
	}
}

func TestEventType_UnmarshalJSON_InvalidValue(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		expectedError string // This should match the error message from the UnmarshalJSON method
	}{
		{
			name:          "NonString",
			jsonData:      `123`,
			expectedError: "json: cannot unmarshal number into Go value of type string",
		},
		{
			name:          "MalformedJSON",
			jsonData:      `"UNCLOSED_STRING`,
			expectedError: "unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var et EventType
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

func TestEventType_UnmarshalJSON_MalformedInput(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		expectedError string
	}{
		{
			name:          "InvalidEventTypeString",
			jsonData:      `"INVALID_EVENT_TYPE"`, // Note: The error message in event.go is "invalid ErrorCode: %s". It should probably be "invalid EventType: %s".
			expectedError: "invalid EventType: INVALID_EVENT_TYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var et EventType
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
