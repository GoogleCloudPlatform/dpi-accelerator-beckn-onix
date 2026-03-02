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
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/internal/event"
	"github.com/google/dpi-accelerator-beckn-onix/internal/log"
	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"

	pubsubpb "cloud.google.com/go/pubsub/apiv1/pubsubpb"
	"cloud.google.com/go/pubsub/pstest"
	"github.com/google/go-cmp/cmp"
	"github.com/beckn/beckn-onix/pkg/plugin/definition"
	"google.golang.org/api/option"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc"
	"github.com/DATA-DOG/go-sqlmock"
)

const testdataDir = "testdata"

// Helper to set the global configPath for the duration of a test case in TestRunError
func setTestConfigPath(t *testing.T, path string) func() {
	t.Helper()
	originalConfigPath := configPath
	configPath = path
	return func() {
		configPath = originalConfigPath
	}
}

// mockSignValidator is a mock implementation of definition.SignValidator for testing.
type mockSignValidator struct {
}

func (m *mockSignValidator) Validate(ctx context.Context, body []byte, header string, publicKeyBase64 string) error {
	return nil
}

func TestInitConfigSuccess(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
		expectedConfig *config
	}{
		{
			name:     "valid config",
			filePath: filepath.Join(testdataDir, "config_valid.yaml"),
			expectedConfig: &config{
				Log:      &log.Config{Level: "INFO"},
				Server:   &serverConfig{Host: "localhost", Port: 8080},
				Timeouts: &timeoutConfig{Read: 5 * time.Second, Write: 10 * time.Second, Idle: 120 * time.Second, Shutdown: 15 * time.Second},
				DB: &repository.Config{
					User:           "user",
					Name:           "dbname",
					ConnectionName: "host:port",
				},
				Event: &event.Config{ProjectID: "test-project", TopicID: "test-topic"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := initConfig(tt.filePath)
			if err != nil {
				t.Fatalf("initConfig() error = %v, wantErr nil", err)
			}
			if cfg == nil {
				t.Fatal("initConfig() cfg is nil, want non-nil")
			}

			if diff := cmp.Diff(tt.expectedConfig, cfg); diff != "" {
				t.Errorf("initConfig() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInitConfigError(t *testing.T) {
	tests := []struct {
		name          string
		filePath      string
		expectedError string
	}{
		{
			name:          "file not found",
			filePath:      filepath.Join(testdataDir, "non_existent_config.yaml"),
			expectedError: "failed to read config file",
		},
		{
			name:          "malformed yaml",
			filePath:      filepath.Join(testdataDir, "config_malformed.yaml"),
			expectedError: "failed to unmarshal config data",
		},
		{
			name:          "missing server section",
			filePath:      filepath.Join(testdataDir, "config_missing_server_section.yaml"),
			expectedError: "missing required config section: server",
		},
		{
			name:          "missing log section",
			filePath:      filepath.Join(testdataDir, "config_missing_log_section.yaml"),
			expectedError: "missing required config section: log",
		},
		{
			name:          "missing timeouts section",
			filePath:      filepath.Join(testdataDir, "config_missing_timeouts_section.yaml"),
			expectedError: "missing required config section: timeouts",
		},
		{
			name:          "invalid server port zero",
			filePath:      filepath.Join(testdataDir, "config_invalid_port_zero.yaml"),
			expectedError: "invalid server port: 0",
		},
		{
			name:          "invalid server port negative",
			filePath:      filepath.Join(testdataDir, "config_invalid_port_negative.yaml"),
			expectedError: "invalid server port: -1",
		},
		{
			name:          "invalid server port too large",
			filePath:      filepath.Join(testdataDir, "config_invalid_port_toolarge.yaml"),
			expectedError: "invalid server port: 65536",
		},
		{
			name:          "missing event section",
			filePath:      filepath.Join(testdataDir, "config_missing_event_section.yaml"),
			expectedError: "missing required config section: event",
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

func TestConfigValidSuccess(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config
	}{
		{
			name: "valid config",
			cfg: &config{
				Log:      &log.Config{Level: "INFO"},
				Server:   &serverConfig{Host: "localhost", Port: 8080},
				Timeouts: &timeoutConfig{Read: 1 * time.Second, Write: 1 * time.Second, Idle: 1 * time.Second, Shutdown: 1 * time.Second},
				DB: &repository.Config{
					User:           "user",
					Name:           "dbname",
					ConnectionName: "host:port",
				},
				Event: &event.Config{ProjectID: "test", TopicID: "test"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.valid()
			if err != nil {
				t.Errorf("config.valid() error = %v, wantErr nil", err)
			}
		})
	}
}

func TestConfigValidError(t *testing.T) {
	// Base valid parts to construct error cases
	validLogCfg := &log.Config{Level: "INFO"}
	validTimeoutsCfg := &timeoutConfig{Read: 1 * time.Second, Write: 1 * time.Second, Idle: 1 * time.Second, Shutdown: 1 * time.Second}
	validServerCfg := &serverConfig{Host: "localhost", Port: 8080}
	validDBCfg := &repository.Config{
		User:           "user",
		Name:           "dbname",
		ConnectionName: "host:port",
	}
	validEventCfg := &event.Config{ProjectID: "test", TopicID: "test"}

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
			name:          "missing log config",
			cfg:           &config{Timeouts: validTimeoutsCfg, Server: validServerCfg, DB: validDBCfg, Event: validEventCfg},
			expectedError: "missing required config section: log",
		},
		{
			name:          "missing server config",
			cfg:           &config{Log: validLogCfg, Timeouts: validTimeoutsCfg, DB: validDBCfg, Event: validEventCfg},
			expectedError: "missing required config section: server",
		},
		{
			name:          "missing timeouts config",
			cfg:           &config{Log: validLogCfg, Server: validServerCfg, DB: validDBCfg, Event: validEventCfg},
			expectedError: "missing required config section: timeouts",
		},
		{
			name:          "invalid server port (0)",
			cfg:           &config{Log: validLogCfg, Timeouts: validTimeoutsCfg, Server: &serverConfig{Host: "localhost", Port: 0}, DB: validDBCfg, Event: validEventCfg},
			expectedError: "invalid server port: 0",
		},
		{
			name:          "invalid server port (-1)",
			cfg:           &config{Log: validLogCfg, Timeouts: validTimeoutsCfg, Server: &serverConfig{Host: "localhost", Port: -1}, DB: validDBCfg, Event: validEventCfg},
			expectedError: "invalid server port: -1",
		},
		{
			name:          "invalid server port (65536)",
			cfg:           &config{Log: validLogCfg, Timeouts: validTimeoutsCfg, Server: &serverConfig{Host: "localhost", Port: 65536}, DB: validDBCfg, Event: validEventCfg},
			expectedError: "invalid server port: 65536",
		},
		{
			name:          "missing event config",
			cfg:           &config{Log: validLogCfg, Timeouts: validTimeoutsCfg, Server: validServerCfg, DB: validDBCfg},
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

func TestNewServerSuccess(t *testing.T) {
	ctx := context.Background()
	_, clientOpts, cleanupPubsub := setUpTestPubsub(ctx, t, "test-topic")
	defer cleanupPubsub()

	cfg := &config{
		Log:      &log.Config{Level: "DEBUG"},
		Server:   &serverConfig{Host: "127.0.0.1", Port: 9090},
		Timeouts: &timeoutConfig{Read: 5 * time.Second, Write: 10 * time.Second, Idle: 15 * time.Second, Shutdown: 20 * time.Second},
		DB: &repository.Config{
			User:           "user",
			Name:           "dbname",
			ConnectionName: "host:port",
		},
		Event: &event.Config{ProjectID: testProject, TopicID: "test-topic", Opts: clientOpts},
	}

	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer mockDB.Close()

	mockSV := &mockSignValidator{}

	server, err := newServer(ctx, cfg, mockDB, mockSV)
	if err != nil {
		t.Fatalf("newServer() error = %v, wantErr nil", err)
	}
	if server == nil {
		t.Fatal("newServer() returned nil server with no error")
	}

	expectedAddr := net.JoinHostPort(cfg.Server.Host, fmt.Sprintf("%d", cfg.Server.Port))
	if server.Addr != expectedAddr {
		t.Errorf("server.Addr got %s, want %s", server.Addr, expectedAddr)
	}
	if server.ReadTimeout != cfg.Timeouts.Read {
		t.Errorf("server.ReadTimeout got %v, want %v", server.ReadTimeout, cfg.Timeouts.Read)
	}
	if server.WriteTimeout != cfg.Timeouts.Write {
		t.Errorf("server.WriteTimeout got %v, want %v", server.WriteTimeout, cfg.Timeouts.Write)
	}
	if server.IdleTimeout != cfg.Timeouts.Idle {
		t.Errorf("server.IdleTimeout got %v, want %v", server.IdleTimeout, cfg.Timeouts.Idle)
	}
	if server.Handler == nil {
		t.Error("server.Handler is nil, want non-nil router")
	}
}

func TestNewServerError(t *testing.T) {
	cfg := &config{ // A minimal valid config for other parts
		Log:      &log.Config{Level: "INFO"},
		Server:   &serverConfig{Host: "localhost", Port: 8080},
		Timeouts: &timeoutConfig{Read: 1 * time.Second, Write: 1 * time.Second, Idle: 1 * time.Second, Shutdown: 1 * time.Second},
		DB: &repository.Config{
			User:           "user",
			Name:           "dbname",
			ConnectionName: "host:port",
		},
		Event: &event.Config{ProjectID: "test", TopicID: "test"},
	}
	mockSV := &mockSignValidator{}

	tests := []struct {
		name          string
		db            *sql.DB // Allow passing nil DB
		sv            definition.SignValidator
		expectedError string
	}{
		{
			name:          "nil db for repository.NewRegistry",
			db:            nil,
			sv:            mockSV,
			expectedError: "failed to create registry repository: sql.DB is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := newServer(context.Background(), cfg, tt.db, tt.sv)
			if err == nil {
				t.Fatalf("newServer() error = nil, wantErr containing %q", tt.expectedError)
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("newServer() error = %q, want error containing %q", err.Error(), tt.expectedError)
			}
			if server != nil {
				t.Errorf("newServer() server = %v, want nil on error", server)
			}
		})
	}
}

func TestRunError(t *testing.T) {
	// Suppress os.Exit calls during tests of run()
	// This is a common pattern but has limitations.
	// It won't prevent the exit if it happens in a different goroutine
	// not managed by the test's main goroutine.
	// For the setup errors in run(), this should be effective.
	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()
	osExit = func(code int) {
		// In tests, we don't want to actually exit.
		// We can panic to stop the test here if an unexpected exit occurs.
		panic(fmt.Sprintf("os.Exit(%d) called during test", code))
	}

	tests := []struct {
		name          string
		configRelPath string
		expectedError string
	}{
		{
			name:          "initConfig fails - file not found",
			configRelPath: "non_existent_for_run.yaml", // Does not need to be in testdata
			expectedError: "failed to read config file",
		},
		{
			name:          "initConfig fails - malformed yaml",
			configRelPath: "config_malformed.yaml",
			expectedError: "failed to unmarshal config data",
		},
		{
			name:          "initConfig fails - missing server section",
			configRelPath: "config_missing_server_section.yaml",
			expectedError: "missing required config section: server",
		},
		{
			name:          "initConfig fails - invalid port",
			configRelPath: "config_invalid_port_zero.yaml",
			expectedError: "invalid server port: 0",
		},
		{
			name:          "log.Setup fails - invalid log level",
			configRelPath: "config_invalid_log_level.yaml",
			expectedError: "invalid log level: INVALIDLEVEL", // This error comes from log.Setup via log.valid
		},
		{
			name:          "initConfig fails - missing event section",
			configRelPath: "config_missing_event_section.yaml",
			expectedError: "missing required config section: event",
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testConfigFilePath string
			if strings.HasSuffix(tt.configRelPath, ".yaml") && tt.configRelPath != "non_existent_for_run.yaml" {
				testConfigFilePath = filepath.Join(testdataDir, tt.configRelPath)
			} else {
				testConfigFilePath = tt.configRelPath
			}

			cleanup := setTestConfigPath(t, testConfigFilePath)
			defer cleanup()
			err := run(ctx)

			if err == nil {
				t.Fatalf("run() with config %s, err = nil, wantErr containing %q", tt.configRelPath, tt.expectedError)
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("run() with config %s, error = %q, want error containing %q", tt.configRelPath, err.Error(), tt.expectedError)
			}
		})
	}
}

const testProject = "test-project"

func setUpTestPubsub(ctx context.Context, t *testing.T, topicID string, opts ...pstest.ServerReactorOption) (*pstest.Server, []option.ClientOption, func()) {
	t.Helper()
	psSrv := pstest.NewServer(opts...)
	topic, err := psSrv.GServer.CreateTopic(ctx, &pubsubpb.Topic{Name: "projects/" + testProject + "/topics/" + topicID})
	if err != nil {
		t.Fatalf("failed to create pubsub topic: %v, err: %v", topic.GetName(), err)
	}

	conn, err := grpc.NewClient(psSrv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to create grpc client: %v, err: %v", psSrv.Addr, err)
	}

	return psSrv, []option.ClientOption{option.WithGRPCConn(conn)}, func() { conn.Close(); psSrv.Close() }
}

func TestRun_NewServerFails_Error(t *testing.T) {
	oldOsExit := osExit
	defer func() { osExit = oldOsExit }()
	osExit = func(code int) {
		panic(fmt.Sprintf("os.Exit(%d) called during test", code))
	}

	ctx := context.Background()
	configRelPath := "config_valid_for_newserver_fail.yaml"
	expectedError := "failed to open database connection: db connection error"

	testConfigFilePath := filepath.Join(testdataDir, configRelPath)
	cleanup := setTestConfigPath(t, testConfigFilePath)
	defer cleanup()

	// Mock NewConnectionPool to simulate a DB connection failure.
	originalNewConnectionPool := newConnectionPool
	newConnectionPool = func(ctx context.Context, cfg *repository.Config) (*sql.DB, func() error, error) {
		return nil, func() error { return nil }, fmt.Errorf("db connection error")
	}
	defer func() { newConnectionPool = originalNewConnectionPool }()

	err := run(ctx)

	if err == nil {
		t.Fatalf("run() with config %s, err = nil, wantErr containing %q", configRelPath, expectedError)
	}
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("run() with config %s, error = %q, want error containing %q", configRelPath, err.Error(), expectedError)
	}
}

// To allow mocking os.Exit in tests for run()
var osExit = os.Exit
