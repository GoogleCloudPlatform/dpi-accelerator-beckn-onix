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

// EventType defines the type for various events in the system.
type EventType string

// Defines the specific event types used within the system.
const (
	// EventTypeNewSubscriptionRequest signals a new subscription request event.
	EventTypeNewSubscriptionRequest EventType = "NEW_SUBSCRIPTION_REQUEST"
	// EventTypeUpdateSubscriptionRequest signals an update to an existing subscription request event.
	EventTypeUpdateSubscriptionRequest EventType = "UPDATE_SUBSCRIPTION_REQUEST"
	// EventTypeSubscriptionRequestApproved signals that a subscription request has been approved.
	EventTypeSubscriptionRequestApproved EventType = "SUBSCRIPTION_REQUEST_APPROVED"
	// EventTypeSubscriptionRequestRejected signals that a subscription request has been rejected.
	EventTypeSubscriptionRequestRejected EventType = "SUBSCRIPTION_REQUEST_REJECTED"
	// EventTypeOnSubscribeRecieved signals am OnSubscribe call recieved event.
	EventTypeOnSubscribeRecieved EventType = "ON_SUBSCRIBE_RECIEVED"
)

var validEventTypes = map[EventType]bool{
	EventTypeNewSubscriptionRequest:      true,
	EventTypeUpdateSubscriptionRequest:   true,
	EventTypeSubscriptionRequestApproved: true,
	EventTypeSubscriptionRequestRejected: true,
	EventTypeOnSubscribeRecieved:         true,
}

// MarshalJSON implements the json.Marshaler interface for EventType.
func (e EventType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(e))
}

// UnmarshalJSON implements the json.Unmarshaler interface for EventType.
func (e *EventType) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*e = EventType(str)
	if !validEventTypes[*e] {
		return fmt.Errorf("invalid EventType: %s", str)
	}
	return nil
}
