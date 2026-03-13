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

// Package oidcauth provides a middleware for validating OIDC tokens.
package oidcauth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"google3/third_party/golang/github_com/beckn/beckn_onix/v/v1/pkg/log/log"
	"google.golang.org/api/idtoken"
)

// Config represents the configuration for the OIDC validation middleware.
type Config struct {
	Audience string
}

type contextKey struct{}

var oidcPayloadKey = contextKey{}

// idtokenValidate is a package-level variable to allow mocking in tests.
var idtokenValidate = idtoken.Validate

// FromContext returns the OIDC payload stored in the context, if any.
func FromContext(ctx context.Context) (*idtoken.Payload, bool) {
	payload, ok := ctx.Value(oidcPayloadKey).(*idtoken.Payload)
	return payload, ok
}

// New returns a middleware that processes the incoming request,
// extracts the Bearer token, and validates it using Google's OIDC implementation.
func New(ctx context.Context, cfg map[string]string) (func(http.Handler) http.Handler, error) {
	config := &Config{
		Audience: cfg["audience"],
	}

	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header is required", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]

			payload, err := idtokenValidate(r.Context(), token, config.Audience)
			if err != nil {
				log.Errorf(r.Context(), err, "invalid oidc token")
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Add the token payload to the request context
			reqCtx := context.WithValue(r.Context(), oidcPayloadKey, payload)
			r = r.WithContext(reqCtx)

			next.ServeHTTP(w, r)
		})
	}, nil
}

func validateConfig(cfg *Config) error {
	if cfg == nil {
		return errors.New("config cannot be nil")
	}

	if cfg.Audience == "" {
		return errors.New("audience is required")
	}

	return nil
}
