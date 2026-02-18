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
	"errors"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/pubsub" //lint:ignore SA1019 v2 is not yet available in google3, see yaqs/2071311681450934272
	"google.golang.org/api/option"
)

// Config holds the Pub/Sub configuration.
type Config struct {
	ProjectID string
	TopicIDs  []string
}

// publisher is a concrete implementation of a Beck-Onix Pub/Sub publisher plugin.
type publisher struct {
	client *pubsub.Client
	topics map[string]*pubsub.Topic
	config *Config
}

var (
	ErrProjectMissing = errors.New("invalid config: missing required field 'Project'")
	ErrTopicMissing   = errors.New("invalid config: missing required field 'Topic'")
	ErrEmptyConfig    = errors.New("invalid config: empty config")
	ErrTopicNotFound  = errors.New("topic not found")
)

func validate(cfg *Config) error {
	if cfg == nil {
		return ErrEmptyConfig
	}
	if strings.TrimSpace(cfg.ProjectID) == "" {
		return ErrProjectMissing
	}
	if len(cfg.TopicIDs) == 0 {
		return ErrTopicMissing
	}
	return nil
}

// New creates a new Publisher instance. It creates a Pub/Sub client, checks that the
// requested topics exist, and returns a Publisher.
func New(ctx context.Context, cfg *Config, opts ...option.ClientOption) (*publisher, func() error, error) {
	if err := validate(cfg); err != nil {
		return nil, nil, fmt.Errorf("invalid config: %w", err)
	}

	client, err := pubsub.NewClient(ctx, cfg.ProjectID, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	topics := make(map[string]*pubsub.Topic)
	for _, topicID := range cfg.TopicIDs {
		topic := client.Topic(topicID)
		exists, err := topic.Exists(ctx)
		if err != nil || !exists {
			_ = client.Close()
			return nil, nil, fmt.Errorf("topic %s not found: %w", topicID, err)
		}
		topics[topicID] = topic
	}
	p := &publisher{
		client: client,
		topics: topics,
		config: cfg,
	}
	return p, p.close, nil
}

// Publish sends a message to Google Cloud Pub/Sub to a specific topic.
func (p *publisher) Publish(ctx context.Context, topicID string, msg []byte) error {
	topic, ok := p.topics[topicID]
	if !ok {
		return fmt.Errorf("topic %s not found in publisher", topicID)
	}

	pubsubMsg := &pubsub.Message{
		Data: msg,
	}

	result := topic.Publish(ctx, pubsubMsg)
	id, err := result.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish message to topic %s: %w", topicID, err)
	}

	log.Printf("Published message with ID: %s to topic: %s\n", id, topicID)
	return nil
}

// Close closes the underlying Pub/Sub client.
func (p *publisher) close() error {
	return p.client.Close()
}
