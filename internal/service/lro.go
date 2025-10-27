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

package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/dpi-accelerator-beckn-onix/pkg/model"
)

// lroRepository defines the interface for database operations related to LROs.
type lroRepository interface {
	InsertOperation(ctx context.Context, lro *model.LRO) (*model.LRO, error)
	GetOperation(ctx context.Context, id string) (*model.LRO, error)
}

type lroService struct {
	repo lroRepository
}

// NewLROService creates a new lroService.
func NewLROService(repo lroRepository) (*lroService, error) {
	if repo == nil {
		slog.Error("NewLROService: LRORepository cannot be nil")
		return nil, errors.New("LRORepository cannot be nil")
	}
	return &lroService{repo: repo}, nil
}

// Create persists a new LRO record.
func (s *lroService) Create(ctx context.Context, lro *model.LRO) (*model.LRO, error) {
	slog.InfoContext(ctx, "LROService: Creating new LRO", "operation_id", lro.OperationID, "type", lro.Type)

	createdLRO, err := s.repo.InsertOperation(ctx, lro)
	if err != nil {
		slog.ErrorContext(ctx, "LROService: Failed to insert LRO into repository", "error", err, "operation_id", lro.OperationID)
		return nil, err
	}
	return createdLRO, nil
}

// Get retrieves an LRO by its ID.
func (s *lroService) Get(ctx context.Context, id string) (*model.LRO, error) {
	slog.InfoContext(ctx, "LROService: Getting LRO", "operation_id", id)
	lro, err := s.repo.GetOperation(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "LROService: Failed to get LRO from repository", "error", err, "operation_id", id)
		return nil, err
	}
	return lro, nil
}
