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

package pubsubpublisher

import (
	"context"
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
	client, err := pubsub.NewClient(ctx, projectID,
		option.WithEndpoint(srv.Addr),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("failed to create pubsub client: %v", err)
	}
	return client, srv
}

func TestPublisherSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, srv := setupTestServer(t, ctx, "test-project")
	defer client.Close()
	defer srv.Close()

	_, err := client.CreateTopic(ctx, "test-topic1")
	if err != nil {
		t.Fatalf("failed to setup topic1: %v", err)
	}
	_, err = client.CreateTopic(ctx, "test-topic2")
	if err != nil {
		t.Fatalf("failed to setup topic2: %v", err)
	}

	config := &Config{
		ProjectID: "test-project",
		TopicIDs:  []string{"test-topic1", "test-topic2"},
	}

	pub, closer, err := New(ctx, config,
		option.WithEndpoint(srv.Addr),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("unexpected error during publisher creation: %v", err)
	}
	defer func() {
		if err := closer(); err != nil {
			t.Logf("error closing publisher: %v", err)
		}
	}()

	msg := []byte("hello world")
	err = pub.Publish(ctx, "test-topic1", msg)
	if err != nil {
		t.Fatalf("unexpected error during publish to test-topic1: %v", err)
	}

	err = pub.Publish(ctx, "test-topic2", msg)
	if err != nil {
		t.Fatalf("unexpected error during publish to test-topic2: %v", err)
	}
}

func TestPublisherError(t *testing.T) {
	testCases := []struct {
		name            string
		config          *Config
		setupTopics     func(ctx context.Context, client *pubsub.Client) error
		useCancelledCtx bool
		errorContains   string
		publishTopic    string
	}{
		{
			name: "Topic does not exist",
			config: &Config{
				ProjectID: "test-project",
				TopicIDs:  []string{"nonexistent-topic"},
			},
			errorContains: "topic nonexistent-topic not found",
		},
		{
			name: "Invalid config missing topics",
			config: &Config{
				ProjectID: "test-project",
			},
			errorContains: "missing required field",
		},
		{
			name: "Publish with cancelled context",
			config: &Config{
				ProjectID: "test-project",
				TopicIDs:  []string{"test-topic"},
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
			config: &Config{
				ProjectID: "test-project",
				TopicIDs:  []string{"test-topic"},
			},
			setupTopics: func(ctx context.Context, client *pubsub.Client) error {
				_, err := client.CreateTopic(ctx, "test-topic")
				return err
			},
			errorContains: "not found in publisher",
			publishTopic:  "non-existent",
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

			if tc.setupTopics != nil {
				if err := tc.setupTopics(ctx, client); err != nil {
					t.Fatalf("failed to setup topic: %v", err)
				}
			}

			pub, closer, err := New(ctx, tc.config,
				option.WithEndpoint(srv.Addr),
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
			)

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
