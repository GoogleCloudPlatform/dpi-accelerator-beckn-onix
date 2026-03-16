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

// The oidcauth_plugin command provides the entry point for the oidcauth plugin.
package main

import (
	"context"

	"github.com/google/dpi-accelerator-beckn-onix/plugins/oidcauthtransportwrapper"
	"github.com/beckn-one/beckn-onix/pkg/plugin/definition"
)

type oidcProvider struct{}

// New creates a new instance of the oidcauthtransportwrapper plugin.
func (p *oidcProvider) New(ctx context.Context, config map[string]any) (definition.TransportWrapper, func(), error) {
	return oidcauthtransportwrapper.New(ctx, config)
}

// Provider is the exported symbol for the oidcauthtransportwrapper plugin.
var Provider = oidcProvider{}

func main() {}
