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
	"time"
)

// LROStatus defines the set of possible statuses for an LRO.
type LROStatus string

// Defines the valid OperationStatus values.
const (
	// LROStatusPending indicates that the long-running operation is currently in progress.
	LROStatusPending LROStatus = "PENDING"
	// LROStatusApproved indicates that the long-running operation has been successfully completed and approved.
	LROStatusApproved LROStatus = "APPROVED"
	// LROStatusFailure indicates that the long-running operation has failed.
	LROStatusFailure LROStatus = "FAILURE"
	// LROStatusRejected indicates that the long-running operation has been rejected or failed.
	LROStatusRejected LROStatus = "REJECTED"
)

// OperationType defines the set of possible types for an LRO.
type OperationType string

// Defines the valid OperationType values.
const (
	// OperationTypeCreateSubscription signifies an LRO related to creating a new subscription.
	OperationTypeCreateSubscription OperationType = "CREATE_SUBSCRIPTION"
	// OperationTypeUpdateSubscription signifies an LRO related to updating an existing subscription.
	OperationTypeUpdateSubscription OperationType = "UPDATE_SUBSCRIPTION"
)

type LRO struct {
	OperationID   string          `json:"operation_id"`
	Status        LROStatus       `json:"status,omitempty"`
	Type          OperationType   `json:"type,omitempty"`
	RetryCount    int             `json:"retry_count,omitempty"`
	RequestJSON   json.RawMessage `json:"request_json,omitempty"`
	ResultJSON    json.RawMessage `json:"result_json,omitempty"`
	ErrorDataJSON json.RawMessage `json:"error_data_json,omitempty"`
	CreatedAt     time.Time       `json:"created_at,omitempty"`
	UpdatedAt     time.Time       `json:"updated_at,omitempty"`
}
