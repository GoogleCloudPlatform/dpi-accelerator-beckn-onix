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
	"errors"
	"testing"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

func TestEventPublisher_PublishNewSubscriptionRequestEvent(t *testing.T) {
	ctx := context.Background()
	req := &model.SubscriptionRequest{}

	expectedMsgID := "test-msg-id"
	expectedErr := errors.New("test error")

	m := &EventPublisher{
		NewSubscriptionMsgID: expectedMsgID,
		NewSubscriptionErr:   expectedErr,
	}

	msgID, err := m.PublishNewSubscriptionRequestEvent(ctx, req)

	if msgID != expectedMsgID {
		t.Errorf("PublishNewSubscriptionRequestEvent() msgID = %v, want %v", msgID, expectedMsgID)
	}
	if err != expectedErr {
		t.Errorf("PublishNewSubscriptionRequestEvent() error = %v, wantErr %v", err, expectedErr)
	}
}

func TestEventPublisher_PublishUpdateSubscriptionRequestEvent(t *testing.T) {
	ctx := context.Background()
	req := &model.SubscriptionRequest{}

	expectedMsgID := "test-msg-id"
	expectedErr := errors.New("test error")

	m := &EventPublisher{
		UpdateSubscriptionMsgID: expectedMsgID,
		UpdateSubscriptionErr:   expectedErr,
	}

	msgID, err := m.PublishUpdateSubscriptionRequestEvent(ctx, req)

	if msgID != expectedMsgID {
		t.Errorf("PublishUpdateSubscriptionRequestEvent() msgID = %v, want %v", msgID, expectedMsgID)
	}
	if err != expectedErr {
		t.Errorf("PublishUpdateSubscriptionRequestEvent() error = %v, wantErr %v", err, expectedErr)
	}
}

func TestEventPublisher_PublishSubscriptionRequestApprovedEvent(t *testing.T) {
	ctx := context.Background()
	req := &model.LRO{}
	expectedMsgID := "test-msg-id"
	expectedErr := errors.New("test error")

	m := &EventPublisher{
		ApproveSubscriptionMsgID: expectedMsgID,
		ApproveSubscriptionErr:   expectedErr,
	}

	msgID, err := m.PublishSubscriptionRequestApprovedEvent(ctx, req)

	if msgID != expectedMsgID {
		t.Errorf("PublishSubscriptionRequestApprovedEvent() msgID = %v, want %v", msgID, expectedMsgID)
	}
	if err != expectedErr {
		t.Errorf("PublishSubscriptionRequestApprovedEvent() error = %v, wantErr %v", err, expectedErr)
	}
}

func TestEventPublisher_PublishSubscriptionRequestRejectedEvent(t *testing.T) {
	ctx := context.Background()
	req := &model.LRO{}
	expectedMsgID := "test-msg-id"
	expectedErr := errors.New("test error")

	m := &EventPublisher{
		RejectSubscriptionMsgID: expectedMsgID,
		RejectSubscriptionErr:   expectedErr,
	}

	msgID, err := m.PublishSubscriptionRequestRejectedEvent(ctx, req)

	if msgID != expectedMsgID {
		t.Errorf("PublishSubscriptionRequestRejectedEvent() msgID = %v, want %v", msgID, expectedMsgID)
	}
	if err != expectedErr {
		t.Errorf("PublishSubscriptionRequestRejectedEvent() error = %v, wantErr %v", err, expectedErr)
	}
}

func TestEventPublisher_PublishOnSubscribeRecievedEvent(t *testing.T) {
	ctx := context.Background()
	lroID := "test-lro-id"

	expectedMsgID := "test-msg-id"
	expectedErr := errors.New("test error")

	m := &EventPublisher{
		OnSubscribeRecievedMsgID: expectedMsgID,
		OnSubscribeRecievedErr:   expectedErr,
	}

	msgID, err := m.PublishOnSubscribeRecievedEvent(ctx, lroID)

	if msgID != expectedMsgID {
		t.Errorf("PublishOnSubscribeRecievedEvent() msgID = %v, want %v", msgID, expectedMsgID)
	}
	if err != expectedErr {
		t.Errorf("PublishOnSubscribeRecievedEvent() error = %v, wantErr %v", err, expectedErr)
	}
}
