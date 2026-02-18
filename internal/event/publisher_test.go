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

package event

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"cloud.google.com/go/pubsub/pstest"
	"cloud.google.com/go/pubsub" //lint:ignore SA1019 v2 is not yet available in google3, see yaqs/2071311681450934272
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	pb "cloud.google.com/go/pubsub/apiv1/pubsubpb"
)

const testProject = "test-project"

func setUpTestPubsub(ctx context.Context, t *testing.T, topicID string, opts ...pstest.ServerReactorOption) (*pstest.Server, []option.ClientOption, func()) {
	t.Helper()
	psSrv := pstest.NewServer(opts...)
	topic, err := psSrv.GServer.CreateTopic(ctx, &pb.Topic{Name: "projects/" + testProject + "/topics/" + topicID})
	if err != nil {
		t.Fatalf("failed to create pubsub topic: %v, err: %v", topic.GetName(), err)
	}

	conn, err := grpc.NewClient(psSrv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to create grpc client: %v, err: %v", psSrv.Addr, err)
	}

	return psSrv, []option.ClientOption{option.WithGRPCConn(conn)}, func() { conn.Close(); psSrv.Close() }
}

func TestNewPublisherSuccess(t *testing.T) {
	_, opts, cleanup := setUpTestPubsub(context.Background(), t, "test-topic")
	defer cleanup()
	cfg := &Config{TopicID: "test-topic", ProjectID: testProject, Opts: opts}

	got, close, err := NewPublisher(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewPublisher(%v) = %v, want nil", cfg, err)
	}
	if got == nil {
		t.Fatalf("NewPublisher(%v) = nil, want non-nil", cfg)
	}
	if got.client == nil || got.topic == nil {
		t.Fatalf("NewPublisher(%v) clients not initialized", cfg)
	}
	defer close()
}

func TestNewPublisherFailure(t *testing.T) {
	_, opts, cleanup := setUpTestPubsub(context.Background(), t, "test-topic")
	defer cleanup()
	tc := []struct {
		name string
		cfg  *Config
	}{
		{
			name: "invalid_config",
			cfg:  &Config{},
		},
		{
			name: "pubsub_error",
			cfg:  &Config{TopicID: "invalid-topic", ProjectID: testProject, Opts: opts},
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := NewPublisher(context.Background(), tc.cfg); err == nil {
				t.Fatalf("NewPublisher(%v) returned nil error, want non-nil", tc.cfg)
			}
		})
	}
}

func TestInitPSSuccess(t *testing.T) {
	testTopic := "test-topic"
	_, opts, cleanup := setUpTestPubsub(context.Background(), t, testTopic)
	defer cleanup()

	gotClient, gotTopic, err := initPS(context.Background(), testProject, testTopic, opts)
	if err != nil {
		t.Fatalf("initPS(%v, %v, %v) = %v, want nil", testProject, testTopic, opts, err)
	}
	if gotClient == nil || gotTopic == nil {
		t.Fatalf("initPS(%v, %v, %v) = nil, want non-nil", testProject, testTopic, opts)
	}
	if gotTopic.ID() != testTopic {
		t.Fatalf("initPS(%v, %v, %v) = %v, want %v", testProject, testTopic, opts, gotTopic.ID(), testTopic)
	}
	defer gotClient.Close()
	defer gotTopic.Stop()
}

func TestInitPSFailure(t *testing.T) {
	_, opts, cleanup := setUpTestPubsub(context.Background(), t, "test-topic")
	defer cleanup()
	tc := []struct {
		name string
		pID  string
		tID  string
		opts []option.ClientOption
	}{
		{
			name: "newClient_error",
			pID:  testProject,
			tID:  "test-topic",
		},
		{
			name: "topic_error",
			pID:  testProject,
			tID:  "invalid-topic",
			opts: opts,
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := initPS(context.Background(), tc.pID, tc.tID, tc.opts); err == nil {
				t.Fatalf("initPS(%v, %v, %v) returned nil error, want non-nil", tc.pID, tc.tID, tc.opts)
			}
		})
	}
}

func TestTopicFailure(t *testing.T) {
	testTopic := "test-topic"
	tc := []struct {
		name string
		tID  string
		opts []pstest.ServerReactorOption
	}{
		{
			name: "exists_error",
			tID:  testTopic,
			opts: []pstest.ServerReactorOption{pstest.WithErrorInjection("GetTopic", http.StatusServiceUnavailable, "error")},
		},
		{
			name: "invalid_topic",
			tID:  "invalid-topic",
		},
	}
	ctx := context.Background()

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			_, opts, cleanup := setUpTestPubsub(context.Background(), t, testTopic, tc.opts...)
			defer cleanup()
			cl, err := pubsub.NewClient(ctx, testProject, opts...)
			if err != nil {
				t.Fatalf("failed to create pubsub client: %v, err: %v", testProject, err)
			}
			defer cl.Close()
			if _, err := topic(ctx, cl, tc.tID); err == nil {
				t.Fatalf("topic(%v) returned nil error, want non-nil", tc.tID)
			}
		})
	}
}

func TestPublishSuccess(t *testing.T) {
	ctx := context.Background()
	testTopic := "test-topic"
	psSrv, opts, cleanup := setUpTestPubsub(context.Background(), t, testTopic)
	defer cleanup()
	cfg := &Config{TopicID: testTopic, ProjectID: testProject, Opts: opts}
	publisher, closer, err := NewPublisher(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewPublisher(%v) = %v, want nil", cfg, err)
	}
	defer closer()
	wantMsg := "test-msg-id"
	testMsg := &pubsub.Message{}
	psSrv.SetAutoPublishResponse(false)
	psSrv.AddPublishResponse(&pb.PublishResponse{MessageIds: []string{wantMsg}}, nil)

	gotMsg, err := publisher.Publish(ctx, testMsg)
	if err != nil {
		t.Errorf("publish(%v) = %v, want nil", testMsg, err)
	}
	if gotMsg != wantMsg {
		t.Errorf("publish(%v) = %v, want %v", testMsg, gotMsg, wantMsg)
	}
}

func TestPublishError(t *testing.T) {
	ctx := context.Background()
	testTopic := "test-topic"
	psSrv, opts, cleanup := setUpTestPubsub(context.Background(), t, testTopic)
	defer cleanup()
	cfg := &Config{TopicID: testTopic, ProjectID: testProject, Opts: opts}
	publisher, closer, err := NewPublisher(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewPublisher(%v) = %v, want nil", cfg, err)
	}
	defer closer()
	testMsg := &pubsub.Message{}
	wantErr := status.Errorf(codes.DataLoss, "obscure error")
	psSrv.SetAutoPublishResponse(false)
	psSrv.AddPublishResponse(&pb.PublishResponse{MessageIds: []string{}}, wantErr)

	_, gotErr := publisher.Publish(ctx, testMsg)
	if d := cmp.Diff(wantErr, gotErr, cmpopts.EquateErrors()); d != "" {
		t.Errorf("publish(%v) returned diff (-want +got):\n%s", testMsg, d)
	}
}

func TestValidateFailure(t *testing.T) {
	tc := []struct {
		name      string
		cfg       *Config
		wantError error
	}{
		{
			name:      "nil_config",
			wantError: ErrMissingConfig,
		},
		{
			name:      "missing_topic_id",
			cfg:       &Config{ProjectID: testProject},
			wantError: ErrMissingTopicID,
		},
		{
			name:      "missing_project_id",
			cfg:       &Config{TopicID: "missing-topic"},
			wantError: ErrMissingProjectID,
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			if err := validate(tc.cfg); err != tc.wantError {
				t.Fatalf("validate(%v) = %v, want %v", tc.cfg, err, tc.wantError)
			}
		})
	}
}

const (
	testTopic     = "test-topic"
	testTopicName = "projects/" + testProject + "/topics/" + testTopic
)

var msgCmpOpts = []cmp.Option{
	cmpopts.IgnoreUnexported(pstest.Message{}),
	cmpopts.IgnoreFields(pstest.Message{}, "PublishTime", "ID"),
}

func setUpPublisher(ctx context.Context, t *testing.T) (*publisher, *pstest.Server, func()) {
	t.Helper()
	psSrv, opts, cleanup := setUpTestPubsub(ctx, t, testTopic)
	cfg := &Config{TopicID: testTopic, ProjectID: testProject, Opts: opts}

	publisher, close, err := NewPublisher(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewPublisher(%v) = %v, want nil", cfg, err)
	}
	return publisher, psSrv, func() {
		close()
		cleanup()
	}
}

func TestPublishMsg(t *testing.T) {
	ctx := context.Background()
	testMsgID := "testMsgID"
	publisher, psSrv, cleanup := setUpPublisher(ctx, t)
	defer cleanup()
	want := testMsgID
	psSrv.AddPublishResponse(&pb.PublishResponse{MessageIds: []string{testMsgID}}, nil)
	psSrv.SetAutoPublishResponse(false)
	testData := &model.SubscriptionResponse{MessageID: "testMessageID"}
	got, err := publisher.publishMsg(ctx, model.EventTypeNewSubscriptionRequest, testData)
	if err != nil {
		t.Fatalf("publishMsg(%v, %v) returned error: %v, want nil", model.EventTypeNewSubscriptionRequest, testData, err)
	}
	if want != got {
		t.Errorf("publishMsg(%v, %v) = %v, want %v", model.EventTypeNewSubscriptionRequest, testData, got, want)
	}
}

func TestPublishNewSubscriptionRequestEvent(t *testing.T) {
	ctx := context.Background()
	publisher, psSrv, cleanup := setUpPublisher(ctx, t)
	defer cleanup()
	req := &model.SubscriptionRequest{MessageID: "testMessageID"}

	byts, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal testData: %v", err)
	}
	want := &pstest.Message{
		Attributes: map[string]string{
			"event_type": "NEW_SUBSCRIPTION_REQUEST",
		},
		Topic: testTopicName,
		Data:  byts,
	}
	if _, err := publisher.PublishNewSubscriptionRequestEvent(ctx, req); err != nil {
		t.Fatalf("PublishNewSubscriptionRequestEvent() returned an unexpected error: %v", err)
	}
	got := psSrv.Messages()[0]
	if d := cmp.Diff(want, got, msgCmpOpts...); d != "" {
		t.Errorf("PublishNewSubscriptionRequestEvent(%v) returned diff (-want +got):\n%s", req, d)
	}
}

func TestPublishUpdateSubscriptionRequestEvent(t *testing.T) {
	ctx := context.Background()
	publisher, psSrv, cleanup := setUpPublisher(ctx, t)
	defer cleanup()
	req := &model.SubscriptionRequest{MessageID: "testMessageID"}
	byts, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal testData: %v", err)
	}
	want := &pstest.Message{
		Attributes: map[string]string{
			"event_type": "UPDATE_SUBSCRIPTION_REQUEST",
		},
		Topic: testTopicName,
		Data:  byts,
	}
	if _, err := publisher.PublishUpdateSubscriptionRequestEvent(ctx, req); err != nil {
		t.Fatalf("PublishUpdateSubscriptionRequestEvent() returned an unexpected error: %v", err)
	}
	got := psSrv.Messages()[0]
	if d := cmp.Diff(want, got, msgCmpOpts...); d != "" {
		t.Errorf("PublishUpdateSubscriptionRequestEvent(%v) returned diff (-want +got):\n%s", req, d)
	}
}

func TestPublishSubscriptionRequestApprovedEvent(t *testing.T) {
	ctx := context.Background()
	publisher, psSrv, cleanup := setUpPublisher(ctx, t)
	req := &model.LRO{OperationID: "testOperationID"}
	defer cleanup()

	byts, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal testData: %v", err)
	}
	want := &pstest.Message{
		Attributes: map[string]string{
			"event_type": "SUBSCRIPTION_REQUEST_APPROVED",
		},
		Topic: testTopicName,
		Data:  byts,
	}
	if _, err := publisher.PublishSubscriptionRequestApprovedEvent(ctx, req); err != nil {
		t.Fatalf("PublishSubscriptionRequestApprovedEvent() returned an unexpected error: %v", err)
	}
	got := psSrv.Messages()[0]
	if d := cmp.Diff(want, got, msgCmpOpts...); d != "" {
		t.Errorf("PublishSubscriptionRequestApprovedEvent(%v) returned diff (-want +got):\n%s", req, d)
	}
}

func TestPublishSubscriptionRequestRejectedEvent(t *testing.T) {
	ctx := context.Background()
	publisher, psSrv, cleanup := setUpPublisher(ctx, t)
	req := &model.LRO{OperationID: "testOperationID"}
	defer cleanup()

	byts, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal testData: %v", err)
	}
	want := &pstest.Message{
		Attributes: map[string]string{
			"event_type": "SUBSCRIPTION_REQUEST_REJECTED",
		},
		Topic: testTopicName,
		Data:  byts,
	}
	if _, err := publisher.PublishSubscriptionRequestRejectedEvent(ctx, req); err != nil {
		t.Fatalf("PublishSubscriptionRequestRejectedEvent() returned an unexpected error: %v", err)
	}
	got := psSrv.Messages()[0]
	if d := cmp.Diff(want, got, msgCmpOpts...); d != "" {
		t.Errorf("PublishSubscriptionRequestRejectedEvent(%v) returned diff (-want +got):\n%s", req, d)
	}
}

func TestPublishOnSubscribeRecievedEvent(t *testing.T) {
	ctx := context.Background()
	publisher, psSrv, cleanup := setUpPublisher(ctx, t)
	defer cleanup()
	lroID := "test-lro-id"
	eventData := &OnSubscribeRecievedEvent{OperationID: lroID}

	byts, err := json.Marshal(eventData)
	if err != nil {
		t.Fatalf("failed to marshal testData: %v", err)
	}
	want := &pstest.Message{
		Attributes: map[string]string{
			"event_type": "ON_SUBSCRIBE_RECIEVED",
		},
		Topic: testTopicName,
		Data:  byts,
	}
	if _, err := publisher.PublishOnSubscribeRecievedEvent(ctx, lroID); err != nil {
		t.Fatalf("PublishOnSubscribeRecievedEvent() returned an unexpected error: %v", err)
	}
	if len(psSrv.Messages()) == 0 {
		t.Fatal("PublishOnSubscribeRecievedEvent did not publish a message")
	}
	got := psSrv.Messages()[0]
	if d := cmp.Diff(want, got, msgCmpOpts...); d != "" {
		t.Errorf("PublishOnSubscribeRecievedEvent(%v) returned diff (-want +got):\n%s", lroID, d)
	}
}
