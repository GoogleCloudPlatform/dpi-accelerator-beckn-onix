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

package mock

import (
	"context"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// EventPublisher is a mock implementation of an event publisher, for testing.
// It allows setting specific return values (message ID and error) for each of its methods.
type EventPublisher struct {
	// NewSubscriptionMsgID is the message ID to return for PublishNewSubscriptionRequestEvent.
	NewSubscriptionMsgID string
	// NewSubscriptionErr is the error to return for PublishNewSubscriptionRequestEvent.
	NewSubscriptionErr error

	// UpdateSubscriptionMsgID is the message ID to return for PublishUpdateSubscriptionRequestEvent.
	UpdateSubscriptionMsgID string
	// UpdateSubscriptionErr is the error to return for PublishUpdateSubscriptionRequestEvent.
	UpdateSubscriptionErr error

	// ApproveSubscriptionMsgID is the message ID to return for PublishSubscriptionRequestApprovedEvent.
	ApproveSubscriptionMsgID string
	// ApproveSubscriptionErr is the error to return for PublishSubscriptionRequestApprovedEvent.
	ApproveSubscriptionErr error

	// RejectSubscriptionMsgID is the message ID to return for PublishSubscriptionRequestRejectedEvent.
	RejectSubscriptionMsgID string
	// RejectSubscriptionErr is the error to return for PublishSubscriptionRequestRejectedEvent.
	RejectSubscriptionErr error

	// OnSubscribeRecievedMsgID is the message ID to return for PublishOnSubscribeRecievedEvent.
	OnSubscribeRecievedMsgID string
	// OnSubscribeRecievedErr is the error to return for PublishOnSubscribeRecievedEvent.
	OnSubscribeRecievedErr error
}

// PublishNewSubscriptionRequestEvent mocks the publishing of a new subscription request event.
func (m *EventPublisher) PublishNewSubscriptionRequestEvent(ctx context.Context, req *model.SubscriptionRequest) (string, error) {
	return m.NewSubscriptionMsgID, m.NewSubscriptionErr
}

// PublishUpdateSubscriptionRequestEvent mocks the publishing of an update subscription request event.
func (m *EventPublisher) PublishUpdateSubscriptionRequestEvent(ctx context.Context, req *model.SubscriptionRequest) (string, error) {
	return m.UpdateSubscriptionMsgID, m.UpdateSubscriptionErr
}

// PublishSubscriptionRequestApprovedEvent mocks the publishing of a subscription request approved event.
func (m *EventPublisher) PublishSubscriptionRequestApprovedEvent(ctx context.Context, req *model.LRO) (string, error) {
	return m.ApproveSubscriptionMsgID, m.ApproveSubscriptionErr
}

// PublishSubscriptionRequestRejectedEvent mocks the publishing of a subscription request rejected event.
func (m *EventPublisher) PublishSubscriptionRequestRejectedEvent(ctx context.Context, req *model.LRO) (string, error) {
	return m.RejectSubscriptionMsgID, m.RejectSubscriptionErr
}

// PublishOnSubscribeRecievedEvent mocks the publishing of an on_subscribe received event.
func (m *EventPublisher) PublishOnSubscribeRecievedEvent(ctx context.Context, lroID string) (string, error) {
	return m.OnSubscribeRecievedMsgID, m.OnSubscribeRecievedErr
}
