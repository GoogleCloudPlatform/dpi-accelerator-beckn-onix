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
	"io"
	"log/slog"
	"net/http"

	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"
	"github.com/google/dpi-accelerator-beckn-onix/internal/service"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// subscriptionService defines the interface for subscription-related operations.
type subscriptionService interface {
	Create(context.Context, *model.SubscriptionRequest) (*model.LRO, error)
	Update(context.Context, *model.SubscriptionRequest) (*model.LRO, error)
}

type authenticator interface {
	AuthenticatedReq(ctx context.Context, bodyBytes []byte, authHeader string) (*model.SubscriptionRequest, *model.AuthError)
}

// subscriptionHandler handles HTTP requests for the /subscribe endpoint.
type subscriptionHandler struct {
	subService subscriptionService
	// signValidator service.signValidator // Type from service package
	auth authenticator // Type from service package
}

// NewSubscriptionHandler creates a new SubscribeHandler.
func NewSubscriptionHandler(ss subscriptionService, auth authenticator) (*subscriptionHandler, error) {
	if ss == nil {
		slog.Error("NewSubscriptionHandler: subscriptionService dependency is nil.")
		return nil, errors.New("subscriptionService dependency is nil")
	}

	if auth == nil {
		slog.Error("NewSubscriptionHandler: authenticator dependency is nil.")
		return nil, errors.New("authenticator dependency is nil")
	}
	return &subscriptionHandler{subService: ss, auth: auth}, nil
}

// writeJSONError is a helper function to construct and write standardized JSON error responses.
func writeJSONError(w http.ResponseWriter, statusCode int, errType model.ErrorType, errCode model.ErrorCode, errMsg, errPath, realmForAuthHeader string) {
	w.Header().Set("Content-Type", "application/json")
	if statusCode == http.StatusUnauthorized {
		w.Header().Set(model.UnauthorizedHeaderSubscriber, service.UnauthorizedHeader(realmForAuthHeader))
	}
	errResp := model.ErrorResponse{
		Error: model.Error{
			Type:    errType,
			Code:    errCode,
			Message: errMsg,
			Path:    errPath,
		},
	}
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		slog.Error("Failed to encode error response", "error", err)
	}
}

// Create handles POST requests to the /subscribe endpoint to create a new subscription.
func (h *subscriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.InfoContext(ctx, "SubscribeHandler: Received create request", "method", r.Method, "path", r.URL.Path)
	slog.DebugContext(ctx, "SubscribeHandler: Attempting to decode create request body")

	var subReq model.SubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&subReq); err != nil {
		slog.ErrorContext(ctx, "SubscribeHandler: Failed to decode request body for create", "error", err)
		writeJSONError(w, http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeInvalidJSON, "Invalid request body: "+err.Error(), "", "")
		return
	}
	defer r.Body.Close()
	slog.DebugContext(ctx, "SubscribeHandler: Create request body decoded", "subscriber_id", subReq.SubscriberID, "message_id", subReq.MessageID)

	// Call the subscription service
	lro, err := h.subService.Create(ctx, &subReq)

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		slog.ErrorContext(ctx, "SubscribeHandler: Error from SubscriptionService during create", "error", err, "message_id", subReq.MessageID)
		if errors.Is(err, repository.ErrOperationAlreadyExists) { // Check if it's a duplicate request error
			writeJSONError(w, http.StatusConflict, model.ErrorTypeConflictError, model.ErrorCodeDuplicateRequest, "Duplicate request: An operation with this message_id already exists or is in progress.", "", "")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, model.ErrorTypeInternalError, model.ErrorCodeInternalServerError, "Failed to process subscription request.", "", "")
		return
	}
	slog.DebugContext(ctx, "SubscribeHandler: LRO created successfully for create request", "operation_id", lro.OperationID, "status", lro.Status)

	w.WriteHeader(http.StatusOK)
	response := model.SubscriptionResponse{
		Status:    model.SubscriptionStatusUnderSubscription,
		MessageID: lro.OperationID,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(ctx, "SubscribeHandler: Failed to encode subscription response for create", "error", err, "message_id", lro.OperationID)
	}
}

// Update handles PATCH requests to the /subscribe endpoint to update an existing subscription.
func (h *subscriptionHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.InfoContext(ctx, "SubscribeHandler: Received update request", "method", r.Method, "path", r.URL.Path)
	slog.DebugContext(ctx, "SubscribeHandler: Starting authenticatedReq for update")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.ErrorContext(ctx, "SubscribeHandler: Failed to read request body for update", "error", err)
		// Not using newAuthError here as this is an I/O error before auth logic.
		writeJSONError(w, http.StatusInternalServerError, model.ErrorTypeInternalError, model.ErrorCodeInternalServerError, "Failed to read request body.", "", "")
		return
	}
	r.Body.Close()

	authHeader := r.Header.Get("Authorization")
	subReq, authErr := h.auth.AuthenticatedReq(ctx, bodyBytes, authHeader)
	if authErr != nil {
		writeJSONError(w, authErr.StatusCode, authErr.ErrorType, authErr.ErrorCode, authErr.Message, "", authErr.SubscriberID)
		return
	}

	lro, err := h.subService.Update(ctx, subReq)

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		slog.ErrorContext(ctx, "SubscribeHandler: Error from SubscriptionService during update", "error", err, "message_id", subReq.MessageID)
		if errors.Is(err, repository.ErrOperationAlreadyExists) { // Check if it's a duplicate request error
			writeJSONError(w, http.StatusConflict, model.ErrorTypeConflictError, model.ErrorCodeDuplicateRequest, "Duplicate request: An operation with this message_id already exists or is in progress for update.", "", "")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, model.ErrorTypeInternalError, model.ErrorCodeInternalServerError, "Failed to process subscription update request.", "", "")

		return
	}
	slog.DebugContext(ctx, "SubscribeHandler: LRO created successfully for update request", "operation_id", lro.OperationID, "status", lro.Status)

	w.WriteHeader(http.StatusOK)
	response := model.SubscriptionResponse{
		Status:    model.SubscriptionStatusUnderSubscription,
		MessageID: lro.OperationID,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(ctx, "SubscribeHandler: Failed to encode subscription response for update", "error", err, "message_id", lro.OperationID)
	}
}
