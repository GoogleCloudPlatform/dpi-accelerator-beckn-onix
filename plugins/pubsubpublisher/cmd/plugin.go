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
	"strings"

	"github.com/google/dpi-accelerator-beckn-onix/plugins/pubsubpublisher"

	"github.com/beckn/beckn-onix/pkg/plugin/definition"
)

type pubsubPublisherProvider struct{}

func (p pubsubPublisherProvider) New(ctx context.Context, config map[string]string) (definition.Publisher, func() error, error) {
	projectID := config["project"]
	topicIDsString := config["topics"]

	if projectID == "" {
		return nil, nil, fmt.Errorf("missing project ID in config")
	}

	if topicIDsString == "" {
		return nil, nil, fmt.Errorf("missing topics in config")
	}

	topicIDs := strings.Split(topicIDsString, ",")
	for i, topicID := range topicIDs {
		topicIDs[i] = strings.TrimSpace(topicID)
	}

	cfg := &pubsubpublisher.Config{
		ProjectID: projectID,
		TopicIDs:  topicIDs,
	}
	return pubsubpublisher.New(ctx, cfg)
}

var Provider = pubsubPublisherProvider{}

func main() {}