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
	"testing"
)

func TestProviderNew_Success(t *testing.T) {
	provider := &oidcProvider{}
	config := map[string]string{
		"audience": "test-audience",
	}

	middleware, err := provider.New(context.Background(), config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if middleware == nil {
		t.Fatalf("expected middleware, got nil")
	}
}

func TestProviderNew_Error(t *testing.T) {
	provider := &oidcProvider{}
	config := map[string]string{} // Missing audience

	middleware, err := provider.New(context.Background(), config)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if middleware != nil {
		t.Fatalf("expected nil middleware, got non-nil")
	}
}

func TestMain(t *testing.T) {
	main()
}
