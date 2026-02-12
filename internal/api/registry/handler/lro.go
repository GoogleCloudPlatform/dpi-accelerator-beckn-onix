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
	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
	"github.com/go-chi/chi/v5"
)

type lroService interface {
	Get(ctx context.Context, id string) (*model.LRO, error)
}

// LROHandler handles Long-Running Operation (LRO) status requests.
type LROHandler struct {
	srv lroService
}

// NewLROHandler creates a new LROHandler.
func NewLROHandler(srv lroService) (*LROHandler, error) {
	if srv == nil {
		slog.Error("NewLROHandler: lroService dependency is nil.")
		return nil, errors.New("lroService dependency is nil")
	}
	return &LROHandler{srv: srv}, nil
}

// Get retrieves the status of a Long-Running Operation.
func (h *LROHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	operationID := chi.URLParam(r, "operation_id")
	lro, err := h.srv.Get(ctx, operationID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get LRO from service", "operation_id", operationID, "error", err)
		if errors.Is(err, repository.ErrOperationNotFound) {
			writeJSONError(w, http.StatusNotFound, model.ErrorTypeNotFoundError,
				model.ErrorCodeOperationNotFound, fmt.Sprintf("Operation with id %s not found.", operationID), "", "")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, model.ErrorTypeInternalError, model.ErrorCodeInternalServerError,
			"Failed to retrieve operation status due to an internal error.", "", "")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(lro); err != nil {
		slog.ErrorContext(ctx, "LROHandler: Failed to encode LRO response for get", "error", err, "operation_id", lro.OperationID)
	}
}
