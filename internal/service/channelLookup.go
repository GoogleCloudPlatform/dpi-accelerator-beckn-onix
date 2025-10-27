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
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// taskQueuer defines the interface for queueing tasks.
type taskQueuer interface {
	QueueTxn(ctx context.Context, reqCtx *model.Context, msg []byte, h http.Header) (*model.AsyncTask, error)
}

// authGen defines the interface for generrating auth header.
type authGen interface {
	AuthHeader(ctx context.Context, body []byte, keyID string) (string, error)
}

// lookupClient defines the interface for looking up subscriptions.
type lookupClient interface {
	Lookup(ctx context.Context, request *model.Subscription) ([]model.Subscription, error)
}

// channelLookupProcessor handles tasks that require looking up subscribers
// and then fanning out proxy tasks to them.
type channelLookupProcessor struct {
	subID          string
	maxProxyTasks  int // Maximum number of proxy tasks to generate from a lookup
	registryClient lookupClient
	authGen        authGen
	taskQueuer     taskQueuer
}

// NewLookupTaskProcessor creates a new LookupTaskProcessor.
func NewChannelLookupProcessor(registryClient lookupClient, authGen authGen, tq taskQueuer, subID string, maxProxyTasks int) (*channelLookupProcessor, error) {
	if registryClient == nil {
		slog.Error("NewLookupTaskProcessor: registryClient cannot be nil")
		return nil, fmt.Errorf("registryClient cannot be nil")
	}

	if authGen == nil {
		slog.Error("NewLookupTaskProcessor: authGen cannot be nil")
		return nil, fmt.Errorf("authGen cannot be nil")
	}
	if tq == nil {
		slog.Error("NewLookupTaskProcessor: taskQueuer cannot be nil")
		return nil, fmt.Errorf("taskQueuer cannot be nil")
	}
	if subID == "" {
		slog.Error("NewLookupTaskProcessor: subID cannot be empty")
		return nil, fmt.Errorf("subID cannot be empty")
	}
	if maxProxyTasks <= 0 {
		slog.Warn("NewChannelLookupProcessor: maxProxyTasks is not positive, defaulting to no limit (effectively unlimited)", "provided_max_proxy_tasks", maxProxyTasks)
		maxProxyTasks = 0 // 0 or negative means no limit
	}

	return &channelLookupProcessor{
		registryClient: registryClient,
		authGen:        authGen,
		taskQueuer:     tq,
		maxProxyTasks:  maxProxyTasks,
		subID:          subID,
	}, nil
}

// validateTask checks if the AsyncTask is valid for processing.
func (p *channelLookupProcessor) validateTask(ctx context.Context, task *model.AsyncTask) error {
	if task == nil {
		slog.ErrorContext(ctx, "LookupTaskProcessor: async task cannot be nil")
		return errors.New("async task cannot be nil")
	}
	if task.Type != model.AsyncTaskTypeLookup {
		slog.ErrorContext(ctx, "LookupTaskProcessor: received task with incorrect type", "expected_type", model.AsyncTaskTypeLookup, "actual_type", task.Type)
		return fmt.Errorf("invalid task type for LookupTaskProcessor: expected %s, got %s", model.AsyncTaskTypeLookup, task.Type)
	}
	if len(task.Body) == 0 {
		slog.ErrorContext(ctx, "LookupTaskProcessor: task body for lookup cannot be empty")
		return errors.New("task body for lookup cannot be empty")
	}
	return nil
}

// lookup unmarshals the task body and looks up subscriptions.
func (p *channelLookupProcessor) lookup(ctx context.Context, reqCtx *model.Context) ([]model.Subscription, error) {
	lookupCriteria := &model.Subscription{
		Subscriber: model.Subscriber{
			Domain:       reqCtx.Domain,
			Type:         model.RoleBPP,
			SubscriberID: reqCtx.BppID,
			Location:     reqCtx.Location,
		}}

	slog.DebugContext(ctx, "LookupTaskProcessor: Performing lookup with criteria", "criteria", lookupCriteria)
	subscriptions, err := p.registryClient.Lookup(ctx, lookupCriteria)
	if err != nil {
		slog.ErrorContext(ctx, "LookupTaskProcessor: Failed to lookup subscribers from registry", "error", err)
		return nil, fmt.Errorf("failed to lookup subscribers: %w", err)
	}
	return subscriptions, nil
}

// enqueueProxyTasks iterates through subscriptions, prepares, and enqueues proxy tasks
// using the configured taskQueuer.
func (p *channelLookupProcessor) enqueueProxyTasks(ctx context.Context, subscriptions []model.Subscription, originalTask *model.AsyncTask) error {
	authHeader, err := p.authGen.AuthHeader(ctx, originalTask.Body, p.subID)
	if err != nil {
		slog.ErrorContext(ctx, "LookupTaskProcessor: Failed to prepare signed headers for proxy tasks", "error", err)
		return fmt.Errorf("failed to prepare signed headers for proxy tasks: %w", err)
	}

	headersForProxy := originalTask.Headers.Clone()
	headersForProxy.Set(model.AuthHeaderGateway, authHeader)

	// Randomize the order of subscriptions to distribute load, especially when maxProxyTasks is used.
	rand.Shuffle(len(subscriptions), func(i, j int) {
		subscriptions[i], subscriptions[j] = subscriptions[j], subscriptions[i]
	})

	successfulPublications := 0
	skipped := 0
	var firstError error

	for i, sub := range subscriptions {
		if sub.URL == "" {
			slog.WarnContext(ctx, "LookupTaskProcessor: Skipping subscriber due to empty URL", "subscriber_id", sub.SubscriberID)
			skipped++
			continue
		}

		// Prepare a model.Context for this specific proxy task.
		// QueueTxn will use this to determine task type (PROXY) and target.
		proxyTaskModelContext := originalTask.Context // Start with a copy from the original lookup task.
		proxyTaskModelContext.BppURI = sub.URL        // Set the target BPP URI.
		slog.DebugContext(ctx, "LookupTaskProcessor: Enqueuing new proxy task",
			"target_subscriber_id", sub.SubscriberID,
			"target_bpp_uri", proxyTaskModelContext.BppURI,
			"action_for_queue", proxyTaskModelContext.Action)

		// QueueTxn will create the AsyncTask, set its Type to PROXY, and Target based on BppURI + "/search" (or other action path)
		_, err := p.taskQueuer.QueueTxn(ctx, &proxyTaskModelContext, originalTask.Body, headersForProxy)
		if err != nil {
			errMsg := fmt.Errorf("failed to queue proxy task for subscriber %s (URL: %s): %w", sub.SubscriberID, sub.URL, err)
			slog.ErrorContext(ctx, "LookupTaskProcessor: Error enqueuing proxy task", "error", errMsg)
			if firstError == nil {
				firstError = errMsg // Capture the first error
			}
			skipped++
			continue
		}
		slog.InfoContext(ctx, "LookupTaskProcessor: Successfully queued proxy task", "subscriber_id", sub.SubscriberID, "target_bpp_uri", sub.URL)
		successfulPublications++
		if p.maxProxyTasks > 0 && successfulPublications >= p.maxProxyTasks {
			slog.InfoContext(ctx, "LookupTaskProcessor: Reached maxProxyTasks limit, stopping further proxy task creation for this lookup.", "limit", p.maxProxyTasks, "created_count", successfulPublications, "total_subscriptions_found", len(subscriptions), "subscriptions_skipped_due_to_limit", len(subscriptions)-(i+1))
			break
		}

	}
	slog.InfoContext(ctx, "LookupTaskProcessor: Finished enqueuing proxy tasks", "successful_count", successfulPublications, "skipped_or_failed", skipped)
	return firstError // Return the first error encountered, or nil if all successful
}

// Process handles the given LOOKUP asynchronous task.
// It looks up subscribers based on the task body and queues individual PROXY tasks for each.
func (p *channelLookupProcessor) Process(ctx context.Context, task *model.AsyncTask) error {
	if err := p.validateTask(ctx, task); err != nil {
		return err
	}
	slog.InfoContext(ctx, "LookupTaskProcessor: Processing lookup task", "task.context", task.Context)

	subscriptions, err := p.lookup(ctx, &task.Context)
	if err != nil {
		return err
	}

	// If no subscribers found, nothing more to do.
	if len(subscriptions) == 0 {
		slog.InfoContext(ctx, "LookupTaskProcessor: No subscribers found for the given lookup criteria")
		return nil // No error if no subscribers found, just nothing to do.
	}

	slog.InfoContext(ctx, "LookupTaskProcessor: Found subscribers, preparing to generate proxy tasks", "count", len(subscriptions))
	return p.enqueueProxyTasks(ctx, subscriptions, task)
}
