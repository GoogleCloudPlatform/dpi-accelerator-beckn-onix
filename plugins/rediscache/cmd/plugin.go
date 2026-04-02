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

package main

import (
	"context"
	"fmt"

	"github.com/google/dpi-accelerator-beckn-onix/plugins/rediscache"

	"github.com/beckn/beckn-onix/pkg/plugin/definition"
)

type cacheProvider struct{}

func (cp cacheProvider) New(ctx context.Context, config map[string]string) (definition.Cache, func() error, error) {
	c, closeFunc, err := rediscache.New(ctx, config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create redis cache: %w", err)
	}
	return c, closeFunc, nil
}

var Provider = cacheProvider{}

func main() {}
