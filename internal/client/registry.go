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

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

const (
	lookupPath        = "/lookup"
	subscribePath     = "/subscribe"
	operationsPathFmt = "/operations/%s" // Format string for operation ID
)

// RegistryClientConfig holds configuration for the retryable HTTP client for the Registry.
type RegistryClientConfig struct {
	Timeout             time.Duration `yaml:"timeout"` // Timeout for each individual HTTP request attempt.
	BaseURL             string        `yaml:"baseURL"` // Base URL of the registry service (e.g., "http://localhost:8080")
	MaxIdleConns        int           `yaml:"maxIdleConns"`
	MaxIdleConnsPerHost int           `yaml:"maxIdleConnsPerHost"`
	MaxConnsPerHost     int           `yaml:"maxConnsPerHost"`
	IdleConnTimeout     time.Duration `yaml:"idleConnTimeout"`
}

type httpRegistryClient struct {
	client  *http.Client
	baseURL string
}

// NewRegistryClient creates a new RegistryClient that uses a retryable HTTP client.
func NewRegistryClient(cfg *RegistryClientConfig) (*httpRegistryClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("RegistryClientConfig cannot be nil")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("BaseURL cannot be empty in RegistryClientConfig")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second // Provide a default timeout if not configured
	}

	// Configure a custom transport with connection pooling.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// If MaxIdleConnsPerHost is not set, it defaults to http.DefaultMaxIdleConnsPerHost (currently 2).
	if cfg.MaxIdleConnsPerHost > 0 {
		transport.MaxIdleConnsPerHost = cfg.MaxIdleConnsPerHost
	}
	// If MaxIdleConns is not set, it defaults to 100.
	if cfg.MaxIdleConns > 0 {
		transport.MaxIdleConns = cfg.MaxIdleConns
	}
	// If MaxConnsPerHost is not set, there is no limit.
	if cfg.MaxConnsPerHost > 0 {
		transport.MaxConnsPerHost = cfg.MaxConnsPerHost
	}
	// If IdleConnTimeout is not set, it defaults to 90 seconds.
	if cfg.IdleConnTimeout > 0 {
		transport.IdleConnTimeout = cfg.IdleConnTimeout
	}

	client := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}
	return &httpRegistryClient{
		client:  client,
		baseURL: cfg.BaseURL,
	}, nil
}

// doAPIRequest is a helper function to handle common logic for making API requests.
func (c *httpRegistryClient) doAPIRequest(
	ctx context.Context,
	method string,
	pathFormat string,
	pathArgs []any,
	requestData any, // Will be marshalled to JSON if not nil
	responseData any, // Pointer to struct to unmarshal JSON response
	expectedStatusCode int,
	logAction string, // e.g., "POST /subscribe"
	authHeader string,
) error {
	fullURL := c.baseURL + fmt.Sprintf(pathFormat, pathArgs...)
	slog.DebugContext(ctx, "RegistryClient: Preparing request", "action", logAction, "url", fullURL)

	var reqBodyReader io.Reader
	if requestData != nil {
		requestBytes, err := jsonMarshal(requestData)
		if err != nil {
			slog.ErrorContext(ctx, "RegistryClient: Failed to marshal request", "action", logAction, "error", err)
			return fmt.Errorf("failed to marshal %s request: %w", logAction, err)
		}
		reqBodyReader = bytes.NewBuffer(requestBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBodyReader)
	if err != nil {
		slog.ErrorContext(ctx, "RegistryClient: Failed to create HTTP request", "action", logAction, "error", err)
		return fmt.Errorf("failed to create HTTP request for %s: %w", logAction, err)
	}
	if authHeader != "" {
		req.Header.Set(model.AuthHeaderSubscriber, authHeader)
	}
	if requestData != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	slog.DebugContext(ctx, "RegistryClient: Sending request", "action", logAction, "url", fullURL)
	resp, err := c.client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "RegistryClient: Failed to send request", "action", logAction, "url", fullURL, "error", err)
		return fmt.Errorf("HTTP request to Registry %s failed: %w", logAction, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.ErrorContext(ctx, "RegistryClient: Failed to read response body", "action", logAction, "url", fullURL, "error", err)
		return fmt.Errorf("failed to read Registry %s response body: %w", logAction, err)
	}

	if resp.StatusCode != expectedStatusCode {
		slog.WarnContext(ctx, "RegistryClient: Endpoint returned unexpected status", "action", logAction, "url", fullURL, "status_code", resp.StatusCode, "expected_status_code", expectedStatusCode, "response_body", string(responseBody))
		return fmt.Errorf("registry %s failed with status %d: %s", logAction, resp.StatusCode, string(responseBody))
	}

	if responseData != nil {
		if err := json.Unmarshal(responseBody, responseData); err != nil {
			slog.ErrorContext(ctx, "RegistryClient: Failed to unmarshal response", "action", logAction, "url", fullURL, "error", err, "response_body", string(responseBody))
			return fmt.Errorf("failed to unmarshal Registry %s response: %w", logAction, err)
		}
	}

	slog.DebugContext(ctx, "RegistryClient: Successfully received response", "action", logAction, "url", fullURL)
	return nil
}

// Lookup sends a POST request to the Registry's /lookup endpoint.
func (c *httpRegistryClient) Lookup(ctx context.Context, request *model.Subscription) ([]model.Subscription, error) {
	var subscriptions []model.Subscription
	err := c.doAPIRequest(ctx, http.MethodPost, lookupPath, nil, request, &subscriptions, http.StatusOK, "POST /lookup", "")
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// CreateSubscription sends a POST request to the Registry's /subscribe endpoint to create a new subscription.
func (c *httpRegistryClient) CreateSubscription(ctx context.Context, request *model.SubscriptionRequest) (*model.SubscriptionResponse, error) {
	var subResponse model.SubscriptionResponse
	err := c.doAPIRequest(ctx, http.MethodPost, subscribePath, nil, request, &subResponse, http.StatusOK, "POST /subscribe", "")
	if err != nil {
		return nil, err
	}
	slog.DebugContext(ctx, "RegistryClient: Successfully received POST /subscribe response", "url", c.baseURL+subscribePath, "message_id", subResponse.MessageID)
	return &subResponse, nil
}

// UpdateSubscription sends a PATCH request to the Registry's /subscribe endpoint to update an existing subscription.
func (c *httpRegistryClient) UpdateSubscription(ctx context.Context, request *model.SubscriptionRequest, authHeader string) (*model.SubscriptionResponse, error) {
	var subResponse model.SubscriptionResponse
	err := c.doAPIRequest(ctx, http.MethodPatch, subscribePath, nil, request, &subResponse, http.StatusOK, "PATCH /subscribe", authHeader)
	if err != nil {
		return nil, err
	}
	slog.DebugContext(ctx, "RegistryClient: Successfully received PATCH /subscribe response", "url", c.baseURL+subscribePath, "message_id", subResponse.MessageID)
	return &subResponse, nil
}

// GetOperation sends a GET request to the Registry's /operations/{operation_id} endpoint to retrieve LRO status.
func (c *httpRegistryClient) GetOperation(ctx context.Context, operationID string) (*model.LRO, error) {
	var lro model.LRO
	logAction := fmt.Sprintf("GET /operations/%s", operationID)
	err := c.doAPIRequest(ctx, http.MethodGet, operationsPathFmt, []any{operationID}, nil, &lro, http.StatusOK, logAction, "")
	if err != nil {
		return nil, err
	}
	slog.DebugContext(ctx, "RegistryClient: Successfully received GET /operations response", "url", c.baseURL+fmt.Sprintf(operationsPathFmt, operationID), "operation_id", lro.OperationID)
	return &lro, nil
}
