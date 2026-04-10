// Copyright 2026 Google LLC
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
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestRole_UnmarshalYAML_Success(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		expected Role
	}{
		{
			name:     "ValidRoleBAP",
			yamlData: `BAP`,
			expected: RoleBAP,
		},
		{
			name:     "ValidRoleBPP",
			yamlData: `BPP`,
			expected: RoleBPP,
		},
		{
			name:     "ValidRoleGateway",
			yamlData: `BG`,
			expected: RoleGateway,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var r Role
			err := yaml.Unmarshal([]byte(tc.yamlData), &r)
			if err != nil {
				t.Fatalf("UnmarshalYAML(%q) returned error: %v, want nil", tc.yamlData, err)
			}
			if diff := cmp.Diff(tc.expected, r); diff != "" {
				t.Errorf("UnmarshalYAML(%q) mismatch (-want +got):\n%s", tc.yamlData, diff)
			}
		})
	}
}

func TestSubscriptionStatus_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		status   SubscriptionStatus
		expected string
	}{
		{"Initiated", SubscriptionStatusInitiated, `"INITIATED"`},
		{"Subscribed", SubscriptionStatusSubscribed, `"SUBSCRIBED"`},
		{"Empty", SubscriptionStatusEmpty, `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v, wantErr nil", err)
			}
			if string(got) != tt.expected {
				t.Errorf("MarshalJSON() got = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestSubscriptionStatus_UnmarshalJSON_Success(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected SubscriptionStatus
	}{
		{
			name:     "ValidStatusInitiated",
			jsonData: `"INITIATED"`,
			expected: SubscriptionStatusInitiated,
		},
		{
			name:     "ValidStatusSubscribed",
			jsonData: `"SUBSCRIBED"`,
			expected: SubscriptionStatusSubscribed,
		},
		{
			name:     "ValidStatusEmpty",
			jsonData: `""`,
			expected: SubscriptionStatusEmpty,
		},
		{
			name:     "ValidStatusExpired",
			jsonData: `"EXPIRED"`,
			expected: SubscriptionStatusExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s SubscriptionStatus
			err := json.Unmarshal([]byte(tt.jsonData), &s)
			if err != nil {
				t.Fatalf("UnmarshalJSON(%q) returned error: %v, want nil", tt.jsonData, err)
			}
			if s != tt.expected {
				t.Errorf("UnmarshalJSON(%q) got = %s, want %s", tt.jsonData, s, tt.expected)
			}
		})
	}
}

func TestSubscriptionStatus_UnmarshalJSON_Error(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		expectedError string
	}{
		{
			name:          "InvalidStatus",
			jsonData:      `"INVALID_STATUS"`,
			expectedError: "invalid SubscriptionStatus: INVALID_STATUS",
		},
		{
			name:          "NonStringStatus",
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
			var s SubscriptionStatus
			err := json.Unmarshal([]byte(tt.jsonData), &s)
			if err == nil {
				t.Fatalf("UnmarshalJSON(%q) returned nil error, want error containing %q", tt.jsonData, tt.expectedError)
			}
			if err.Error() != tt.expectedError { // For exact match, otherwise use strings.Contains
				t.Errorf("UnmarshalJSON(%q) returned error %q, want error %q", tt.jsonData, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestRole_UnmarshalYAML_Error(t *testing.T) {
	tests := []struct {
		name          string
		yamlData      string
		expectedError string
	}{
		{
			name:          "InvalidRole",
			yamlData:      `INVALID_ROLE`,
			expectedError: "invalid Role: INVALID_ROLE",
		},
		{
			name:          "NonStringRole", // e.g. a number or a map
			yamlData:      `123`,           // Assuming yaml.v3 unmarshals this to string "123" for the first step
			expectedError: "invalid Role: 123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var r Role
			err := yaml.Unmarshal([]byte(tc.yamlData), &r)
			if err == nil {
				t.Fatalf("UnmarshalYAML(%q) returned nil error, want error %q", tc.yamlData, tc.expectedError)
			}
			if err.Error() != tc.expectedError {
				t.Errorf("UnmarshalYAML(%q) returned error %q, want error %q", tc.yamlData, err.Error(), tc.expectedError)
			}
		})
	}
}

func TestLocation_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    Location
		wantErr bool
	}{
		{
			name:  "ValidJSON",
			input: []byte(`{"id": "loc1", "address": "123 Main St"}`),
			want:  Location{ID: "loc1", Address: "123 Main St"},
		},
		{
			name:  "NullValue",
			input: nil,
			want:  Location{},
		},
		{
			name:    "InvalidType",
			input:   123,
			wantErr: true,
		},
		{
			name:    "InvalidJSON",
			input:   []byte(`{invalid}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var l Location
			err := l.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !cmp.Equal(l, tt.want) {
				t.Errorf("Scan() got = %+v, want %+v", l, tt.want)
			}
		})
	}
}

func TestRole_UnmarshalYAML_DirectError(t *testing.T) {
	// Tests the path where the unmarshal function itself returns an error
	var r Role
	mockErr := fmt.Errorf("unmarshal error")
	err := r.UnmarshalYAML(func(unmarshal any) error {
		return mockErr
	})
	if err != mockErr {
		t.Errorf("UnmarshalYAML() error = %v, want %v", err, mockErr)
	}
}

func TestLocation_Value_FieldCombinations(t *testing.T) {
	// Tests that setting ANY single field makes Value() return a non-nil result
	tests := []struct {
		name     string
		location Location
	}{
		{"ID", Location{ID: "1"}},
		{"Descriptor", Location{Descriptor: &LocationDescriptor{Name: "n"}}},
		{"MapURL", Location{MapURL: "http://map"}},
		{"Gps", Location{Gps: "1,2"}},
		{"City", Location{City: &City{Name: "C"}}},
		{"State", Location{State: &State{Name: "S"}}},
		{"Country", Location{Country: &Country{Name: "CO"}}},
		{"Circle", Location{Circle: &Circle{Gps: "1,1"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.location.Value()
			if err != nil {
				t.Fatalf("Value() error = %v", err)
			}
			if got == nil {
				t.Errorf("Value() for field %s got nil, want []byte", tt.name)
			}
		})
	}
}

func TestLocation_Scan_JsonError(t *testing.T) {
	// Specifically targets the error return from json.Unmarshal
	var l Location
	err := l.Scan([]byte(`{"id": invalid}`))
	if err == nil {
		t.Error("Scan() got nil error, want error for invalid JSON")
	}
}
