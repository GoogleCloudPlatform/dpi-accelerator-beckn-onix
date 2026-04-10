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

// Package main provides a decrypter plugin for the Beckn-Onix platform.
package main

import (
	"context"

	"github.com/google/dpi-accelerator-beckn-onix/plugins/decrypter"
	"github.com/beckn-one/beckn-onix/pkg/plugin/definition"
)

// decrypterProvider implements the definition.decrypterProvider interface.
type decrypterProvider struct{}

// New creates a new Decrypter instance using the provided configuration.
func (dp decrypterProvider) New(ctx context.Context, config map[string]string) (definition.Decrypter, func() error, error) {
	return decrypter.New(ctx)
}

// Provider is the exported symbol that the plugin manager will look for.
var Provider = decrypterProvider{}

func main() {}
