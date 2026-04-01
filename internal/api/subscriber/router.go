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

package subscriber

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// subscriberHandler defines the interface for subscriber HTTP handlers.
type subscriberHandler interface {
	CreateSubscription(w http.ResponseWriter, r *http.Request)
	UpdateSubscription(w http.ResponseWriter, r *http.Request)
	StatusUpdate(w http.ResponseWriter, r *http.Request)
	OnSubscribe(w http.ResponseWriter, r *http.Request)
}

// NewRouter configures and returns the Chi router for subscriber service functionalities.
func NewRouter(sh subscriberHandler) *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.Logger)    // Log API requests
	router.Use(middleware.Recoverer) // Recover from panics
	router.Use(middleware.RequestID) // Add a request ID to the context

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok","service":"subscriber"}`)
	})

	router.Post("/subscribe", sh.CreateSubscription)
	router.Patch("/subscribe", sh.UpdateSubscription)
	router.Post("/updateStatus", sh.StatusUpdate)

	// Catch-all for POST requests to paths ending in /on_subscribe
	router.Post("/*", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/on_subscribe") {
			sh.OnSubscribe(w, r)
			return
		}
		http.NotFound(w, r)
	})
	return router
}
