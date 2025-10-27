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

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/dpi-accelerator-beckn-onix/internal/event"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// subscriberService defines the interface for subscription-related business logic.
type subscriberService interface {
	CreateSubscription(ctx context.Context, req *model.NpSubscriptionRequest) (string, error)
	UpdateSubscription(ctx context.Context, req *model.NpSubscriptionRequest) (string, error)
	UpdateStatus(ctx context.Context, opID string) (model.LROStatus, error)
	OnSubscribe(ctx context.Context, req *model.OnSubscribeRequest) (*model.OnSubscribeResponse, error)
}

// subscriberHandler handles HTTP requests for subscriber operations.
type subscriberHandler struct {
	srv subscriberService
}

// NewSubscriberHandler creates a new subscriberHandler.
func NewSubscriberHandler(srv subscriberService) (*subscriberHandler, error) {
	if srv == nil {
		slog.Error("NewSubscriberHandler: SubscriberService dependency is nil.")
		return nil, errors.New("SubscriberService dependency is nil")
	}
	return &subscriberHandler{srv: srv}, nil
}

// writeSubscriberJSONError is a helper function to construct and write standardized JSON error responses.
func writeSubscriberJSONError(w http.ResponseWriter, statusCode int, errType model.ErrorType, errCode model.ErrorCode, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	errResp := model.ErrorResponse{
		Error: model.Error{
			Type:    errType,
			Code:    errCode,
			Message: errMsg,
		},
	}
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		slog.Error("SubscriberHandler: Failed to encode error response", "error", err)
	}
}

// CreateSubscription handles POST /subscribe requests.
func (h *subscriberHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req model.NpSubscriptionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.ErrorContext(ctx, "SubscriberHandler: Failed to decode create subscription request", "error", err)
		writeSubscriberJSONError(w, http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeInvalidJSON, "Invalid request body: "+err.Error())
		return
	}
	defer r.Body.Close()

	slog.InfoContext(ctx, "SubscriberHandler: Received create subscription request")
	operationID, err := h.srv.CreateSubscription(ctx, &req)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriberHandler: Error creating subscription", "error", err)
		writeSubscriberJSONError(w, http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeBadRequest, err.Error())

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted for LRO
	if err := json.NewEncoder(w).Encode(operationID); err != nil {
		slog.ErrorContext(ctx, "SubscriberHandler: Failed to encode OperationID for create subscription", "error", err, "message_id", operationID)
	}
}

// UpdateSubscription handles PATCH /subscribe/{subscription_id} requests.
func (h *subscriberHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req model.NpSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.ErrorContext(ctx, "SubscriberHandler: Failed to decode update subscription request", "error", err)
		writeSubscriberJSONError(w, http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeInvalidJSON, "Invalid request body: "+err.Error())
		return
	}
	defer r.Body.Close()

	slog.InfoContext(ctx, "SubscriberHandler: Received update subscription request")
	lroID, err := h.srv.UpdateSubscription(ctx, &req)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriberHandler: Error updating subscription", "error", err)
		writeSubscriberJSONError(w, http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted for LRO
	if err := json.NewEncoder(w).Encode(lroID); err != nil {
		slog.ErrorContext(ctx, "SubscriberHandler: Failed to encode LRO response for update subscription", "error", err, "message_id", lroID)
	}
}

// StatusUpdate handles POST /statusUpdate requests.
func (h *subscriberHandler) StatusUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req event.OnSubscribeRecievedEvent

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.ErrorContext(ctx, "SubscriberHandler: Failed to decode status update request", "error", err)
		writeSubscriberJSONError(w, http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeInvalidJSON, "Invalid request body: "+err.Error())
		return
	}
	defer r.Body.Close()

	slog.InfoContext(ctx, "SubscriberHandler: Received status update", "message_id", req.OperationID)
	status, err := h.srv.UpdateStatus(ctx, req.OperationID)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriberHandler: Error processing status update", "message_id", req.OperationID, "error", err)
		writeSubscriberJSONError(w, http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeBadRequest, err.Error())
		return
	}
	slog.InfoContext(ctx, "SubscriberHandler: Subscription updated successfully", "message_id", req.OperationID, "status", status)
	w.WriteHeader(http.StatusOK)
}

// OnSubscribe handles POST /on_subscribe requests from the Registry to the NP.
func (h *subscriberHandler) OnSubscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req model.OnSubscribeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.ErrorContext(ctx, "SubscriberHandler: Failed to decode on_subscribe request", "error", err)
		writeSubscriberJSONError(w, http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeInvalidJSON, "Invalid request body: "+err.Error())
		return
	}
	defer r.Body.Close()

	slog.InfoContext(ctx, "SubscriberHandler: Received on_subscribe request", "message_id", req.MessageID)
	resp, err := h.srv.OnSubscribe(ctx, &req)
	if err != nil {
		slog.ErrorContext(ctx, "SubscriberHandler: Error processing on_subscribe request", "message_id", req.MessageID, "error", err)
		// Beckn spec usually expects an ACK/NACK for /on_subscribe, but here we're returning the error directly.
		// For a more compliant Beckn error, you might return a NACK with an error object.
		writeSubscriberJSONError(w, http.StatusInternalServerError, model.ErrorTypeInternalError, model.ErrorCodeInternalServerError, "Failed to process on_subscribe: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(ctx, "SubscriberHandler: Failed to encode on_subscribe response", "error", err, "message_id", req.MessageID)
	}
}
