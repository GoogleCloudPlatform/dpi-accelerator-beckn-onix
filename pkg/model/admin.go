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

// OperationActionRequest defines the request body for the admin subscription action endpoint.
type OperationActionRequest struct {
	// Action specifies the action to perform on the subscription (APPROVE/REJECT).
	Action OperationAction `json:"action"`

	// OperationID specifies the ID of the target operation.
	OperationID string `json:"operation_id"`

	// Reason provides the rejection reason when rejecting a subscription.
	Reason string `json:"reason,omitempty"`
}

// OperationAction defines the possible actions an admin can take on a subscription.
type OperationAction string

// Defines the valid OperationAction values.
const (
	// OperationActionApproveSubscription represents the action to approve a subscription.
	OperationActionApproveSubscription OperationAction = "APPROVE_SUBSCRIPTION"

	// OperationActionRejectSubscription represents the action to reject a subscription.
	OperationActionRejectSubscription OperationAction = "REJECT_SUBSCRIPTION"
)
