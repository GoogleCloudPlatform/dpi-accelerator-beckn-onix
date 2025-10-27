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

package model

import (
	"net/http"
	"net/url"
)

// AsyncTaskType defines the type of asynchronous task.
type AsyncTaskType string

const (
	// AsyncTaskTypeProxy indicates a task that should be proxied to a target URL.
	AsyncTaskTypeProxy AsyncTaskType = "PROXY"
	// AsyncTaskTypeLookup indicates a task that requires a lookup.
	AsyncTaskTypeLookup AsyncTaskType = "LOOKUP"
)

// AsyncTask holds the details for an asynchronous task.
type AsyncTask struct {
	Type    AsyncTaskType `json:"type"`
	Target  *url.URL      `json:"target"`
	Body    []byte        `json:"body"`
	Headers http.Header   `json:"headers"`
	Context Context       `json:"context,omitempty"`
}

// NpSubscriptionRequest models the request for subscriber service.
type NpSubscriptionRequest struct {
	Subscriber `json:",inline"`
	KeyID      string `json:"key_id"`
	MessageID  string `json:"message_id"`
}
