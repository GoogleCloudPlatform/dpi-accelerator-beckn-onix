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

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

type gatewayAuthValidator interface {
	Validate(ctx context.Context, body []byte, authHeader string) *model.AuthError
}

type taskQueuer interface {
	QueueTxn(ctx context.Context, reqCtx *model.Context, msg []byte, h http.Header) (*model.AsyncTask, error)
}

type gatewayHandler struct {
	authValidator gatewayAuthValidator
	taskQueuer    taskQueuer
}

func NewGatewayHandler(authValidator gatewayAuthValidator, taskQueuer taskQueuer) (*gatewayHandler, error) {
	if authValidator == nil {
		slog.Error("NewGatewayHandler: authValidator dependency is nil.")
		return nil, errors.New("authValidator dependency is nil")
	}
	if taskQueuer == nil {
		slog.Error("NewGatewayHandler: taskQueuer dependency is nil.")
		return nil, errors.New("taskQueuer dependency is nil")
	}
	return &gatewayHandler{authValidator: authValidator, taskQueuer: taskQueuer}, nil
}

func (h *gatewayHandler) ServeHttp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		slog.ErrorContext(ctx, "GatewayHandler: Failed to read request body", "error", err)
		writeGatewayError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to read request body.")
		return
	}
	defer r.Body.Close()

	authHeader := r.Header.Get(model.AuthHeaderSubscriber)
	if authErr := h.authValidator.Validate(ctx, bodyBytes, authHeader); authErr != nil {
		slog.ErrorContext(ctx, "GatewayHandler: Authentication failed", "error", authErr)
		writeGatewayError(w, authErr.StatusCode, string(authErr.ErrorCode), authErr.Message)
		return
	}
	slog.InfoContext(ctx, "GatewayHandler: Authentication successful")

	var txnReq model.TxnRequest
	if err := json.Unmarshal(bodyBytes, &txnReq); err != nil {
		slog.ErrorContext(ctx, "GatewayHandler: Failed to unmarshal request body", "error", err)
		writeGatewayError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body.")
		return
	}
	queuedTask, err := h.taskQueuer.QueueTxn(ctx, &txnReq.Context, bodyBytes, r.Header.Clone())
	if err != nil {
		slog.ErrorContext(ctx, "GatewayHandler: Failed to queue task via QueueTxn", "error", err)
		writeGatewayError(w, http.StatusInternalServerError, "QUEUEING_FAILED", "Failed to queue task.")
		return
	}
	slog.InfoContext(ctx, "GatewayHandler: Task queued successfully via QueueTxn", "task", queuedTask)
	response := model.TxnResponse{Message: model.Message{Ack: model.Ack{Status: model.StatusACK}}}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.ErrorContext(ctx, "GatewayHandler: Failed to write success response", "error", err)
	}
}

func writeGatewayError(w http.ResponseWriter, statusCode int, errorCode string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errResp := model.TxnResponse{
		Message: model.Message{
			Ack: model.Ack{Status: model.StatusNACK},
			Error: &model.Error{
				Code:    model.ErrorCode(errorCode),
				Message: message,
			},
		},
	}
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		slog.Error("writeGatewayError: Failed to encode/write error response", "error", err)
	}
}
