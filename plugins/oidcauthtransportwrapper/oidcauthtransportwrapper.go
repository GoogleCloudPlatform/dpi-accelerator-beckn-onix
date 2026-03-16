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

// Package oidcauthtransportwrapper implements an outbound authentication plugin using Google OIDC tokens.
package oidcauthtransportwrapper

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"google.golang.org/api/idtoken"
	"golang.org/x/oauth2"
)

// idtokenNewTokenSource is a package-level variable to allow mocking in tests.
var idtokenNewTokenSource = idtoken.NewTokenSource

// oidcTransport is an http.RoundTripper that injects a Google-signed OIDC token.
type oidcTransport struct {
	base             http.RoundTripper
	audienceOverride string
	mu               sync.RWMutex
	tokenSources     map[string]oauth2.TokenSource
	ctx              context.Context // Decoupled context for background refreshes.
}

func (t *oidcTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	audience := t.audienceOverride
	if audience == "" {
		scheme := req.URL.Scheme
		if scheme == "" {
			scheme = "https"
		}
		audience = fmt.Sprintf("%s://%s", scheme, req.URL.Host)
	}

	ts, err := t.getOrCreateTokenSource(audience)
	if err != nil {
		return nil, fmt.Errorf("oidc: failed to get token source for aud %s: %w", audience, err)
	}

	token, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("oidc: failed to fetch token: %w", err)
	}

	newReq := req.Clone(req.Context())
	newReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

	return t.base.RoundTrip(newReq)
}

func (t *oidcTransport) getOrCreateTokenSource(audience string) (oauth2.TokenSource, error) {
	t.mu.RLock()
	if ts, ok := t.tokenSources[audience]; ok {
		t.mu.RUnlock()
		return ts, nil
	}
	t.mu.RUnlock()

	// Slow path: initialize source outside the lock using the creation context.
	newTS, err := idtokenNewTokenSource(t.ctx, audience)
	if err != nil {
		return nil, err
	}

	// Lock again to update the map.
	t.mu.Lock()
	defer t.mu.Unlock()

	// Initialize map if it's nil (defensive measure).
	if t.tokenSources == nil {
		t.tokenSources = make(map[string]oauth2.TokenSource)
	}

	if existingTS, ok := t.tokenSources[audience]; ok {
		return existingTS, nil
	}

	// Add the newly created token source.
	t.tokenSources[audience] = newTS
	return newTS, nil
}

// OIDCWrapper wraps an existing transport with OIDC token injection.
type OIDCWrapper struct {
	audienceOverride string
	ctx              context.Context
}

// Wrap returns an http.RoundTripper that wraps the given base transport with OIDC auth.
func (w *OIDCWrapper) Wrap(base http.RoundTripper) http.RoundTripper {
	return &oidcTransport{
		base:             base,
		audienceOverride: w.audienceOverride,
		tokenSources:     make(map[string]oauth2.TokenSource),
		ctx:              w.ctx,
	}
}

// New creates a new OIDCWrapper.
func New(ctx context.Context, config map[string]any) (*OIDCWrapper, func(), error) {
	var override string
	if val, ok := config["audience_override"]; ok {
		override, ok = val.(string)
		if !ok {
			return nil, nil, fmt.Errorf("oidc: config 'audience_override' must be a string, but got %T", val)
		}
	}

	return &OIDCWrapper{
		audienceOverride: override,
		ctx:              ctx,
	}, nil, nil
}
