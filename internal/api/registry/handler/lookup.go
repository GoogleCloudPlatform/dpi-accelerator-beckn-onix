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
	"log/slog"
	"net/http"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

type lookupService interface {
	Lookup(context.Context, *model.Subscription) ([]model.Subscription, error)
}

// lookupHandler handles lookup requests.
type lookupHandler struct {
	lhService lookupService
}

// NewLookupHandler creates a new LookupHandler.
func NewLookupHandler(svc lookupService) *lookupHandler {
	return &lookupHandler{lhService: svc}
}

// Lookup handles the HTTP POST request for subscriber lookup.
// It unmarshals the request body, calls the service layer, and returns JSON response.
func (h *lookupHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	slog.Info("Handler: Received lookup request", "method", r.Method, "path", r.URL.Path)

	var lookupReq model.Subscription

	if err := json.NewDecoder(r.Body).Decode(&lookupReq); err != nil {
		slog.Error("Handler: Failed to unmarshal request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	subscriptions, err := h.lhService.Lookup(r.Context(), &lookupReq)
	if err != nil {
		slog.Error("Handler: Failed to perform lookup", "error", err, "request", lookupReq)
		http.Error(w, "Failed to lookup subscriptions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(subscriptions); err != nil {
		slog.Error("Handler: Failed to encode lookup response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}

	slog.Info("Handler: Lookup request processed successfully", "count", len(subscriptions))
}
