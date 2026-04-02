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
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"

	"cloud.google.com/go/pubsub" //lint:ignore SA1019 v2 is not yet available in google3, see yaqs/2071311681450934272
	"google.golang.org/api/option"
)

var (
	// ErrMissingTopicID occurs if the PubSub topic id is empty.
	ErrMissingTopicID = errors.New("missing pubsub topic id")

	// ErrTopicNotFound occurs if the provided pubsub topic is not found in the provided project.
	ErrTopicNotFound = errors.New("pubdub topic not found")

	// ErrMissingProjectID occurs if the project ID is empty.
	ErrMissingProjectID = errors.New("missing project ID")

	// ErrMissingConfig occurs if the config is nil.
	ErrMissingConfig = errors.New("missing config")
)

// Config describes the connection config for a list given CloudPubSub topics.
type Config struct {
	// Target pubsub topic id.
	TopicID string `yaml:"topicID"`

	// Target project to be used.
	ProjectID string `yaml:"projectID"`

	// Client Option, If provided, these will be used.
	// otherwise it will be populated with defaults.
	Opts []option.ClientOption
}

// publisher is wrapper around Cloud PubSub client.
type publisher struct {
	client *pubsub.Client
	topic  *pubsub.Topic
}

// NewPublisher creates a new Publisher.
// Usage:
//
//	  cfg :=&event.PubsubCfg{
//	                      TopicID: "my-topic-id",
//	                      ProjectID: "my-project-id",
//	                      Opts: myOptions,
//	                      }
//	                  }
//		  p, err := event.NewPublisher(ctx, cfg)
//			 if err != nil {
//			 	 return err
//			 }
//			 defer p.Close()
func NewPublisher(ctx context.Context, cfg *Config) (*publisher, func(), error) {
	slog.DebugContext(ctx, "Creating new pubsub publisher")
	if err := validate(cfg); err != nil {
		return nil, nil, fmt.Errorf("validate(%v): %w", cfg, err)
	}

	cl, tp, err := initPS(ctx, cfg.ProjectID, cfg.TopicID, cfg.Opts)
	if err != nil {
		return nil, nil, fmt.Errorf("conn(%v): %w", cfg, err)
	}
	p := &publisher{
		client: cl,
		topic:  tp,
	}
	slog.DebugContext(ctx, "Successfully initialized publisher")
	return p, func() {
		tp.Stop()
		cl.Close()
	}, nil
}

// Publish publishes the provided message to the configured topics in Cloud PubSub.
func (p *publisher) Publish(ctx context.Context, msg *pubsub.Message) (string, error) {
	res := p.topic.Publish(ctx, msg)
	return res.Get(ctx)
}

func initPS(ctx context.Context, pID string, tID string, opts []option.ClientOption) (*pubsub.Client, *pubsub.Topic, error) {
	cl, err := pubsub.NewClient(ctx, pID, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("pubsub.NewClient(%s): %w", pID, err)
	}
	tp, err := topic(ctx, cl, tID)
	if err != nil {
		return nil, nil, fmt.Errorf("topic(%s): %w", tID, err)
	}
	return cl, tp, nil
}

func topic(ctx context.Context, c *pubsub.Client, id string) (*pubsub.Topic, error) {
	tp := c.Topic(id)
	exists, err := tp.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("topic.Exists: %w", err)
	}
	if !exists {
		return nil, ErrTopicNotFound
	}
	return tp, nil
}

func validate(c *Config) error {
	if c == nil {
		return ErrMissingConfig
	}
	if strings.TrimSpace(c.ProjectID) == "" {
		return ErrMissingProjectID
	}
	if strings.TrimSpace(c.TopicID) == "" {
		return ErrMissingTopicID
	}

	return nil
}

func (p *publisher) publishMsg(ctx context.Context, tp model.EventType, data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("json.Marshal(%v): %w", data, err)
	}
	msg := &pubsub.Message{
		Attributes: map[string]string{"event_type": string(tp)},
		Data:       b,
	}
	return p.Publish(ctx, msg)
}

// PublishNewSubscriptionRequestEvent publishes a new subscription request event to PubSub.
func (p *publisher) PublishNewSubscriptionRequestEvent(ctx context.Context, req *model.SubscriptionRequest) (string, error) {
	return p.publishMsg(ctx, model.EventTypeNewSubscriptionRequest, req)
}

// PublishUpdateSubscriptionRequestEvent publishes an update subscription request event to PubSub.
func (p *publisher) PublishUpdateSubscriptionRequestEvent(ctx context.Context, req *model.SubscriptionRequest) (string, error) {
	return p.publishMsg(ctx, model.EventTypeUpdateSubscriptionRequest, req)
}

// PublishSubscriptionRequestApprovedEvent publishes a subscription request approved event to PubSub.
func (p *publisher) PublishSubscriptionRequestApprovedEvent(ctx context.Context, req *model.LRO) (string, error) {
	return p.publishMsg(ctx, model.EventTypeSubscriptionRequestApproved, req)
}

// PublishSubscriptionRequestRejectedEvent publishes a subscription request rejected event to PubSub.
func (p *publisher) PublishSubscriptionRequestRejectedEvent(ctx context.Context, req *model.LRO) (string, error) {
	return p.publishMsg(ctx, model.EventTypeSubscriptionRequestRejected, req)
}

type OnSubscribeRecievedEvent struct {
	OperationID string `json:"operation_id"`
}

func (p *publisher) PublishOnSubscribeRecievedEvent(ctx context.Context, lroID string) (string, error) {
	return p.publishMsg(ctx, model.EventTypeOnSubscribeRecieved, &OnSubscribeRecievedEvent{OperationID: lroID})
}
