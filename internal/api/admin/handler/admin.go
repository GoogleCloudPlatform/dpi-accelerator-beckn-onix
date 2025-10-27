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
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/dpi-accelerator-beckn-onix/internal/repository"
	"github.com/google/dpi-accelerator-beckn-onix/internal/service"
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// adminService defines the interface for LRO operations relevant to admin actions.
type adminService interface {
	ApproveSubscription(ctx context.Context, req *model.OperationActionRequest) (*model.Subscription, *model.LRO, error)
	RejectSubscription(ctx context.Context, req *model.OperationActionRequest) (*model.LRO, error)
}

// adminHandler handles admin-specific Long-Running Operation (LRO) actions.
type adminHandler struct {
	srv adminService
}

// NewAdminHandler creates a new AdminLROHandler.
func NewAdminHandler(srv adminService) (*adminHandler, error) {
	if srv == nil {
		slog.Error("NewAdminLROHandler: AdminLROService dependency is nil.")
		return nil, errors.New("AdminLROService dependency is nil")
	}
	return &adminHandler{srv: srv}, nil
}

// writeAdminJSONError is a helper function to construct and write standardized JSON error responses for admin API.
func writeAdminJSONError(w http.ResponseWriter, statusCode int, errType model.ErrorType, errCode model.ErrorCode, errMsg string) {
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
		slog.Error("AdminLROHandler: Failed to encode error response", "error", err)
	}
}

// HandleSubscriptionAction processes APPROVE/REJECT actions for a subscription LRO.
func (h *adminHandler) HandleSubscriptionAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req model.OperationActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.ErrorContext(ctx, "AdminLROHandler: Failed to decode request body for action", "error", err)
		writeAdminJSONError(w, http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeInvalidJSON, "Invalid request body: "+err.Error())
		return
	}
	defer r.Body.Close()

	var lro *model.LRO
	var err error

	switch req.Action {
	case model.OperationActionApproveSubscription:
		slog.InfoContext(ctx, "AdminLROHandler: Approving subscription", "operation_id", req.OperationID)
		_, lro, err = h.srv.ApproveSubscription(ctx, &req)
	case model.OperationActionRejectSubscription:
		if req.Reason == "" {
			slog.WarnContext(ctx, "AdminLROHandler: Reason missing for REJECT action", "operation_id", req.OperationID)
			writeAdminJSONError(w, http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeTypeInvalidAction, "Reason is required for REJECT action.")
			return
		}
		slog.InfoContext(ctx, "AdminLROHandler: Rejecting subscription", "operation_id", req.OperationID, "reason", req.Reason)
		lro, err = h.srv.RejectSubscription(ctx, &req)
	default:
		slog.WarnContext(ctx, "AdminLROHandler: Invalid action specified", "operation_id", req.OperationID, "action", req.Action)
		writeAdminJSONError(w, http.StatusBadRequest, model.ErrorTypeValidationError, model.ErrorCodeTypeInvalidAction, "Invalid action specified. Must be 'APPROVE_SUBSCRIPTION' or 'REJECT_SUBSCRIPTION'.")
		return
	}

	if err != nil {
		slog.ErrorContext(ctx, "AdminLROHandler: Error processing subscription action", "operation_id", req.OperationID, "action", req.Action, "error", err)
		if errors.Is(err, repository.ErrOperationNotFound) {
			writeAdminJSONError(w, http.StatusNotFound, model.ErrorTypeNotFoundError, model.ErrorCodeOperationNotFound, fmt.Sprintf("Operation with id %s not found.", req.OperationID))
			return
		}
		if errors.Is(err, service.ErrLROAlreadyProcessed) {
			writeAdminJSONError(w, http.StatusConflict, model.ErrorTypeConflictError, model.ErrorCodeDuplicateRequest, fmt.Sprintf("Operation %s has already been processed.", req.OperationID))
			return
		}
		writeAdminJSONError(w, http.StatusInternalServerError, model.ErrorTypeInternalError, model.ErrorCodeInternalServerError, "Failed to process subscription action due to an internal error.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(lro); err != nil {
		slog.ErrorContext(ctx, "AdminLROHandler: Failed to encode LRO response for action", "error", err, "operation_id", lro.OperationID)
		// Client has already received 200 OK, this error is server-side logging.
	}
}
