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

// Package agentenginetransportwrapper transforms outbound Beckn callback
// requests into the Vertex AI Agent Engine :query envelope and injects an
// OAuth2 access token (cloud-platform scope) so the request can hit
// `aiplatform.googleapis.com`. Non-callback actions are forwarded
// unmodified.
package agentenginetransportwrapper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"google.golang.org/api/impersonate"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2"
)

// Cloud-platform OAuth2 scope required by *.googleapis.com endpoints.
const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

// callbackActionPrefix marks Beckn callback actions.
// Only actions with this prefix are wrapped into the
// :query envelope and signed with an OAuth2 token.
const callbackActionPrefix = "on_"

// Package-level factory vars allow tests to substitute fakes.
var (
	defaultOAuth2TokenSource = func(ctx context.Context, scopes ...string) (oauth2.TokenSource, error) {
		return google.DefaultTokenSource(ctx, scopes...)
	}
	impersonateOAuth2TokenSource = func(ctx context.Context, sa string, scopes []string) (oauth2.TokenSource, error) {
		return impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
			TargetPrincipal: sa,
			Scopes:          scopes,
		})
	}
)

// Wrapper implements definition.TransportWrapper.
type Wrapper struct {
	serviceAccount string
	tokenSrc       oauth2.TokenSource
}

// New parses config, eagerly builds the OAuth2 token source so any auth
// misconfiguration surfaces at startup, and returns a ready Wrapper.
func New(ctx context.Context, config map[string]any) (*Wrapper, func(), error) {
	w := &Wrapper{}
	if v, ok := config["serviceAccount"]; ok {
		w.serviceAccount, ok = v.(string)
		if !ok {
			return nil, nil, fmt.Errorf("agentenginetransportwrapper: config 'serviceAccount' must be a string, got %T", v)
		}
	}

	ts, err := buildTokenSource(ctx, w.serviceAccount)
	if err != nil {
		return nil, nil, fmt.Errorf("agentenginetransportwrapper: build token source: %w", err)
	}
	w.tokenSrc = ts

	return w, nil, nil
}

// buildTokenSource constructs the OAuth2 access TokenSource (cloud-platform
// scope) backed by service-account impersonation when serviceAccount is set,
// otherwise by Application Default Credentials.
func buildTokenSource(ctx context.Context, serviceAccount string) (oauth2.TokenSource, error) {
	if serviceAccount != "" {
		return impersonateOAuth2TokenSource(ctx, serviceAccount, []string{cloudPlatformScope})
	}
	return defaultOAuth2TokenSource(ctx, cloudPlatformScope)
}

// Wrap returns a RoundTripper that transforms callback bodies, injects an
// OAuth2 access token, and forwards to base. Non-callback requests pass
// through unmodified.
func (w *Wrapper) Wrap(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &aeTransport{
		base:     base,
		tokenSrc: w.tokenSrc,
	}
}

type aeTransport struct {
	base     http.RoundTripper
	tokenSrc oauth2.TokenSource
}

func (t *aeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	originalBody, err := readAndCloseBody(req)
	if err != nil {
		return nil, fmt.Errorf("agentenginetransportwrapper: failed to read body: %w", err)
	}

	action, err := extractAction(originalBody)
	if err != nil {
		return nil, fmt.Errorf("agentenginetransportwrapper: %w", err)
	}

	if !strings.HasPrefix(action, callbackActionPrefix) {
		// Non-callback action: forward the original request unmodified.
		newReq := req.Clone(req.Context())
		setBody(newReq, originalBody)
		return t.base.RoundTrip(newReq)
	}

	wrapped, err := wrapEnvelope(action, originalBody)
	if err != nil {
		return nil, fmt.Errorf("agentenginetransportwrapper: %w", err)
	}

	// Clone so the caller's request is left untouched (audit logs depend on it).
	newReq := req.Clone(req.Context())
	setBody(newReq, wrapped)
	newReq.Header.Set("Content-Type", "application/json")

	tok, err := fetchTokenWithContext(req.Context(), t.tokenSrc)
	if err != nil {
		return nil, fmt.Errorf("agentenginetransportwrapper: mint token: %w", err)
	}
	newReq.Header.Set("Authorization", "Bearer "+tok.AccessToken)

	return t.base.RoundTrip(newReq)
}

func fetchTokenWithContext(ctx context.Context, ts oauth2.TokenSource) (*oauth2.Token, error) {
	type result struct {
		tok *oauth2.Token
		err error
	}
	ch := make(chan result, 1)
	go func() {
		tok, err := ts.Token()
		ch <- result{tok, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.tok, r.err
	}
}

func readAndCloseBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	defer req.Body.Close()
	return io.ReadAll(req.Body)
}

func setBody(req *http.Request, body []byte) {
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
}

// envelope is the body shape Vertex AI Agent Engine's :query expects.
type envelope struct {
	ClassMethod string `json:"class_method"`
	Input       input  `json:"input"`
}

type input struct {
	Request json.RawMessage `json:"request"`
}

// becknActionEnvelope is the minimal Beckn body shape needed to read
// `context.action` in a single unmarshal pass. Pointer fields let us
// distinguish missing keys from explicit nulls or wrong types.
type becknActionEnvelope struct {
	Context *struct {
		Action *string `json:"action"`
	} `json:"context"`
}

// extractAction returns context.action from a Beckn JSON body.
func extractAction(body []byte) (string, error) {
	if len(body) == 0 {
		return "", fmt.Errorf("body is empty")
	}

	var env becknActionEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			switch typeErr.Field {
			case "context":
				return "", fmt.Errorf("'context' is not a JSON object: %w", err)
			case "context.action":
				return "", fmt.Errorf("'context.action' is not a JSON string: %w", err)
			}
		}
		return "", fmt.Errorf("body is not a valid JSON object: %w", err)
	}

	if env.Context == nil {
		return "", fmt.Errorf("body is missing top-level 'context' field")
	}
	if env.Context.Action == nil {
		return "", fmt.Errorf("'context.action' field is missing")
	}
	if *env.Context.Action == "" {
		return "", fmt.Errorf("'context.action' is empty")
	}
	return *env.Context.Action, nil
}

func wrapEnvelope(action string, originalBody []byte) ([]byte, error) {
	if action == "" {
		return nil, fmt.Errorf("action is empty")
	}
	if !json.Valid(originalBody) {
		return nil, fmt.Errorf("original body is not valid JSON")
	}

	env := envelope{
		ClassMethod: action,
		Input: input{
			Request: json.RawMessage(originalBody),
		},
	}

	out, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Agent Engine envelope: %w", err)
	}
	return out, nil
}
