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

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/internal/client"
	"github.com/google/dpi-accelerator-beckn-onix/internal/event"
	"github.com/google/dpi-accelerator-beckn-onix/internal/log"
	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"
	"github.com/google/dpi-accelerator-beckn-onix/internal/service"
)

func TestConfig_Valid_Success(t *testing.T) {
	cfg := &config{
		Log:      &log.Config{Level: "INFO"},
		Timeouts: &timeoutConfig{Read: 5 * time.Second, Write: 10 * time.Second, Idle: 120 * time.Second, Shutdown: 15 * time.Second},
		Server:   &serverConfig{Host: "localhost", Port: 8080},
		DB:       &repository.Config{User: "test", Name: "test", ConnectionName: "test-conn"},
		Event:    &event.Config{ProjectID: "test-project", TopicID: "test-topic"},
		Admin:    &service.AdminConfig{OperationRetryMax: 3},
		Setup:    &service.RegistrySelfRegistrationConfig{KeyID: "test-key-id"},
		NPClient: &client.NPClientConfig{Timeout: 10 * time.Second},
	}

	if err := cfg.valid(); err != nil {
		t.Errorf("config.valid() returned error for a valid config: %v", err)
	}
}

func TestInitConfig_Success(t *testing.T) {
	configPath := "testData/valid_config.yaml"

	cfg, err := initConfig(configPath)
	if err != nil {
		t.Fatalf("initConfig() error = %v, wantErr nil", err)
	}
	if cfg == nil {
		t.Fatal("initConfig() cfg is nil, want non-nil")
	}

	// Basic checks for some fields
	if cfg.Log.Level != "INFO" {
		t.Errorf("cfg.Log.Level = %q, want %q", cfg.Log.Level, "INFO")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("cfg.Server.Port = %d, want %d", cfg.Server.Port, 8080)
	}
	if cfg.Admin.OperationRetryMax != 3 {
		t.Errorf("cfg.Admin.OperationRetryMax = %d, want %d", cfg.Admin.OperationRetryMax, 3)
	}
}

func TestInitConfig_Success_DefaultNPClient(t *testing.T) {
	configPath := "testData/config_no_npclient.yaml"
	cfg, err := initConfig(configPath)
	if err != nil {
		t.Fatalf("initConfig() error = %v, wantErr nil", err)
	}
	if cfg == nil {
		t.Fatal("initConfig() cfg is nil, want non-nil")
	}
	if cfg.NPClient == nil {
		t.Fatal("cfg.NPClient is nil, want default NPClientConfig")
	}
}

func TestInitConfig_Error(t *testing.T) {
	invalidYAMLPath := "testData/invalid_yaml.yaml"
	invalidConfigDataPath := "testData/invalid_config_missing_server.yaml"

	tests := []struct {
		name          string
		filePath      string
		expectedError string
	}{
		{
			name:          "file not found",
			filePath:      "testData/non_existent_config.yaml",
			expectedError: "failed to read config file",
		},
		{
			name:          "invalid YAML format",
			filePath:      invalidYAMLPath,
			expectedError: "failed to unmarshal config data",
		},
		{
			name:          "invalid config data (missing server)",
			filePath:      invalidConfigDataPath,
			expectedError: "missing required config section: server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := initConfig(tt.filePath)
			if err == nil {
				t.Fatalf("initConfig() error = nil, wantErr containing %q", tt.expectedError)
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("initConfig() error = %q, want error containing %q", err.Error(), tt.expectedError)
			}
		})
	}
}

func TestConfig_Valid_Error(t *testing.T) {
	// Define base valid parts to construct error cases
	validLogCfg := &log.Config{Level: "INFO"}
	validTimeoutsCfg := &timeoutConfig{Read: 1 * time.Second, Write: 1 * time.Second, Idle: 1 * time.Second, Shutdown: 1 * time.Second}
	validServerCfg := &serverConfig{Host: "localhost", Port: 8080}
	validDBCfg := &repository.Config{User: "test", Name: "test", ConnectionName: "test-conn"}
	validEventCfg := &event.Config{ProjectID: "test-project", TopicID: "test-topic"}
	validNPClientCfg := &client.NPClientConfig{Timeout: 5 * time.Second}
	validAdminCfg := &service.AdminConfig{OperationRetryMax: 1}
	validSetupCfg := &service.RegistrySelfRegistrationConfig{KeyID: "test-key-id"}

	tests := []struct {
		name          string
		cfg           *config
		expectedError string
	}{
		{
			name:          "nil config",
			cfg:           nil,
			expectedError: "config is nil",
		},
		{
			name: "missing log config",
			cfg: &config{
				Timeouts: validTimeoutsCfg,
				Server:   validServerCfg,
				NPClient: validNPClientCfg,
				Admin:    validAdminCfg,
			},
			expectedError: "missing required config section: log",
		},
		{
			name: "missing server config",
			cfg: &config{
				Log:      validLogCfg,
				Timeouts: validTimeoutsCfg,
				NPClient: validNPClientCfg,
				Admin:    validAdminCfg,
			},
			expectedError: "missing required config section: server",
		},
		{
			name: "missing timeouts config",
			cfg: &config{
				Log:      validLogCfg,
				Server:   validServerCfg,
				NPClient: validNPClientCfg,
				Admin:    validAdminCfg,
			},
			expectedError: "missing required config section: timeouts",
		},
		{
			name: "missing db config",
			cfg: &config{
				Log:      validLogCfg,
				Server:   validServerCfg,
				Timeouts: validTimeoutsCfg,
				NPClient: validNPClientCfg,
				Admin:    validAdminCfg,
			},
			expectedError: "missing required config section: db",
		},
		{
			name:          "invalid server port (0)",
			cfg:           &config{Log: validLogCfg, Timeouts: validTimeoutsCfg, Server: &serverConfig{Port: 0}, DB: validDBCfg, NPClient: validNPClientCfg, Admin: validAdminCfg, Event: validEventCfg, Setup: validSetupCfg},
			expectedError: "invalid server port: 0",
		},
		{
			name:          "invalid server port (65536)",
			cfg:           &config{Log: validLogCfg, Timeouts: validTimeoutsCfg, Server: &serverConfig{Port: 65536}, DB: validDBCfg, NPClient: validNPClientCfg, Admin: validAdminCfg, Event: validEventCfg, Setup: validSetupCfg},
			expectedError: "invalid server port: 65536",
		},
		{
			name:          "missing admin config",
			cfg:           &config{Log: validLogCfg, Timeouts: validTimeoutsCfg, Server: validServerCfg, DB: validDBCfg, NPClient: validNPClientCfg, Event: validEventCfg, Setup: validSetupCfg},
			expectedError: "missing required config section: admin",
		},
		{
			name:          "admin.OperationRetryMax is zero",
			cfg:           &config{Log: validLogCfg, Timeouts: validTimeoutsCfg, Server: validServerCfg, DB: validDBCfg, NPClient: validNPClientCfg, Admin: &service.AdminConfig{OperationRetryMax: 0}, Event: validEventCfg, Setup: validSetupCfg},
			expectedError: "admin.OperationRetryMax must be greater than zero",
		},
		{
			name:          "missing event config",
			cfg:           &config{Log: validLogCfg, Timeouts: validTimeoutsCfg, Server: validServerCfg, DB: validDBCfg, Admin: validAdminCfg, Setup: validSetupCfg, NPClient: validNPClientCfg},
			expectedError: "missing required config section: event",
		},
		{
			name:          "missing setup config",
			cfg:           &config{Log: validLogCfg, Timeouts: validTimeoutsCfg, Server: validServerCfg, DB: validDBCfg, Admin: validAdminCfg, Event: validEventCfg, NPClient: validNPClientCfg},
			expectedError: "missing required config section: setup",
		},
		{
			name: "missing encryptionKeyID",
			cfg: &config{
				Log:      validLogCfg,
				Timeouts: validTimeoutsCfg,
				Server:   validServerCfg,
				DB:       validDBCfg,
				Admin:    validAdminCfg,
				Event:    validEventCfg,
				Setup:    &service.RegistrySelfRegistrationConfig{KeyID: ""},
				NPClient: validNPClientCfg,
			},
			expectedError: "encryptionKeyID is missing in setup config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.valid()
			if err == nil {
				t.Fatalf("config.valid() error = nil, wantErr containing %q", tt.expectedError)
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("config.valid() error = %q, want error containing %q", err.Error(), tt.expectedError)
			}
		})
	}
}
