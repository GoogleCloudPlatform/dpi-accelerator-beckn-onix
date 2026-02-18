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

	"github.com/google/dpi-accelerator-beckn-onix/internal/api/gateway"
	"github.com/google/dpi-accelerator-beckn-onix/internal/api/gateway/handler"
	"github.com/google/dpi-accelerator-beckn-onix/internal/client"
	"github.com/google/dpi-accelerator-beckn-onix/internal/log"
	"github.com/google/dpi-accelerator-beckn-onix/internal/service"
	keyManager "github.com/google/dpi-accelerator-beckn-onix/plugins/inmemorysecretkeymanager"
	"github.com/google/dpi-accelerator-beckn-onix/plugins/rediscache"

	yaml "gopkg.in/yaml.v3"

	beckn "github.com/beckn/beckn-onix/core/module/client"
	"github.com/beckn/beckn-onix/pkg/plugin/implementation/signer"
	"github.com/beckn/beckn-onix/pkg/plugin/implementation/signvalidator"
)

// config represents application configuration.
type config struct {
	Log                      *log.Config                  `yaml:"log"`
	Timeouts                 *timeoutConfig               `yaml:"timeouts"`
	Server                   *serverConfig                `yaml:"server"`
	ProjectID                string                       `yaml:"projectID"`
	KeyManagerCacheTTL       *keyManager.CacheTTL         `yaml:"keyManagerCacheTTL"`
	Registry                 *client.RegistryClientConfig `yaml:"registry"`
	RedisAddr                string                       `yaml:"redisAddr"`
	MaxConcurrentFanoutTasks int                          `yaml:"maxConcurrentFanoutTasks"`
	TaskQueueWorkersCount    int                          `yaml:"taskQueueWorkersCount"`
	TaskQueueBufferSize      int                          `yaml:"taskQueueBufferSize"`
	SubscriberID             string                       `yaml:"subscriberID"`
	HTTPClientRetry          *service.RetryConfig         `yaml:"httpClientRetry"`
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
	if c.SubscriberID == "" {
		return fmt.Errorf("missing subscriber ID")
	}
	if c.HTTPClientRetry == nil {
		slog.Warn("Config validation: httpClientRetry section missing, using default retry values.")
		// Provide default values or handle as an error if strict config is required
		c.HTTPClientRetry = &service.RetryConfig{RetryMax: 1, RetryWaitMin: 1 * time.Second, RetryWaitMax: 30 * time.Second}
	}
	if c.KeyManagerCacheTTL == nil {
		slog.Warn("Config validation: keyManagerCacheTTL section missing, using default retry values.")
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

	// Initialize Signature Validator (used by TxnSignValidator)
	sv, svClose, err := signvalidator.New(ctx, &signvalidator.Config{})
	if err != nil {
		return fmt.Errorf("failed to create signature validator: %w", err)
	}
	if svClose != nil {
		defer func() {
			if err := svClose(); err != nil {
				slog.ErrorContext(ctx, "failed to close signature validator", "error", err)
			}
		}()
	}

	redis, closeRedis, err := rediscache.New(ctx, map[string]string{"addr": cfg.RedisAddr})
	if err != nil {
		return fmt.Errorf("failed to create redis cache: %w", err)
	}
	defer func() {
		if err := closeRedis(); err != nil {
			slog.ErrorContext(ctx, "failed to close redis connection", "error", err)
		}
	}()
	rClient := beckn.NewRegisteryClient(&beckn.Config{RegisteryURL: cfg.Registry.BaseURL})

	keyManagerConfig := &keyManager.Config{
		ProjectID: cfg.ProjectID,
		CacheTTL: keyManager.CacheTTL{
			PrivateKeysSeconds: cfg.KeyManagerCacheTTL.PrivateKeysSeconds,
			PublicKeysSeconds:  cfg.KeyManagerCacheTTL.PublicKeysSeconds,
		},
	}

	km, closeKM, err := keyManager.New(ctx, redis, rClient, keyManagerConfig)
	if err != nil {
		return fmt.Errorf("failed to create secrets key manager: %w", err)
	}
	defer func() {
		if err := closeKM(); err != nil {
			slog.ErrorContext(ctx, "failed to close key manager", "error", err)
		}
	}()

	signer, _, err := signer.New(ctx, &signer.Config{})
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}

	// Initialize TxnSignValidator
	txnValidator, err := service.NewTxnSignValidator(sv, km)
	if err != nil {
		return fmt.Errorf("failed to create transaction sign validator: %w", err)
	}

	authGen, err := service.NewAuthGenService(km, signer)
	if err != nil {
		return fmt.Errorf("failed to create auth gen service: %w", err)
	}

	pTaskProcessor, err := service.NewProxyTaskProcessor(authGen, cfg.SubscriberID, *cfg.HTTPClientRetry)
	if err != nil {
		return fmt.Errorf("failed to create proxy task processor: %w", err)
	}
	registryClient, err := client.NewRegistryClient(cfg.Registry)
	if err != nil {
		return fmt.Errorf("failed to create registry client: %w", err)
	}
	channelTaskQ, err := service.NewChannelTaskQueue(cfg.TaskQueueWorkersCount, ctx, pTaskProcessor, nil, cfg.TaskQueueBufferSize) // Lookup processor will be set later
	if err != nil {
		return fmt.Errorf("failed to create channel task queue: %w", err)
	}
	channelTaskQ.StartWorkers()
	defer channelTaskQ.StopWorkers() // Add to graceful shutdown logic

	lTaskProcessor, err := service.NewChannelLookupProcessor(registryClient, authGen, channelTaskQ, cfg.SubscriberID, cfg.MaxConcurrentFanoutTasks)
	if err != nil {
		return fmt.Errorf("failed to create lookup task processor: %w", err)
	}
	channelTaskQ.SetLookupProcessor(lTaskProcessor)

	// Initialize Gateway Handler
	gwHandler, err := handler.NewGatewayHandler(txnValidator, channelTaskQ)
	if err != nil {
		return fmt.Errorf("failed to create gateway handler: %w", err)
	}

	// Initialize HTTP Server
	server := &http.Server{
		Addr:         net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)),
		Handler:      gateway.NewRouter(gwHandler),
		ReadTimeout:  cfg.Timeouts.Read,
		WriteTimeout: cfg.Timeouts.Write,
		IdleTimeout:  cfg.Timeouts.Idle,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("Gateway server starting...", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		slog.Error("FATAL: Gateway server failed to start or encountered an error", "error", err)
		os.Exit(1) // Consider returning error instead of os.Exit for better testability
	case sig := <-quit:
		slog.Info("Shutdown signal received", "signal", sig.String())
	}

	slog.Info("Attempting to shut down Gateway server gracefully...", "timeout", cfg.Timeouts.Shutdown.String())
	shutdownCtx, cancelShutdown := context.WithTimeout(ctx, cfg.Timeouts.Shutdown)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Graceful Gateway server shutdown failed", "error", err)
	} else {
		slog.Info("Gateway server shut down gracefully.")
	}

	slog.Info("Gateway service has stopped.")
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
