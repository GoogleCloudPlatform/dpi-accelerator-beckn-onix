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
)

func TestConfig_Valid_Success(t *testing.T) {
	cfg := &config{
		Log:       &log.Config{Level: "INFO"},
		Timeouts:  &timeoutConfig{Read: 5 * time.Second, Write: 10 * time.Second, Idle: 120 * time.Second, Shutdown: 15 * time.Second},
		Server:    &serverConfig{Host: "localhost", Port: 8080},
		ProjectID: "test-project",
		Registry:  &client.RegistryClientConfig{BaseURL: "http://registry.com"},
		RedisAddr: "localhost:6379",
		RegID:     "registry.beckn.org",
		RegKeyID:  "registry-key-id",
		Event:     &event.Config{ProjectID: "test-project", TopicID: "test-topic"},
	}

	if err := cfg.valid(); err != nil {
		t.Errorf("config.valid() returned error for a valid config: %v", err)
	}
}

func TestInitConfig_Success(t *testing.T) {
	configPath := "testdata/valid_config.yaml"

	cfg, err := initConfig(configPath)
	if err != nil {
		t.Fatalf("initConfig() error = %v, wantErr nil", err)
	}
	if cfg == nil {
		t.Fatal("initConfig() cfg is nil, want non-nil")
	}

	// Basic checks for some fields
	if cfg.Log.Level != "DEBUG" {
		t.Errorf("cfg.Log.Level = %q, want %q", cfg.Log.Level, "DEBUG")
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("cfg.Server.Port = %d, want %d", cfg.Server.Port, 9090)
	}
	if cfg.ProjectID != "test-gcp-project" {
		t.Errorf("cfg.ProjectID = %q, want %q", cfg.ProjectID, "test-gcp-project")
	}
	if cfg.Registry.BaseURL != "http://localhost:8080" {
		t.Errorf("cfg.Registry.BaseURL = %q, want %q", cfg.Registry.BaseURL, "http://localhost:8080")
	}
}

func TestInitConfig_Error(t *testing.T) {
	invalidYAMLPath := "testdata/invalid_yaml.yaml"
	invalidConfigDataPath := "testdata/invalid_config_missing_server.yaml"

	tests := []struct {
		name          string
		filePath      string
		expectedError string
	}{
		{
			name:          "file not found",
			filePath:      "testdata/non_existent_config.yaml",
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
	validRegistryCfg := &client.RegistryClientConfig{BaseURL: "http://registry.com"}
	validEventCfg := &event.Config{ProjectID: "test-project", TopicID: "test-topic"}

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
				Timeouts:  validTimeoutsCfg,
				Server:    validServerCfg,
				ProjectID: "proj",
				Registry:  validRegistryCfg,
				RedisAddr: "redis",
				RegID:     "reg",
				RegKeyID:  "key",
				Event:     validEventCfg,
			},
			expectedError: "missing required config section: log",
		},
		{
			name: "missing server config",
			cfg: &config{
				Log:       validLogCfg,
				Timeouts:  validTimeoutsCfg,
				ProjectID: "proj",
				Registry:  validRegistryCfg,
				RedisAddr: "redis",
				RegID:     "reg",
				RegKeyID:  "key",
				Event:     validEventCfg,
			},
			expectedError: "missing required config section: server",
		},
		{
			name: "missing timeouts config",
			cfg: &config{
				Log:       validLogCfg,
				Server:    validServerCfg,
				ProjectID: "proj",
				Registry:  validRegistryCfg,
				RedisAddr: "redis",
				RegID:     "reg",
				RegKeyID:  "key",
				Event:     validEventCfg,
			},
			expectedError: "missing required config section: timeouts",
		},
		{
			name:          "invalid server port (0)",
			cfg:           &config{Log: validLogCfg, Timeouts: validTimeoutsCfg, Server: &serverConfig{Port: 0}, ProjectID: "proj", Registry: validRegistryCfg, RedisAddr: "redis", RegID: "reg", RegKeyID: "key", Event: validEventCfg},
			expectedError: "invalid server port: 0",
		},
		{
			name:          "invalid server port (65536)",
			cfg:           &config{Log: validLogCfg, Timeouts: validTimeoutsCfg, Server: &serverConfig{Port: 65536}, ProjectID: "proj", Registry: validRegistryCfg, RedisAddr: "redis", RegID: "reg", RegKeyID: "key", Event: validEventCfg},
			expectedError: "invalid server port: 65536",
		},
		{
			name: "missing registry config",
			cfg: &config{
				Log:       validLogCfg,
				Timeouts:  validTimeoutsCfg,
				Server:    validServerCfg,
				ProjectID: "proj",
				RedisAddr: "redis",
				RegID:     "reg",
				RegKeyID:  "key",
				Event:     validEventCfg,
			},
			expectedError: "missing required config section: registry",
		},
		{
			name: "missing registry base URL",
			cfg: &config{
				Log:       validLogCfg,
				Timeouts:  validTimeoutsCfg,
				Server:    validServerCfg,
				ProjectID: "proj",
				Registry:  &client.RegistryClientConfig{BaseURL: ""},
				RedisAddr: "redis",
				RegID:     "reg",
				RegKeyID:  "key",
				Event:     validEventCfg,
			},
			expectedError: "missing registry base URL",
		},
		{
			name: "missing project ID",
			cfg: &config{
				Log:       validLogCfg,
				Timeouts:  validTimeoutsCfg,
				Server:    validServerCfg,
				Registry:  validRegistryCfg,
				RedisAddr: "redis",
				RegID:     "reg",
				RegKeyID:  "key",
				Event:     validEventCfg,
			},
			expectedError: "missing project ID",
		},
		{
			name: "missing redis address",
			cfg: &config{
				Log:       validLogCfg,
				Timeouts:  validTimeoutsCfg,
				Server:    validServerCfg,
				ProjectID: "proj",
				Registry:  validRegistryCfg,
				RegID:     "reg",
				RegKeyID:  "key",
				Event:     validEventCfg,
			},
			expectedError: "missing redis address",
		},
		{
			name: "missing regID",
			cfg: &config{
				Log:       validLogCfg,
				Timeouts:  validTimeoutsCfg,
				Server:    validServerCfg,
				ProjectID: "proj",
				Registry:  validRegistryCfg,
				RedisAddr: "redis",
				RegKeyID:  "key",
				Event:     validEventCfg,
			},
			expectedError: "missing regId (Registry ID)",
		},
		{
			name: "missing regKeyID",
			cfg: &config{
				Log:       validLogCfg,
				Timeouts:  validTimeoutsCfg,
				Server:    validServerCfg,
				ProjectID: "proj",
				Registry:  validRegistryCfg,
				RedisAddr: "redis",
				RegID:     "reg",
				Event:     validEventCfg,
			},
			expectedError: "missing regKeyId (Registry Key ID for decryption)",
		},
		{
			name: "missing event config",
			cfg: &config{
				Log:       validLogCfg,
				Timeouts:  validTimeoutsCfg,
				Server:    validServerCfg,
				ProjectID: "proj",
				Registry:  validRegistryCfg,
				RedisAddr: "redis",
				RegID:     "reg",
				RegKeyID:  "key",
			},
			expectedError: "missing required config section: event",
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
