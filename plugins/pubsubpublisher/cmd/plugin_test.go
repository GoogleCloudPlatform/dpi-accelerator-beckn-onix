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
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/pstest"
	"cloud.google.com/go/pubsub" //lint:ignore SA1019 v2 is not yet available in google3, see yaqs/2071311681450934272
	"google.golang.org/api/option"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc"
)

func setupTestServer(t *testing.T, ctx context.Context, projectID string) (*pubsub.Client, *pstest.Server) {
	srv := pstest.NewServer()
	conn, err := grpc.NewClient(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to create new gRPC client: %v", err)
	}
	client, err := pubsub.NewClient(ctx, projectID, option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("failed to create pubsub client: %v", err)
	}
	return client, srv
}

func TestProviderNewSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, srv := setupTestServer(t, ctx, "test-project")
	defer client.Close()
	defer srv.Close()

	os.Setenv("PUBSUB_EMULATOR_HOST", srv.Addr)
	defer os.Unsetenv("PUBSUB_EMULATOR_HOST")

	_, err := client.CreateTopic(ctx, "test-topic1")
	if err != nil {
		t.Fatalf("failed to setup topic1: %v", err)
	}
	_, err = client.CreateTopic(ctx, "test-topic2")
	if err != nil {
		t.Fatalf("failed to setup topic2: %v", err)
	}

	config := map[string]string{
		"project": "test-project",
		"topics":  "test-topic1, test-topic2",
	}

	pub, closer, err := Provider.New(ctx, config)
	if err != nil {
		t.Fatalf("unexpected error during publisher creation: %v", err)
	}
	defer func() {
		if err := closer(); err != nil {
			t.Logf("error closing publisher: %v", err)
		}
	}()

	msg := []byte("hello from plugin")
	err = pub.Publish(ctx, "test-topic1", msg)
	if err != nil {
		t.Fatalf("unexpected error during publish to test-topic1: %v", err)
	}
	err = pub.Publish(ctx, "test-topic2", msg)
	if err != nil {
		t.Fatalf("unexpected error during publish to test-topic2: %v", err)
	}
}

func TestProviderNewError(t *testing.T) {
	testCases := []struct {
		name            string
		config          map[string]string
		setupTopics     func(ctx context.Context, client *pubsub.Client) error
		useCancelledCtx bool
		errorContains   string
		publishTopic    string
	}{
		{
			name: "Invalid config missing topics",
			config: map[string]string{
				"project": "test-project",
			},
			errorContains: "missing topics in config",
		},
		{
			name: "Topic does not exist",
			config: map[string]string{
				"project": "test-project",
				"topics":  "nonexistent-topic",
			},
			errorContains: "topic nonexistent-topic not found",
		},
		{
			name: "Publish with cancelled context",
			config: map[string]string{
				"project": "test-project",
				"topics":  "test-topic",
			},
			setupTopics: func(ctx context.Context, client *pubsub.Client) error {
				_, err := client.CreateTopic(ctx, "test-topic")
				return err
			},
			useCancelledCtx: true,
			errorContains:   "context",
			publishTopic:    "test-topic",
		},
		{
			name: "Publish to non-existent topic in publisher instance",
			config: map[string]string{
				"project": "test-project",
				"topics":  "test-topic",
			},
			setupTopics: func(ctx context.Context, client *pubsub.Client) error {
				_, err := client.CreateTopic(ctx, "test-topic")
				return err
			},
			errorContains: "not found in publisher",
			publishTopic:  "non-existent",
		},
		{
			name: "Invalid config missing project ID",
			config: map[string]string{
				"topics": "test-topic",
			},
			errorContains: "missing project ID in config",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			client, srv := setupTestServer(t, ctx, "test-project")
			defer client.Close()
			defer srv.Close()

			os.Setenv("PUBSUB_EMULATOR_HOST", srv.Addr)
			defer os.Unsetenv("PUBSUB_EMULATOR_HOST")

			if tc.setupTopics != nil {
				if err := tc.setupTopics(ctx, client); err != nil {
					t.Fatalf("failed to setup topic: %v", err)
				}
			}

			pub, closer, err := Provider.New(ctx, tc.config)

			if err == nil {
				if !tc.useCancelledCtx && tc.publishTopic == "" {
					t.Fatalf("expected error, but publisher creation succeeded")
				}
				defer func() {
					if err := closer(); err != nil {
						t.Logf("error closing publisher: %v", err)
					}
				}()
				pubCtx := ctx
				if tc.useCancelledCtx {
					var cancelPub context.CancelFunc
					pubCtx, cancelPub = context.WithCancel(ctx)
					cancelPub()
				}
				err = pub.Publish(pubCtx, tc.publishTopic, []byte("test"))
				if err == nil {
					t.Fatalf("expected publish error, but publish succeeded")
				}
			}
			if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
				t.Fatalf("expected error containing %q, got %v", tc.errorContains, err)
			}
		})
	}
}
