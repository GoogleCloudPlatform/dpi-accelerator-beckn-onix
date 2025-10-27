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
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"github.com/google/uuid"
)

type repo interface {
	EncryptionKey(ctx context.Context, subID, keyID string) (string, error)
	InsertSubscription(context.Context, *model.Subscription) (*model.Subscription, error)
}

type encrInitializer interface {
	Init(ctx context.Context) (string, error)
}

// RegistrySelfRegistrationConfig holds the configuration for the registry's self-registration.
type RegistrySelfRegistrationConfig struct {
	KeyID        string `yaml:"keyID"`        // UniqueKeyID for the registry's own keyset (e.g., used in Secret Manager).
	SubscriberID string `yaml:"subscriberID"` // The registry's own subscriber ID.
	URL          string `yaml:"url"`          // The registry's own URL.
	Domain       string `yaml:"domain"`       // The registry's own domain.
}

// Validate checks if the configuration fields are valid.
func (c *RegistrySelfRegistrationConfig) Validate() error {

	if c == nil {
		return errors.New("RegistrySelfRegistrationConfig cannot be nil")
	}
	if c.KeyID == "" {
		return errors.New("RegistrySelfRegistrationConfig: KeyID cannot be empty")
	}
	if c.SubscriberID == "" {
		return errors.New("RegistrySelfRegistrationConfig: SubscriberID cannot be empty")
	}
	if c.URL == "" {
		return errors.New("RegistrySelfRegistrationConfig: URL cannot be empty")
	}
	if c.Domain == "" {
		return errors.New("RegistrySelfRegistrationConfig: Domain cannot be empty")
	}
	return nil
}

// registrySetupService handles the initial key registration logic.
type registrySetupService struct {
	repo    repo
	encInit encrInitializer
	cfg     *RegistrySelfRegistrationConfig
}

// NewRegistrySetupService is the constructor for the service.
func NewRegistrySetupService(dbRepo repo, encInit encrInitializer, cfg *RegistrySelfRegistrationConfig) (*registrySetupService, error) {
	if dbRepo == nil {
		slog.Error("NewRegistrySetupService: dbRepo cannot be nil")
		return nil, fmt.Errorf("dbRepo cannot be nil")
	}
	if encInit == nil {
		slog.Error("NewRegistrySetupService: encInit cannot be nil")
		return nil, fmt.Errorf("encInit cannot be nil")
	}
	if err := cfg.Validate(); err != nil {
		slog.Error("NewRegistrySetupService: Invalid RegistrySelfRegistrationConfig", "error", err)
		return nil, fmt.Errorf("invalid RegistrySelfRegistrationConfig: %w", err)
	}
	return &registrySetupService{
		repo:    dbRepo,
		encInit: encInit,
		cfg:     cfg,
	}, nil
}

// It must be called at application startup.
func (s *registrySetupService) SelfRegister(ctx context.Context) error {
	slog.InfoContext(ctx, "RegistrySetupService: Checking if registry's own encryption key exists in DB", "subscriber_id", s.cfg.SubscriberID, "key_id", s.cfg.KeyID)
	_, err := s.repo.EncryptionKey(ctx, s.cfg.SubscriberID, s.cfg.KeyID)

	// If there's no error, the key exists.
	if err == nil {
		slog.InfoContext(ctx, "RegistrySetupService: Registry key already exists in DB. No self-registration needed.", "subscriber_id", s.cfg.SubscriberID, "key_id", s.cfg.KeyID)
		return nil
	}

	// Using errors.Is for specific error types is preferred.
	if errors.Is(err, repository.ErrEncrKeyNotFound) {
		slog.InfoContext(ctx, "RegistrySetupService: Registry key not found in DB. Initializing and self-registering.", "subscriber_id", s.cfg.SubscriberID, "key_id", s.cfg.KeyID)

		slog.InfoContext(ctx, "RegistrySetupService: Initializing keys via encrInitializer", "key_id_for_secret_manager", s.cfg.KeyID)
		registryEncrPublicKey, initErr := s.encInit.Init(ctx)
		if initErr != nil {
			slog.ErrorContext(ctx, "RegistrySetupService: Failed to initialize keys via encrInitializer", "error", initErr, "key_id_for_secret_manager", s.cfg.KeyID)
			return fmt.Errorf("failed to initialize registry keys: %w", initErr)
		}
		if registryEncrPublicKey == "" {
			slog.ErrorContext(ctx, "RegistrySetupService: encrInitializer returned an empty public key", "key_id_for_secret_manager", s.cfg.KeyID)
			return fmt.Errorf("encrInitializer returned an empty public key for keyID %s", s.cfg.KeyID)
		}
		slog.InfoContext(ctx, "RegistrySetupService: Keys initialized successfully. Public encryption key obtained.", "key_id_for_secret_manager", s.cfg.KeyID)

		now := time.Now().UTC()
		registrySubscription := &model.Subscription{
			Subscriber:       model.Subscriber{SubscriberID: s.cfg.SubscriberID, URL: s.cfg.URL, Type: model.RoleRegistry, Domain: s.cfg.Domain},
			KeyID:            s.cfg.KeyID,
			EncrPublicKey:    registryEncrPublicKey,
			SigningPublicKey: "", // encryptionService.Init typically only handles encryption keys
			ValidFrom:        now,
			ValidUntil:       now.AddDate(100, 0, 0), // Valid for 100 years
			Status:           model.SubscriptionStatusSubscribed,
			Nonce:            uuid.NewString(),
		}

		slog.InfoContext(ctx, "RegistrySetupService: Inserting self-subscription into DB", "subscriber_id", s.cfg.SubscriberID, "key_id", s.cfg.KeyID)
		if _, insertErr := s.repo.InsertSubscription(ctx, registrySubscription); insertErr != nil {
			slog.ErrorContext(ctx, "RegistrySetupService: Failed to insert self-subscription into DB", "error", insertErr, "subscriber_id", s.cfg.SubscriberID)
			return fmt.Errorf("failed to insert self-subscription for registry: %w", insertErr)
		}
		slog.InfoContext(ctx, "RegistrySetupService: Registry self-registration successful.", "subscriber_id", s.cfg.SubscriberID, "key_id", s.cfg.KeyID)
		return nil
	}
	// If err is not "not found", then it's some other unexpected DB error.
	slog.ErrorContext(ctx, "RegistrySetupService: Error checking for registry key in DB", "error", err, "subscriber_id", s.cfg.SubscriberID, "key_id", s.cfg.KeyID)
	return fmt.Errorf("error checking for registry key %s for subscriber %s: %w", s.cfg.KeyID, s.cfg.SubscriberID, err)
}
