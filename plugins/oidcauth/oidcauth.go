// Copyright 2026 Google LLC
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

	"github.com/beckn-one/beckn-onix/pkg/log"
	"google.golang.org/api/idtoken"
)

// Config represents the configuration for the OIDC validation middleware.
type Config struct {
	AllowedAudience string   `yaml:"allowedAudience"`
	AllowedIssuers  []string `yaml:"allowedIssuers"`
	AllowedSAs      []string `yaml:"allowedSAs"`
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
func New(ctx context.Context, config *Config) (func(http.Handler) http.Handler, error) {
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

			payload, err := validateToken(r.Context(), token, config)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
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

	if cfg.AllowedAudience == "" {
		return errors.New("allowed audience is required")
	}

	if len(cfg.AllowedIssuers) == 0 {
		return errors.New("allowed issuers are required")
	}

	return nil
}

func validateToken(ctx context.Context, token string, config *Config) (*idtoken.Payload, error) {
	payload, err := idtokenValidate(ctx, token, config.AllowedAudience)
	if err != nil {
		log.Errorf(ctx, err, "invalid oidc token")
		return nil, errors.New("Invalid token")
	}

	if isGoogleIssuer(payload.Issuer) {
		// For Google issuers, validate against the allowed SAs
		if !isSAAuthorized(ctx, payload.Claims, config.AllowedSAs) {
			log.Errorf(ctx, nil, "token failed sa validation")
			return nil, errors.New("Unauthorized: invalid sa")
		}
	} else {
		// For non-Google issuers, validate against the allowed issuer
		if !isIssuerAuthorized(ctx, payload.Issuer, config.AllowedIssuers) {
			log.Errorf(ctx, nil, "token failed issuer validation")
			return nil, errors.New("Unauthorized: invalid issuer")
		}
	}

	return payload, nil
}

func isGoogleIssuer(issuer string) bool {
	return issuer == "https://accounts.google.com" || issuer == "accounts.google.com"
}

func isSAAuthorized(ctx context.Context, claims map[string]any, allowedSAs []string) bool {
	if len(allowedSAs) == 0 {
		log.Errorf(ctx, nil, "no allowed SAs configured")
		return false
	}
	emailClaim, ok := claims["email"].(string)
	if !ok || emailClaim == "" {
		log.Errorf(ctx, nil, "no email claim found in token")
		return false
	}
	for _, allowedSA := range allowedSAs {
		if strings.EqualFold(emailClaim, allowedSA) {
			return true
		}
	}
	log.Errorf(ctx, nil, "email %q not found in allowed SAs", emailClaim)
	return false
}

func isIssuerAuthorized(ctx context.Context, tokenIssuer string, allowedIssuers []string) bool {
	if len(allowedIssuers) == 0 {
		log.Errorf(ctx, nil, "no allowed issuers configured")
		return false
	}
	for _, allowedIssuer := range allowedIssuers {
		if tokenIssuer == strings.TrimSpace(allowedIssuer) {
			return true
		}
	}
	log.Errorf(ctx, nil, "token issuer %q not found in allowed issuers", tokenIssuer)
	return false
}
