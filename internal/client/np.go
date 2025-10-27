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
	"log/slog"
	"net/http"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

const (
	onSubscribePath = "/on_subscribe"
)

// NPClientConfig holds configuration for the retryable HTTP client.
type NPClientConfig struct {
	Timeout time.Duration `yaml:"timeout"` // Timeout for each individual HTTP request attempt.
}

// DefaultNPClientConfig provides a sensible default configuration.
func DefaultNPClientConfig() NPClientConfig {
	return NPClientConfig{ //nolint:gomnd // Default configuration values
		Timeout: 10 * time.Second, // Timeout for each attempt
	}
}

type httpNPClient struct {
	client *http.Client
}

// NewNPClient creates a new NPClient that uses a retryable HTTP client.
func NewNPClient(cfg NPClientConfig) *httpNPClient {
	client := &http.Client{
		Timeout: cfg.Timeout,
	}
	return &httpNPClient{
		client: client,
	}
}

var jsonMarshal = json.Marshal

// OnSubscribe sends a request to the Network Participant's (NP) /on_subscribe endpoint.
// It handles request marshaling, sending the HTTP request with retries, and decoding the response.
func (c *httpNPClient) OnSubscribe(ctx context.Context, callbackURL string, request *model.OnSubscribeRequest) (*model.OnSubscribeResponse, error) {
	slog.InfoContext(ctx, "NPClient: Preparing /on_subscribe request", "url", callbackURL)

	requestBody, err := jsonMarshal(request)
	if err != nil {
		slog.ErrorContext(ctx, "NPClient: Failed to marshal /on_subscribe request", "error", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL+onSubscribePath, bytes.NewBuffer(requestBody))
	if err != nil {
		slog.ErrorContext(ctx, "NPClient: Failed to create /on_subscribe HTTP request", "error", err)
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	slog.InfoContext(ctx, "NPClient: Sending /on_subscribe request", "url", callbackURL)
	resp, err := c.client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "NPClient: Failed to send /on_subscribe request", "url", callbackURL, "error", err)
		return nil, fmt.Errorf("HTTP request to NP failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "NPClient: /on_subscribe callback returned non-OK status", "url", callbackURL, "status_code", resp.StatusCode)
		return nil, fmt.Errorf("NP callback failed with status %d", resp.StatusCode)
	}

	var onSubscribeResponse model.OnSubscribeResponse
	if err := json.NewDecoder(resp.Body).Decode(&onSubscribeResponse); err != nil {
		slog.ErrorContext(ctx, "NPClient: Failed to decode /on_subscribe response", "url", callbackURL, "error", err)
		return nil, fmt.Errorf("failed to decode NP response: %w", err)
	}

	slog.InfoContext(ctx, "NPClient: Successfully received /on_subscribe response", "url", callbackURL)
	return &onSubscribeResponse, nil
}
