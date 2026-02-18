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

	"github.com/google/dpi-accelerator-beckn-onix/internal/api/subscriber/handler"
	"github.com/google/dpi-accelerator-beckn-onix/internal/api/subscriber"
	"github.com/google/dpi-accelerator-beckn-onix/internal/client"
	"github.com/google/dpi-accelerator-beckn-onix/internal/event"
	"github.com/google/dpi-accelerator-beckn-onix/internal/log"
	"github.com/google/dpi-accelerator-beckn-onix/internal/service"
	decryption "github.com/google/dpi-accelerator-beckn-onix/plugins/decrypter"
	keyManager "github.com/google/dpi-accelerator-beckn-onix/plugins/inmemorysecretkeymanager"
	"github.com/google/dpi-accelerator-beckn-onix/plugins/rediscache"
	becknclient "github.com/beckn/beckn-onix/core/module/client"
	"github.com/beckn/beckn-onix/pkg/plugin/implementation/signer"
	yaml "gopkg.in/yaml.v3"
)

// config represents application configuration for the subscriber service.
type config struct {
	Log                *log.Config                  `yaml:"log"`
	Timeouts           *timeoutConfig               `yaml:"timeouts"`
	Server             *serverConfig                `yaml:"server"`
	ProjectID          string                       `yaml:"projectID"`
	KeyManagerCacheTTL *keyManager.CacheTTL         `yaml:"keyManagerCacheTTL"`
	Registry           *client.RegistryClientConfig `yaml:"registry"`
	RedisAddr          string                       `yaml:"redisAddr"`
	RegID              string                       `yaml:"regID"`    // Registry's ID
	RegKeyID           string                       `yaml:"regKeyID"` // Registry's public key ID for decryption
	Event              *event.Config                `yaml:"event"`
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
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Registry == nil {
		return fmt.Errorf("missing required config section: registry")
	}
	if c.Registry.BaseURL == "" {
		return fmt.Errorf("missing registry base URL")
	}
	if c.ProjectID == "" {
		return fmt.Errorf("missing project ID")
	}
	if c.RedisAddr == "" {
		return fmt.Errorf("missing redis address")
	}
	if c.RegID == "" {
		return fmt.Errorf("missing regId (Registry ID)")
	}
	if c.RegKeyID == "" {
		return fmt.Errorf("missing regKeyId (Registry Key ID for decryption)")
	}
	if c.Event == nil {
		return fmt.Errorf("missing required config section: event")
	}
	if c.KeyManagerCacheTTL == nil {
		slog.Warn("Config validation: cacheTTL section missing, using default retry values.")
		// Provide default values or handle as an error if strict config is required
		c.KeyManagerCacheTTL = &keyManager.CacheTTL{PrivateKeysSeconds: 5, PublicKeysSeconds: 3600}
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

	redis, closeRedis, err := rediscache.New(ctx, map[string]string{"addr": cfg.RedisAddr})
	if err != nil {
		return fmt.Errorf("failed to create redis cache: %w", err)
	}
	defer func() {
		if err := closeRedis(); err != nil {
			slog.Error("failed to close redis", "error", err)
		}
	}()

	becknRegClient := becknclient.NewRegisteryClient(&becknclient.Config{RegisteryURL: cfg.Registry.BaseURL})
	keyManagerConfig := &keyManager.Config{
		ProjectID: cfg.ProjectID,
		CacheTTL: keyManager.CacheTTL{
			PrivateKeysSeconds: cfg.KeyManagerCacheTTL.PrivateKeysSeconds,
			PublicKeysSeconds:  cfg.KeyManagerCacheTTL.PublicKeysSeconds,
		},
	}

	km, closeKM, err := keyManager.New(ctx, redis, becknRegClient, keyManagerConfig)
	if err != nil {
		return fmt.Errorf("failed to create secrets key manager: %w", err)
	}
	defer func() {
		if err := closeKM(); err != nil {
			slog.Error("failed to close key manager", "error", err)
		}
	}()

	// Initialize Decrypter
	dec, _, err := decryption.New(ctx)
	if err != nil {
		return fmt.Errorf("failed to create decrypter: %w", err)
	}

	registryClient, err := client.NewRegistryClient(cfg.Registry)
	if err != nil {
		return fmt.Errorf("failed to create registry client: %w", err)
	}

	signer, sCloser, err := signer.New(ctx, &signer.Config{})
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}
	defer func() {
		if err := sCloser(); err != nil {
			slog.Error("failed to close signer", "error", err)
		}
	}()

	evPub, close, err := event.NewPublisher(ctx, cfg.Event)
	if err != nil {
		return fmt.Errorf("failed to create event publisher: %w", err)
	}
	defer close()

	authGen, err := service.NewAuthGenService(km, signer)
	if err != nil {
		return fmt.Errorf("failed to create auth gen service: %w", err)
	}
	// Initialize Subscriber Service
	subService, err := service.NewSubscriberService(registryClient, km, dec, evPub, authGen, cfg.RegID, cfg.RegKeyID)
	if err != nil {
		return fmt.Errorf("failed to create subscriber service: %w", err)
	}

	// Initialize Subscriber Handler
	subHandler, err := handler.NewSubscriberHandler(subService)
	if err != nil {
		return fmt.Errorf("failed to create subscriber handler: %w", err)
	}

	// Initialize HTTP Server
	server := &http.Server{
		Addr:         net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)),
		Handler:      subscriber.NewRouter(subHandler),
		ReadTimeout:  cfg.Timeouts.Read,
		WriteTimeout: cfg.Timeouts.Write,
		IdleTimeout:  cfg.Timeouts.Idle,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("Subscriber server starting...", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		slog.Error("FATAL: Subscriber server failed to start or encountered an error", "error", err)
		os.Exit(1)
	case sig := <-quit:
		slog.Info("Shutdown signal received", "signal", sig.String())
	}

	slog.Info("Attempting to shut down Subscriber server gracefully...", "timeout", cfg.Timeouts.Shutdown.String())
	shutdownCtx, cancelShutdown := context.WithTimeout(ctx, cfg.Timeouts.Shutdown)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Graceful Subscriber server shutdown failed", "error", err)
	} else {
		slog.Info("Subscriber server shut down gracefully.")
	}

	slog.Info("Subscriber service has stopped.")
	return nil
}

var configPath string

func main() {
	ctx := context.Background()
	configPath = os.Getenv("CONFIG_FILE")
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
