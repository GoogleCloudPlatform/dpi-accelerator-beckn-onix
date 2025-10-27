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
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// taskProcessor is an interface that task processors (like proxyProcessor or lookupProcessor) should implement.
// This is typically defined where your processors are, or can be defined here if it's a shared concept.
type taskProcessor interface {
	Process(ctx context.Context, task *model.AsyncTask) error
}

// channelQueueItem wraps an AsyncTask with its original request context.
type channelQueueItem struct {
	originalCtx context.Context
	task        *model.AsyncTask
}

// ChannelTaskQueue implements an in-memory task queue using Go channels and a worker goroutine.
type ChannelTaskQueue struct {
	taskChannel     chan channelQueueItem
	proxyProcessor  taskProcessor
	lookupProcessor taskProcessor
	numWorkers      int

	workerCtx    context.Context
	workerCancel context.CancelFunc
	wg           sync.WaitGroup
}

// NewChannelTaskQueue creates a new ChannelTaskQueue.
// parentCtx is the context for the worker's lifecycle.
// proxyP and lookupP are the processors for different task types.
// bufferSize determines the capacity of the task channel.
func NewChannelTaskQueue(
	numWorkers int,
	parentCtx context.Context,
	proxyP taskProcessor,
	lookupP taskProcessor,
	bufferSize int,
) (*ChannelTaskQueue, error) {
	if proxyP == nil {
		slog.Error("NewChannelTaskQueue: proxyProcessor cannot be nil")
		return nil, fmt.Errorf("proxyProcessor cannot be nil")
	}
	if numWorkers <= 0 {
		slog.Warn("NewChannelTaskQueue: numWorkers is not positive, defaulting to 1", "provided_num_workers", numWorkers)
		numWorkers = 1
	}
	if bufferSize <= 0 {
		slog.Warn("NewChannelTaskQueue: bufferSize is not positive, defaulting to 100", "provided_buffer_size", bufferSize)
		bufferSize = 100 // Default buffer size
	}

	workerCtx, workerCancel := context.WithCancel(parentCtx)

	return &ChannelTaskQueue{
		taskChannel:     make(chan channelQueueItem, bufferSize),
		proxyProcessor:  proxyP,
		lookupProcessor: lookupP,
		numWorkers:      numWorkers,
		workerCtx:       workerCtx,
		workerCancel:    workerCancel,
	}, nil
}

// SetLookupProcessor sets the lookup processor for the ChannelTaskQueue.
// This is used to break the initialization cycle.
func (ctq *ChannelTaskQueue) SetLookupProcessor(lookupP taskProcessor) {
	if lookupP == nil {
		slog.Error("ChannelTaskQueue.SetLookupProcessor: lookupProcessor cannot be nil when setting")
	}
	ctq.lookupProcessor = lookupP
}

// QueueTxn creates an AsyncTask based on the request context and body,
// then sends it to an internal channel for asynchronous processing by a worker goroutine.
// This method implements the taskQueuer interface.
func (ctq *ChannelTaskQueue) QueueTxn(ctx context.Context, reqCtx *model.Context, body []byte, h http.Header) (*model.AsyncTask, error) {
	if reqCtx == nil {
		slog.ErrorContext(ctx, "ChannelTaskQueue.QueueTxn: request context (model.Context) cannot be nil")
		return nil, fmt.Errorf("request context (model.Context) is nil")
	}

	task := &model.AsyncTask{
		Body:    body, // Store the raw body
		Headers: h.Clone(),
		Context: *reqCtx,
	}
	// Determine task type and target based on action
	switch reqCtx.Action {
	case "search":
		if reqCtx.BppURI == "" {
			task.Type = model.AsyncTaskTypeLookup
			// Target for lookup is not set here; it's determined by the LookupTaskProcessor
		} else {
			task.Type = model.AsyncTaskTypeProxy
			targetURL, err := url.Parse(reqCtx.BppURI)
			if err != nil {
				slog.ErrorContext(ctx, "ChannelTaskQueue.QueueTxn: Failed to parse BppURI for search", "error", err, "bpp_uri", reqCtx.BppURI)
				return nil, fmt.Errorf("failed to parse BppURI for search: %w", err)
			}
			task.Target = targetURL.JoinPath("search")
		}
	case "on_search":
		if reqCtx.BapURI == "" {
			slog.ErrorContext(ctx, "ChannelTaskQueue.QueueTxn: BapURI missing for on_search")
			return nil, fmt.Errorf("BapURI is required for /on_search")
		}
		task.Type = model.AsyncTaskTypeProxy
		targetURL, err := url.Parse(reqCtx.BapURI)
		if err != nil {
			slog.ErrorContext(ctx, "ChannelTaskQueue.QueueTxn: Failed to parse BapURI for on_search", "error", err, "bap_uri", reqCtx.BapURI)
			return nil, fmt.Errorf("failed to parse BapURI for on_search: %w", err)
		}
		task.Target = targetURL.JoinPath("on_search")
	default:
		slog.ErrorContext(ctx, "ChannelTaskQueue.QueueTxn: Unknown action type", "action", reqCtx.Action)
		return nil, fmt.Errorf("unknown action type: %s", reqCtx.Action)
	}

	item := channelQueueItem{
		originalCtx: ctx, // Propagate the original request's context
		task:        task,
	}
	slog.DebugContext(ctx, "Queuing task", "action", reqCtx.Action, "type", task.Type, "target", task.Target)

	select {
	case ctq.taskChannel <- item:
		slog.InfoContext(ctx, "ChannelTaskQueue.QueueTxn: Task successfully sent to channel", "action", reqCtx.Action, "type", task.Type)
		return task, nil
	case <-ctq.workerCtx.Done():
		slog.ErrorContext(ctx, "ChannelTaskQueue.QueueTxn: Worker is shutting down, cannot queue task", "action", reqCtx.Action)
		return nil, fmt.Errorf("worker is shutting down, cannot queue task")
	default:
		// This case is for a full buffered channel if we want non-blocking behavior.
		// For now, if the channel is full, it will block until space is available or workerCtx is done.
		// If non-blocking is desired with task dropping:
		// slog.WarnContext(ctx, "ChannelTaskQueue.QueueTxn: Task channel is full, dropping task", "action", reqCtx.Action)
		// return nil, fmt.Errorf("task channel is full, task dropped")

		// Blocking send (current behavior with buffered channel):
		ctq.taskChannel <- item
		slog.InfoContext(ctx, "ChannelTaskQueue.QueueTxn: Task successfully sent to channel (after block)", "action", reqCtx.Action, "type", task.Type)
		return task, nil
	}
}

// StartWorkers launches the background worker goroutines that process tasks from the channel.
func (ctq *ChannelTaskQueue) StartWorkers() {
	slog.InfoContext(ctq.workerCtx, "ChannelTaskQueue: Starting workers...", "num_workers", ctq.numWorkers)
	for i := 0; i < ctq.numWorkers; i++ {
		ctq.wg.Add(1)
		go func(workerID int) {
			defer ctq.wg.Done()
			slog.InfoContext(ctq.workerCtx, "ChannelTaskQueue Worker: Starting...", "worker_id", workerID)
			for {
				select {
				case item, ok := <-ctq.taskChannel:
					if !ok {
						slog.InfoContext(ctq.workerCtx, "ChannelTaskQueue Worker: Task channel closed, stopping.", "worker_id", workerID)
						return
					}
					// Log receipt of the task with its original context for correlation
					slog.InfoContext(item.originalCtx, "ChannelTaskQueue Worker: Received task", "worker_id", workerID, "type", item.task.Type, "target", item.task.Target)

					var err error
					// Use the worker's context for the actual processing, so it's not prematurely canceled.
					// The item.originalCtx can still be used for extracting request-scoped values if needed by the processors,
					// but the primary cancellation for the Process method should come from workerCtx.
					processingCtx := ctq.workerCtx

					switch item.task.Type {
					case model.AsyncTaskTypeProxy:
						if ctq.proxyProcessor == nil {
							slog.ErrorContext(item.originalCtx, "ChannelTaskQueue Worker: proxyProcessor is nil, cannot process PROXY task", "worker_id", workerID)
							continue
						}
						err = ctq.proxyProcessor.Process(processingCtx, item.task)
					case model.AsyncTaskTypeLookup:
						if ctq.lookupProcessor == nil {
							slog.ErrorContext(item.originalCtx, "ChannelTaskQueue Worker: lookupProcessor is nil, cannot process LOOKUP task", "worker_id", workerID)
							continue
						}
						err = ctq.lookupProcessor.Process(processingCtx, item.task)
					default:
						slog.ErrorContext(item.originalCtx, "ChannelTaskQueue Worker: Unknown task type received", "worker_id", workerID, "type", item.task.Type)
					}
					if err != nil {
						slog.ErrorContext(item.originalCtx, "ChannelTaskQueue Worker: Error processing task", "worker_id", workerID, "type", item.task.Type, "error", err)
					} else {
						slog.InfoContext(item.originalCtx, "ChannelTaskQueue Worker: Task processed successfully", "worker_id", workerID, "type", item.task.Type)
					}
				case <-ctq.workerCtx.Done():
					slog.InfoContext(ctq.workerCtx, "ChannelTaskQueue Worker: Context cancelled, stopping.", "worker_id", workerID)
					return
				}
			}
		}(i)
	}
}

// StopWorkers signals the worker goroutines to stop and waits for them to finish.
func (ctq *ChannelTaskQueue) StopWorkers() {
	slog.InfoContext(ctq.workerCtx, "ChannelTaskQueue: StopWorkers called, signaling workers to stop.")
	ctq.workerCancel() // Signal the worker to stop by cancelling its context

	// It's generally safer to close the channel *after* the worker goroutine has exited
	// or is guaranteed not to write to it. However, since the worker reads,
	// closing it here can also signal the worker if it's in a blocking read.
	// The select on workerCtx.Done() is the primary stop signal.
	// Let the worker detect context cancellation and then exit, then close channel.
	// For robust shutdown, ensure all worker loops exit before closing the channel,
	// or handle potential panics if sending to a closed channel (though QueueTxn checks workerCtx.Done()).

	// Wait for the worker to finish processing and exit its loop.
	ctq.wg.Wait()

	// Now it's safe to close the channel as the worker is no longer reading from it.
	close(ctq.taskChannel)
	slog.InfoContext(ctq.workerCtx, "ChannelTaskQueue: All workers stopped and channel closed.")
}
