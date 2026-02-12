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
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/internal/api/admin"
	"github.com/google/dpi-accelerator-beckn-onix/internal/api/admin/handler"
	"github.com/google/dpi-accelerator-beckn-onix/internal/client"
	"github.com/google/dpi-accelerator-beckn-onix/internal/event"
	"github.com/google/dpi-accelerator-beckn-onix/internal/log"
	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"
	"github.com/google/dpi-accelerator-beckn-onix/internal/service"

	"gopkg.in/yaml.v3"

	"github.com/google/dpi-accelerator-beckn-onix/plugins/encrypter"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/beckn/beckn-onix/pkg/plugin/definition"
)

// config represents application configuration.
type config struct {
	Log      *log.Config                             `yaml:"log"`
	Timeouts *timeoutConfig                          `yaml:"timeouts"`
	Server   *serverConfig                           `yaml:"server"`
	DB       *repository.Config                      `yaml:"db"`
	NPClient *client.NPClientConfig                  `yaml:"npClient"`
	Admin    *service.AdminConfig                    `yaml:"admin"`
	Event    *event.Config                           `yaml:"event"`
	Setup    *service.RegistrySelfRegistrationConfig `yaml:"setup"`
}

type serverConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type timeoutConfig struct {
	Read     time.Duration `yaml:"read"`
	Write    time.Duration `yaml:"write"`
	Idle     time.Duration `yaml:"idle"`
	Shutdown time.Duration `yaml:"shutdown"`
}

// initConfig reads configuration from a YAML file.
func initConfig(filePath string) (*config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	var cfg config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config data: %w", err)
	}
	if cfg.NPClient == nil {
		c := client.DefaultNPClientConfig()
		cfg.NPClient = &c
	}
	if err := cfg.valid(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// valid checks if the configuration is valid.
func (c *config) valid() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.Log == nil {
		return fmt.Errorf("missing required config section: log")
	}
	if c.Server == nil {
		return fmt.Errorf("missing required config section: server")
	}
	if c.Timeouts == nil {
		return fmt.Errorf("missing required config section: timeouts")
	}
	if c.DB == nil {
		return fmt.Errorf("missing required config section: db")
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Admin == nil {
		return fmt.Errorf("missing required config section: admin")
	}
	if c.Admin.OperationRetryMax <= 0 {
		return fmt.Errorf("admin.OperationRetryMax must be greater than zero")
	}
	if c.Event == nil {
		return fmt.Errorf("missing required config section: event")
	}
	if c.Setup == nil {
		return fmt.Errorf("missing required config section: setup")
	}
	if c.Setup.KeyID == "" {
		return fmt.Errorf("encryptionKeyID is missing in setup config")
	}
	return nil
}

// run starts the HTTP server and handles graceful shutdown.
func run(ctx context.Context) error {
	cfg, err := initConfig(configPath)
	if err != nil {
		return err
	}
	if err := log.Setup(cfg.Log); err != nil {
		return err
	}
	db, dbCleanUp, err := newConnectionPool(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer func() {
		if err := dbCleanUp(); err != nil {
			slog.Error("failed to clean up database connection", "error", err)
		}
	}()
	encry, _, err := encrypter.New(ctx)
	if err != nil {
		return fmt.Errorf("failed to create signature validator: %w", err)
	}
	sm, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create secret manager client for encryption service: %w", err)
	}
	defer sm.Close()
	server, err := newServer(ctx, cfg, db, encry, sm)
	if err != nil {
		return err
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("Registry server starting...", "address", net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	//Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		slog.Error("FATAL: Registry server failed to start or encountered an error", "error", err)
		os.Exit(1)
	case sig := <-quit:
		slog.Info("Shutdown signal received", "signal", sig.String())
	}

	slog.Info("Attempting to shut down server gracefully...", "timeout", cfg.Timeouts.Shutdown.String())
	shutdownCtx, cancelShutdown := context.WithTimeout(ctx, cfg.Timeouts.Shutdown)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Graceful server shutdown failed", "error", err)
	} else {
		slog.Info("Registry server shut down gracefully.")
	}

	slog.Info("Registry service has stopped.")
	return nil
}

var configPath string
var newConnectionPool = repository.NewConnectionPool

func newServer(ctx context.Context, cfg *config, db *sql.DB, encyr definition.Encrypter, sm *secretmanager.Client) (*http.Server, error) {

	regRepo, err := repository.NewRegistry(db)
	if err != nil {
		slog.Error("Failed to create registry repository", "error", err)
		return nil, fmt.Errorf("failed to create registry repository: %w", err)
	}
	encSrv, err := service.NewEcryptionService(ctx, encyr, sm, cfg.Event.ProjectID, cfg.Setup.KeyID)
	if err != nil {
		slog.Error("Failed to create encryption service", "error", err)
		return nil, fmt.Errorf("failed to create encryption service: %w", err)
	}
	setup, err := service.NewRegistrySetupService(regRepo, encSrv, cfg.Setup)
	if err != nil {
		slog.Error("Failed to create registry setup service", "error", err)
		return nil, fmt.Errorf("failed to create registry setup service: %w", err)
	}
	if err := setup.SelfRegister(ctx); err != nil {
		slog.Error("Failed to self register", "error", err)
		return nil, fmt.Errorf("failed to self register: %w", err)
	}
	evPub, _, err := event.NewPublisher(ctx, cfg.Event)
	if err != nil {
		return nil, fmt.Errorf("failed to create event publisher: %w", err)
	}
	adminSrv, err := service.NewAdminService(regRepo,
		service.NewChallengeService(),
		encSrv,
		client.NewNPClient(*cfg.NPClient),
		evPub,
		cfg.Admin)
	if err != nil {
		slog.Error("Failed to create admin service", "error", err)
		return nil, fmt.Errorf("failed to create admin service: %w", err)
	}
	h, err := handler.NewAdminHandler(adminSrv)
	if err != nil {
		slog.Error("Failed to create admin handler", "error", err)
		return nil, fmt.Errorf("failed to create admin handler: %w", err)
	}
	return &http.Server{
		Addr:         net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)),
		Handler:      admin.NewRouter(h),
		ReadTimeout:  cfg.Timeouts.Read,
		WriteTimeout: cfg.Timeouts.Write,
		IdleTimeout:  cfg.Timeouts.Idle,
	}, nil
}

func main() {
	ctx := context.Background()
	configPath = os.Getenv("CONFIG_FILE")

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
