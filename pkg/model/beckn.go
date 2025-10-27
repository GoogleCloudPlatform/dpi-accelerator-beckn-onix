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
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Subscriber represents a unique operational configuration of a trusted platform on a network.
type Subscriber struct {
	// Added db:"column_name" tags for database mapping
	SubscriberID string    `json:"subscriber_id,omitzero" db:"subscriber_id"`
	URL          string    `json:"url,omitzero" format:"uri" db:"url"`
	Type         Role      `json:"type,omitzero" enum:"BAP,BPP,BG" db:"type"`
	Domain       string    `json:"domain,omitzero" db:"domain"`
	Location     *Location `json:"location,omitzero" db:"location"`
}

// Subscription represents subscription details of a network participant.
type Subscription struct {
	Subscriber `json:",inline"`
	// Added db:"column_name" tags for these fields.
	KeyID              string             `json:"key_id,omitzero" format:"uuid" db:"key_id"`
	SigningPublicKey   string             `json:"signing_public_key,omitzero" db:"signing_public_key"`
	EncrPublicKey      string             `json:"encr_public_key,omitzero" db:"encr_public_key"`
	ValidFrom          time.Time          `json:"valid_from,omitzero" format:"date-time" db:"valid_from"`
	ValidUntil         time.Time          `json:"valid_until,omitzero" format:"date-time" db:"valid_until"`
	Status             SubscriptionStatus `json:"status,omitzero" enum:"INITIATED,UNDER_SUBSCRIPTION,SUBSCRIBED,EXPIRED,UNSUBSCRIBED,INVALID_SSL" db:"status"`
	Created            time.Time          `json:"created,omitzero" format:"date-time" db:"created_at"`
	Updated            time.Time          `json:"updated,omitzero" format:"date-time" db:"updated_at"`
	Nonce              string             `json:"nonce,omitzero" db:"nonce"`
	ExtendedAttributes json.RawMessage    `json:"extended_attributes,omitzero"`
}

// SubscriptionRequest represents the data structure for a new subscription request.
// It embeds the Subscription details and includes a MessageID for tracking.
type SubscriptionRequest struct {
	Subscription `json:",inline"`
	// MessageID is a unique identifier for this specific request message.
	MessageID string `json:"message_id,omitzero"`
}

// SubscriptionStatus defines the set of possible statuses for a subscription operation's immediate response.
type SubscriptionStatus string

// Defines the valid SubscriptionStatus values for the immediate /subscribe response.
const (
	// Values from the original SubscriberStatus
	// SubscriptionStatusEmpty represents an uninitialized or empty status.
	// It's added to support incoming requests which might not have the status field set.
	SubscriptionStatusEmpty SubscriptionStatus = "" // Added for supporting Incoming request which do not have status set.
	// SubscriptionStatusInitiated indicates that the subscription process has been initiated.
	SubscriptionStatusInitiated SubscriptionStatus = "INITIATED"
	// SubscriptionStatusUnderSubscription indicates that the subscription is currently being processed or is pending approval.
	SubscriptionStatusUnderSubscription SubscriptionStatus = "UNDER_SUBSCRIPTION"
	// SubscriptionStatusSubscribed indicates that the participant is actively subscribed to the network.
	SubscriptionStatusSubscribed SubscriptionStatus = "SUBSCRIBED"
	// SubscriptionStatusExpired indicates that the subscription has expired.
	SubscriptionStatusExpired SubscriptionStatus = "EXPIRED"
	// SubscriptionStatusUnsubscribed indicates that the participant has unsubscribed from the network.
	SubscriptionStatusUnsubscribed SubscriptionStatus = "UNSUBSCRIBED"
	// SubscriptionStatusInvalidSSL indicates that the subscription is inactive due to an invalid SSL certificate.
	SubscriptionStatusInvalidSSL SubscriptionStatus = "INVALID_SSL"
)

var validSubscriptionStatuses = map[SubscriptionStatus]bool{
	SubscriptionStatusEmpty:             true,
	SubscriptionStatusInitiated:         true,
	SubscriptionStatusUnderSubscription: true,
	SubscriptionStatusSubscribed:        true,
	SubscriptionStatusExpired:           true,
	SubscriptionStatusUnsubscribed:      true,
	SubscriptionStatusInvalidSSL:        true,
}

// MarshalJSON implements the json.Marshaler interface for SubscriptionStatus.
func (s SubscriptionStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

// UnmarshalJSON implements the json.Unmarshaler interface for SubscriptionStatus.
func (s *SubscriptionStatus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	status := SubscriptionStatus(str)
	if !validSubscriptionStatuses[status] {
		return fmt.Errorf("invalid SubscriptionStatus: %s", str)
	}
	*s = status
	return nil
}

// SubscriptionResponse is the successful response structure for /subscribe POST and PATCH.
type SubscriptionResponse struct {
	Status    SubscriptionStatus `json:"status"`
	MessageID string             `json:"message_id"`
}

// AuthHeaderSubscriber is the standard HTTP header key for subscriber authorization.
const (
	AuthHeaderSubscriber string = "Authorization"
	// UnauthorizedHeaderSubscriber is the standard HTTP header key used in responses when authorization fails,
	// typically indicating how to authenticate.
	UnauthorizedHeaderSubscriber string = "WWW-Authenticate"
	// AuthHeaderGateway
	AuthHeaderGateway string = "X-Gateway-Authorization"
)

// Role defines the functional type of a participant in the network.
type Role string

const (
	// RoleBAP represents a Buyer App Participant (BAP) in the network.
	RoleBAP Role = "BAP"
	// RoleBPP represents a Buyer Platform Participant (BPP) in the network.
	RoleBPP Role = "BPP"
	// RoleGateway represents a Gateway that facilitates communication in the network.
	RoleGateway Role = "BG"
	// RoleRegistry represents the Registry itself.
	RoleRegistry Role = "REGISTRY"
)

var validRoles = map[Role]bool{
	RoleBAP:      true,
	RoleBPP:      true,
	RoleGateway:  true,
	RoleRegistry: true,
}

// UnmarshalYAML implements custom YAML unmarshalling for Role to ensure only valid values are accepted.
func (r *Role) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var roleName string
	if err := unmarshal(&roleName); err != nil {
		return err
	}
	role := Role(roleName)
	if !validRoles[role] {
		return fmt.Errorf("invalid Role: %s", roleName)
	}
	*r = role
	return nil
}

// Gps represents a GPS coordinate as a string, typically in "latitude,longitude" format.
type Gps string

// Location describes the physical location of an entity.
type Location struct {
	ID          string              `json:"id,omitempty"`
	Descriptor  *LocationDescriptor `json:"descriptor,omitempty"`
	MapURL      string              `json:"map_url,omitempty" format:"uri"`
	Gps         Gps                 `json:"gps,omitempty"`
	Address     string              `json:"address,omitempty"`
	City        *City               `json:"city,omitempty"`
	District    string              `json:"district,omitempty"`
	State       *State              `json:"state,omitempty"`
	Country     *Country            `json:"country,omitempty"`
	AreaCode    string              `json:"area_code,omitempty"`
	Circle      *Circle             `json:"circle,omitempty"`
	Polygon     string              `json:"polygon,omitempty"`
	ThreeDSpace string              `json:"3dspace,omitempty"`
	Rating      string              `json:"rating,omitempty"`
}

// Scan implements the sql.Scanner interface for Location.
// It converts database JSONB ([]byte) into a model.Location struct.
func (l *Location) Scan(value interface{}) error {
	if value == nil {
		*l = Location{} // Set to zero value (empty Location) if DB value is NULL.
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Scan source was not []byte; got %T", value)
	}
	// Unmarshal the JSON bytes into the Location struct.
	return json.Unmarshal(bytes, l)
}

// Value implements the driver.Valuer interface for Location.
// It converts a model.Location struct into a JSONB ([]byte) for the database.
// This is now explicitly designed to store NULL if the Location struct is empty.
func (l Location) Value() (driver.Value, error) {
	// Handle the case where Location is empty/zero-value to store NULL in DB.
	// This check ensures that if all relevant fields are empty, it stores NULL.
	if l.ID == "" && l.Descriptor == nil && l.MapURL == "" && l.Gps == "" &&
		l.Address == "" && l.City == nil && l.District == "" && l.State == nil &&
		l.Country == nil && l.AreaCode == "" && l.Circle == nil && l.Polygon == "" &&
		l.ThreeDSpace == "" && l.Rating == "" {
		return nil, nil
	}
	// Otherwise, marshal the Location struct into JSON bytes.
	return json.Marshal(l)
}

type LocationDescriptor struct {
	Name           string                `json:"name,omitempty"`
	Code           string                `json:"code,omitempty"`
	ShortDesc      string                `json:"short_desc,omitempty"`
	LongDesc       string                `json:"long_desc,omitempty"`
	AdditionalDesc *AdditionalDescriptor `json:"additional_desc,omitempty"`
	Media          []*MediaFile          `json:"media,omitempty"`
	Images         []*Image              `json:"images,omitempty"`
}

// AdditionalDescriptor provides supplementary details about a location, often via a URL.
type AdditionalDescriptor struct {
	URL         string `json:"url,omitempty"`
	ContentType string `json:"content_type,omitempty" enum:"text/plain,text/html,application/json"`
}

// MediaFile Represents a media file with its URL and signature.
type MediaFile struct {
	Mimetype  string `json:"mimetype,omitempty"`
	URL       string `json:"url,omitempty" format:"uri"`
	Signature string `json:"signature,omitempty"`
	Dsa       string `json:"dsa,omitempty"`
}

// Image Describes an image with URL, size, and dimensions.
type Image struct {
	URL      string `json:"url,omitempty" format:"uri"`
	SizeType string `json:"size_type,omitempty" enum:"xs,sm,md,lg,xl,custom"`
	Width    string `json:"width,omitempty"`
	Height   string `json:"height,omitempty"`
}

// City represents a city, typically identified by its name and a standardized code.
type City struct {
	Name string `json:"name,omitempty"`
	Code string `json:"code,omitempty"`
}

// State represents a bounded geopolitical region of governance within a country.
type State struct {
	Name string `json:"name,omitempty"`
	Code string `json:"code,omitempty"`
}

// Country represents a country, identified by its name and a standardized code.
type Country struct {
	Name string `json:"name,omitempty"`
	Code string `json:"code,omitempty"`
}

// Circle describes a circular geographical region defined by a central GPS coordinate and a radius.
type Circle struct {
	Gps    Gps     `json:"gps,omitempty"`
	Radius *Scalar `json:"radius,omitempty"`
}

// Scalar Describes a scalar value with type, value, and optional range.
type Scalar struct {
	Type           string       `json:"type,omitempty" enum:"CONSTANT,VARIABLE"`
	Value          string       `json:"value,omitempty" pattern:"[+-]?([0-9]*[.])?[0-9]+"`
	EstimatedValue string       `json:"estimated_value,omitempty"`
	ComputedValue  string       `json:"computed_value,omitempty"`
	Range          *ScalarRange `json:"range,omitempty"`
	Unit           string       `json:"unit,omitempty"`
}

// ScalarRange Defines a range for scalar values.
type ScalarRange struct {
	Min string `json:"min,omitempty"`
	Max string `json:"max,omitempty"`
}

// OnSubscribeRequest defines the Beckn message body for the /on_subscribe callback.
type OnSubscribeRequest struct {
	MessageID string `json:"message_id"`
	Challenge string `json:"challenge"` // Encrypted challenge string
}

// OnSubscribeResponse defines the expected response from the NP's /on_subscribe callback.
// This is a simplified version; a full Beckn response would be more complex.
type OnSubscribeResponse struct {
	Answer string `json:"answer"` // Decrypted challenge string
}

// AuthHeader holds the components from the parsed Authorization header.
type AuthHeader struct {
	SubscriberID string
	UniqueID     string
	Algorithm    string
}

// Context provides a high-level overview of the transaction.
type Context struct {
	Domain        string    `json:"domain,omitempty"`         // Domain code
	Location      *Location `json:"location,omitempty"`       // Transaction fulfillment location
	Action        string    `json:"action,omitempty"`         // Beckn protocol method
	Version       string    `json:"version,omitempty"`        // Protocol version
	BapID         string    `json:"bap_id,omitempty"`         // Subscriber ID of BAP
	BapURI        string    `json:"bap_uri,omitempty"`        // Subscriber URL of BAP (URI format)
	BppID         string    `json:"bpp_id,omitempty"`         // Subscriber ID of BPP
	BppURI        string    `json:"bpp_uri,omitempty"`        // Subscriber URL of BPP (URI format)
	TransactionID string    `json:"transaction_id,omitempty"` // Unique value for user session (UUID format)
	MessageID     string    `json:"message_id,omitempty"`     // Unique value for request/callback cycle (UUID format)
	Timestamp     string    `json:"timestamp,omitempty"`      // Time of request generation (RFC3339 format)
	Key           string    `json:"key,omitempty"`            // Encryption public key of sender
	TTL           string    `json:"ttl,omitempty"`            // Duration in ISO8601 format for message validity
}

// Status represents the acknowledgment status in a response.
type Status string

const (
	// StatusACK indicates a successful acknowledgment.
	StatusACK Status = "ACK"
	// StatusNACK indicates a negative acknowledgment or failure.
	StatusNACK Status = "NACK"
)

// Ack represents an acknowledgment response.
type Ack struct {
	// Status holds the acknowledgment status (ACK/NACK).
	Status Status `json:"status"`
}

// Message represents the structure of a response message.
type Message struct {
	// Ack contains the acknowledgment status.
	Ack Ack `json:"ack"`
	// Error holds error details, if any, in the response.
	Error *Error `json:"error,omitempty"`
}

// TxnResponse represents the main response structure.
type TxnResponse struct {
	Message Message `json:"message"`
}

type TxnRequest struct {
	Context Context `json:"context"`
}
