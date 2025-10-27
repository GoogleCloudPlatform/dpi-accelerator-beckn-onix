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

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// lroCreator defines the interface for creating LROs.
type lroCreator interface {
	Create(ctx context.Context, lro *model.LRO) (*model.LRO, error)
}

// subscriptionRepository defines the interface for fetching subscriber data.
type subscriptionRepository interface {
	GetSubscriberSigningKey(ctx context.Context, subscriberID string, domain string, subType model.Role, keyID string) (string, error)
	Lookup(ctx context.Context, filter *model.Subscription) ([]model.Subscription, error)
}

// subscriptionEventPublisher defines the interface for publishing subscription events.
// This is exported for testing purposes.
type subscriptionEventPublisher interface {
	PublishNewSubscriptionRequestEvent(ctx context.Context, req *model.SubscriptionRequest) (string, error)
	PublishUpdateSubscriptionRequestEvent(ctx context.Context, req *model.SubscriptionRequest) (string, error)
}

type subscriptionService struct {
	lroCreator             lroCreator
	subscriptionRepository subscriptionRepository
	evPublisher            subscriptionEventPublisher
}

// NewSubscriptionService creates a new subscriptionService.
func NewSubscriptionService(lroCreator lroCreator, subscriptionRepository subscriptionRepository, evPub subscriptionEventPublisher) (*subscriptionService, error) {
	if lroCreator == nil {
		slog.Error("NewSubscriptionService: lroCreator cannot be nil")
		return nil, errors.New("lroCreator cannot be nil")
	}
	if subscriptionRepository == nil {
		slog.Error("NewSubscriptionService: subscriptionRepository cannot be nil")
		return nil, errors.New("subscriptionRepository cannot be nil")
	}
	if evPub == nil {
		slog.Error("NewSubscriptionService: eventPublisher cannot be nil")
		return nil, errors.New("eventPublisher cannot be nil")
	}
	return &subscriptionService{lroCreator: lroCreator, subscriptionRepository: subscriptionRepository, evPublisher: evPub}, nil
}

// Lookup retrieves subscriptions based on the provided filter criteria.
func (s *subscriptionService) Lookup(ctx context.Context, filter *model.Subscription) ([]model.Subscription, error) {
	slog.Info("SubscriptionService: Handling lookup request", "filter", filter)

	// Call the repository layer to perform the database lookup.
	subscriptions, err := s.subscriptionRepository.Lookup(ctx, filter)
	if err != nil {
		slog.Error("SubscriptionService: Failed to perform lookup in repository", "error", err, "filter", filter)
		return nil, fmt.Errorf("failed to lookup subscriptions: %w", err)
	}

	slog.Info("SubscriptionService: Lookup successful", "count", len(subscriptions))
	return subscriptions, nil
}

// createLRO is a helper method to construct and persist an LRO.
func (s *subscriptionService) createLRO(ctx context.Context, operationType model.OperationType, req *model.SubscriptionRequest) (*model.LRO, error) {
	requestBytes, err := json.Marshal(req)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriptionService: Failed to marshal request for LRO", "error", err, "operation_id", req.MessageID, "type", operationType)
		return nil, fmt.Errorf("failed to marshal request for LRO type %s: %w", operationType, err)
	}

	newLRO := &model.LRO{
		OperationID: req.MessageID,
		Type:        operationType,
		RequestJSON: requestBytes,
		Status:      model.LROStatusPending,
	}

	createdLRO, err := s.lroCreator.Create(ctx, newLRO)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriptionService: Failed to create LRO via lroCreator", "error", err, "operation_id", newLRO.OperationID, "type", newLRO.Type)
		return nil, fmt.Errorf("failed to initiate LRO type %s: %w", newLRO.Type, err)
	}
	return createdLRO, nil
}

// Create handles the business logic for creating a new subscription.
// It creates an LRO to track this operation.
func (s *subscriptionService) Create(ctx context.Context, req *model.SubscriptionRequest) (*model.LRO, error) {
	if req == nil {
		slog.ErrorContext(ctx, "SubscriptionService: Create called with nil request")
		return nil, errors.New("subscription request cannot be nil")
	}
	slog.InfoContext(ctx, "SubscriptionService: Handling create subscription request", "message_id", req.MessageID)

	createdLRO, err := s.createLRO(ctx, model.OperationTypeCreateSubscription, req)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "SubscriptionService: LRO created for new subscription", "operation_id", createdLRO.OperationID, "status", createdLRO.Status)
	if evID, err := s.evPublisher.PublishNewSubscriptionRequestEvent(ctx, req); err != nil {
		slog.ErrorContext(ctx, "SubscriptionService: Failed to publish new subscription request event", "error", err)
	} else {
		slog.InfoContext(ctx, "SubscriptionService: Published new subscription request event", "message_id", req.MessageID, "event_id", evID)
	}
	return createdLRO, nil

}

// Update handles the business logic for updating an existing subscription.
func (s *subscriptionService) Update(ctx context.Context, req *model.SubscriptionRequest) (*model.LRO, error) {
	if req == nil {
		slog.ErrorContext(ctx, "SubscriptionService: Update called with nil request")
		return nil, errors.New("subscription request cannot be nil")
	}
	slog.InfoContext(ctx, "SubscriptionService: Handling update subscription request", "message_id", req.MessageID)

	createdLRO, err := s.createLRO(ctx, model.OperationTypeUpdateSubscription, req)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "SubscriptionService: LRO created for subscription update", "operation_id", createdLRO.OperationID, "status", createdLRO.Status)
	if evID, err := s.evPublisher.PublishUpdateSubscriptionRequestEvent(ctx, req); err != nil {
		slog.ErrorContext(ctx, "SubscriptionService: Failed to publish update subscription request event", "error", err)
	} else {
		slog.InfoContext(ctx, "SubscriptionService: Published update subscription request event", "message_id", req.MessageID, "event_id", evID)
	}
	return createdLRO, nil
}

// GetSigningPublicKey fetches the subscriber's public signing key.
func (s *subscriptionService) GetSigningPublicKey(ctx context.Context, subscriberID string, domain string, role model.Role, keyID string) (string, error) {
	slog.InfoContext(ctx, "SubscriptionService: Fetching signing public key", "subscriber_id", subscriberID, "domain", domain, "type", role, "key_id", keyID)
	return s.subscriptionRepository.GetSubscriberSigningKey(ctx, subscriberID, domain, role, keyID)
}
