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

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
	"github.com/google/uuid"

	becknmodel "github.com/beckn/beckn-onix/pkg/model"
)

// Error definitions for the subscriber service
var (
	ErrMissingSubscriberID     = errors.New("subscriber_id is required")
	ErrMissingDomain           = errors.New("domain is required")
	ErrMissingType             = errors.New("type is required")
	ErrMissingOperationID      = errors.New("operation_id is required")
	ErrLRONotFound             = errors.New("lro not found")
	ErrLRONotApproved          = errors.New("lro status is not approved")
	ErrKeyGenerationFailed     = errors.New("key generation failed")
	ErrKeyFetchFailed          = errors.New("key fetch failed")
	ErrKeyStoreFailed          = errors.New("key store failed")
	ErrRegistryOperationFailed = errors.New("registry operation failed")
	ErrSigningFailed           = errors.New("signing failed")
)

// registryClient defines the interface for interacting with the registry component
// for subscription and LRO management.
type registryClient interface {
	CreateSubscription(ctx context.Context, req *model.SubscriptionRequest) (*model.SubscriptionResponse, error)
	UpdateSubscription(ctx context.Context, req *model.SubscriptionRequest, authHeader string) (*model.SubscriptionResponse, error)
	GetOperation(ctx context.Context, operationID string) (*model.LRO, error)
}

// onSubscribeEventPublisher defines the interface for publishing an OnSubscribeRecievedEvent.
// This can be implemented by event.Publisher.
type onSubscribeEventPublisher interface {
	PublishOnSubscribeRecievedEvent(ctx context.Context, lroID string) (string, error)
}

// keyManager defines the interface for key management operations needed by subscriberService.
type keyManager interface {
	Keyset(ctx context.Context, keyID string) (*becknmodel.Keyset, error)
	GenerateKeyset() (*becknmodel.Keyset, error)
	InsertKeyset(ctx context.Context, keyID string, keyset *becknmodel.Keyset) error
	DeleteKeyset(ctx context.Context, keyID string) error
	LookupNPKeys(ctx context.Context, subscriberID, uniqueKeyID string) (signingPublicKey string, encrPublicKey string, err error)
}

// decrypter defines the interface for decryption operations needed by subscriberService.
type decrypter interface {
	Decrypt(ctx context.Context, data string, privateKeyBase64, publicKeyBase64 string) (string, error)
}

type subscriberService struct {
	registry registryClient
	keyMgr   keyManager
	dec      decrypter
	evPub    onSubscribeEventPublisher
	authGen  authGen
	regID    string
	regKeyID string // Public encryption key of the Registry, used as sender key in decryption
}

// NewSubscriberService creates a new subscriberService.
func NewSubscriberService(
	registry registryClient,
	keyMgr keyManager,
	dec decrypter,
	evPub onSubscribeEventPublisher,
	authGen authGen,
	regID, regKeyID string,
) (*subscriberService, error) {
	if registry == nil {
		return nil, errors.New("registryClient cannot be nil")
	}
	if keyMgr == nil {
		return nil, errors.New("keyManager cannot be nil")
	}
	if dec == nil {
		return nil, errors.New("decrypter cannot be nil")
	}
	if evPub == nil {
		return nil, errors.New("eventPublisher (onSubscribeEventPublisher) cannot be nil")
	}
	if authGen == nil {
		return nil, errors.New("authGen cannot be nil")
	}
	if regID == "" {
		return nil, errors.New("regID cannot be empty")
	}
	if regKeyID == "" {
		return nil, errors.New("regKeyID cannot be empty")
	}
	return &subscriberService{
		registry: registry,
		keyMgr:   keyMgr,
		dec:      dec,
		evPub:    evPub,
		regID:    regID,
		regKeyID: regKeyID,
		authGen:  authGen,
	}, nil
}

func (s *subscriberService) validateSubscriptionRequest(req *model.NpSubscriptionRequest) error {
	if req.SubscriberID == "" {
		return ErrMissingSubscriberID
	}
	if req.Domain == "" {
		return ErrMissingDomain
	}
	if req.Type == "" {
		return ErrMissingType
	}
	return nil
}

func (s *subscriberService) keySet(ctx context.Context, req *model.NpSubscriptionRequest) (*becknmodel.Keyset, error) {
	var keys *becknmodel.Keyset
	var err error

	if req.KeyID != "" {
		slog.InfoContext(ctx, "SubscriberService: Fetching existing keyset", "key_id", req.KeyID)
		keys, err = s.keyMgr.Keyset(ctx, req.KeyID)
		if err != nil {
			slog.ErrorContext(ctx, "SubscriberService: Failed to fetch keyset", "key_id", req.KeyID, "error", err)
			return nil, fmt.Errorf("%w: %v", ErrKeyFetchFailed, err)
		}
	} else {
		slog.InfoContext(ctx, "SubscriberService: Generating new keyset", "subscriber_id", req.SubscriberID)
		keys, err = s.keyMgr.GenerateKeyset()
		if err != nil {
			slog.ErrorContext(ctx, "SubscriberService: Failed to generate new keyset", "error", err)
			return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
		}
		slog.InfoContext(ctx, "SubscriberService: New keyset generated", "subscriber_id", req.SubscriberID, "new_key_id", req.KeyID)
	}
	keys.SubscriberID = req.SubscriberID
	return keys, nil
}

func subscriptionRequest(npReq *model.NpSubscriptionRequest, keys *becknmodel.Keyset) *model.SubscriptionRequest {
	now := time.Now().UTC()
	return &model.SubscriptionRequest{
		MessageID: npReq.MessageID,
		Subscription: model.Subscription{
			Subscriber: model.Subscriber{
				SubscriberID: npReq.SubscriberID,
				URL:          npReq.URL,
				Domain:       npReq.Domain,
				Type:         npReq.Type,
				Location:     npReq.Location,
			},
			KeyID:            keys.UniqueKeyID,
			SigningPublicKey: keys.SigningPublic,
			EncrPublicKey:    keys.EncrPublic,
			ValidFrom:        now,
			ValidUntil:       now.AddDate(100, 0, 0), // Valid for 100 years
			Nonce:            uuid.NewString(),
		},
	}
}

// CreateSubscription handles the logic for creating a new subscription.
func (s *subscriberService) CreateSubscription(ctx context.Context, req *model.NpSubscriptionRequest) (string, error) {
	if err := s.validateSubscriptionRequest(req); err != nil {
		return "", err
	}
	if req.MessageID == "" {
		req.MessageID = uuid.NewString()
		slog.InfoContext(ctx, "SubscriberService: Generated new MessageID for CreateSubscription", "message_id", req.MessageID)
	}

	keys, err := s.keySet(ctx, req)
	if err != nil {
		return "", err
	}

	if err := s.keyMgr.InsertKeyset(ctx, req.MessageID, keys); err != nil {
		slog.ErrorContext(ctx, "SubscriberService: Failed to insert keyset ", "subscriber_id", req.SubscriberID, "key_id", keys.UniqueKeyID, "error", err)
		return "", fmt.Errorf("%w: %v", ErrKeyStoreFailed, err)
	}

	resp, err := s.registry.CreateSubscription(ctx, subscriptionRequest(req, keys))
	if err != nil {
		slog.ErrorContext(ctx, "SubscriberService: Registry CreateSubscription failed", "error", err)
		return "", fmt.Errorf("%w: %v", ErrRegistryOperationFailed, err)
	}

	slog.InfoContext(ctx, "SubscriberService: CreateSubscription successful", "message_id", resp.MessageID, "status", resp.Status)
	return resp.MessageID, nil
}

// UpdateSubscription handles the logic for updating an existing subscription.
func (s *subscriberService) UpdateSubscription(ctx context.Context, req *model.NpSubscriptionRequest) (string, error) {
	if err := s.validateSubscriptionRequest(req); err != nil {
		return "", err
	}
	if req.MessageID == "" {
		req.MessageID = uuid.NewString()
		slog.InfoContext(ctx, "SubscriberService: Generated new MessageID for UpdateSubscription", "message_id", req.MessageID)
	}

	keys, err := s.keySet(ctx, req)
	if err != nil {
		return "", err
	}

	if err := s.keyMgr.InsertKeyset(ctx, req.MessageID, keys); err != nil {
		slog.ErrorContext(ctx, "SubscriberService: Failed to insert keyset after registry update", "subscriber_id", req.SubscriberID, "key_id", keys.UniqueKeyID, "error", err)
		return "", fmt.Errorf("%w: %v", ErrKeyStoreFailed, err)
	}
	sreq := subscriptionRequest(req, keys)
	authHeader, err := s.authHeader(ctx, sreq)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriberService: Failed to generate auth header", "error", err)
		return "", fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
	}
	resp, err := s.registry.UpdateSubscription(ctx, sreq, authHeader)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriberService: Registry UpdateSubscription failed", "error", err)
		return "", fmt.Errorf("%w: %v", ErrRegistryOperationFailed, err)
	}
	slog.InfoContext(ctx, "SubscriberService: UpdateSubscription successful", "message_id", resp.MessageID, "status", resp.Status)
	return resp.MessageID, nil
}

// UpdateStatus checks the status of an LRO.
func (s *subscriberService) UpdateStatus(ctx context.Context, operationID string) (model.LROStatus, error) {
	if operationID == "" {
		slog.ErrorContext(ctx, "SubscriberService: Missing operation ID for status update")
		return "", ErrMissingOperationID
	}

	lro, err := s.registry.GetOperation(ctx, operationID)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriberService: Failed to get LRO for status update", "message_id", operationID, "error", err)
		return "", fmt.Errorf("%w: %v", ErrLRONotFound, err)
	}
	if lro == nil {
		slog.WarnContext(ctx, "SubscriberService: LRO not found for status update", "message_id", operationID)
		return "", ErrLRONotFound
	}

	if lro.Status != model.LROStatusApproved {
		slog.WarnContext(ctx, "SubscriberService: LRO status is not approved", "message_id", operationID, "status", lro.Status)
		return lro.Status, ErrLRONotApproved
	}

	keys, err := s.keyMgr.Keyset(ctx, operationID)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriberService: Failed to fetch keyset for status update", "error", err)
		return "", fmt.Errorf("%w: %v", ErrKeyFetchFailed, err)
	}
	if err := s.keyMgr.InsertKeyset(ctx, keys.SubscriberID, keys); err != nil {
		slog.ErrorContext(ctx, "SubscriberService: Failed to insert keyset after update status", "subscriber_id", keys.SubscriberID, "key_id", keys.UniqueKeyID, "error", err)
		return "", fmt.Errorf("%w: %v", ErrKeyStoreFailed, err)
	}
	if err := s.keyMgr.DeleteKeyset(ctx, operationID); err != nil {
		slog.WarnContext(ctx, "SubscriberService: Failed to delete keyset after update status", "message_id", operationID, "error", err)
	}
	slog.InfoContext(ctx, "SubscriberService: LRO status approved", "message_id", operationID, "status", lro.Status)
	return lro.Status, nil
}

// OnSubscribe handles an incoming on_subscribe request from the Registry.
// It decrypts the challenge, publishes an event, and returns the decrypted answer.
func (s *subscriberService) OnSubscribe(ctx context.Context, req *model.OnSubscribeRequest) (*model.OnSubscribeResponse, error) {
	slog.InfoContext(ctx, "SubscriberService: Received OnSubscribe request", "message_id", req.MessageID)

	if req.MessageID == "" {
		slog.ErrorContext(ctx, "SubscriberService: MessageID is required for OnSubscribe")
		return nil, errors.New("message_id is required")
	}
	if req.Challenge == "" {
		slog.ErrorContext(ctx, "SubscriberService: Challenge is required for OnSubscribe")
		return nil, errors.New("challenge is required")
	}
	// Get subscribers keyset using the MessageID (which is the operation_id for the subscription LRO)
	// This keyset should contain the NP's private encryption key.
	keys, err := s.keyMgr.Keyset(ctx, req.MessageID)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriberService: Failed to fetch keyset for OnSubscribe", "message_id", req.MessageID, "error", err)
		return nil, fmt.Errorf("failed to retrieve keys for message_id %s: %w", req.MessageID, err)
	}
	if keys.EncrPrivate == "" {
		slog.ErrorContext(ctx, "SubscriberService: Encryption private key not found in keyset for OnSubscribe", "message_id", req.MessageID)
		return nil, fmt.Errorf("encryption private key not found for message_id %s", req.MessageID)
	}

	// Decrypt the challenge string
	// The challenge was encrypted by the Registry (sender) using the NP's public key.
	// The NP (receiver) decrypts it using its own private key and the Registry's public key.
	_, regKey, err := s.keyMgr.LookupNPKeys(ctx, s.regID, s.regKeyID)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriberService: Failed to lookup registry keys", "message_id", req.MessageID, "error", err)
		return nil, fmt.Errorf("failed to lookup registry keys for message_id %s: %w", req.MessageID, err)
	}
	if regKey == "" {
		slog.ErrorContext(ctx, "SubscriberService: Registry public key not found", "message_id", req.MessageID)
		return nil, fmt.Errorf("registry public key not found for message_id %s", req.MessageID)
	}
	decryptedAnswer, err := s.dec.Decrypt(ctx, req.Challenge, keys.EncrPrivate, regKey)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriberService: Failed to decrypt challenge", "message_id", req.MessageID, "error", err)
		return nil, fmt.Errorf("failed to decrypt challenge for message_id %s: %w", req.MessageID, err)
	}

	// Publish an OnSubscribeRecievedEvent
	// This event indicates that the NP has received and processed the /on_subscribe call.
	eventID, err := s.evPub.PublishOnSubscribeRecievedEvent(ctx, req.MessageID)
	if err != nil {
		// Log the error but proceed, as responding to the Registry is the primary path.
		slog.WarnContext(ctx, "SubscriberService: Failed to publish OnSubscribeRecievedEvent", "message_id", req.MessageID, "error", err)
	} else {
		slog.InfoContext(ctx, "SubscriberService: Published OnSubscribeRecievedEvent", "message_id", req.MessageID, "event_id", eventID)
	}

	// Respond with the decrypted answer
	response := &model.OnSubscribeResponse{Answer: decryptedAnswer}
	slog.InfoContext(ctx, "SubscriberService: Successfully processed OnSubscribe request", "message_id", req.MessageID)
	return response, nil
}

func (s *subscriberService) authHeader(ctx context.Context, req *model.SubscriptionRequest) (string, error) {

	body, err := json.Marshal(req)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriberService: Failed to marshal request body", "error", err)
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}
	return s.authGen.AuthHeader(ctx, body, req.SubscriberID)

}
