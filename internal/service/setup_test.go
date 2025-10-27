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
	"strings"
	"testing"

	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// mockSetupRepo is a mock implementation of the repo interface in setup.go.
type mockSetupRepo struct {
	encryptionKeyToReturn      string
	encryptionKeyErr           error
	insertSubscriptionToReturn *model.Subscription
	insertSubscriptionErr      error

	// To verify calls
	insertSubscriptionCalledWith *model.Subscription
}

func (m *mockSetupRepo) EncryptionKey(ctx context.Context, subID, keyID string) (string, error) {
	return m.encryptionKeyToReturn, m.encryptionKeyErr
}

func (m *mockSetupRepo) InsertSubscription(ctx context.Context, sub *model.Subscription) (*model.Subscription, error) {
	m.insertSubscriptionCalledWith = sub
	if m.insertSubscriptionToReturn != nil {
		// Return a copy to avoid race conditions if the caller modifies it
		ret := *m.insertSubscriptionToReturn
		return &ret, m.insertSubscriptionErr
	}
	return nil, m.insertSubscriptionErr
}

// mockEncrInitializer is a mock implementation of the encrInitializer interface.
type mockEncrInitializer struct {
	publicKeyToReturn string
	initErr           error
}

func (m *mockEncrInitializer) Init(ctx context.Context) (string, error) {
	return m.publicKeyToReturn, m.initErr
}

func TestRegistrySelfRegistrationConfig_Validate(t *testing.T) {
	validConfig := &RegistrySelfRegistrationConfig{
		KeyID:        "reg-key",
		SubscriberID: "registry.example.com",
		URL:          "https://registry.example.com",
		Domain:       "beckn:retail:1.0.0",
	}

	tests := []struct {
		name       string
		config     *RegistrySelfRegistrationConfig
		wantErrMsg string
	}{
		{"valid config", validConfig, ""},
		{"nil config", nil, "RegistrySelfRegistrationConfig cannot be nil"},
		{"empty KeyID", &RegistrySelfRegistrationConfig{SubscriberID: "id", URL: "url", Domain: "domain"}, "RegistrySelfRegistrationConfig: KeyID cannot be empty"},
		{"empty SubscriberID", &RegistrySelfRegistrationConfig{KeyID: "key", URL: "url", Domain: "domain"}, "RegistrySelfRegistrationConfig: SubscriberID cannot be empty"},
		{"empty URL", &RegistrySelfRegistrationConfig{KeyID: "key", SubscriberID: "id", Domain: "domain"}, "RegistrySelfRegistrationConfig: URL cannot be empty"},
		{"empty Domain", &RegistrySelfRegistrationConfig{KeyID: "key", SubscriberID: "id", URL: "url"}, "RegistrySelfRegistrationConfig: Domain cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErrMsg != "" {
				if err == nil || err.Error() != tt.wantErrMsg {
					t.Errorf("Validate() error = %v, wantErrMsg %q", err, tt.wantErrMsg)
				}
			} else if err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestNewRegistrySetupService(t *testing.T) {
	mockRepo := &mockSetupRepo{}
	mockEncInit := &mockEncrInitializer{}
	validCfg := &RegistrySelfRegistrationConfig{
		KeyID: "reg-key", SubscriberID: "reg-id", URL: "reg-url", Domain: "reg-domain",
	}
	invalidCfg := &RegistrySelfRegistrationConfig{} // Missing fields

	tests := []struct {
		name       string
		dbRepo     repo
		encInit    encrInitializer
		cfg        *RegistrySelfRegistrationConfig
		wantErrMsg string
	}{
		{"success", mockRepo, mockEncInit, validCfg, ""},
		{"nil dbRepo", nil, mockEncInit, validCfg, "dbRepo cannot be nil"},
		{"nil encInit", mockRepo, nil, validCfg, "encInit cannot be nil"},
		{"invalid config", mockRepo, mockEncInit, invalidCfg, "invalid RegistrySelfRegistrationConfig: RegistrySelfRegistrationConfig: KeyID cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewRegistrySetupService(tt.dbRepo, tt.encInit, tt.cfg)
			if tt.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("NewRegistrySetupService() error = %v, want error containing %q", err, tt.wantErrMsg)
				}
				if service != nil {
					t.Error("NewRegistrySetupService() service should be nil on error")
				}
			} else {
				if err != nil {
					t.Errorf("NewRegistrySetupService() unexpected error = %v", err)
				}
				if service == nil {
					t.Error("NewRegistrySetupService() service is nil, want non-nil")
				}
			}
		})
	}
}

func TestRegistrySetupService_SelfRegister(t *testing.T) {
	ctx := context.Background()
	validCfg := &RegistrySelfRegistrationConfig{
		KeyID:        "reg-key",
		SubscriberID: "registry.example.com",
		URL:          "https://registry.example.com",
		Domain:       "beckn:retail:1.0.0",
	}

	tests := []struct {
		name          string
		mockRepoSetup func(*mockSetupRepo)
		mockEncSetup  func(*mockEncrInitializer)
		wantErrMsg    string
		wantInsert    bool // whether InsertSubscription should be called
	}{
		{
			name: "key already exists",
			mockRepoSetup: func(m *mockSetupRepo) {
				m.encryptionKeyToReturn = "existing-key"
				m.encryptionKeyErr = nil
			},
			wantErrMsg: "",
			wantInsert: false,
		},
		{
			name: "successful self-registration",
			mockRepoSetup: func(m *mockSetupRepo) {
				m.encryptionKeyErr = repository.ErrEncrKeyNotFound
				m.insertSubscriptionToReturn = &model.Subscription{} // success
			},
			mockEncSetup: func(m *mockEncrInitializer) {
				m.publicKeyToReturn = "new-public-key"
			},
			wantErrMsg: "",
			wantInsert: true,
		},
		{
			name: "encrInitializer Init fails",
			mockRepoSetup: func(m *mockSetupRepo) {
				m.encryptionKeyErr = repository.ErrEncrKeyNotFound
			},
			mockEncSetup: func(m *mockEncrInitializer) {
				m.initErr = errors.New("init failed")
			},
			wantErrMsg: "failed to initialize registry keys: init failed",
			wantInsert: false,
		},
		{
			name: "encrInitializer returns empty key",
			mockRepoSetup: func(m *mockSetupRepo) {
				m.encryptionKeyErr = repository.ErrEncrKeyNotFound
			},
			mockEncSetup: func(m *mockEncrInitializer) {
				m.publicKeyToReturn = ""
			},
			wantErrMsg: "encrInitializer returned an empty public key",
			wantInsert: false,
		},
		{
			name: "insert subscription fails",
			mockRepoSetup: func(m *mockSetupRepo) {
				m.encryptionKeyErr = repository.ErrEncrKeyNotFound
				m.insertSubscriptionErr = errors.New("db insert failed")
			},
			mockEncSetup: func(m *mockEncrInitializer) {
				m.publicKeyToReturn = "new-public-key"
			},
			wantErrMsg: "failed to insert self-subscription for registry: db insert failed",
			wantInsert: true, // It was called, but failed.
		},
		{
			name: "unexpected DB error on key check",
			mockRepoSetup: func(m *mockSetupRepo) {
				m.encryptionKeyErr = errors.New("unexpected db error")
			},
			wantErrMsg: "error checking for registry key",
			wantInsert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockSetupRepo{}
			mockEncInit := &mockEncrInitializer{}

			if tt.mockRepoSetup != nil {
				tt.mockRepoSetup(mockRepo)
			}
			if tt.mockEncSetup != nil {
				tt.mockEncSetup(mockEncInit)
			}

			service, _ := NewRegistrySetupService(mockRepo, mockEncInit, validCfg)
			err := service.SelfRegister(ctx)

			if tt.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("SelfRegister() error = %v, want error containing %q", err, tt.wantErrMsg)
				}
			} else if err != nil {
				t.Errorf("SelfRegister() unexpected error = %v", err)
			}

			if tt.wantInsert {
				if mockRepo.insertSubscriptionCalledWith == nil {
					t.Error("Expected InsertSubscription to be called, but it was not")
				} else {
					insertedSub := mockRepo.insertSubscriptionCalledWith
					if insertedSub.SubscriberID != validCfg.SubscriberID {
						t.Errorf("InsertSubscription called with wrong SubscriberID. Got %s, want %s", insertedSub.SubscriberID, validCfg.SubscriberID)
					}
					if insertedSub.KeyID != validCfg.KeyID {
						t.Errorf("InsertSubscription called with wrong KeyID. Got %s, want %s", insertedSub.KeyID, validCfg.KeyID)
					}
					if insertedSub.Status != model.SubscriptionStatusSubscribed {
						t.Errorf("InsertSubscription called with wrong Status. Got %s, want %s", insertedSub.Status, model.SubscriptionStatusSubscribed)
					}
				}
			} else if mockRepo.insertSubscriptionCalledWith != nil {
				t.Error("Expected InsertSubscription NOT to be called, but it was")
			}
		})
	}
}
