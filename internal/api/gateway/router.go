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

package gateway

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// gatewayHandler defines the interface for handling gateway requests.
type gatewayHandler interface {
	ServeHttp(w http.ResponseWriter, r *http.Request)
}

// NewRouter configures and returns the Chi router for the Registry service.
func NewRouter(gh gatewayHandler) *chi.Mux {
	router := chi.NewRouter()

	// Standard middleware stack
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger) // Chi's structured logger
	router.Use(middleware.Recoverer)
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// Beckn specific routes
	// Group for routes that might share common Beckn-specific middleware or prefixes
	router.Post("/search", gh.ServeHttp)
	router.Post("/on_search", gh.ServeHttp)

	return router
}
