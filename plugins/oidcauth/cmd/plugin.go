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

// Package main provides the entry point for the oidcauth plugin.
package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/dpi-accelerator-beckn-onix/plugins/oidcauth"
)

type oidcProvider struct{}

// New creates a new instance of the oidcauth plugin.
func (p oidcProvider) New(ctx context.Context, cfg map[string]string) (func(http.Handler) http.Handler, error) {
	return oidcauth.New(ctx, config(cfg))
}

func config(cfg map[string]string) *oidcauth.Config {
	c := &oidcauth.Config{
		AllowedAudience: cfg["allowed_audience"],
	}

	for _, iss := range strings.Split(cfg["allowed_issuers"], ",") {
		if iss = strings.TrimSpace(iss); iss != "" {
			c.AllowedIssuers = append(c.AllowedIssuers, iss)
		}
	}

	for _, sa := range strings.Split(cfg["allowed_sas"], ",") {
		if sa = strings.TrimSpace(sa); sa != "" {
			c.AllowedSAs = append(c.AllowedSAs, sa)
		}
	}

	return c
}

// Provider is the exported symbol used by the plugin framework.
var Provider = oidcProvider{}

func main() {}
