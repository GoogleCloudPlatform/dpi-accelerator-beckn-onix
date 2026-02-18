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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"github.com/hashicorp/go-retryablehttp"
)

// httpClient defines an interface for making HTTP requests, allowing for
// standard or retryable clients.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// RetryConfig holds configuration for the retryable HTTP client.
type RetryConfig struct {
	RetryMax            int           `yaml:"retryMax"`            // Maximum number of retries.
	RetryWaitMin        time.Duration `yaml:"waitMin"`             // Minimum time to wait before retrying.
	RetryWaitMax        time.Duration `yaml:"waitMax"`             // Maximum time to wait before retrying.
	Timeout             time.Duration `yaml:"timeout"`             // Timeout for each HTTP request.
	MaxIdleConns        int           `yaml:"maxIdleConns"`        // Maximum total idle connections.
	MaxIdleConnsPerHost int           `yaml:"maxIdleConnsPerHost"` // Maximum idle connections per host.
	MaxConnsPerHost     int           `yaml:"maxConnsPerHost"`     // Maximum connections per host.
	IdleConnTimeout     time.Duration `yaml:"idleConnTimeout"`     // Timeout for idle connections.
}

// proxyTaskProcessor makes HTTP POST calls for asynchronous proxy tasks.
type proxyTaskProcessor struct {
	client httpClient // Changed from *http.Client to httpClient interface
	auth   authGen
	keyID  string
}

// NewProxyTaskProcessor creates a new proxyTaskProcessor.
func NewProxyTaskProcessor(auth authGen, keyID string, retryCfg RetryConfig) (*proxyTaskProcessor, error) {
	if auth == nil {
		slog.Error("NewProxyTaskProcessor: authGen cannot be nil")
		return nil, errors.New("authGen cannot be nil")
	}
	if keyID == "" {
		slog.Error("NewProxyTaskProcessor: keyID cannot be empty")
		return nil, errors.New("keyID cannot be empty")
	}

	// Configure a custom transport with connection pooling.
	// Use the default values if no config given.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// If MaxIdleConnsPerHost is not set, it defaults to http.DefaultMaxIdleConnsPerHost (currently 2).
	if retryCfg.MaxIdleConnsPerHost > 0 {
		transport.MaxIdleConnsPerHost = retryCfg.MaxIdleConnsPerHost
	}
	// If MaxIdleConns is not set, it defaults to 100.
	if retryCfg.MaxIdleConns > 0 {
		transport.MaxIdleConns = retryCfg.MaxIdleConns
	}
	// If MaxConnsPerHost is not set, there is no limit.
	if retryCfg.MaxConnsPerHost > 0 {
		transport.MaxConnsPerHost = retryCfg.MaxConnsPerHost
	}
	// If IdleConnTimeout is not set, it defaults to 90 seconds.
	if retryCfg.IdleConnTimeout > 0 {
		transport.IdleConnTimeout = retryCfg.IdleConnTimeout
	}

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = retryCfg.RetryMax
	retryClient.RetryWaitMin = retryCfg.RetryWaitMin
	retryClient.RetryWaitMax = retryCfg.RetryWaitMax
	retryClient.Logger = nil

	// Set the underlying http.Client to use our custom transport and timeout.
	retryClient.HTTPClient = &http.Client{
		Transport: transport,
		Timeout:   retryCfg.Timeout,
	}

	return &proxyTaskProcessor{client: retryClient.StandardClient(), auth: auth, keyID: keyID}, nil
}

// validateTask checks if the AsyncTask is valid for processing.
func (p *proxyTaskProcessor) validateTask(ctx context.Context, task *model.AsyncTask) error {
	if task == nil {
		slog.ErrorContext(ctx, "ProxyTaskProcessor: async task cannot be nil")
		return errors.New("async task cannot be nil")
	}
	if task.Target == nil {
		slog.ErrorContext(ctx, "ProxyTaskProcessor: async task target URL cannot be nil")
		return errors.New("async task target URL cannot be nil")
	}
	if task.Headers == nil {
		slog.ErrorContext(ctx, "ProxyTaskProcessor: async task headers cannot be nil")
		return errors.New("async task headers cannot be nil")
	}
	return nil
}

// httpReq creates and configures an HTTP request from the AsyncTask.
func (p *proxyTaskProcessor) httpReq(ctx context.Context, task *model.AsyncTask) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, task.Target.String(), bytes.NewReader(task.Body))
	if err != nil {
		slog.ErrorContext(ctx, "ProxyTaskProcessor: Failed to create HTTP request", "error", err, "target", task.Target.String())
		return nil, fmt.Errorf("failed to create HTTP request for %s: %w", task.Target.String(), err)
	}
	req.Header = task.Headers.Clone()
	// Ensure Content-Type is set if body is present and not already in headers.
	if len(task.Body) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Only attempt to add auth header if it's not already present.
	if req.Header.Get(model.AuthHeaderGateway) != "" {
		return req, nil
	}
	slog.InfoContext(ctx, "ProxyTaskProcessor: Generating auth header", "target", task.Target.String(), "key_id", p.keyID)
	authHeader, err := p.auth.AuthHeader(ctx, task.Body, p.keyID)
	if err != nil {
		slog.ErrorContext(ctx, "ProxyTaskProcessor: Failed to generate auth header", "error", err)
		return nil, fmt.Errorf("failed to generate auth header: %w", err)
	}
	req.Header.Set(model.AuthHeaderGateway, authHeader)
	return req, nil
}

// proxy sends the HTTP request, reads, and parses the response.
func (p *proxyTaskProcessor) proxy(ctx context.Context, req *http.Request) error {
	targetURLStr := req.URL.String()
	resp, err := p.client.Do(req)

	if err != nil {
		slog.ErrorContext(ctx, "ProxyTaskProcessor: HTTP request failed", "error", err, "target", targetURLStr)
		return fmt.Errorf("HTTP request to %s failed: %w", targetURLStr, err)
	}
	defer resp.Body.Close()

	slog.InfoContext(ctx, "ProxyTaskProcessor: Received response", "target", targetURLStr, "status_code", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		respBodyBytes, _ := io.ReadAll(resp.Body) // Read body for error context
		slog.ErrorContext(ctx, "ProxyTaskProcessor: Unexpected HTTP status code", "target", targetURLStr, "status_code", resp.StatusCode, "response_body", string(respBodyBytes))
		return fmt.Errorf("unexpected status code %d from %s. Body: %s", resp.StatusCode, targetURLStr, string(respBodyBytes))
	}

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.ErrorContext(ctx, "ProxyTaskProcessor: Failed to read response body", "error", err, "target", targetURLStr)
		return fmt.Errorf("failed to read response body from %s: %w", targetURLStr, err)
	}

	var txnResponse model.TxnResponse
	if err := json.Unmarshal(respBodyBytes, &txnResponse); err != nil {
		slog.ErrorContext(ctx, "ProxyTaskProcessor: Failed to unmarshal response body into TxnResponse", "error", err, "target", targetURLStr, "response_body", string(respBodyBytes))
		return fmt.Errorf("failed to unmarshal response body from %s into model.TxnResponse: %w. Body: %s", targetURLStr, err, string(respBodyBytes))
	}
	if txnResponse.Message.Ack.Status != model.StatusACK {
		slog.WarnContext(ctx, "ProxyTaskProcessor: Response status is not ACK", "target", targetURLStr, "ack_status", txnResponse.Message.Ack.Status, "response_message", txnResponse.Message)
		errMsg := "response status is not ACK"
		if txnResponse.Message.Error != nil {
			errMsg = fmt.Sprintf("response status is NACK from %s: Code=%s, Message=%s", targetURLStr, txnResponse.Message.Error.Code, txnResponse.Message.Error.Message)
		}
		return errors.New(errMsg)
	}
	return nil
}

// Process handles the given asynchronous task by making an HTTP POST request
// to the task's target URL. It expects a 200 OK response with a model.TxnResponse
// body indicating an ACK status.
func (p *proxyTaskProcessor) Process(ctx context.Context, task *model.AsyncTask) error {
	if err := p.validateTask(ctx, task); err != nil {
		return err
	}
	slog.InfoContext(ctx, "ProxyTaskProcessor: Processing task", "target", task.Target.String(), "type", task.Type)

	req, err := p.httpReq(ctx, task)
	if err != nil {
		return err
	}

	if err := p.proxy(ctx, req); err != nil {
		return err
	}

	slog.InfoContext(ctx, "ProxyTaskProcessor: Task processed successfully and received ACK", "target", task.Target.String())
	return nil
}
